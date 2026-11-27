package handles

import (
	"net/http"
	"strconv"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/schema"
)

// ownedFarm busca a fazenda do {id} da rota. Quem nao e dono (e nao e admin) recebe "nao achou",
// para nao descobrir quais fazendas existem.
func (h *Handles) ownedFarm(r *http.Request, p *Principal) (*operations.Farm, error) {
	id, err := pathID(r)

	if err != nil {
		return nil, err
	}

	return h.farmFor(r, p, id)
}

func (h *Handles) farmFor(r *http.Request, p *Principal, id int64) (*operations.Farm, error) {
	farm, err := h.db.Farms.GetByID(r.Context(), id)

	if err != nil {
		return nil, err
	}

	if !p.CanAccessUser(farm.UserID) {
		return nil, errNotFound
	}

	return farm, nil
}

// createFarm: POST /farms. A fazenda e do usuario logado.
func (h *Handles) createFarm(w http.ResponseWriter, r *http.Request, p *Principal) {
	var request schema.CreateFarmRequest

	if err := decodeJSON(w, r, &request); err != nil {
		h.fail(w, err)

		return
	}

	farm, err := request.ToModel(p.User.ID)

	if err != nil {
		h.fail(w, err)

		return
	}

	if err := h.db.Farms.Create(r.Context(), farm); err != nil {
		h.fail(w, err)

		return
	}

	writeJSON(w, http.StatusCreated, schema.FarmFromModel(farm))
}

// listFarms: GET /farms. Lista as do usuario logado; o admin pode pedir as de outro com ?user_id=.
func (h *Handles) listFarms(w http.ResponseWriter, r *http.Request, p *Principal) {
	userID := p.User.ID

	if raw := r.URL.Query().Get("user_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)

		if err != nil || id <= 0 {
			h.fail(w, bodyError("user_id", "must be a positive number"))

			return
		}

		if !p.CanAccessUser(id) {
			h.fail(w, errForbidden)

			return
		}

		userID = id
	}

	farms, err := h.db.Farms.ListByUser(r.Context(), userID)

	if err != nil {
		h.fail(w, err)

		return
	}

	writeJSON(w, http.StatusOK, schema.FarmsFromModel(farms))
}

// getFarm: GET /farms/{id}.
func (h *Handles) getFarm(w http.ResponseWriter, r *http.Request, p *Principal) {
	farm, err := h.ownedFarm(r, p)

	if err != nil {
		h.fail(w, err)

		return
	}

	writeJSON(w, http.StatusOK, schema.FarmFromModel(farm))
}

// updateFarm: PATCH /farms/{id}.
func (h *Handles) updateFarm(w http.ResponseWriter, r *http.Request, p *Principal) {
	farm, err := h.ownedFarm(r, p)

	if err != nil {
		h.fail(w, err)

		return
	}

	var request schema.UpdateFarmRequest

	if err := decodeJSON(w, r, &request); err != nil {
		h.fail(w, err)

		return
	}

	if err := request.Apply(farm); err != nil {
		h.fail(w, err)

		return
	}

	if err := h.db.Farms.Update(r.Context(), farm); err != nil {
		h.fail(w, err)

		return
	}

	writeJSON(w, http.StatusOK, schema.FarmFromModel(farm))
}

// deleteFarm: DELETE /farms/{id}.
func (h *Handles) deleteFarm(w http.ResponseWriter, r *http.Request, p *Principal) {
	farm, err := h.ownedFarm(r, p)

	if err != nil {
		h.fail(w, err)

		return
	}

	if err := h.db.Farms.Delete(r.Context(), farm.ID); err != nil {
		h.fail(w, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
