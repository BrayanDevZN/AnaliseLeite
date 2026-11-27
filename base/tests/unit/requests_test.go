package unit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
)

const testKey = "chave-secreta-de-teste"

// pipelineSeen e o que o servidor de teste (o falso pipeline) recebeu na ultima chamada.
type pipelineSeen struct {
	method, path, query, auth, contentType string
	body                                   []byte
}

// newFakePipeline sobe um servidor que imita o pipeline: exige a chave e responde por rota.
func newFakePipeline(t *testing.T) (*httptest.Server, *pipelineSeen) {
	t.Helper()

	seen := &pipelineSeen{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.method, seen.path, seen.query = r.Method, r.URL.Path, r.URL.RawQuery
		seen.auth, seen.contentType = r.Header.Get("Authorization"), r.Header.Get("Content-Type")
		seen.body, _ = io.ReadAll(r.Body)

		w.Header().Set("Content-Type", "application/json")

		if seen.auth != "Bearer "+testKey {
			w.WriteHeader(http.StatusUnauthorized)
			io.WriteString(w, `{"detail":"Invalid credential"}`)

			return
		}

		switch r.Method + " " + r.URL.Path {
		case "GET /health":
			io.WriteString(w, `{"status": true}`)

		case "GET /pipeline/":
			w.WriteHeader(http.StatusCreated) // o pipeline responde 201
			io.WriteString(w, `[{"ccs": 100.5}]`)

		case "POST /model/":
			io.WriteString(w, `"aprovado"`)

		case "POST /model/file":
			io.WriteString(w, `[{"resultado":"aprovado"},{"resultado":"reprovado"}]`)

		case "GET /model/":
			io.WriteString(w, `["anl_v1","anl_v2"]`)

		default:
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, `{"detail":"Not Found"}`)
		}
	}))

	t.Cleanup(server.Close)

	return server, seen
}

func newTestRequests(t *testing.T, url string) (*backend.Requests, *bytes.Buffer) {
	t.Helper()

	var logs bytes.Buffer

	requests, err := backend.NewRequests(backend.NewLoggerWriter(&logs), backend.WithModelURL(url), backend.WithAPIKey(testKey))

	if err != nil {
		t.Fatal(err)
	}

	return requests, &logs
}

func TestRequestsReadsTheEnvironment(t *testing.T) {
	server, _ := newFakePipeline(t)

	t.Setenv(backend.ModelURLEnv, server.URL+"/")
	t.Setenv(backend.ModelKeyEnv, testKey)

	requests, err := backend.NewRequests(backend.NewLoggerWriter(io.Discard))

	if err != nil {
		t.Fatal(err)
	}

	// a barra no fim da URL nao pode duplicar na rota
	healthy, err := requests.Health(context.Background())

	if err != nil || !healthy {
		t.Fatalf("esperava o pipeline de pe, veio %v %v", healthy, err)
	}
}

func TestRequestsMissingEnvironment(t *testing.T) {
	t.Setenv(backend.ModelURLEnv, "")
	t.Setenv(backend.ModelKeyEnv, "")

	_, err := backend.NewRequests(backend.NewLoggerWriter(io.Discard))

	var missing *backend.MissingSecretsError

	if !errors.As(err, &missing) || len(missing.Names) != 2 {
		t.Fatalf("esperava faltar MODEL_URL e ML_KEY, veio %v", err)
	}

	// so a chave falta
	t.Setenv(backend.ModelURLEnv, "http://localhost:8000")

	_, err = backend.NewRequests(backend.NewLoggerWriter(io.Discard))

	if !errors.As(err, &missing) || len(missing.Names) != 1 || missing.Names[0] != backend.ModelKeyEnv {
		t.Fatalf("esperava faltar so a ML_KEY, veio %v", err)
	}
}

func TestRequestsInvalidURL(t *testing.T) {
	for _, bad := range []string{"", "localhost:8000", "ftp://localhost", "http://", "://x"} {
		_, err := backend.NewRequests(backend.NewLoggerWriter(io.Discard), backend.WithModelURL(bad), backend.WithAPIKey("k"))

		if !errors.Is(err, backend.ErrInvalidModelURL) {
			t.Errorf("URL %q deveria dar ErrInvalidModelURL, veio %v", bad, err)
		}
	}
}

func TestRequestsRoutes(t *testing.T) {
	ctx := context.Background()
	server, seen := newFakePipeline(t)
	requests, logs := newTestRequests(t, server.URL)

	t.Run("health", func(t *testing.T) {
		healthy, err := requests.Health(ctx)

		if err != nil || !healthy {
			t.Fatalf("esperava true, veio %v %v", healthy, err)
		}

		if seen.method != "GET" || seen.path != "/health" || seen.auth != "Bearer "+testKey {
			t.Fatalf("chamada errada: %+v", seen)
		}
	})

	t.Run("pipeline", func(t *testing.T) {
		data, err := requests.Pipeline(ctx, backend.LayerRaw)

		if err != nil {
			t.Fatal(err)
		}

		if string(data) != `[{"ccs": 100.5}]` || seen.path != "/pipeline/" || seen.query != "layer=raw" {
			t.Fatalf("chamada errada: %s %+v", data, seen)
		}

		// sem camada, nao manda o query (o pipeline usa o padrao)
		if _, err := requests.Pipeline(ctx, ""); err != nil || seen.query != "" {
			t.Fatalf("sem camada nao deveria ter query: %v %q", err, seen.query)
		}

		if _, err := requests.Pipeline(ctx, "gold"); !errors.Is(err, backend.ErrInvalidLayer) {
			t.Fatalf("camada invalida deveria dar ErrInvalidLayer, veio %v", err)
		}
	})

	t.Run("predict", func(t *testing.T) {
		temperature, state := 4.5, "MG"

		result, err := requests.Predict(ctx, backend.ModelInput{Temperature: &temperature, State: &state}, "anl_v2")

		if err != nil || result != "aprovado" {
			t.Fatalf("esperava aprovado, veio %q %v", result, err)
		}

		if seen.method != "POST" || seen.path != "/model/" || seen.query != "model=anl_v2" || seen.contentType != "application/json" {
			t.Fatalf("chamada errada: %+v", seen)
		}

		// so os campos preenchidos vao no JSON
		var sent map[string]any

		if err := json.Unmarshal(seen.body, &sent); err != nil {
			t.Fatal(err)
		}

		if len(sent) != 2 || sent["temperature"] != 4.5 || sent["state"] != "MG" {
			t.Fatalf("corpo errado: %s", seen.body)
		}

		// temperatura 0 e um valor valido e nao pode sumir
		zero := 0.0

		if _, err := requests.Predict(ctx, backend.ModelInput{Temperature: &zero}, ""); err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(string(seen.body), `"temperature":0`) || seen.query != "" {
			t.Fatalf("temperatura 0 deveria ir no corpo, sem query: %s %q", seen.body, seen.query)
		}
	})

	t.Run("predict file", func(t *testing.T) {
		rows, err := requests.PredictFile(ctx, "coletas.csv", strings.NewReader("a;b\n1;2\n"), "anl_v1")

		if err != nil {
			t.Fatal(err)
		}

		if len(rows) != 2 || rows[1]["resultado"] != "reprovado" {
			t.Fatalf("linhas erradas: %v", rows)
		}

		if seen.method != "POST" || seen.path != "/model/file" || seen.query != "model=anl_v1" {
			t.Fatalf("chamada errada: %+v", seen)
		}

		request, _ := http.NewRequest("POST", "/", bytes.NewReader(seen.body))

		request.Header.Set("Content-Type", seen.contentType)

		if err := request.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("o corpo deveria ser multipart: %v", err)
		}

		file, header, err := request.FormFile("file")

		if err != nil {
			t.Fatalf("faltou o campo file: %v", err)
		}

		content, _ := io.ReadAll(file)

		if header.Filename != "coletas.csv" || string(content) != "a;b\n1;2\n" {
			t.Fatalf("arquivo errado: %q %q", header.Filename, content)
		}

		if _, err := requests.PredictFile(ctx, "coletas.xlsx", strings.NewReader("x"), ""); !errors.Is(err, backend.ErrInvalidFile) {
			t.Fatalf("arquivo invalido deveria dar ErrInvalidFile, veio %v", err)
		}
	})

	t.Run("models", func(t *testing.T) {
		models, err := requests.Models(ctx)

		if err != nil || len(models) != 2 || models[0] != "anl_v1" {
			t.Fatalf("modelos errados: %v %v", models, err)
		}

		if seen.method != "GET" || seen.path != "/model/" {
			t.Fatalf("chamada errada: %+v", seen)
		}
	})

	t.Run("logs never have the key or the url", func(t *testing.T) {
		text := logs.String()

		if !strings.Contains(text, "Starting to list models") || !strings.Contains(text, "Finished to list models with success") {
			t.Fatalf("faltaram os logs de comeco e fim:\n%s", text)
		}

		if strings.Contains(text, testKey) || strings.Contains(text, server.URL) {
			t.Fatalf("a chave ou a URL apareceu nos logs:\n%s", text)
		}
	})
}

func TestRequestsErrors(t *testing.T) {
	ctx := context.Background()
	server, _ := newFakePipeline(t)

	// chave errada: o pipeline responde 401 com {"detail": ...}
	wrong, err := backend.NewRequests(backend.NewLoggerWriter(io.Discard), backend.WithModelURL(server.URL), backend.WithAPIKey("errada"))

	if err != nil {
		t.Fatal(err)
	}

	_, err = wrong.Models(ctx)

	var requestError *backend.RequestError

	if !errors.As(err, &requestError) || requestError.StatusCode != 401 || requestError.Detail != "Invalid credential" {
		t.Fatalf("esperava RequestError 401, veio %v", err)
	}

	// resposta que nao e JSON
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "nao e json")
	}))

	defer broken.Close()

	requests, _ := newTestRequests(t, broken.URL)

	if _, err := requests.Models(ctx); err == nil {
		t.Fatal("resposta que nao e JSON deveria dar erro")
	}

	// erro do servidor sem {"detail": ...} usa o texto do status
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))

	defer failing.Close()

	requests, _ = newTestRequests(t, failing.URL)

	_, err = requests.Models(ctx)

	if !errors.As(err, &requestError) || requestError.StatusCode != 502 || requestError.Detail != "Bad Gateway" {
		t.Fatalf("esperava RequestError 502, veio %v", err)
	}

	// contexto cancelado
	requests, _ = newTestRequests(t, server.URL)

	canceled, cancel := context.WithCancel(ctx)

	cancel()

	if _, err := requests.Models(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("esperava context.Canceled, veio %v", err)
	}

	// pipeline fora do ar
	server.Close()

	if _, err := requests.Health(ctx); err == nil {
		t.Fatal("pipeline fora do ar deveria dar erro")
	}
}

func TestRequestsSend(t *testing.T) {
	ctx := context.Background()
	server, seen := newFakePipeline(t)
	requests, logs := newTestRequests(t, server.URL)

	t.Run("get with query and headers", func(t *testing.T) {
		response, err := requests.Send(ctx, backend.Request{
			Method:  "get",
			URL:     "model/",
			Query:   map[string][]string{"x": {"1"}},
			Headers: map[string]string{"X-Teste": "sim"},
		})

		if err != nil {
			t.Fatal(err)
		}

		var models []string

		if err := response.JSON(&models); err != nil || len(models) != 2 {
			t.Fatalf("resposta errada: %s %v", response.Body, err)
		}

		if seen.method != "GET" || seen.path != "/model/" || seen.query != "x=1" || seen.auth != "Bearer "+testKey {
			t.Fatalf("chamada errada: %+v", seen)
		}
	})

	t.Run("query already in the url", func(t *testing.T) {
		_, err := requests.Send(ctx, backend.Request{Method: "GET", URL: "/model/?a=1", Query: map[string][]string{"b": {"2"}}})

		must(t, err)

		if seen.query != "a=1&b=2" {
			t.Fatalf("esperava a=1&b=2, veio %q", seen.query)
		}
	})

	t.Run("bodies", func(t *testing.T) {
		// qualquer valor que nao seja texto ou bytes vira JSON
		_, err := requests.Send(ctx, backend.Request{Method: "POST", URL: "/model/", Body: map[string]any{"ccs": 10}})

		must(t, err)

		if seen.contentType != "application/json" || string(seen.body) != `{"ccs":10}` {
			t.Fatalf("JSON errado: %q %s", seen.contentType, seen.body)
		}

		// texto vai como esta, e o Content-Type e o que voce disser
		_, err = requests.Send(ctx, backend.Request{
			Method: "POST", URL: "/model/", Body: "a;b\n1;2", Headers: map[string]string{"Content-Type": "text/csv"},
		})

		must(t, err)

		if seen.contentType != "text/csv" || string(seen.body) != "a;b\n1;2" {
			t.Fatalf("texto errado: %q %s", seen.contentType, seen.body)
		}

		_, err = requests.Send(ctx, backend.Request{Method: "PUT", URL: "/model/", Body: []byte{1, 2}})

		must(t, err)

		if seen.method != "PUT" || !bytes.Equal(seen.body, []byte{1, 2}) || seen.contentType != "" {
			t.Fatalf("bytes errados: %+v", seen)
		}

		_, err = requests.Send(ctx, backend.Request{Method: "POST", URL: "/model/", Body: strings.NewReader("reader")})

		must(t, err)

		if string(seen.body) != "reader" {
			t.Fatalf("reader errado: %s", seen.body)
		}

		if _, err := requests.Send(ctx, backend.Request{Method: "POST", URL: "/model/", Body: make(chan int)}); err == nil {
			t.Fatal("corpo que nao vira JSON deveria dar erro")
		}
	})

	t.Run("any status comes back without error", func(t *testing.T) {
		response, err := requests.Send(ctx, backend.Request{Method: "GET", URL: "/nao-existe"})

		if err != nil || response.StatusCode != 404 {
			t.Fatalf("esperava 404 sem erro, veio %v %v", response, err)
		}

		// chave errada: 401, tambem sem erro
		response, err = requests.Send(ctx, backend.Request{Method: "GET", URL: "/health", Headers: map[string]string{"Authorization": "Bearer outra"}})

		if err != nil || response.StatusCode != 401 {
			t.Fatalf("o Authorization do chamador deveria valer mais: %v %v", response, err)
		}
	})

	t.Run("the key goes only to the model host", func(t *testing.T) {
		// URL completa do mesmo host: leva a chave
		_, err := requests.Send(ctx, backend.Request{Method: "GET", URL: server.URL + "/health"})

		must(t, err)

		if seen.auth != "Bearer "+testKey {
			t.Fatalf("o mesmo host deveria receber a chave, veio %q", seen.auth)
		}

		// outro host: nunca leva a chave
		var otherAuth string

		other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			otherAuth = r.Header.Get("Authorization")

			io.WriteString(w, "ok")
		}))

		defer other.Close()

		response, err := requests.Send(ctx, backend.Request{Method: "GET", URL: other.URL + "/x"})

		if err != nil || string(response.Body) != "ok" {
			t.Fatalf("resposta errada: %v %v", response, err)
		}

		if otherAuth != "" {
			t.Fatalf("a chave vazou para outro host: %q", otherAuth)
		}

		// mas um Authorization que voce mesmo pos vai
		_, err = requests.Send(ctx, backend.Request{Method: "GET", URL: other.URL, Headers: map[string]string{"Authorization": "Bearer meu"}})

		must(t, err)

		if otherAuth != "Bearer meu" {
			t.Fatalf("o Authorization do chamador deveria ir, veio %q", otherAuth)
		}
	})

	t.Run("invalid requests", func(t *testing.T) {
		if _, err := requests.Send(ctx, backend.Request{URL: "/health"}); !errors.Is(err, backend.ErrInvalidMethod) {
			t.Fatalf("sem metodo deveria dar ErrInvalidMethod, veio %v", err)
		}

		for _, bad := range []string{"", "ftp://localhost/x", "http://", "://x"} {
			if _, err := requests.Send(ctx, backend.Request{Method: "GET", URL: bad}); !errors.Is(err, backend.ErrInvalidRequestURL) {
				t.Errorf("URL %q deveria dar ErrInvalidRequestURL, veio %v", bad, err)
			}
		}

		if _, err := requests.Send(ctx, backend.Request{Method: "GE T", URL: "/health"}); err == nil {
			t.Fatal("metodo invalido deveria dar erro")
		}
	})

	t.Run("logs", func(t *testing.T) {
		text := logs.String()

		if !strings.Contains(text, "Starting to send GET /model/") {
			t.Fatalf("faltou o log da requisicao livre:\n%s", text)
		}

		// nem a chave, nem a URL, nem o query vao para os logs
		if strings.Contains(text, testKey) || strings.Contains(text, server.URL) || strings.Contains(text, "a=1") {
			t.Fatalf("dado sensivel nos logs:\n%s", text)
		}
	})
}

func TestRequestsPredictImage(t *testing.T) {
	ctx := context.Background()

	var seenPath, seenContentType string
	var seenBody []byte

	// o falso pipeline responde conforme o ?model=: cada valor imita um caminho do agente de imagem
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath, seenContentType = r.URL.Path, r.Header.Get("Content-Type")
		seenBody, _ = io.ReadAll(r.Body)

		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Query().Get("model") {
		case "rows":
			io.WriteString(w, `[{"ccs":280,"predict":"aprovado"},{"ccs":510,"predict":"reprovado"}]`)

		case "text":
			io.WriteString(w, `"Foto de um latao de leite, sem numeros."`)

		case "invalid":
			w.WriteHeader(http.StatusConflict)
			io.WriteString(w, `{"detail":"Invalid image"}`)

		case "down":
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, `{"detail":"boom"}`)

		default:
			io.WriteString(w, `"Reprovado"`)
		}
	}))

	t.Cleanup(server.Close)

	requests, _ := newTestRequests(t, server.URL)

	t.Run("one sample", func(t *testing.T) {
		result, err := requests.PredictImage(ctx, "laudo.jpg", strings.NewReader("JPEGDATA"), "")

		if err != nil || result.Result != "reprovado" || result.Rows != nil || result.Text != "" {
			t.Fatalf("resposta errada: %+v %v", result, err)
		}

		if seenPath != "/model/image" {
			t.Fatalf("rota errada: %s", seenPath)
		}

		request, _ := http.NewRequest("POST", "/", bytes.NewReader(seenBody))

		request.Header.Set("Content-Type", seenContentType)

		if err := request.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("o corpo deveria ser multipart: %v", err)
		}

		file, header, err := request.FormFile("image")

		if err != nil {
			t.Fatalf("faltou o campo image: %v", err)
		}

		content, _ := io.ReadAll(file)

		if header.Filename != "laudo.jpg" || string(content) != "JPEGDATA" {
			t.Fatalf("imagem errada: %q %q", header.Filename, content)
		}
	})

	t.Run("several samples", func(t *testing.T) {
		result, err := requests.PredictImage(ctx, "tabela.png", strings.NewReader("x"), "rows")

		if err != nil || len(result.Rows) != 2 || result.Rows[1]["predict"] != "reprovado" || result.Result != "" {
			t.Fatalf("resposta errada: %+v %v", result, err)
		}
	})

	t.Run("text analysis", func(t *testing.T) {
		result, err := requests.PredictImage(ctx, "latao.jpg", strings.NewReader("x"), "text")

		if err != nil || !strings.HasPrefix(result.Text, "Foto de um latao") || result.Result != "" || result.Rows != nil {
			t.Fatalf("resposta errada: %+v %v", result, err)
		}
	})

	t.Run("invalid image", func(t *testing.T) {
		if _, err := requests.PredictImage(ctx, "cachorro.jpg", strings.NewReader("x"), "invalid"); !errors.Is(err, backend.ErrInvalidImage) {
			t.Fatalf("409 deveria virar ErrInvalidImage, veio %v", err)
		}
	})

	t.Run("pipeline failure", func(t *testing.T) {
		_, err := requests.PredictImage(ctx, "a.jpg", strings.NewReader("x"), "down")

		var requestError *backend.RequestError

		if errors.Is(err, backend.ErrInvalidImage) || !errors.As(err, &requestError) || requestError.StatusCode != 500 {
			t.Fatalf("500 deveria continuar sendo RequestError, veio %v", err)
		}
	})
}
