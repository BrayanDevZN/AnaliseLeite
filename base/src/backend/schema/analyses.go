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

// CreateAnalysisRequest e o corpo de POST /farms/{id}/analyses.
// Ou vai JSON com Sample, ou vai multipart com o arquivo (campo "file"; tank_id e model viram campos do form).
// Exatamente um dos dois: Sample vai para contents.content e o arquivo para contents.file.
type CreateAnalysisRequest struct {
	TankID *int64 `json:"tank_id,omitempty"`

	// Model e o nome do modelo (vazio usa o padrao do pipeline).
	Model string `json:"model,omitempty"`

	Sample *Sample     `json:"sample,omitempty"`
	File   *FileUpload `json:"-"`
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

	switch {
	case r.Sample != nil && r.File != nil:
		v.add("body", "send either a sample or a file, not both")

	case r.Sample == nil && r.File == nil:
		v.add("body", "send a sample or a file")

	case r.Sample != nil:
		mergeErrors(&v, r.Sample.Validate())

	default:
		mergeErrors(&v, r.File.Validate())
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

// ToContent valida e monta o conteudo do banco: com amostra, so content; com arquivo, so file.
func (r *CreateAnalysisRequest) ToContent(analysisID int64) (*operations.Content, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	content := &operations.Content{AnalysisID: analysisID}

	if r.File != nil {
		content.File = r.File.Data

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
	ID        int64            `json:"id"`
	FarmID    int64            `json:"farm_id"`
	TankID    *int64           `json:"tank_id"`
	CreatedAt time.Time        `json:"created_at"`
	Content   *ContentResponse `json:"content,omitempty"`
	Result    string           `json:"result,omitempty"`
	Rows      []map[string]any `json:"rows,omitempty"`
}

// AnalysisFromModel converte a analise do banco para a resposta.
func AnalysisFromModel(analysis *operations.Analysis) AnalysisResponse {
	return AnalysisResponse{
		ID:        analysis.ID,
		FarmID:    analysis.FarmID,
		TankID:    analysis.TankID,
		CreatedAt: analysis.CreatedAt,
	}
}

// AnalysesFromModel converte uma lista de analises do banco para a resposta (nunca nil).
func AnalysesFromModel(analyses []operations.Analysis) []AnalysisResponse {
	responses := make([]AnalysisResponse, 0, len(analyses))

	for i := range analyses {
		responses = append(responses, AnalysisFromModel(&analyses[i]))
	}

	return responses
}

// AnalysisRoutes sao as rotas de analises. A de criar recebe JSON (amostra) ou multipart (arquivo).
var AnalysisRoutes = []Route{
	{Method: http.MethodPost, Path: "/farms/{id}/analyses", Description: "cria uma analise e roda o modelo", Request: CreateAnalysisRequest{}, Response: AnalysisResponse{}},
	{Method: http.MethodGet, Path: "/farms/{id}/analyses", Description: "lista as analises da fazenda", Response: []AnalysisResponse{}},
	{Method: http.MethodGet, Path: "/analyses/{id}", Description: "busca uma analise", Response: AnalysisResponse{}},
	{Method: http.MethodGet, Path: "/analyses/{id}/contents", Description: "lista os conteudos da analise", Response: []ContentResponse{}},
	{Method: http.MethodDelete, Path: "/analyses/{id}", Description: "apaga uma analise"},
}
