package handles

import (
	"errors"
	"net/http"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/schema"
)

// register: POST /auth/register. A senha vira hash bcrypt antes de ir para o banco.
func (h *Handles) register(w http.ResponseWriter, r *http.Request, _ *Principal) {
	var request schema.RegisterRequest

	if err := decodeJSON(w, r, &request); err != nil {
		h.fail(w, err)

		return
	}

	user, err := request.ToUser(h.hasher.Hash)

	if err != nil {
		h.fail(w, err)

		return
	}

	if err := h.db.Users.Create(r.Context(), user); err != nil {
		h.fail(w, err)

		return
	}

	writeJSON(w, http.StatusCreated, schema.UserFromModel(user))
}

// login: POST /auth/login. Os tokens voltam em cookies HttpOnly.
// Email que nao existe, conta sem senha e senha errada dao a mesma resposta (e o mesmo tempo).
func (h *Handles) login(w http.ResponseWriter, r *http.Request, _ *Principal) {
	var request schema.LoginRequest

	if err := decodeJSON(w, r, &request); err != nil {
		h.fail(w, err)

		return
	}

	if err := request.Validate(); err != nil {
		h.fail(w, err)

		return
	}

	user, err := h.db.Users.GetByEmail(r.Context(), request.Email)

	if err != nil && !errors.Is(err, operations.ErrNotFound) {
		h.fail(w, err)

		return
	}

	hash := h.dummyHash

	if err == nil && user.Password != nil {
		hash = *user.Password
	}

	valid, verifyErr := h.hasher.Verify(request.Password, hash)

	if verifyErr != nil {
		h.logger.Error("Stored password hash is invalid")
	}

	if err != nil || user.Password == nil || verifyErr != nil || !valid {
		h.fail(w, ErrUnauthorized)

		return
	}

	h.startSession(w, user)
}

// refreshSession: POST /auth/refresh. Troca o refresh token (cookie) por tokens novos.
// O refresh token usado e invalidado, entao cada um so serve uma vez.
func (h *Handles) refreshSession(w http.ResponseWriter, r *http.Request, _ *Principal) {
	token, err := h.readRefresh(r)

	if err != nil {
		h.fail(w, err)

		return
	}

	if h.revoked.revoked(r.Context(), token.tokenID) {
		h.fail(w, errInvalidRefresh)

		return
	}

	user, err := h.db.Users.GetByID(r.Context(), token.userID)

	if errors.Is(err, operations.ErrNotFound) {
		h.fail(w, ErrUnauthorized)

		return
	}

	if err != nil {
		h.fail(w, err)

		return
	}

	h.revoked.revoke(r.Context(), token.tokenID, token.expiresAt)

	h.startSession(w, user)
}

// logout: POST /auth/logout. Invalida o refresh token e limpa os cookies. Sempre responde 200.
// O token de acesso nao tem como ser invalidado; ele vence sozinho (e de vida curta).
func (h *Handles) logout(w http.ResponseWriter, r *http.Request, _ *Principal) {
	if token, err := h.readRefresh(r); err == nil {
		h.revoked.revoke(r.Context(), token.tokenID, token.expiresAt)
	}

	h.clearCookies(w)

	writeJSON(w, http.StatusOK, schema.MessageResponse{Message: "logged out"})
}

// startSession poe os cookies dos tokens na resposta e devolve a sessao.
func (h *Handles) startSession(w http.ResponseWriter, user *operations.User) {
	if err := h.issueSession(w, user); err != nil {
		h.fail(w, err)

		return
	}

	writeJSON(w, http.StatusOK, schema.NewSessionResponse(h.access.TTL(), schema.UserFromModel(user)))
}

// me: GET /me.
func (h *Handles) me(w http.ResponseWriter, r *http.Request, p *Principal) {
	writeJSON(w, http.StatusOK, schema.UserFromModel(p.User))
}

// updateMe: PATCH /me. Quem nao e admin nao pode mudar o proprio papel. A senha nova vira hash bcrypt.
func (h *Handles) updateMe(w http.ResponseWriter, r *http.Request, p *Principal) {
	var request schema.UpdateUserRequest

	if err := decodeJSON(w, r, &request); err != nil {
		h.fail(w, err)

		return
	}

	if request.Role != nil && !p.IsAdmin() {
		h.fail(w, errForbidden)

		return
	}

	user := *p.User

	if err := request.Apply(&user, h.hasher.Hash); err != nil {
		h.fail(w, err)

		return
	}

	if err := h.db.Users.Update(r.Context(), &user); err != nil {
		h.fail(w, err)

		return
	}

	writeJSON(w, http.StatusOK, schema.UserFromModel(&user))
}
