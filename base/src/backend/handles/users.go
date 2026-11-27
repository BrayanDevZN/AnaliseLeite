package handles

import (
	"net/http"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/schema"
)

// createUser: POST /users (admin). A senha, se vier, vira hash bcrypt.
func (h *Handles) createUser(w http.ResponseWriter, r *http.Request, _ *Principal) {
	var request schema.CreateUserRequest

	if err := decodeJSON(w, r, &request); err != nil {
		h.fail(w, err)

		return
	}

	user, err := request.ToModel(h.hasher.Hash)

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

// listUsers: GET /users (admin).
func (h *Handles) listUsers(w http.ResponseWriter, r *http.Request, _ *Principal) {
	users, err := h.db.Users.List(r.Context())

	if err != nil {
		h.fail(w, err)

		return
	}

	writeJSON(w, http.StatusOK, schema.UsersFromModel(users))
}

// getUser: GET /users/{id} (admin).
func (h *Handles) getUser(w http.ResponseWriter, r *http.Request, _ *Principal) {
	id, err := pathID(r)

	if err != nil {
		h.fail(w, err)

		return
	}

	user, err := h.db.Users.GetByID(r.Context(), id)

	if err != nil {
		h.fail(w, err)

		return
	}

	writeJSON(w, http.StatusOK, schema.UserFromModel(user))
}

// updateUser: PATCH /users/{id} (admin). A senha nova vira hash bcrypt.
func (h *Handles) updateUser(w http.ResponseWriter, r *http.Request, _ *Principal) {
	id, err := pathID(r)

	if err != nil {
		h.fail(w, err)

		return
	}

	var request schema.UpdateUserRequest

	if err := decodeJSON(w, r, &request); err != nil {
		h.fail(w, err)

		return
	}

	user, err := h.db.Users.GetByID(r.Context(), id)

	if err != nil {
		h.fail(w, err)

		return
	}

	if err := request.Apply(user, h.hasher.Hash); err != nil {
		h.fail(w, err)

		return
	}

	if err := h.db.Users.Update(r.Context(), user); err != nil {
		h.fail(w, err)

		return
	}

	writeJSON(w, http.StatusOK, schema.UserFromModel(user))
}

// deleteUser: DELETE /users/{id} (admin).
func (h *Handles) deleteUser(w http.ResponseWriter, r *http.Request, _ *Principal) {
	id, err := pathID(r)

	if err != nil {
		h.fail(w, err)

		return
	}

	if err := h.db.Users.Delete(r.Context(), id); err != nil {
		h.fail(w, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
