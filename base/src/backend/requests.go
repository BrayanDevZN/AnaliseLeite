package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// ModelURLEnv e a variavel de ambiente com a URL do microservico do modelo (o pipeline).
	ModelURLEnv = "MODEL_URL"

	// ModelKeyEnv e a variavel de ambiente com a chave de API que o pipeline exige (a mesma ML_KEY dele).
	ModelKeyEnv = "ML_KEY"

	// defaultRequestTimeout e o tempo maximo de cada chamada ao pipeline.
	defaultRequestTimeout = 60 * time.Second

	// imageRequestTimeout e o tempo maximo da chamada de imagem: o agente do pipeline le a imagem com um modelo
	// de visao e, se falhar, tenta de novo ate 3 vezes, entao demora bem mais que as outras rotas.
	imageRequestTimeout = 110 * time.Second

	// maxResponseSize e o maior corpo de resposta que e lido (em bytes).
	maxResponseSize = 64 << 20
)

// Camadas de dados da rota GET /pipeline/.
const (
	LayerProcessed = "processed"
	LayerCleaned   = "cleaned"
	LayerRaw       = "raw"
)

var (
	ErrInvalidModelURL = errors.New("invalid model url")
	ErrInvalidLayer    = errors.New("invalid layer, expected processed, cleaned or raw")
	ErrInvalidFile     = errors.New("expected a csv or parquet file")

	// ErrInvalidImage quando o agente do pipeline diz que a imagem nao serve (nao tem relacao com leite).
	ErrInvalidImage = errors.New("the image is not related to milk quality")

	ErrInvalidMethod     = errors.New("invalid request method")
	ErrInvalidRequestURL = errors.New("invalid request url")
)

// RequestError e o erro devolvido quando o pipeline responde com um status que nao e 2xx.
type RequestError struct {
	StatusCode int
	Detail     string
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("model service returned status %d: %s", e.StatusCode, e.Detail)
}

// ModelInput sao os dados de uma amostra para o modelo (corpo da rota POST /model/).
// Todos os campos sao opcionais; quais sao obrigatorios depende do modelo escolhido.
type ModelInput struct {
	Temperature *float64 `json:"temperature,omitempty"`
	State       *string  `json:"state,omitempty"`
	Data        *string  `json:"data,omitempty"`
	CCS         *float64 `json:"ccs,omitempty"`
	CPP         *float64 `json:"cpp,omitempty"`
	City        *string  `json:"cidade,omitempty"`
	Tank        *string  `json:"tanque,omitempty"`
}

// ImageResult e a resposta do agente de imagem do pipeline (POST /model/image). So um dos campos vem preenchido:
// Result para uma amostra lida ("aprovado" ou "reprovado"), Rows para varias amostras (uma linha por amostra, com
// a coluna "predict") e Text quando a imagem nao tem numeros e o agente devolveu uma analise em texto.
type ImageResult struct {
	Result string
	Rows   []map[string]any
	Text   string
}

// RequestsOption muda o comportamento de NewRequests.
type RequestsOption func(*Requests)

// WithModelURL usa essa URL em vez de ler a MODEL_URL.
func WithModelURL(modelURL string) RequestsOption {
	return func(r *Requests) {
		r.baseURL = modelURL
		r.hasURL = true
	}
}

// WithAPIKey usa essa chave em vez de ler a ML_KEY.
func WithAPIKey(key string) RequestsOption {
	return func(r *Requests) {
		r.key = key
		r.hasKey = true
	}
}

// WithHTTPClient troca o cliente HTTP (para mudar o timeout, o transporte, etc.).
func WithHTTPClient(client *http.Client) RequestsOption {
	return func(r *Requests) {
		r.client = client
	}
}

// Requests faz as chamadas ao pipeline (o microservico do modelo), uma por rota.
// E seguro usar de varias goroutines ao mesmo tempo.
type Requests struct {
	logger  *Logger
	baseURL string
	base    *url.URL
	key     string
	hasURL  bool
	hasKey  bool
	client  *http.Client
}

// NewRequests le a MODEL_URL e a ML_KEY do ambiente e cria o cliente do pipeline.
// Se alguma faltar, devolve um *MissingSecretsError com o nome de todas as que faltam.
// Se logger for nil, os logs vao para o terminal. A chave e a URL nunca vao para os logs.
func NewRequests(logger *Logger, options ...RequestsOption) (*Requests, error) {
	if logger == nil {
		logger = NewLogger()
	}

	requests := &Requests{
		logger: logger,
		client: &http.Client{Timeout: defaultRequestTimeout},
	}

	for _, option := range options {
		option(requests)
	}

	var required []string

	if !requests.hasURL {
		required = append(required, ModelURLEnv)
	}

	if !requests.hasKey {
		required = append(required, ModelKeyEnv)
	}

	if len(required) > 0 {
		logger.Info("Reading %s...", strings.Join(required, ", "))

		config, err := NewConfig(required, required)

		if err != nil {
			logger.Error("%v", err)

			return nil, err
		}

		secrets := config.GetSecrets()

		if !requests.hasURL {
			requests.baseURL = *secrets[ModelURLEnv]
		}

		if !requests.hasKey {
			requests.key = *secrets[ModelKeyEnv]
		}

		logger.Success("%s loaded with success", strings.Join(required, ", "))
	}

	parsed, err := url.Parse(requests.baseURL)

	// o erro do url.Parse repete a URL inteira, entao ele nao e repassado
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		logger.Error("%v", ErrInvalidModelURL)

		return nil, ErrInvalidModelURL
	}

	requests.baseURL = strings.TrimRight(requests.baseURL, "/")
	requests.base = parsed

	return requests, nil
}

// Health chama GET /health e diz se o pipeline esta de pe.
func (r *Requests) Health(ctx context.Context) (healthy bool, err error) {
	defer r.begin("check model service health")(&err)

	var body struct {
		Status bool `json:"status"`
	}

	if err := r.do(ctx, http.MethodGet, "/health", nil, "", nil, &body); err != nil {
		return false, err
	}

	return body.Status, nil
}

// Pipeline chama GET /pipeline/ e devolve os dados da camada (processed, cleaned ou raw).
// O JSON e uma lista de registros, ou um mapa de nome para lista de registros.
// Com layer vazio, o pipeline usa a camada padrao (processed).
func (r *Requests) Pipeline(ctx context.Context, layer string) (data json.RawMessage, err error) {
	defer r.begin("get pipeline data")(&err)

	query := url.Values{}

	switch layer {
	case "":
	case LayerProcessed, LayerCleaned, LayerRaw:
		query.Set("layer", layer)

	default:
		return nil, ErrInvalidLayer
	}

	if err := r.do(ctx, http.MethodGet, "/pipeline/", query, "", nil, &data); err != nil {
		return nil, err
	}

	return data, nil
}

// Predict chama POST /model/ e devolve a resposta do modelo para uma amostra ("aprovado" ou "reprovado").
// Com model vazio, o pipeline usa o modelo padrao.
func (r *Requests) Predict(ctx context.Context, input ModelInput, model string) (result string, err error) {
	defer r.begin("predict one sample")(&err)

	body, err := json.Marshal(input)

	if err != nil {
		return "", fmt.Errorf("encode model input: %w", err)
	}

	if err := r.do(ctx, http.MethodPost, "/model/", modelQuery(model), "application/json", bytes.NewReader(body), &result); err != nil {
		return "", err
	}

	return result, nil
}

// PredictFile chama POST /model/file e devolve a resposta do modelo para cada linha do arquivo.
// O arquivo precisa terminar em .csv (separado por ;) ou .parquet. Com model vazio, usa o modelo padrao.
func (r *Requests) PredictFile(ctx context.Context, filename string, file io.Reader, model string) (rows []map[string]any, err error) {
	defer r.begin("predict a file")(&err)

	if !strings.HasSuffix(filename, ".csv") && !strings.HasSuffix(filename, ".parquet") {
		return nil, ErrInvalidFile
	}

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filename)

	if err != nil {
		return nil, fmt.Errorf("build upload: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("build upload: %w", err)
	}

	if err := r.do(ctx, http.MethodPost, "/model/file", modelQuery(model), writer.FormDataContentType(), &body, &rows); err != nil {
		return nil, err
	}

	return rows, nil
}

// PredictImage chama POST /model/image: o agente do pipeline le a imagem (laudo, tabela, anotacao), extrai os dados
// e prever. A resposta depende do que ele achou (veja ImageResult). Se a imagem nao tem relacao com leite, devolve
// ErrInvalidImage. Com model vazio, usa o modelo padrao.
func (r *Requests) PredictImage(ctx context.Context, filename string, image io.Reader, model string) (result *ImageResult, err error) {
	defer r.begin("predict an image")(&err)

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("image", filename)

	if err != nil {
		return nil, fmt.Errorf("build upload: %w", err)
	}

	if _, err := io.Copy(part, image); err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("build upload: %w", err)
	}

	// o agente demora mais que as outras rotas: usa uma copia do cliente com o prazo maior (so se o atual for menor)
	slow := *r

	if r.client.Timeout > 0 && r.client.Timeout < imageRequestTimeout {
		slow.client = &http.Client{Transport: r.client.Transport, CheckRedirect: r.client.CheckRedirect, Jar: r.client.Jar, Timeout: imageRequestTimeout}
	}

	var raw json.RawMessage

	if err := slow.do(ctx, http.MethodPost, "/model/image", modelQuery(model), writer.FormDataContentType(), &body, &raw); err != nil {
		var requestError *RequestError

		// o pipeline responde 409 quando o agente considera a imagem invalida
		if errors.As(err, &requestError) && requestError.StatusCode == http.StatusConflict {
			return nil, ErrInvalidImage
		}

		return nil, err
	}

	return decodeImageResult(raw)
}

// decodeImageResult separa as tres formas da resposta do agente: texto (uma amostra ou analise) ou lista de linhas.
func decodeImageResult(raw json.RawMessage) (*ImageResult, error) {
	trimmed := bytes.TrimSpace(raw)

	if len(trimmed) > 0 && trimmed[0] == '[' {
		var rows []map[string]any

		if err := json.Unmarshal(trimmed, &rows); err != nil {
			return nil, fmt.Errorf("decode model service response: %w", err)
		}

		return &ImageResult{Rows: rows}, nil
	}

	var text string

	if err := json.Unmarshal(trimmed, &text); err != nil {
		return nil, fmt.Errorf("decode model service response: %w", err)
	}

	text = strings.TrimSpace(text)

	switch strings.ToLower(text) {
	case "aprovado", "reprovado":
		return &ImageResult{Result: strings.ToLower(text)}, nil
	}

	return &ImageResult{Text: text}, nil
}

// Models chama GET /model/ e devolve os nomes dos modelos disponiveis.
func (r *Requests) Models(ctx context.Context) (models []string, err error) {
	defer r.begin("list models")(&err)

	if err := r.do(ctx, http.MethodGet, "/model/", nil, "", nil, &models); err != nil {
		return nil, err
	}

	return models, nil
}

// modelQuery monta o query ?model=... (vazio se nao tiver modelo).
func modelQuery(model string) url.Values {
	query := url.Values{}

	if model != "" {
		query.Set("model", model)
	}

	return query
}

// begin loga o comeco da chamada e devolve a funcao que loga o fim.
func (r *Requests) begin(format string, args ...any) func(*error) {
	action := fmt.Sprintf(format, args...)

	r.logger.Info("Starting to %s...", action)

	return func(err *error) {
		if *err != nil {
			r.logger.Error("Could not %s: %v", action, *err)

			return
		}

		r.logger.Success("Finished to %s with success", action)
	}
}

// do faz a chamada com a chave no header Authorization e decodifica o JSON da resposta em out.
func (r *Requests) do(ctx context.Context, method, path string, query url.Values, contentType string, body io.Reader, out any) error {
	target := r.baseURL + path

	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	header := http.Header{}

	header.Set("Authorization", "Bearer "+r.key)
	header.Set("Accept", "application/json")

	if contentType != "" {
		header.Set("Content-Type", contentType)
	}

	response, err := r.roundTrip(ctx, method, target, header, body)

	if err != nil {
		return err
	}

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return &RequestError{StatusCode: response.StatusCode, Detail: errorDetail(response.StatusCode, response.Body)}
	}

	if err := json.Unmarshal(response.Body, out); err != nil {
		return fmt.Errorf("decode model service response: %w", err)
	}

	return nil
}

// roundTrip envia a requisicao e le a resposta inteira (de qualquer status).
func (r *Requests) roundTrip(ctx context.Context, method, target string, header http.Header, body io.Reader) (*Response, error) {
	request, err := http.NewRequestWithContext(ctx, method, target, body)

	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	for name, values := range header {
		request.Header[name] = values
	}

	response, err := r.client.Do(request)

	if err != nil {
		return nil, fmt.Errorf("call model service: %w", err)
	}

	defer response.Body.Close()

	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize))

	if err != nil {
		return nil, fmt.Errorf("read model service response: %w", err)
	}

	return &Response{StatusCode: response.StatusCode, Header: response.Header, Body: data}, nil
}

// Request e uma requisicao livre para o Send.
type Request struct {
	// Method e o metodo HTTP (GET, POST, PUT, PATCH, DELETE...).
	Method string

	// URL pode ser um caminho (ex.: "/model/"), que vai para a MODEL_URL, ou uma URL completa
	// (http:// ou https://). So as requisicoes para o proprio host da MODEL_URL levam a chave.
	URL string

	// Query e acrescentado na URL (opcional).
	Query url.Values

	// Headers sao os headers extras (opcional). Um Authorization seu vale mais que a chave automatica.
	Headers map[string]string

	// Body e o corpo (opcional): io.Reader, []byte e string vao como estao; qualquer outro valor
	// vira JSON (e o Content-Type vira application/json, se voce nao definiu outro).
	Body any
}

// Response e a resposta de uma requisicao livre, de qualquer status.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// JSON decodifica o corpo da resposta em out.
func (r *Response) JSON(out any) error {
	if err := json.Unmarshal(r.Body, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

// Send envia uma requisicao livre: voce diz o metodo, a URL, os dados e os headers.
// Diferente dos outros metodos, qualquer status HTTP (inclusive 4xx e 5xx) volta como Response
// sem erro; o erro so vem se a requisicao nao puder ser feita ou a resposta nao puder ser lida.
// A chave (ML_KEY) so e enviada para o host da MODEL_URL, nunca para outro host.
func (r *Requests) Send(ctx context.Context, request Request) (response *Response, err error) {
	method := strings.ToUpper(strings.TrimSpace(request.Method))

	if method == "" {
		return nil, ErrInvalidMethod
	}

	target, sameHost, path, err := r.resolve(request.URL)

	if err != nil {
		return nil, err
	}

	defer r.begin("send %s %s", method, path)(&err)

	if len(request.Query) > 0 {
		separator := "?"

		if strings.Contains(target, "?") {
			separator = "&"
		}

		target += separator + request.Query.Encode()
	}

	body, contentType, err := encodeBody(request.Body)

	if err != nil {
		return nil, err
	}

	header := http.Header{}

	if sameHost {
		header.Set("Authorization", "Bearer "+r.key)
	}

	if contentType != "" {
		header.Set("Content-Type", contentType)
	}

	for name, value := range request.Headers {
		header.Set(name, value)
	}

	return r.roundTrip(ctx, method, target, header, body)
}

// resolve acha a URL final. Devolve tambem se ela e do host da MODEL_URL e o caminho (para o log).
func (r *Requests) resolve(raw string) (target string, sameHost bool, path string, err error) {
	parsed, err := url.Parse(raw)

	// o erro do url.Parse repete a URL inteira, entao ele nao e repassado
	if err != nil || raw == "" {
		return "", false, "", ErrInvalidRequestURL
	}

	if parsed.IsAbs() {
		if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return "", false, "", ErrInvalidRequestURL
		}

		sameHost = strings.EqualFold(parsed.Scheme, r.base.Scheme) && strings.EqualFold(parsed.Host, r.base.Host)

		return raw, sameHost, parsed.Path, nil
	}

	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}

	return r.baseURL + raw, true, parsed.Path, nil
}

// encodeBody transforma o corpo no que vai na requisicao e devolve o Content-Type automatico.
func encodeBody(body any) (io.Reader, string, error) {
	switch value := body.(type) {
	case nil:
		return nil, "", nil

	case io.Reader:
		return value, "", nil

	case []byte:
		return bytes.NewReader(value), "", nil

	case string:
		return strings.NewReader(value), "", nil
	}

	data, err := json.Marshal(body)

	if err != nil {
		return nil, "", fmt.Errorf("encode request body: %w", err)
	}

	return bytes.NewReader(data), "application/json", nil
}

// errorDetail pega a mensagem do erro do FastAPI ({"detail": ...}).
func errorDetail(status int, data []byte) string {
	var body struct {
		Detail json.RawMessage `json:"detail"`
	}

	detail := ""

	if json.Unmarshal(data, &body) == nil && len(body.Detail) > 0 {
		var text string

		if json.Unmarshal(body.Detail, &text) == nil {
			detail = text
		} else {
			detail = string(body.Detail)
		}
	}

	if detail == "" {
		detail = http.StatusText(status)
	}

	if len(detail) > 500 {
		detail = detail[:500] + "..."
	}

	return detail
}
