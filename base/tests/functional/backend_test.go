// Package functional tem os testes funcionais: falam com o backend e o pipeline de verdade, por HTTP.
//
// Para rodar, suba o pipeline e o backend na sua maquina (com o base/.env preenchido) e rode, dentro de base/:
//
//	go test -count=1 -v ./tests/functional/
//
// A URL e sempre a local: http://localhost:<PORT>, com a PORT do .env (padrao 8080). Quem entra e o admin do
// .env (ADMIN_EMAIL e ADMIN_PASSWORD). O teste cria um usuario proprio e apaga tudo que criou no fim.
// Sem o backend no ar o teste e pulado; no CI (variavel CI definida) ele falha.
package functional

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/handles"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/schema"
)

// testParquet e o arquivo de teste que o pipeline ja usa nos testes funcionais.
const testParquet = "../../src/pipeline/storage/test/analise_leite_teste.parquet"

// api e um cliente do backend com cookies, como um navegador.
type api struct {
	t      *testing.T
	url    string
	client *http.Client
}

func newAPI(t *testing.T, url string) *api {
	t.Helper()

	jar, err := cookiejar.New(nil)

	if err != nil {
		t.Fatal(err)
	}

	return &api{t: t, url: url, client: &http.Client{Jar: jar, Timeout: 30 * time.Second}}
}

// send faz a requisicao e devolve o status, o corpo e a resposta. body pode ser um valor (vira JSON) ou *bytes.Buffer.
func (a *api) send(method, path string, body any, headers ...string) (int, []byte, *http.Response) {
	a.t.Helper()

	var reader io.Reader

	contentType := ""

	switch value := body.(type) {
	case nil:
	case *bytes.Buffer:
		reader = value

	default:
		data, err := json.Marshal(value)

		if err != nil {
			a.t.Fatal(err)
		}

		reader, contentType = bytes.NewReader(data), "application/json"
	}

	request, err := http.NewRequest(method, a.url+path, reader)

	if err != nil {
		a.t.Fatal(err)
	}

	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}

	for i := 0; i+1 < len(headers); i += 2 {
		request.Header.Set(headers[i], headers[i+1])
	}

	response, err := a.client.Do(request)

	if err != nil {
		a.t.Fatalf("%s %s: %v", method, path, err)
	}

	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)

	if err != nil {
		a.t.Fatal(err)
	}

	if response.StatusCode == http.StatusTooManyRequests {
		a.t.Fatalf("%s %s: o rate limit do backend bloqueou (429). Espere %s segundo(s) e rode de novo; cada execucao usa boa parte do limite publico por IP",
			method, path, response.Header.Get("Retry-After"))
	}

	return response.StatusCode, data, response
}

// call faz a requisicao, confere o status e, se out nao for nil, le o JSON da resposta nele.
func (a *api) call(method, path string, body any, want int, out any) {
	a.t.Helper()

	status, data, _ := a.send(method, path, body)

	if status != want {
		a.t.Fatalf("%s %s: esperava %d, veio %d: %s", method, path, want, status, data)
	}

	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			a.t.Fatalf("%s %s: resposta que nao e o JSON esperado: %v: %s", method, path, err, data)
		}
	}
}

// expect so confere o status.
func (a *api) expect(method, path string, body any, want int) {
	a.t.Helper()

	a.call(method, path, body, want, nil)
}

// cookie devolve o cookie que o navegador mandaria para esse caminho (o refresh_token so vai para /auth).
func (a *api) cookie(name, path string) *http.Cookie {
	parsed, _ := http.NewRequest(http.MethodGet, a.url+path, nil)

	for _, cookie := range a.client.Jar.Cookies(parsed.URL) {
		if cookie.Name == name {
			return cookie
		}
	}

	return nil
}

// target devolve a URL local do backend e as credenciais do admin, lidas do ambiente (e do base/.env).
// Pula o teste (ou falha, no CI) se o backend nao estiver no ar ou faltar o admin.
func target(t *testing.T) (url, adminEmail, adminPassword string) {
	t.Helper()

	config, err := backend.NewConfig([]string{"PORT", "ADMIN_EMAIL", "ADMIN_PASSWORD"}, nil)

	if err != nil {
		t.Fatal(err)
	}

	values := config.GetSecrets()

	port := "8080"

	if values["PORT"] != nil {
		port = *values["PORT"]
	}

	unavailable := t.Skipf

	if os.Getenv("CI") != "" {
		unavailable = t.Fatalf
	}

	if values["ADMIN_EMAIL"] == nil || values["ADMIN_PASSWORD"] == nil {
		unavailable("faltam ADMIN_EMAIL e ADMIN_PASSWORD no ambiente ou no base/.env")
	}

	url = "http://localhost:" + port

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	defer cancel()

	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, url+"/health", nil)

	response, err := http.DefaultClient.Do(request)

	if err != nil {
		unavailable("o backend nao esta no ar em %s (rode go run . dentro de base/): %v", url, err)
	}

	response.Body.Close()

	return url, *values["ADMIN_EMAIL"], *values["ADMIN_PASSWORD"]
}

func randomID(t *testing.T) string {
	t.Helper()

	data := make([]byte, 6)

	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}

	return hex.EncodeToString(data)
}

func TestBackend(t *testing.T) {
	url, adminEmail, adminPassword := target(t)

	anonymous := newAPI(t, url)
	admin := newAPI(t, url)
	user := newAPI(t, url)
	other := newAPI(t, url)

	t.Run("health", func(t *testing.T) {
		var health schema.HealthResponse

		anonymous.call("GET", "/health", nil, 200, &health)

		if !health.Status {
			t.Fatal("o backend deveria estar de pe")
		}
	})

	t.Run("protected routes need login", func(t *testing.T) {
		anonymous.expect("GET", "/me", nil, 401)
		anonymous.expect("GET", "/farms", nil, 401)
		anonymous.expect("GET", "/users", nil, 401)
		anonymous.expect("POST", "/auth/refresh", nil, 401)
	})

	t.Run("admin login", func(t *testing.T) {
		admin.expect("POST", "/auth/login", schema.LoginRequest{Email: adminEmail, Password: "senha-errada-123"}, 401)

		status, data, response := admin.send("POST", "/auth/login", schema.LoginRequest{Email: adminEmail, Password: adminPassword})

		if status != 200 {
			t.Fatalf("login do admin: esperava 200, veio %d: %s", status, data)
		}

		var session schema.SessionResponse

		if err := json.Unmarshal(data, &session); err != nil || session.User.Role != schema.RoleAdmin || session.ExpiresIn <= 0 {
			t.Fatalf("sessao errada: %v %s", err, data)
		}

		// os tokens vao so em cookies HttpOnly, nunca no corpo
		for _, name := range []string{handles.AccessCookie, handles.RefreshCookie} {
			found := false

			for _, cookie := range response.Cookies() {
				if cookie.Name == name {
					found = true

					if !cookie.HttpOnly {
						t.Errorf("o cookie %s deveria ser HttpOnly", name)
					}

					if cookie.Secure {
						t.Fatalf("o cookie %s veio com Secure e nao funciona em http: ponha COOKIE_INSECURE=true no .env", name)
					}

					if strings.Contains(string(data), cookie.Value) {
						t.Errorf("o token do cookie %s apareceu no corpo da resposta", name)
					}
				}
			}

			if !found {
				t.Fatalf("faltou o cookie %s", name)
			}
		}

		var me schema.UserResponse

		admin.call("GET", "/me", nil, 200, &me)

		if me.Email != adminEmail {
			t.Fatalf("o /me deveria ser o admin, veio %s", me.Email)
		}
	})

	// um usuario so deste teste, apagado no fim
	email := "functional-" + randomID(t) + "@example.com"
	password := "senha-funcional-" + randomID(t)

	var created schema.UserResponse

	t.Run("register and login as a normal user", func(t *testing.T) {
		anonymous.call("POST", "/auth/register", schema.RegisterRequest{Name: "Funcional", Email: email, Password: password}, 201, &created)

		if created.Role != schema.RoleUser || !created.HasPassword {
			t.Fatalf("usuario criado errado: %+v", created)
		}

		anonymous.expect("POST", "/auth/register", schema.RegisterRequest{Name: "Funcional", Email: email, Password: password}, 409)
		anonymous.expect("POST", "/auth/register", schema.RegisterRequest{Name: "", Email: "invalido", Password: "1"}, 422)

		user.expect("POST", "/auth/login", schema.LoginRequest{Email: email, Password: password}, 200)
		user.expect("GET", "/users", nil, 403)
		user.expect("PATCH", "/me", map[string]string{"role": "admin"}, 403)
	})

	t.Cleanup(func() {
		if created.ID != 0 {
			admin.send("DELETE", fmt.Sprintf("/users/%d", created.ID), nil)
		}
	})

	if created.ID == 0 {
		t.Fatal("sem o usuario de teste nao da para seguir")
	}

	var farm schema.FarmResponse

	var tank schema.TankResponse

	var otherUser schema.UserResponse

	t.Cleanup(func() {
		if otherUser.ID != 0 {
			admin.send("DELETE", fmt.Sprintf("/users/%d", otherUser.ID), nil)
		}
	})

	t.Run("farms and tanks", func(t *testing.T) {
		user.call("POST", "/farms", schema.CreateFarmRequest{Name: "Fazenda Funcional", City: "Uberaba", State: "mg"}, 201, &farm)

		if farm.UserID != created.ID || farm.State != "MG" {
			t.Fatalf("fazenda errada: %+v", farm)
		}

		user.expect("POST", "/farms", schema.CreateFarmRequest{Name: "x", City: "y", State: "XX"}, 422)

		farmPath := fmt.Sprintf("/farms/%d", farm.ID)

		user.call("PATCH", farmPath, schema.UpdateFarmRequest{City: ptr("Araxa")}, 200, &farm)

		if farm.City != "Araxa" {
			t.Fatalf("o PATCH nao mudou a cidade: %+v", farm)
		}

		var farms []schema.FarmResponse

		user.call("GET", "/farms", nil, 200, &farms)

		if len(farms) != 1 || farms[0].ID != farm.ID {
			t.Fatalf("esperava so a fazenda do usuario: %+v", farms)
		}

		// outro usuario nao ve a fazenda; o admin ve
		otherEmail, otherPassword := "functional-"+randomID(t)+"@example.com", "senha-outro-"+randomID(t)

		anonymous.call("POST", "/auth/register", schema.RegisterRequest{Name: "Outro", Email: otherEmail, Password: otherPassword}, 201, &otherUser)
		other.expect("POST", "/auth/login", schema.LoginRequest{Email: otherEmail, Password: otherPassword}, 200)

		other.expect("GET", farmPath, nil, 404)
		admin.expect("GET", farmPath, nil, 200)

		user.call("POST", farmPath+"/tanks", schema.CreateTankRequest{Type: "expansao", CapacityLiters: 1500}, 201, &tank)

		if tank.FarmID != farm.ID {
			t.Fatalf("tanque errado: %+v", tank)
		}

		user.expect("POST", farmPath+"/tanks", schema.CreateTankRequest{Type: "x", CapacityLiters: 0}, 422)
		user.expect("GET", fmt.Sprintf("/tanks/%d", tank.ID), nil, 200)
	})

	// a limpeza e feita pelo admin, porque no fim o usuario de teste ja saiu (logout)
	t.Cleanup(func() {
		if tank.ID != 0 {
			admin.send("DELETE", fmt.Sprintf("/tanks/%d", tank.ID), nil)
		}

		if farm.ID != 0 {
			admin.send("DELETE", fmt.Sprintf("/farms/%d", farm.ID), nil)
		}
	})

	if farm.ID == 0 || tank.ID == 0 {
		t.Fatal("sem a fazenda e o tanque de teste nao da para seguir")
	}

	var analysisIDs []int64

	t.Cleanup(func() {
		for _, id := range analysisIDs {
			admin.send("DELETE", fmt.Sprintf("/analyses/%d", id), nil)
		}
	})

	analysesPath := fmt.Sprintf("/farms/%d/analyses", farm.ID)

	t.Run("analysis with a sample", func(t *testing.T) {
		sample := &schema.Sample{Temperature: ptr(3.6), CCS: ptr(252.0), CPP: ptr(273.0), Date: ptr("2025-09-01"),
			State: ptr("MG"), City: ptr("Uberaba"), Tank: ptr("TC01")}

		status, data, _ := user.send("POST", analysesPath, schema.CreateAnalysisRequest{TankID: &tank.ID, Sample: sample})

		if status == http.StatusServiceUnavailable {
			t.Skipf("o pipeline nao esta no ar (rode o pipeline na porta 8000): %s", data)
		}

		if status != 201 {
			t.Fatalf("esperava 201, veio %d: %s", status, data)
		}

		var analysis schema.AnalysisResponse

		if err := json.Unmarshal(data, &analysis); err != nil {
			t.Fatal(err)
		}

		analysisIDs = append(analysisIDs, analysis.ID)

		if analysis.Result != "aprovado" && analysis.Result != "reprovado" {
			t.Fatalf("resposta do modelo inesperada: %q", analysis.Result)
		}

		if analysis.Content == nil || analysis.Content.Kind != "content" || analysis.TankID == nil || *analysis.TankID != tank.ID {
			t.Fatalf("analise errada: %+v", analysis)
		}

		// arquivo e amostra juntos, ou nenhum dos dois, nao pode
		user.expect("POST", analysesPath, schema.CreateAnalysisRequest{}, 422)
		user.expect("POST", analysesPath, schema.CreateAnalysisRequest{TankID: ptr(int64(999999)), Sample: sample}, 422)

		var contents []schema.ContentResponse

		user.call("GET", fmt.Sprintf("/analyses/%d/contents", analysis.ID), nil, 200, &contents)

		if len(contents) != 1 || contents[0].Kind != "content" || !strings.Contains(string(contents[0].Content), "temperature") {
			t.Fatalf("conteudo errado: %+v", contents)
		}
	})

	t.Run("analysis with a file", func(t *testing.T) {
		file := readTestFile(t)

		status, data := uploadAnalysis(t, user, analysesPath, tank.ID, "", file)

		if status == http.StatusServiceUnavailable {
			t.Skipf("o pipeline nao esta no ar: %s", data)
		}

		if status != 201 {
			t.Fatalf("esperava 201, veio %d: %.300s", status, data)
		}

		analysis := decodeAnalysis(t, data)

		analysisIDs = append(analysisIDs, analysis.ID)

		if len(analysis.Rows) == 0 || analysis.Content == nil || analysis.Content.Kind != "file" || analysis.Content.FileSize != len(file) {
			t.Fatalf("analise de arquivo errada: %d linhas, conteudo %+v", len(analysis.Rows), analysis.Content)
		}

		if _, ok := analysis.Rows[0]["predict"]; !ok {
			t.Fatalf("faltou a coluna predict: %v", analysis.Rows[0])
		}
	})

	// como o tests/functional/api.py do pipeline: cada modelo recebe o arquivo de teste e uma amostra
	t.Run("every model with the test file and a sample", func(t *testing.T) {
		var models schema.ModelsResponse

		status, data, _ := user.send("GET", "/models", nil)

		if status == http.StatusServiceUnavailable {
			t.Skipf("o pipeline nao esta no ar: %s", data)
		}

		if status != 200 || json.Unmarshal(data, &models) != nil || len(models.Models) == 0 {
			t.Fatalf("GET /models: %d %s", status, data)
		}

		file := readTestFile(t)
		sample := &schema.Sample{Temperature: ptr(3.6), CCS: ptr(252.0), CPP: ptr(273.0), Date: ptr("2025-09-01"),
			State: ptr("MG"), City: ptr("Uberaba"), Tank: ptr("TC01")}

		rowsPerModel := 0

		for _, model := range models.Models {
			t.Run(model, func(t *testing.T) {
				// o arquivo de teste inteiro
				status, data := uploadAnalysis(t, user, analysesPath, tank.ID, model, file)

				if status != 201 {
					t.Fatalf("arquivo: esperava 201, veio %d: %.300s", status, data)
				}

				fromFile := decodeAnalysis(t, data)

				analysisIDs = append(analysisIDs, fromFile.ID)

				if len(fromFile.Rows) == 0 || fromFile.Content == nil || fromFile.Content.Kind != "file" {
					t.Fatalf("analise de arquivo errada: %d linhas, conteudo %+v", len(fromFile.Rows), fromFile.Content)
				}

				for i, row := range fromFile.Rows {
					if predict, _ := row["predict"].(string); predict != "aprovado" && predict != "reprovado" {
						t.Fatalf("linha %d: predict inesperado: %v", i, row["predict"])
					}
				}

				// todos os modelos devolvem uma linha para cada linha do arquivo
				if rowsPerModel == 0 {
					rowsPerModel = len(fromFile.Rows)
				} else if len(fromFile.Rows) != rowsPerModel {
					t.Fatalf("esperava %d linhas como nos outros modelos, veio %d", rowsPerModel, len(fromFile.Rows))
				}

				// a amostra, com o mesmo modelo
				status, data, _ = user.send("POST", analysesPath, schema.CreateAnalysisRequest{TankID: &tank.ID, Model: model, Sample: sample})

				if status != 201 {
					t.Fatalf("amostra: esperava 201, veio %d: %.300s", status, data)
				}

				fromSample := decodeAnalysis(t, data)

				analysisIDs = append(analysisIDs, fromSample.ID)

				if fromSample.Result != "aprovado" && fromSample.Result != "reprovado" {
					t.Fatalf("amostra: resposta do modelo inesperada: %q", fromSample.Result)
				}
			})
		}

		// um modelo que nao existe e recusado e nao deixa analise no banco
		before := len(analysisIDs)

		status, _ = uploadAnalysis(t, user, analysesPath, tank.ID, "modelo-que-nao-existe", file)

		if status < 400 {
			t.Fatalf("modelo inexistente deveria dar erro, veio %d", status)
		}

		var analyses []schema.AnalysisResponse

		user.call("GET", analysesPath, nil, 200, &analyses)

		if len(analyses) != before {
			t.Fatalf("a analise com modelo inexistente nao deveria ser gravada: %d no banco, %d esperadas", len(analyses), before)
		}
	})

	t.Run("list analyses and models", func(t *testing.T) {
		if len(analysisIDs) == 0 {
			t.Skip("nenhuma analise foi criada (pipeline fora do ar)")
		}

		var analyses []schema.AnalysisResponse

		user.call("GET", analysesPath, nil, 200, &analyses)

		if len(analyses) != len(analysisIDs) {
			t.Fatalf("esperava %d analises, veio %d", len(analysisIDs), len(analyses))
		}

		// outro usuario nao ve as analises
		other.expect("GET", analysesPath, nil, 404)

		var models schema.ModelsResponse

		user.call("GET", "/models", nil, 200, &models)

		if len(models.Models) == 0 {
			t.Fatal("o pipeline deveria ter modelos")
		}
	})

	t.Run("refresh rotates the token and logout revokes it", func(t *testing.T) {
		oldRefresh := user.cookie(handles.RefreshCookie, "/auth/refresh")

		if oldRefresh == nil {
			t.Fatal("faltou o cookie de refresh para /auth")
		}

		user.expect("POST", "/auth/refresh", nil, 200)

		newRefresh := user.cookie(handles.RefreshCookie, "/auth/refresh")

		if newRefresh == nil || newRefresh.Value == oldRefresh.Value {
			t.Fatal("o refresh token deveria ser trocado")
		}

		user.expect("GET", "/me", nil, 200)

		// o refresh antigo, usado de novo, e recusado
		reuse := newAPI(t, url)

		reuse.expect("POST", "/auth/refresh", nil, 401)

		status, _, _ := reuse.send("POST", "/auth/refresh", nil, "Cookie", handles.RefreshCookie+"="+oldRefresh.Value)

		if status != 401 {
			t.Fatalf("o refresh token ja usado deveria dar 401, veio %d", status)
		}

		user.expect("POST", "/auth/logout", nil, 200)
		user.expect("POST", "/auth/refresh", nil, 401)
		user.expect("GET", "/me", nil, 401)
	})
}

// readTestFile le o arquivo de teste do pipeline.
func readTestFile(t *testing.T) []byte {
	t.Helper()

	file, err := os.ReadFile(testParquet)

	if err != nil {
		t.Fatalf("arquivo de teste: %v", err)
	}

	return file
}

// uploadAnalysis envia o arquivo como multipart para POST /farms/{id}/analyses (model vazio usa o padrao).
func uploadAnalysis(t *testing.T, client *api, path string, tankID int64, model string, file []byte) (int, []byte) {
	t.Helper()

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	writer.WriteField("tank_id", fmt.Sprint(tankID))

	if model != "" {
		writer.WriteField("model", model)
	}

	part, err := writer.CreateFormFile("file", "analise_leite_teste.parquet")

	if err != nil {
		t.Fatal(err)
	}

	part.Write(file)
	writer.Close()

	status, data, _ := client.send("POST", path, &body, "Content-Type", writer.FormDataContentType())

	return status, data
}

func decodeAnalysis(t *testing.T, data []byte) schema.AnalysisResponse {
	t.Helper()

	var analysis schema.AnalysisResponse

	if err := json.Unmarshal(data, &analysis); err != nil {
		t.Fatalf("resposta que nao e uma analise: %v: %.300s", err, data)
	}

	return analysis
}

func ptr[T any](value T) *T {
	return &value
}
