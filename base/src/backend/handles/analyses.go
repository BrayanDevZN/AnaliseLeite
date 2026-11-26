package handles

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/schema"
)

// multipartOverhead e o espaco a mais, alem do arquivo, que o corpo multipart pode ter (campos e cabecalhos).
const multipartOverhead = 1 << 20

// ownedAnalysis busca a analise do {id} da rota, se o usuario pode ver a fazenda dela.
func (h *Handles) ownedAnalysis(r *http.Request, p *Principal) (*operations.Analysis, error) {
	id, err := pathID(r)

	if err != nil {
		return nil, err
	}

	analysis, err := h.db.Analyses.GetByID(r.Context(), id)

	if err != nil {
		return nil, err
	}

	if _, err := h.farmFor(r, p, analysis.FarmID); err != nil {
		return nil, err
	}

	return analysis, nil
}

// readAnalysisRequest le o corpo de POST /farms/{id}/analyses: JSON (amostra) ou multipart (arquivo).
// No multipart o arquivo vem no campo "file", e tank_id e model sao campos do form.
func readAnalysisRequest(w http.ResponseWriter, r *http.Request) (*schema.CreateAnalysisRequest, error) {
	var request schema.CreateAnalysisRequest

	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))

	if mediaType != "multipart/form-data" {
		if err := decodeJSON(w, r, &request); err != nil {
			return nil, err
		}

		return &request, nil
	}

	r.Body = http.MaxBytesReader(w, r.Body, schema.MaxFileSize+multipartOverhead)

	if err := r.ParseMultipartForm(multipartOverhead); err != nil {
		return nil, bodyError("body", "is not a valid upload or is too big")
	}

	defer r.MultipartForm.RemoveAll()

	request.Model = r.FormValue("model")

	if raw := r.FormValue("tank_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)

		if err != nil {
			return nil, bodyError("tank_id", "must be a number")
		}

		request.TankID = &id
	}

	if r.MultipartForm.Value["sample"] != nil {
		return nil, bodyError("sample", "is not accepted in an upload, send only the file")
	}

	file, header, err := r.FormFile("file")

	if err != nil {
		return &request, nil
	}

	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, schema.MaxFileSize+1))

	if err != nil {
		return nil, bodyError("file", "could not be read")
	}

	request.File = &schema.FileUpload{Filename: header.Filename, Data: data}

	return &request, nil
}

// createAnalysis: POST /farms/{id}/analyses.
// Valida, roda o modelo e so depois grava: se o modelo falhar, nada fica no banco.
// Com amostra, so contents.content e preenchido; com arquivo, so contents.file.
func (h *Handles) createAnalysis(w http.ResponseWriter, r *http.Request, p *Principal) {
	farm, err := h.ownedFarm(r, p)

	if err != nil {
		h.fail(w, err)

		return
	}

	request, err := readAnalysisRequest(w, r)

	if err != nil {
		h.fail(w, err)

		return
	}

	analysis, err := request.ToAnalysis(farm.ID)

	if err != nil {
		h.fail(w, err)

		return
	}

	if analysis.TankID != nil {
		tank, err := h.db.Tanks.GetByID(r.Context(), *analysis.TankID)

		if errors.Is(err, operations.ErrNotFound) || (err == nil && tank.FarmID != farm.ID) {
			h.fail(w, bodyError("tank_id", "is not a tank of this farm"))

			return
		}

		if err != nil {
			h.fail(w, err)

			return
		}
	}

	if h.requests == nil {
		h.fail(w, errNoModel)

		return
	}

	response := schema.AnalysisResponse{}

	if request.File != nil {
		response.Rows, err = h.requests.PredictFile(r.Context(), request.File.Filename, bytes.NewReader(request.File.Data), request.Model)
	} else {
		response.Result, err = h.requests.Predict(r.Context(), request.Sample.ToModelInput(), request.Model)
	}

	if err != nil {
		h.fail(w, h.modelError(err))

		return
	}

	// guarda o resultado do modelo junto com a analise (para a lista e o dashboard)
	analysis.Model = request.Model

	if request.File != nil {
		analysis.Kind = operations.KindFile
		analysis.Approved, analysis.Rejected = countPredictions(response.Rows)
	} else {
		analysis.Kind = operations.KindSample

		switch strings.ToLower(response.Result) {
		case "aprovado":
			analysis.Approved = 1

		case "reprovado":
			analysis.Rejected = 1
		}
	}

	if err := h.db.Analyses.Create(r.Context(), analysis); err != nil {
		h.fail(w, err)

		return
	}

	content, err := request.ToContent(analysis.ID)

	if err == nil {
		err = h.db.Contents.Create(r.Context(), content)
	}

	if err != nil {
		// sem o conteudo a analise nao serve; desfaz para nao sobrar uma analise vazia
		if deleteErr := h.db.Analyses.Delete(r.Context(), analysis.ID); deleteErr != nil {
			h.logger.Error("Could not undo analysis after a failure: %v", deleteErr)
		}

		h.fail(w, err)

		return
	}

	created := schema.AnalysisFromModel(analysis)
	contentResponse := schema.ContentFromModel(content)

	created.Content = &contentResponse
	created.Result = response.Result
	created.Rows = response.Rows

	writeJSON(w, http.StatusCreated, created)
}

// countPredictions conta quantas linhas o modelo aprovou e reprovou (a coluna "predict").
func countPredictions(rows []map[string]any) (approved, rejected int) {
	for _, row := range rows {
		switch strings.ToLower(fmt.Sprint(row["predict"])) {
		case "aprovado":
			approved++

		case "reprovado":
			rejected++
		}
	}

	return approved, rejected
}

// listAnalyses: GET /farms/{id}/analyses.
func (h *Handles) listAnalyses(w http.ResponseWriter, r *http.Request, p *Principal) {
	farm, err := h.ownedFarm(r, p)

	if err != nil {
		h.fail(w, err)

		return
	}

	analyses, err := h.db.Analyses.ListByFarm(r.Context(), farm.ID)

	if err != nil {
		h.fail(w, err)

		return
	}

	writeJSON(w, http.StatusOK, schema.AnalysesFromModel(analyses))
}

// getAnalysis: GET /analyses/{id}.
func (h *Handles) getAnalysis(w http.ResponseWriter, r *http.Request, p *Principal) {
	analysis, err := h.ownedAnalysis(r, p)

	if err != nil {
		h.fail(w, err)

		return
	}

	writeJSON(w, http.StatusOK, schema.AnalysisFromModel(analysis))
}

// listContents: GET /analyses/{id}/contents.
func (h *Handles) listContents(w http.ResponseWriter, r *http.Request, p *Principal) {
	analysis, err := h.ownedAnalysis(r, p)

	if err != nil {
		h.fail(w, err)

		return
	}

	contents, err := h.db.Contents.ListByAnalysis(r.Context(), analysis.ID)

	if err != nil {
		h.fail(w, err)

		return
	}

	writeJSON(w, http.StatusOK, schema.ContentsFromModel(contents))
}

// deleteAnalysis: DELETE /analyses/{id}. Apaga tambem os conteudos dela.
func (h *Handles) deleteAnalysis(w http.ResponseWriter, r *http.Request, p *Principal) {
	analysis, err := h.ownedAnalysis(r, p)

	if err != nil {
		h.fail(w, err)

		return
	}

	contents, err := h.db.Contents.ListByAnalysis(r.Context(), analysis.ID)

	if err != nil {
		h.fail(w, err)

		return
	}

	for i := range contents {
		if err := h.db.Contents.Delete(r.Context(), contents[i].ID); err != nil {
			h.fail(w, err)

			return
		}
	}

	if err := h.db.Analyses.Delete(r.Context(), analysis.ID); err != nil {
		h.fail(w, err)

		return
	}

	h.rows.drop(analysis.ID)

	w.WriteHeader(http.StatusNoContent)
}

// rowsCacheSize e rowsCacheTTL limitam o cache das respostas do modelo para arquivos ja analisados.
const (
	rowsCacheSize = 6
	rowsCacheTTL  = 15 * time.Minute
)

type cachedRows struct {
	rows []map[string]any
	at   time.Time
}

// rowsCache guarda por pouco tempo a resposta do modelo para um arquivo, para abrir de novo a mesma analise
// (ou baixar o relatorio logo depois) nao rodar o modelo outra vez.
type rowsCache struct {
	mu    sync.Mutex
	items map[int64]cachedRows
}

func (c *rowsCache) get(id int64) ([]map[string]any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, found := c.items[id]

	if !found || time.Since(item.at) > rowsCacheTTL {
		delete(c.items, id)

		return nil, false
	}

	return item.rows, true
}

func (c *rowsCache) put(id int64, rows []map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.items == nil {
		c.items = make(map[int64]cachedRows)
	}

	// passou do limite: tira o mais antigo
	if len(c.items) >= rowsCacheSize {
		var oldest int64
		var oldestAt time.Time

		for key, item := range c.items {
			if oldestAt.IsZero() || item.at.Before(oldestAt) {
				oldest, oldestAt = key, item.at
			}
		}

		delete(c.items, oldest)
	}

	c.items[id] = cachedRows{rows: rows, at: time.Now()}
}

func (c *rowsCache) drop(id int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, id)
}

// uploadName da o nome que o pipeline espera para o arquivo guardado: o nome original nao fica no banco,
// entao ele e deduzido do conteudo (todo parquet comeca com "PAR1").
func uploadName(data []byte) string {
	if bytes.HasPrefix(data, []byte("PAR1")) {
		return "upload.parquet"
	}

	return "upload.csv"
}

// analysisRows: GET /analyses/{id}/rows. Roda o modelo de novo no arquivo guardado da analise e devolve uma linha por
// amostra (com a coluna "predict"), como na resposta de quando a analise foi feita. So para analises de arquivo.
func (h *Handles) analysisRows(w http.ResponseWriter, r *http.Request, p *Principal) {
	analysis, err := h.ownedAnalysis(r, p)

	if err != nil {
		h.fail(w, err)

		return
	}

	if analysis.Kind != operations.KindFile {
		h.fail(w, errNotFound)

		return
	}

	if rows, found := h.rows.get(analysis.ID); found {
		writeJSON(w, http.StatusOK, rows)

		return
	}

	if h.requests == nil {
		h.fail(w, errNoModel)

		return
	}

	contents, err := h.db.Contents.ListByAnalysis(r.Context(), analysis.ID)

	if err != nil {
		h.fail(w, err)

		return
	}

	for i := range contents {
		if contents[i].File == nil {
			continue
		}

		rows, err := h.requests.PredictFile(r.Context(), uploadName(contents[i].File), bytes.NewReader(contents[i].File), analysis.Model)

		if err != nil {
			h.fail(w, h.modelError(err))

			return
		}

		h.rows.put(analysis.ID, rows)
		writeJSON(w, http.StatusOK, rows)

		return
	}

	h.fail(w, errNotFound)
}
