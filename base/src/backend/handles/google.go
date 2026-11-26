package handles

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/schema"
)

// Login com o Google: fluxo Authorization Code com PKCE, todo no backend.
//
//	GET /auth/google           manda o navegador para a tela de login do Google
//	GET /auth/google/callback  o Google volta para ca; o backend troca o code pelo ID token, acha ou cria o
//	                           usuario, poe os cookies de sessao (os mesmos do login com senha) e redireciona
//
// Variaveis (do ambiente ou do base/.env). As tres primeiras ligam o login com o Google; sem elas as rotas
// respondem 503, e se so algumas existirem o backend nao inicia:
//
//	GOOGLE_CLIENT_ID      ID do cliente OAuth criado no Google Cloud Console
//	GOOGLE_CLIENT_SECRET  segredo do cliente OAuth
//	GOOGLE_REDIRECT_URL   URL deste callback, igual a cadastrada no Google (ex.: http://localhost:8080/auth/google/callback)
//	GOOGLE_SUCCESS_URL    para onde o navegador vai depois de entrar (opcional, padrao "/")
//	GOOGLE_FAILURE_URL    para onde vai se der erro, com ?error=<motivo> (opcional, padrao GOOGLE_SUCCESS_URL)
const (
	googleStateCookie  = "google_oauth"
	googleStateTTL     = 10 * time.Minute
	googleCallbackPath = "/auth/google/callback"
	googleMaxBody      = 1 << 20
)

// googleEndpoints sao os enderecos do Google.
var googleEndpoints = struct{ auth, token string }{
	auth:  "https://accounts.google.com/o/oauth2/v2/auth",
	token: "https://oauth2.googleapis.com/token",
}

var (
	errNoGoogle = &httpError{http.StatusServiceUnavailable, "google login is not configured"}

	errGoogleEmailNotVerified = errors.New("google email is not verified")
	errGoogleEmailNotAllowed  = errors.New("google email is not from an allowed domain")
)

// googleAuth e a configuracao do login com o Google.
type googleAuth struct {
	clientID     string
	clientSecret string
	redirectURL  string
	successURL   string
	failureURL   string
	client       *http.Client
}

// newGoogleAuth le as variaveis do Google. Devolve nil (sem erro) se o login com o Google nao esta configurado.
func newGoogleAuth(logger *backend.Logger) (*googleAuth, error) {
	config, err := backend.NewConfig([]string{
		"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "GOOGLE_REDIRECT_URL", "GOOGLE_SUCCESS_URL", "GOOGLE_FAILURE_URL",
	}, nil)

	if err != nil {
		return nil, err
	}

	values := config.GetSecrets()

	text := func(name, fallback string) string {
		if value := values[name]; value != nil {
			return *value
		}

		return fallback
	}

	required := []string{"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "GOOGLE_REDIRECT_URL"}

	var missing []string

	for _, name := range required {
		if values[name] == nil {
			missing = append(missing, name)
		}
	}

	switch {
	case len(missing) == len(required):
		logger.Warning("Google login is disabled: %s are not set", strings.Join(required, ", "))

		return nil, nil

	case len(missing) > 0:
		return nil, fmt.Errorf("google login needs %s too", strings.Join(missing, ", "))
	}

	google := &googleAuth{
		clientID:     *values["GOOGLE_CLIENT_ID"],
		clientSecret: *values["GOOGLE_CLIENT_SECRET"],
		redirectURL:  *values["GOOGLE_REDIRECT_URL"],
		successURL:   text("GOOGLE_SUCCESS_URL", "/"),
		client:       &http.Client{Timeout: 10 * time.Second},
	}

	google.failureURL = text("GOOGLE_FAILURE_URL", google.successURL)

	if err := google.validate(); err != nil {
		return nil, err
	}

	logger.Success("Google login is enabled")

	return google, nil
}

// validate confere as URLs. O redirect do final do login vem so daqui, nunca da requisicao,
// entao nao tem como alguem de fora mandar o usuario para outro site.
func (g *googleAuth) validate() error {
	callback, err := url.Parse(g.redirectURL)

	if err != nil || (callback.Scheme != "http" && callback.Scheme != "https") || callback.Host == "" {
		return errors.New("GOOGLE_REDIRECT_URL must be a full http or https url")
	}

	if !strings.HasSuffix(callback.Path, googleCallbackPath) {
		return fmt.Errorf("GOOGLE_REDIRECT_URL must end with %s", googleCallbackPath)
	}

	for name, value := range map[string]string{"GOOGLE_SUCCESS_URL": g.successURL, "GOOGLE_FAILURE_URL": g.failureURL} {
		parsed, err := url.Parse(value)

		relative := err == nil && parsed.Scheme == "" && parsed.Host == "" && strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//")
		absolute := err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""

		if !relative && !absolute {
			return fmt.Errorf("%s must be a path like / or a full http or https url", name)
		}
	}

	return nil
}

// googleStart: GET /auth/google. Guarda o state e o verifier do PKCE num cookie HttpOnly de 10 minutos
// e manda o navegador para o Google.
func (h *Handles) googleStart(w http.ResponseWriter, r *http.Request, _ *Principal) {
	if h.google == nil {
		h.fail(w, errNoGoogle)

		return
	}

	state, err := randomToken()

	if err != nil {
		h.fail(w, err)

		return
	}

	verifier, err := randomToken()

	if err != nil {
		h.fail(w, err)

		return
	}

	// o cookie precisa ser Lax: o Google volta por uma navegacao de outro site, e Strict nao mandaria o cookie
	http.SetCookie(w, &http.Cookie{
		Name:     googleStateCookie,
		Value:    state + "." + verifier,
		Path:     "/auth/google",
		Domain:   h.config.CookieDomain,
		MaxAge:   int(googleStateTTL.Seconds()),
		HttpOnly: true,
		Secure:   !h.config.InsecureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	challenge := sha256.Sum256([]byte(verifier))

	query := url.Values{
		"client_id":             {h.google.clientID},
		"redirect_uri":          {h.google.redirectURL},
		"response_type":         {"code"},
		"scope":                 {"openid email profile"},
		"state":                 {state},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
		"prompt":                {"select_account"},
	}

	w.Header().Set("Cache-Control", "no-store")

	http.Redirect(w, r, googleEndpoints.auth+"?"+query.Encode(), http.StatusFound)
}

// googleCallback: GET /auth/google/callback. Qualquer falha redireciona para GOOGLE_FAILURE_URL com ?error=<motivo>.
func (h *Handles) googleCallback(w http.ResponseWriter, r *http.Request, _ *Principal) {
	if h.google == nil {
		h.fail(w, errNoGoogle)

		return
	}

	w.Header().Set("Cache-Control", "no-store")

	cookie, cookieErr := r.Cookie(googleStateCookie)

	// o cookie vale uma vez so
	http.SetCookie(w, &http.Cookie{
		Name:     googleStateCookie,
		Path:     "/auth/google",
		Domain:   h.config.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   !h.config.InsecureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	query := r.URL.Query()

	if query.Get("error") != "" {
		h.googleFailure(w, r, "access_denied", nil)

		return
	}

	var state, verifier string

	if cookieErr == nil {
		state, verifier, _ = strings.Cut(cookie.Value, ".")
	}

	received := query.Get("state")

	if state == "" || verifier == "" || subtle.ConstantTimeCompare([]byte(state), []byte(received)) != 1 {
		h.googleFailure(w, r, "invalid_state", nil)

		return
	}

	code := query.Get("code")

	if code == "" {
		h.googleFailure(w, r, "invalid_request", nil)

		return
	}

	claims, err := h.google.exchange(r.Context(), code, verifier)

	if err != nil {
		h.googleFailure(w, r, "exchange_failed", err)

		return
	}

	user, err := h.googleUser(r.Context(), claims)

	switch {
	case errors.Is(err, errGoogleEmailNotVerified):
		h.googleFailure(w, r, "email_not_verified", nil)

		return

	case errors.Is(err, errGoogleEmailNotAllowed):
		h.googleFailure(w, r, "email_not_allowed", nil)

		return

	case err != nil:
		h.googleFailure(w, r, "server_error", err)

		return
	}

	if err := h.issueSession(w, user); err != nil {
		h.googleFailure(w, r, "server_error", err)

		return
	}

	http.Redirect(w, r, h.google.successURL, http.StatusFound)
}

// googleFailure redireciona para a pagina de erro com o motivo. O erro de verdade vai so para o log.
func (h *Handles) googleFailure(w http.ResponseWriter, r *http.Request, reason string, cause error) {
	if cause != nil {
		h.logger.Error("Google login failed (%s): %v", reason, cause)
	} else {
		h.logger.Warning("Google login refused (%s)", reason)
	}

	target, err := url.Parse(h.google.failureURL)

	if err != nil {
		http.Error(w, "google login failed", http.StatusBadGateway)

		return
	}

	query := target.Query()

	query.Set("error", reason)

	target.RawQuery = query.Encode()

	http.Redirect(w, r, target.String(), http.StatusFound)
}

// googleClaims sao os dados do ID token que o backend usa.
type googleClaims struct {
	Issuer        string `json:"iss"`
	Audience      string `json:"aud"`
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified any    `json:"email_verified"`
	Name          string `json:"name"`
	Expires       int64  `json:"exp"`
}

// exchange troca o code pelo ID token (com o verifier do PKCE) e confere os dados dele.
// O token vem direto do Google, por TLS, na resposta desta chamada; por isso, como a especificacao do OpenID
// Connect permite nesse caso, a assinatura nao e conferida de novo: o emissor, o publico (nosso client id) e a
// validade sao.
func (g *googleAuth) exchange(ctx context.Context, code, verifier string) (*googleClaims, error) {
	form := url.Values{
		"code":          {code},
		"client_id":     {g.clientID},
		"client_secret": {g.clientSecret},
		"redirect_uri":  {g.redirectURL},
		"grant_type":    {"authorization_code"},
		"code_verifier": {verifier},
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, googleEndpoints.token, strings.NewReader(form.Encode()))

	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")

	response, err := g.client.Do(request)

	if err != nil {
		return nil, fmt.Errorf("call google: %w", err)
	}

	defer response.Body.Close()

	data, err := io.ReadAll(io.LimitReader(response.Body, googleMaxBody))

	if err != nil {
		return nil, fmt.Errorf("read google response: %w", err)
	}

	// o corpo do erro nao vai para o log: pode ter dados do pedido
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google token endpoint answered %d", response.StatusCode)
	}

	var body struct {
		IDToken string `json:"id_token"`
	}

	if err := json.Unmarshal(data, &body); err != nil || body.IDToken == "" {
		return nil, errors.New("google did not return an id token")
	}

	return g.readIDToken(body.IDToken)
}

// readIDToken le o corpo do ID token e confere emissor, publico e validade.
func (g *googleAuth) readIDToken(token string) (*googleClaims, error) {
	parts := strings.Split(token, ".")

	if len(parts) != 3 {
		return nil, errors.New("malformed id token")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])

	if err != nil {
		return nil, errors.New("malformed id token")
	}

	var claims googleClaims

	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, errors.New("malformed id token")
	}

	if claims.Issuer != "https://accounts.google.com" && claims.Issuer != "accounts.google.com" {
		return nil, errors.New("id token has an unexpected issuer")
	}

	if subtle.ConstantTimeCompare([]byte(claims.Audience), []byte(g.clientID)) != 1 {
		return nil, errors.New("id token was issued for another client")
	}

	if claims.Expires <= time.Now().Add(-30*time.Second).Unix() {
		return nil, errors.New("id token expired")
	}

	if claims.Subject == "" || claims.Email == "" {
		return nil, errors.New("id token has no subject or email")
	}

	return &claims, nil
}

// verified diz se o Google garante que o email e da pessoa (vem como true ou "true").
func (c *googleClaims) verified() bool {
	switch value := c.EmailVerified.(type) {
	case bool:
		return value

	case string:
		return value == "true"
	}

	return false
}

// googleUser acha o usuario pelo email ou cria um (sem senha, papel user).
// Quem ja tem conta com senha e entra pelo Google com o mesmo email cai na MESMA conta: o Google garante que o
// email e dele (so entra com email verificado), e a senha continua valendo.
// Atencao: o cadastro com senha nao confirma o email. Se alguem se cadastrar com o email de outra pessoa, essa
// pessoa, ao entrar pelo Google, cai numa conta cuja senha quem se cadastrou conhece. O conserto de verdade e
// confirmar o email no cadastro.
func (h *Handles) googleUser(ctx context.Context, claims *googleClaims) (*operations.User, error) {
	if !claims.verified() {
		return nil, errGoogleEmailNotVerified
	}

	email := strings.ToLower(strings.TrimSpace(claims.Email))

	user, err := h.db.Users.GetByEmail(ctx, email)

	if err == nil {
		return user, nil
	}

	if !errors.Is(err, operations.ErrNotFound) {
		return nil, err
	}

	// so a CRIACAO de uma conta exige o dominio aceito: quem ja tem conta (de qualquer dominio) segue entrando
	if !schema.IsAllowedEmail(email) {
		return nil, errGoogleEmailNotAllowed
	}

	name := strings.TrimSpace(claims.Name)

	if name == "" {
		name, _, _ = strings.Cut(email, "@")
	}

	if utf8.RuneCountInString(name) > 120 {
		name = string([]rune(name)[:120])
	}

	created := &operations.User{Name: name, Email: email, Role: schema.RoleUser}

	if err := h.db.Users.Create(ctx, created); err != nil {
		// duas entradas ao mesmo tempo: a outra criou primeiro
		if isConflict(err) {
			return h.googleUser(ctx, claims)
		}

		return nil, err
	}

	return created, nil
}

// randomToken gera 32 bytes aleatorios em base64 sem padding (43 caracteres, o que o PKCE aceita).
func randomToken() (string, error) {
	data := make([]byte, 32)

	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(data), nil
}
