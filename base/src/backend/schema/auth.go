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
	v.gmail("email", r.Email)
	v.newPassword("password", r.Password)

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

// SessionResponse e a resposta de POST /auth/login e POST /auth/refresh.
// Os tokens nunca vao no corpo: vao em cookies HttpOnly. Aqui so vem quanto tempo falta para o
// token de acesso vencer e quem esta logado.
type SessionResponse struct {
	// ExpiresIn e em quantos segundos o token de acesso vence.
	ExpiresIn int          `json:"expires_in"`
	User      UserResponse `json:"user"`
}

// NewSessionResponse monta a resposta de login a partir do tempo de vida do token e do usuario.
func NewSessionResponse(ttl time.Duration, user UserResponse) SessionResponse {
	return SessionResponse{ExpiresIn: int(ttl.Seconds()), User: user}
}

// MessageResponse e uma resposta so com uma mensagem.
type MessageResponse struct {
	Message string `json:"message"`
}

// AuthRoutes sao as rotas de login, logout, renovacao do token e cadastro.
// O login e a renovacao devolvem os tokens em cookies HttpOnly (access_token e refresh_token).
var AuthRoutes = []Route{
	{Method: http.MethodPost, Path: "/auth/register", Description: "cria uma conta", Public: true, Request: RegisterRequest{}, Response: UserResponse{}},
	{Method: http.MethodPost, Path: "/auth/login", Description: "entra com email e senha (tokens nos cookies)", Public: true, Request: LoginRequest{}, Response: SessionResponse{}},
	{Method: http.MethodPost, Path: "/auth/refresh", Description: "troca o refresh token por tokens novos (cookies)", Public: true, Response: SessionResponse{}},
	{Method: http.MethodPost, Path: "/auth/logout", Description: "sai: invalida o refresh token e limpa os cookies", Public: true, Response: MessageResponse{}},
	{Method: http.MethodGet, Path: "/auth/google", Description: "comeca o login com o Google (redireciona para o Google)", Public: true},
	{Method: http.MethodGet, Path: "/auth/google/callback", Description: "volta do Google: entra e redireciona (tokens nos cookies)", Public: true},
	{Method: http.MethodGet, Path: "/me", Description: "dados do usuario logado", Response: UserResponse{}},
	{Method: http.MethodPatch, Path: "/me", Description: "atualiza o proprio usuario (nao muda o papel)", Request: UpdateUserRequest{}, Response: UserResponse{}},
}
