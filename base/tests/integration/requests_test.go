package integration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
)

// localModelURL e onde o compose.yml publica o pipeline (porta 8000 da maquina).
const localModelURL = "http://localhost:8000"

// testParquet e o arquivo de teste que o pipeline ja usa nos testes funcionais.
const testParquet = "../../src/pipeline/storage/test/analise_leite_teste.parquet"

// newPipelineRequests poe a URL local do pipeline no ambiente (MODEL_URL) e cria o cliente,
// que le a MODEL_URL e a ML_KEY (a mesma chave que o pipeline usa) do ambiente.
// Sem a ML_KEY ou com o pipeline fora do ar, o teste e pulado, e no CI ele falha.
func newPipelineRequests(t *testing.T) *backend.Requests {
	t.Helper()

	ci := os.Getenv("CI") != ""

	if os.Getenv(backend.ModelKeyEnv) == "" {
		if ci {
			t.Fatalf("faltou a variavel %s no CI", backend.ModelKeyEnv)
		}

		t.Skipf("a variavel %s nao esta definida", backend.ModelKeyEnv)
	}

	t.Setenv(backend.ModelURLEnv, localModelURL)

	requests, err := backend.NewRequests(quiet())

	must(t, err)

	healthy, err := requests.Health(context.Background())

	if err != nil || !healthy {
		if ci {
			t.Fatalf("o pipeline nao respondeu em %s: %v", localModelURL, err)
		}

		t.Skipf("o pipeline nao esta no ar em %s (suba com docker compose -f compose.yml up -d): %v", localModelURL, err)
	}

	return requests
}

func TestPipelineService(t *testing.T) {
	ctx := context.Background()
	requests := newPipelineRequests(t)

	models, err := requests.Models(ctx)

	must(t, err)

	if len(models) == 0 {
		t.Fatal("o pipeline deveria ter modelos")
	}

	t.Run("pipeline raw data", func(t *testing.T) {
		data, err := requests.Pipeline(ctx, backend.LayerRaw)

		must(t, err)

		if !json.Valid(data) || len(data) < 3 {
			t.Fatalf("esperava dados em JSON, veio %.200s", data)
		}
	})

	t.Run("predict one sample with every model", func(t *testing.T) {
		temperature, ccs, cpp := 3.6, 252.0, 273.0
		date, state, city, tank := "2025-09-01", "MG", "Uberaba", "TC01"

		input := backend.ModelInput{Temperature: &temperature, CCS: &ccs, CPP: &cpp, Data: &date, State: &state, City: &city, Tank: &tank}

		for _, model := range append(models, "") {
			result, err := requests.Predict(ctx, input, model)

			if err != nil {
				t.Fatalf("modelo %q: %v", model, err)
			}

			if result != "aprovado" && result != "reprovado" {
				t.Fatalf("modelo %q: resposta inesperada %q", model, result)
			}
		}
	})

	t.Run("predict a file with every model", func(t *testing.T) {
		for _, model := range models {
			file, err := os.Open(testParquet)

			must(t, err)

			rows, err := requests.PredictFile(ctx, "analise_leite_teste.parquet", file, model)

			file.Close()

			if err != nil {
				t.Fatalf("modelo %q: %v", model, err)
			}

			if len(rows) == 0 {
				t.Fatalf("modelo %q: esperava linhas", model)
			}

			if _, ok := rows[0]["predict"]; !ok {
				t.Fatalf("modelo %q: faltou a coluna predict: %v", model, rows[0])
			}
		}
	})

	t.Run("unknown model", func(t *testing.T) {
		_, err := requests.Predict(ctx, backend.ModelInput{}, "modelo-que-nao-existe")

		var requestError *backend.RequestError

		if !errors.As(err, &requestError) {
			t.Fatalf("esperava RequestError, veio %v", err)
		}
	})

	t.Run("wrong key", func(t *testing.T) {
		wrong, err := backend.NewRequests(quiet(), backend.WithModelURL(localModelURL), backend.WithAPIKey("chave-errada"))

		must(t, err)

		_, err = wrong.Models(ctx)

		var requestError *backend.RequestError

		if !errors.As(err, &requestError) || requestError.StatusCode != 401 {
			t.Fatalf("esperava RequestError 401, veio %v", err)
		}
	})
}
