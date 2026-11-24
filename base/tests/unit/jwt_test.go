package unit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/auth"
)

const (
	testSecret      = "uma-chave-de-teste-com-mais-de-trinta-e-dois-caracteres"
	otherTestSecret = "outra-chave-de-teste-tambem-com-mais-de-32-caracteres"
)

var fixedNow = time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)

// newTestJWT cria um JWT com a chave de teste, um TTL de 1 hora e um relogio parado em fixedNow.
func newTestJWT(t *testing.T) (*auth.JWT, func() string) {
	t.Helper()

	t.Setenv(auth.SecretName, testSecret)

	logger, buffer := newTestLogger()

	jwt, err := auth.NewJWT(logger, time.Hour)

	if err != nil {
		t.Fatalf("nao esperava erro: %v", err)
	}

	jwt.SetClock(func() time.Time { return fixedNow })

	return jwt, buffer.String
}

func encode(t *testing.T, value any) string {
	t.Helper()

	data, err := json.Marshal(value)

	if err != nil {
		t.Fatal(err)
	}

	return base64.RawURLEncoding.EncodeToString(data)
}

// signed monta um token com a chave de teste, de qualquer header e payload.
func signed(t *testing.T, head map[string]any, body map[string]any) string {
	t.Helper()

	input := encode(t, head) + "." + encode(t, body)

	mac := hmac.New(sha256.New, []byte(testSecret))
	mac.Write([]byte(input))

	return input + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func TestJWTSecretNameIsJWTSecret(t *testing.T) {
	if auth.SecretName != "JWT_SECRET" {
		t.Fatalf("esperava JWT_SECRET, veio %q", auth.SecretName)
	}
}

func TestJWTCreateAndReadRoundTrip(t *testing.T) {
	jwt, _ := newTestJWT(t)

	token, err := jwt.Create("usuario-42")

	if err != nil {
		t.Fatalf("nao esperava erro ao criar: %v", err)
	}

	claims, err := jwt.Read(token)

	if err != nil {
		t.Fatalf("nao esperava erro ao ler: %v", err)
	}

	if claims.Subject != "usuario-42" {
		t.Errorf("subject: esperava usuario-42, veio %q", claims.Subject)
	}

	if !claims.IssuedAt.Equal(fixedNow) {
		t.Errorf("emitido em: esperava %v, veio %v", fixedNow, claims.IssuedAt)
	}

	if !claims.ExpiresAt.Equal(fixedNow.Add(time.Hour)) {
		t.Errorf("vence em: esperava %v, veio %v", fixedNow.Add(time.Hour), claims.ExpiresAt)
	}
}

func TestJWTTokenHasThreeParts(t *testing.T) {
	jwt, _ := newTestJWT(t)

	token, _ := jwt.Create("a")

	if parts := strings.Split(token, "."); len(parts) != 3 {
		t.Fatalf("esperava 3 partes separadas por ponto, veio %d", len(parts))
	}
}

func TestJWTHeaderIsHS256(t *testing.T) {
	jwt, _ := newTestJWT(t)

	token, _ := jwt.Create("a")

	data, err := base64.RawURLEncoding.DecodeString(strings.Split(token, ".")[0])

	if err != nil {
		t.Fatal(err)
	}

	var head map[string]string

	if err := json.Unmarshal(data, &head); err != nil {
		t.Fatal(err)
	}

	if head["alg"] != "HS256" || head["typ"] != "JWT" {
		t.Fatalf("header errado: %v", head)
	}
}

func TestJWTPayloadHasStandardClaims(t *testing.T) {
	jwt, _ := newTestJWT(t)

	token, _ := jwt.Create("usuario-1")

	data, _ := base64.RawURLEncoding.DecodeString(strings.Split(token, ".")[1])

	var body map[string]any

	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatal(err)
	}

	if body["sub"] != "usuario-1" {
		t.Errorf("sub errado: %v", body["sub"])
	}

	if int64(body["iat"].(float64)) != fixedNow.Unix() {
		t.Errorf("iat errado: %v", body["iat"])
	}

	if int64(body["exp"].(float64)) != fixedNow.Add(time.Hour).Unix() {
		t.Errorf("exp errado: %v", body["exp"])
	}
}

func TestJWTDifferentSubjectsGiveDifferentTokens(t *testing.T) {
	jwt, _ := newTestJWT(t)

	first, _ := jwt.Create("a")
	second, _ := jwt.Create("b")

	if first == second {
		t.Fatal("subjects diferentes deveriam gerar tokens diferentes")
	}
}

func TestJWTCreateEmptySubject(t *testing.T) {
	jwt, _ := newTestJWT(t)

	token, err := jwt.Create("")

	if !errors.Is(err, auth.ErrEmptySubject) {
		t.Fatalf("esperava ErrEmptySubject, veio %v", err)
	}

	if token != "" {
		t.Fatalf("com erro, o token deveria ser vazio, veio %q", token)
	}
}

func TestJWTExpiredToken(t *testing.T) {
	jwt, _ := newTestJWT(t)

	token, _ := jwt.Create("a")

	jwt.SetClock(func() time.Time { return fixedNow.Add(2 * time.Hour) })

	claims, err := jwt.Read(token)

	if !errors.Is(err, auth.ErrExpiredToken) {
		t.Fatalf("esperava ErrExpiredToken, veio %v", err)
	}

	if claims != nil {
		t.Fatal("com erro, os dados deveriam ser nil")
	}
}

func TestJWTExpiryBoundary(t *testing.T) {
	jwt, _ := newTestJWT(t)

	token, _ := jwt.Create("a")

	jwt.SetClock(func() time.Time { return fixedNow.Add(time.Hour - time.Second) })

	if _, err := jwt.Read(token); err != nil {
		t.Fatalf("um segundo antes de vencer ainda deveria valer: %v", err)
	}

	jwt.SetClock(func() time.Time { return fixedNow.Add(time.Hour) })

	if _, err := jwt.Read(token); !errors.Is(err, auth.ErrExpiredToken) {
		t.Fatalf("no instante exato do vencimento o token deveria estar vencido, veio %v", err)
	}
}

func TestJWTTamperedPayload(t *testing.T) {
	jwt, _ := newTestJWT(t)

	token, _ := jwt.Create("usuario-comum")

	parts := strings.Split(token, ".")
	parts[1] = encode(t, map[string]any{"sub": "administrador", "iat": fixedNow.Unix(), "exp": fixedNow.Add(24 * time.Hour).Unix()})

	if _, err := jwt.Read(strings.Join(parts, ".")); !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("um payload alterado deveria ser invalido, veio %v", err)
	}
}

func TestJWTTamperedSignature(t *testing.T) {
	jwt, _ := newTestJWT(t)

	token, _ := jwt.Create("a")

	parts := strings.Split(token, ".")
	signature := []byte(parts[2])

	// troca o primeiro caractere da assinatura
	if signature[0] == 'A' {
		signature[0] = 'B'
	} else {
		signature[0] = 'A'
	}

	parts[2] = string(signature)

	if _, err := jwt.Read(strings.Join(parts, ".")); !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("uma assinatura alterada deveria ser invalida, veio %v", err)
	}
}

func TestJWTTokenFromAnotherSecretIsRejected(t *testing.T) {
	first, _ := newTestJWT(t)

	token, _ := first.Create("a")

	t.Setenv(auth.SecretName, otherTestSecret)

	logger, _ := newTestLogger()
	second, err := auth.NewJWT(logger, time.Hour)

	if err != nil {
		t.Fatal(err)
	}

	second.SetClock(func() time.Time { return fixedNow })

	if _, err := second.Read(token); !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("um token assinado com outra chave deveria ser invalido, veio %v", err)
	}
}

func TestJWTRejectsAlgorithmNone(t *testing.T) {
	jwt, _ := newTestJWT(t)

	body := map[string]any{"sub": "administrador", "iat": fixedNow.Unix(), "exp": fixedNow.Add(time.Hour).Unix()}

	for name, token := range map[string]string{
		"sem assinatura":      encode(t, map[string]any{"alg": "none", "typ": "JWT"}) + "." + encode(t, body) + ".",
		"alg None":            encode(t, map[string]any{"alg": "None", "typ": "JWT"}) + "." + encode(t, body) + ".",
		"com assinatura lixo": encode(t, map[string]any{"alg": "none", "typ": "JWT"}) + "." + encode(t, body) + ".abc",
	} {
		if _, err := jwt.Read(token); !errors.Is(err, auth.ErrInvalidToken) {
			t.Errorf("%s: esperava ErrInvalidToken, veio %v", name, err)
		}
	}
}

func TestJWTRejectsOtherAlgorithmEvenWithValidSignature(t *testing.T) {
	jwt, _ := newTestJWT(t)

	body := map[string]any{"sub": "a", "iat": fixedNow.Unix(), "exp": fixedNow.Add(time.Hour).Unix()}

	token := signed(t, map[string]any{"alg": "HS512", "typ": "JWT"}, body)

	if _, err := jwt.Read(token); !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("o header diz HS512, so HS256 deveria ser aceito, veio %v", err)
	}
}

func TestJWTAcceptsATokenSignedByTheSameRules(t *testing.T) {
	jwt, _ := newTestJWT(t)

	body := map[string]any{"sub": "montado-na-mao", "iat": fixedNow.Unix(), "exp": fixedNow.Add(time.Hour).Unix()}

	claims, err := jwt.Read(signed(t, map[string]any{"alg": "HS256", "typ": "JWT"}, body))

	if err != nil || claims.Subject != "montado-na-mao" {
		t.Fatalf("um token valido montado a mao deveria ser aceito (claims=%v err=%v)", claims, err)
	}
}

func TestJWTRejectsTokenWithoutSubject(t *testing.T) {
	jwt, _ := newTestJWT(t)

	body := map[string]any{"iat": fixedNow.Unix(), "exp": fixedNow.Add(time.Hour).Unix()}

	if _, err := jwt.Read(signed(t, map[string]any{"alg": "HS256", "typ": "JWT"}, body)); !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("um token sem sub deveria ser invalido, veio %v", err)
	}
}

func TestJWTRejectsTokenWithoutExpiration(t *testing.T) {
	jwt, _ := newTestJWT(t)

	body := map[string]any{"sub": "a", "iat": fixedNow.Unix()}

	if _, err := jwt.Read(signed(t, map[string]any{"alg": "HS256", "typ": "JWT"}, body)); !errors.Is(err, auth.ErrExpiredToken) {
		t.Fatalf("um token sem exp nunca deveria valer para sempre, veio %v", err)
	}
}

func TestJWTMalformedTokens(t *testing.T) {
	jwt, _ := newTestJWT(t)

	for name, token := range map[string]string{
		"vazio":            "",
		"uma parte":        "abc",
		"duas partes":      "abc.def",
		"quatro partes":    "a.b.c.d",
		"so pontos":        "..",
		"base64 invalido":  "!!!.!!!.!!!",
		"json invalido":    base64.RawURLEncoding.EncodeToString([]byte("nao-e-json")) + "." + base64.RawURLEncoding.EncodeToString([]byte("nao-e-json")) + ".abc",
		"texto com espaco": "um token com espacos",
	} {
		claims, err := jwt.Read(token)

		if !errors.Is(err, auth.ErrInvalidToken) {
			t.Errorf("%s: esperava ErrInvalidToken, veio %v", name, err)
		}

		if claims != nil {
			t.Errorf("%s: com erro, os dados deveriam ser nil", name)
		}
	}
}

func TestJWTMissingSecretIsAnError(t *testing.T) {
	t.Setenv(auth.SecretName, "")

	logger, logs := newTestLogger()

	jwt, err := auth.NewJWT(logger, time.Hour)

	var missing *backend.MissingSecretsError

	if !errors.As(err, &missing) {
		t.Fatalf("esperava *MissingSecretsError, veio %v", err)
	}

	if len(missing.Names) != 1 || missing.Names[0] != auth.SecretName {
		t.Errorf("deveria apontar %s como a secret que falta: %v", auth.SecretName, missing.Names)
	}

	if jwt != nil {
		t.Error("sem a secret, o JWT deveria ser nil")
	}

	if !strings.Contains(logs.String(), "[ERROR] => ") {
		t.Errorf("esperava um log de erro: %q", logs.String())
	}
}

func TestJWTWeakSecretIsAnError(t *testing.T) {
	t.Setenv(auth.SecretName, "curta-demais")

	logger, _ := newTestLogger()

	if _, err := auth.NewJWT(logger, time.Hour); !errors.Is(err, auth.ErrWeakSecret) {
		t.Fatalf("esperava ErrWeakSecret, veio %v", err)
	}
}

func TestJWTSecretWithMinimumLengthIsAccepted(t *testing.T) {
	t.Setenv(auth.SecretName, strings.Repeat("x", 32))

	logger, _ := newTestLogger()

	if _, err := auth.NewJWT(logger, time.Hour); err != nil {
		t.Fatalf("32 caracteres deveriam bastar: %v", err)
	}
}

func TestJWTInvalidTTL(t *testing.T) {
	t.Setenv(auth.SecretName, testSecret)

	logger, _ := newTestLogger()

	for _, ttl := range []time.Duration{0, -time.Second} {
		if _, err := auth.NewJWT(logger, ttl); !errors.Is(err, auth.ErrInvalidTTL) {
			t.Errorf("ttl %v: esperava ErrInvalidTTL, veio %v", ttl, err)
		}
	}
}

func TestJWTWithNilLoggerDoesNotPanic(t *testing.T) {
	t.Setenv(auth.SecretName, testSecret)

	jwt, err := auth.NewJWT(nil, time.Hour)

	if err != nil {
		t.Fatalf("nao esperava erro: %v", err)
	}

	token, _ := jwt.Create("a")

	if _, err := jwt.Read(token); err != nil {
		t.Fatalf("nao esperava erro: %v", err)
	}
}

func TestJWTLogsWithoutLeakingSecretOrToken(t *testing.T) {
	jwt, logs := newTestJWT(t)

	token, _ := jwt.Create("usuario-1")
	jwt.Read(token)
	jwt.Read("lixo")

	output := logs()

	for _, want := range []string{
		"Loading JWT secret", "JWT secret loaded with success",
		"Generating token", "Token generated with success",
		"Reading token", "Token read with success",
		"[WARNING] => ", "Token rejected",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("faltou %q nos logs: %q", want, output)
		}
	}

	for _, secret := range []string{testSecret, token, strings.Split(token, ".")[2]} {
		if strings.Contains(output, secret) {
			t.Fatalf("os logs nao podem conter a chave nem o token: %q", output)
		}
	}
}

func TestJWTLogsAnErrorWhenSubjectIsEmpty(t *testing.T) {
	jwt, logs := newTestJWT(t)

	jwt.Create("")

	if !strings.Contains(logs(), "[ERROR] => ") {
		t.Fatalf("esperava um log de erro: %q", logs())
	}
}

func TestJWTLogsExpiredTokenAsWarning(t *testing.T) {
	jwt, logs := newTestJWT(t)

	token, _ := jwt.Create("a")

	jwt.SetClock(func() time.Time { return fixedNow.Add(2 * time.Hour) })
	jwt.Read(token)

	if !strings.Contains(logs(), "token expired") {
		t.Fatalf("esperava o aviso de token vencido: %q", logs())
	}
}

func TestJWTConcurrentUse(t *testing.T) {
	jwt, _ := newTestJWT(t)

	var wait sync.WaitGroup

	for i := 0; i < 50; i++ {
		wait.Add(1)

		go func() {
			defer wait.Done()

			token, err := jwt.Create("usuario")

			if err != nil {
				t.Errorf("erro ao criar: %v", err)

				return
			}

			if _, err := jwt.Read(token); err != nil {
				t.Errorf("erro ao ler: %v", err)
			}
		}()
	}

	wait.Wait()
}
