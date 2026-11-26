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
