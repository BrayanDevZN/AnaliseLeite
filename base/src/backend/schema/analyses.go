package schema

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
)

// MaxFileSize e o maior arquivo aceito numa analise (em bytes).
const MaxFileSize = 20 << 20

// dateLayouts sao os formatos de data aceitos.
var dateLayouts = []string{"2006-01-02", "2006-01-02 15:04:05", time.RFC3339}

// Sample sao os dados de uma amostra de leite para o modelo. Os nomes dos campos sao os do pipeline.
// E o que fica guardado em contents.content.
type Sample struct {
	Temperature *float64 `json:"temperature,omitempty"`
	State       *string  `json:"state,omitempty"`
	Date        *string  `json:"data,omitempty"`
	CCS         *float64 `json:"ccs,omitempty"`
	CPP         *float64 `json:"cpp,omitempty"`
	City        *string  `json:"cidade,omitempty"`
	Tank        *string  `json:"tanque,omitempty"`
}

// Validate valida so os campos enviados (quais sao obrigatorios depende do modelo, e o pipeline confere isso).
func (s *Sample) Validate() error {
	var v validator

	if s.Temperature != nil {
		v.number("sample.temperature", *s.Temperature, -20, 100)
	}

	if s.CCS != nil {
		v.number("sample.ccs", *s.CCS, 0, 1e9)
	}

	if s.CPP != nil {
		v.number("sample.cpp", *s.CPP, 0, 1e9)
	}

	if s.State != nil {
		upper := strings.ToUpper(strings.TrimSpace(*s.State))

		s.State = &upper

		v.state("sample.state", upper)
	}

	if s.City != nil {
		trimmed := strings.TrimSpace(*s.City)

		s.City = &trimmed

		v.text("sample.cidade", trimmed, maxTextLength)
	}

	if s.Tank != nil {
		trimmed := strings.TrimSpace(*s.Tank)

		s.Tank = &trimmed

		v.text("sample.tanque", trimmed, maxTextLength)
	}

	if s.Date != nil {
		trimmed := strings.TrimSpace(*s.Date)

		s.Date = &trimmed

		if !validDate(trimmed) {
			v.add("sample.data", "must be a date like 2025-09-01")
		}
	}

	if *s == (Sample{}) {
		v.add("sample", "must have at least one field")
	}

	return v.err()
}

// ToModelInput converte a amostra para o formato que o cliente do pipeline envia.
func (s *Sample) ToModelInput() backend.ModelInput {
	return backend.ModelInput{
		Temperature: s.Temperature,
		State:       s.State,
		Data:        s.Date,
		CCS:         s.CCS,
		CPP:         s.CPP,
		City:        s.City,
		Tank:        s.Tank,
	}
}

func validDate(value string) bool {
	for _, layout := range dateLayouts {
		if _, err := time.Parse(layout, value); err == nil {
			return true
		}
	}

	return false
}

// FileUpload e o arquivo de uma analise (csv separado por ; ou parquet). Nao vem do JSON:
// o handler preenche a partir do multipart.
type FileUpload struct {
	Filename string `json:"-"`
	Data     []byte `json:"-"`
}

// Validate confere a extensao e o tamanho do arquivo.
func (f *FileUpload) Validate() error {
	var v validator

	if !strings.HasSuffix(f.Filename, ".csv") && !strings.HasSuffix(f.Filename, ".parquet") {
		v.add("file", "must be a .csv or .parquet file")
	}

	switch {
	case len(f.Data) == 0:
		v.add("file", "is empty")

	case len(f.Data) > MaxFileSize:
		v.add("file", "is too big")
	}

	return v.err()
}

// ValidateImage confere o tamanho e se o conteudo e mesmo uma imagem (pelo cabecalho do arquivo, nao pela extensao).
func (f *FileUpload) ValidateImage() error {
	var v validator

	switch {
	case len(f.Data) == 0:
		v.add("image", "is empty")

	case len(f.Data) > MaxFileSize:
		v.add("image", "is too big")

	case !strings.HasPrefix(http.DetectContentType(f.Data), "image/"):
		v.add("image", "must be an image (jpg, png or webp)")
	}

	return v.err()
}

// CreateAnalysisRequest e o corpo de POST /farms/{id}/analyses.
// Ou vai JSON com Sample, ou vai multipart com o arquivo (campo "file") ou com a imagem (campo "image"); tank_id e
// model viram campos do form. Exatamente um dos tres: Sample vai para contents.content, e o arquivo e a imagem para
// contents.file.
type CreateAnalysisRequest struct {
	TankID *int64 `json:"tank_id,omitempty"`

	// Model e o nome do modelo (vazio usa o padrao do pipeline).
	Model string `json:"model,omitempty"`

	Sample *Sample     `json:"sample,omitempty"`
	File   *FileUpload `json:"-"`
	Image  *FileUpload `json:"-"`
}

// Validate valida os campos e confere que so um entre Sample e File foi enviado.
func (r *CreateAnalysisRequest) Validate() error {
	var v validator

	r.Model = strings.TrimSpace(r.Model)

	if r.Model != "" && !modelName.MatchString(r.Model) {
		v.add("model", "is not a valid model name")
	}

	if r.TankID != nil && *r.TankID <= 0 {
		v.add("tank_id", "must be greater than zero")
	}

	sent := 0

	for _, present := range []bool{r.Sample != nil, r.File != nil, r.Image != nil} {
		if present {
			sent++
		}
	}

	switch {
	case sent > 1:
		v.add("body", "send only one of sample, file or image")

	case sent == 0:
		v.add("body", "send a sample, a file or an image")

	case r.Sample != nil:
		mergeErrors(&v, r.Sample.Validate())

	case r.File != nil:
		mergeErrors(&v, r.File.Validate())

	default:
		mergeErrors(&v, r.Image.ValidateImage())
	}

	return v.err()
}

// ToAnalysis valida e monta a analise do banco, da fazenda dela.
func (r *CreateAnalysisRequest) ToAnalysis(farmID int64) (*operations.Analysis, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	return &operations.Analysis{FarmID: farmID, TankID: r.TankID}, nil
}

// ToContent valida e monta o conteudo do banco: com amostra, so content; com arquivo ou imagem, so file.
func (r *CreateAnalysisRequest) ToContent(analysisID int64) (*operations.Content, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	content := &operations.Content{AnalysisID: analysisID}

	if r.File != nil {
		content.File = r.File.Data

		return content, nil
	}

	if r.Image != nil {
		content.File = r.Image.Data

		return content, nil
	}

	data, err := json.Marshal(r.Sample)

	if err != nil {
		return nil, err
	}

	content.Content = data

	return content, nil
}

// mergeErrors poe os erros de um validador interno no validador de fora.
func mergeErrors(v *validator, err error) {
	if validation, ok := AsValidationError(err); ok {
		v.errors = append(v.errors, validation.Errors...)
	}
}

// ContentResponse e o conteudo de uma analise nas respostas. O arquivo nao vai no JSON, so o tamanho dele.
type ContentResponse struct {
	ID         int64           `json:"id"`
	AnalysisID int64           `json:"analysis_id"`
	Kind       string          `json:"kind"`
	Content    json.RawMessage `json:"content,omitempty"`
	FileSize   int             `json:"file_size,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// ContentFromModel converte o conteudo do banco para a resposta ("content" ou "file").
func ContentFromModel(content *operations.Content) ContentResponse {
	response := ContentResponse{
		ID:         content.ID,
		AnalysisID: content.AnalysisID,
		Kind:       "content",
		CreatedAt:  content.CreatedAt,
	}

	if content.File != nil {
		response.Kind = "file"
		response.FileSize = len(content.File)
	} else {
		response.Content = content.Content
	}

	return response
}

// ContentsFromModel converte uma lista de conteudos do banco para a resposta (nunca nil).
func ContentsFromModel(contents []operations.Content) []ContentResponse {
	responses := make([]ContentResponse, 0, len(contents))

	for i := range contents {
		responses = append(responses, ContentFromModel(&contents[i]))
	}

	return responses
}

// AnalysisResponse e uma analise nas respostas.
// Result (uma amostra) ou Rows (um arquivo) trazem a resposta do modelo e so aparecem logo ao criar;
// o banco nao guarda o resultado.
type AnalysisResponse struct {
	ID        int64     `json:"id"`
	FarmID    int64     `json:"farm_id"`
	FarmName  string    `json:"farm_name,omitempty"`
	TankID    *int64    `json:"tank_id"`
	CreatedAt time.Time `json:"created_at"`

	// Kind e "sample" (uma amostra), "file" (um arquivo) ou "image" (uma imagem). Model e o modelo usado ("" e o padrao).
	Kind  string `json:"kind"`
	Model string `json:"model"`

	// Approved e Rejected sao quantas amostras o modelo aprovou e reprovou. Os dois 0 quer dizer que o resultado
	// nao foi guardado (analises antigas).
	Approved int `json:"approved"`
	Rejected int `json:"rejected"`

	Content *ContentResponse `json:"content,omitempty"`

	// Result ("aprovado" ou "reprovado") vem nas analises de uma amostra (e nas de imagem com uma amostra). Rows (uma resposta por linha do
	// arquivo) so vem logo ao criar: o banco guarda so as contagens.
	Result string           `json:"result,omitempty"`
	Rows   []map[string]any `json:"rows,omitempty"`

	// Text e a analise em texto que o agente de imagem devolve quando a imagem e sobre leite mas nao tem numeros.
	// Como Result e Rows, so vem logo ao criar.
	Text string `json:"text,omitempty"`
}

// AnalysisFromModel converte a analise do banco para a resposta.
func AnalysisFromModel(analysis *operations.Analysis) AnalysisResponse {
	response := AnalysisResponse{
		ID:        analysis.ID,
		FarmID:    analysis.FarmID,
		TankID:    analysis.TankID,
		CreatedAt: analysis.CreatedAt,
		Kind:      analysis.Kind,
		Model:     analysis.Model,
		Approved:  analysis.Approved,
		Rejected:  analysis.Rejected,
	}

	if response.Kind == "" {
		response.Kind = operations.KindSample
	}

	// uma amostra (ou uma imagem que trouxe so uma) tem um unico resultado
	if response.Kind == operations.KindSample || (response.Kind == operations.KindImage && analysis.Approved+analysis.Rejected == 1) {
		switch {
		case analysis.Approved == 1 && analysis.Rejected == 0:
			response.Result = "aprovado"

		case analysis.Rejected == 1 && analysis.Approved == 0:
			response.Result = "reprovado"
		}
	}

	return response
}

// UserAnalysisFromModel converte uma analise com o nome da fazenda dela.
func UserAnalysisFromModel(item *operations.UserAnalysis) AnalysisResponse {
	response := AnalysisFromModel(&item.Analysis)

	response.FarmName = item.FarmName

	return response
}

// AnalysisListResponse e uma pagina de analises, com o total para paginar.
type AnalysisListResponse struct {
	Items []AnalysisResponse `json:"items"`
	Total int                `json:"total"`
}

// DayStats e o resumo de um dia (no fuso de quem pediu).
type DayStats struct {
	// Date e o dia no formato 2025-09-01.
	Date     string `json:"date"`
	Analyses int    `json:"analyses"`
	Approved int    `json:"approved"`
	Rejected int    `json:"rejected"`
}

// StatsResponse e o resumo do usuario para o dashboard (a pagina de inicio).
type StatsResponse struct {
	Farms    int `json:"farms"`
	Tanks    int `json:"tanks"`
	Analyses int `json:"analyses"`

	// Approved e Rejected somam as amostras (um arquivo conta uma por linha).
	Approved int `json:"approved"`
	Rejected int `json:"rejected"`

	// Days sao os ultimos 7 dias, do mais antigo para hoje.
	Days []DayStats `json:"days"`
}

// AnalysesFromModel converte uma lista de analises do banco para a resposta (nunca nil).
func AnalysesFromModel(analyses []operations.Analysis) []AnalysisResponse {
	responses := make([]AnalysisResponse, 0, len(analyses))

	for i := range analyses {
		responses = append(responses, AnalysisFromModel(&analyses[i]))
	}

	return responses
}

// AnalysisRoutes sao as rotas de analises. A de criar recebe JSON (amostra) ou multipart (arquivo ou imagem).
var AnalysisRoutes = []Route{
	{Method: http.MethodPost, Path: "/farms/{id}/analyses", Description: "cria uma analise e roda o modelo (JSON com amostra, ou multipart com file ou image)", Request: CreateAnalysisRequest{}, Response: AnalysisResponse{}},
	{Method: http.MethodGet, Path: "/farms/{id}/analyses", Description: "lista as analises da fazenda", Response: []AnalysisResponse{}},
	{Method: http.MethodGet, Path: "/analyses", Description: "lista as analises de todas as fazendas do usuario (?limit=&offset=)", Response: AnalysisListResponse{}},
	{Method: http.MethodGet, Path: "/stats", Description: "resumo do usuario para o dashboard (?offset= e o fuso em minutos a leste de UTC)", Response: StatsResponse{}},
	{Method: http.MethodGet, Path: "/analyses/{id}", Description: "busca uma analise", Response: AnalysisResponse{}},
	{Method: http.MethodGet, Path: "/analyses/{id}/contents", Description: "lista os conteudos da analise", Response: []ContentResponse{}},
	{Method: http.MethodGet, Path: "/analyses/{id}/rows", Description: "roda o modelo de novo no arquivo da analise e devolve uma linha por amostra (so analises de arquivo)", Response: []map[string]any{}},
	{Method: http.MethodDelete, Path: "/analyses/{id}", Description: "apaga uma analise"},
}
