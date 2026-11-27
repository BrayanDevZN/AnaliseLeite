// Package handles tem os handlers das rotas do backend (as rotas ficam descritas em schema).
//
// O depends.go e a injecao de dependencias de login: valida o token do usuario e entrega o payload dele,
// parecido com o Depends do FastAPI. O handles.go liga cada handler a sua rota e aplica o login e o papel.
package handles

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/auth"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/schema"
)

// Nomes dos cookies dos tokens.
const (
	AccessCookie  = "access_token"
	RefreshCookie = "refresh_token"
)

// ErrUnauthorized quando falta o token, ele e invalido ou vencido, ou o usuario dele nao existe mais.
var ErrUnauthorized = errors.New("unauthorized")

// Payload e o que o token de acesso carrega.
type Payload struct {
	UserID    int64
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// Principal e quem esta logado: o payload do token e o usuario atual do banco
// (assim o papel e sempre o de agora, e um usuario apagado perde o acesso na hora).
type Principal struct {
	Payload *Payload
	User    *operations.User
}

// IsAdmin diz se o usuario e admin.
func (p *Principal) IsAdmin() bool {
	return p.User.Role == schema.RoleAdmin
}

// CanAccessUser diz se o usuario pode mexer nos dados do usuario id (ele mesmo, ou qualquer um se for admin).
func (p *Principal) CanAccessUser(id int64) bool {
	return p.IsAdmin() || p.User.ID == id
}

// Depends valida o token do usuario. Use Payload para so conferir o token, ou Principal para
// conferir o token e carregar o usuario do banco.
type Depends struct {
	access *auth.JWT
	users  *database.Users
}

// NewDepends cria o Depends com o JWT dos tokens de acesso e as operacoes de usuarios.
func NewDepends(access *auth.JWT, users *database.Users) *Depends {
	return &Depends{access: access, users: users}
}

// Token acha o token de acesso da requisicao: primeiro no cookie e, se nao tiver, no header
// Authorization: Bearer (para quem nao e navegador).
func (d *Depends) Token(r *http.Request) string {
	if cookie, err := r.Cookie(AccessCookie); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")

	if found && strings.EqualFold(scheme, "Bearer") {
		return strings.TrimSpace(token)
	}

	return ""
}

// Payload valida o token da requisicao e devolve o payload dele.
// Devolve ErrUnauthorized se faltar, estiver adulterado ou vencido. O refresh token nao serve aqui:
// o subject dele nao e so o id do usuario.
func (d *Depends) Payload(r *http.Request) (*Payload, error) {
	token := d.Token(r)

	if token == "" {
		return nil, ErrUnauthorized
	}

	claims, err := d.access.Read(token)

	if err != nil {
		return nil, ErrUnauthorized
	}

	id, err := strconv.ParseInt(claims.Subject, 10, 64)

	if err != nil || id <= 0 {
		return nil, ErrUnauthorized
	}

	return &Payload{UserID: id, IssuedAt: claims.IssuedAt, ExpiresAt: claims.ExpiresAt}, nil
}

// Principal valida o token e carrega o usuario do banco (pelo cache, se houver).
func (d *Depends) Principal(ctx context.Context, r *http.Request) (*Principal, error) {
	payload, err := d.Payload(r)

	if err != nil {
		return nil, err
	}

	user, err := d.users.GetByID(ctx, payload.UserID)

	if errors.Is(err, operations.ErrNotFound) {
		return nil, ErrUnauthorized
	}

	if err != nil {
		return nil, err
	}

	return &Principal{Payload: payload, User: user}, nil
}

// Identify devolve o id do usuario logado como texto, ou "" se nao tem login valido.
// Serve para o Identify do rate limit (midlleware.Config.Identify): so confere o token, sem ir ao banco.
func (d *Depends) Identify(r *http.Request) string {
	payload, err := d.Payload(r)

	if err != nil {
		return ""
	}

	return strconv.FormatInt(payload.UserID, 10)
}
