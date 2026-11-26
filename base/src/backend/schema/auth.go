package schema

import (
	"net/http"
	"strings"
	"time"
)

// RegisterRequest e o corpo de POST /auth/register (cadastro publico, sempre cria um usuario comum).
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate tira os espacos, poe o email em minusculas e valida os campos.
func (r *RegisterRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.Email = normalizeEmail(r.Email)

	var v validator

	v.text("name", r.Name, maxNameLength)
	v.email("email", r.Email)
	v.password("password", r.Password)

	return v.err()
}

// LoginRequest e o corpo de POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate poe o email em minusculas e confere que os dois campos vieram.
// Nao confere o tamanho da senha: quem erra a senha so recebe "credenciais invalidas".
func (r *LoginRequest) Validate() error {
	r.Email = normalizeEmail(r.Email)

	var v validator

	v.email("email", r.Email)

	if r.Password == "" {
		v.add("password", "is required")
	}

	return v.err()
}

// TokenResponse e a resposta de POST /auth/login.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`

	// ExpiresIn e em quantos segundos o token vence.
	ExpiresIn int `json:"expires_in"`
}

// NewTokenResponse monta a resposta de login a partir do token e do tempo de vida dele.
func NewTokenResponse(token string, ttl time.Duration) TokenResponse {
	return TokenResponse{AccessToken: token, TokenType: "bearer", ExpiresIn: int(ttl.Seconds())}
}

// AuthRoutes sao as rotas de login e cadastro.
var AuthRoutes = []Route{
	{Method: http.MethodPost, Path: "/auth/register", Description: "cria uma conta", Public: true, Request: RegisterRequest{}, Response: UserResponse{}},
	{Method: http.MethodPost, Path: "/auth/login", Description: "entra com email e senha", Public: true, Request: LoginRequest{}, Response: TokenResponse{}},
	{Method: http.MethodGet, Path: "/me", Description: "dados do usuario logado", Response: UserResponse{}},
}
