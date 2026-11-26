package handles

import (
	"net/http"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/schema"
)

// ownedTank busca o tanque do {id} da rota, se o usuario pode ver a fazenda dele.
func (h *Handles) ownedTank(r *http.Request, p *Principal) (*operations.Tank, error) {
	id, err := pathID(r)

	if err != nil {
		return nil, err
	}

	tank, err := h.db.Tanks.GetByID(r.Context(), id)

	if err != nil {
		return nil, err
	}

	if _, err := h.farmFor(r, p, tank.FarmID); err != nil {
		return nil, err
	}

	return tank, nil
}

// createTank: POST /farms/{id}/tanks.
func (h *Handles) createTank(w http.ResponseWriter, r *http.Request, p *Principal) {
	farm, err := h.ownedFarm(r, p)

	if err != nil {
		h.fail(w, err)

		return
	}

	var request schema.CreateTankRequest

	if err := decodeJSON(w, r, &request); err != nil {
		h.fail(w, err)

		return
	}

	tank, err := request.ToModel(farm.ID)

	if err != nil {
		h.fail(w, err)

		return
	}

	if err := h.db.Tanks.Create(r.Context(), tank); err != nil {
		h.fail(w, err)

		return
	}

	writeJSON(w, http.StatusCreated, schema.TankFromModel(tank))
}

// listTanks: GET /farms/{id}/tanks.
func (h *Handles) listTanks(w http.ResponseWriter, r *http.Request, p *Principal) {
	farm, err := h.ownedFarm(r, p)

	if err != nil {
		h.fail(w, err)

		return
	}

	tanks, err := h.db.Tanks.ListByFarm(r.Context(), farm.ID)

	if err != nil {
		h.fail(w, err)

		return
	}

	writeJSON(w, http.StatusOK, schema.TanksFromModel(tanks))
}

// getTank: GET /tanks/{id}.
func (h *Handles) getTank(w http.ResponseWriter, r *http.Request, p *Principal) {
	tank, err := h.ownedTank(r, p)

	if err != nil {
		h.fail(w, err)

		return
	}

	writeJSON(w, http.StatusOK, schema.TankFromModel(tank))
}

// updateTank: PATCH /tanks/{id}.
func (h *Handles) updateTank(w http.ResponseWriter, r *http.Request, p *Principal) {
	tank, err := h.ownedTank(r, p)

	if err != nil {
		h.fail(w, err)

		return
	}

	var request schema.UpdateTankRequest

	if err := decodeJSON(w, r, &request); err != nil {
		h.fail(w, err)

		return
	}

	if err := request.Apply(tank); err != nil {
		h.fail(w, err)

		return
	}

	if err := h.db.Tanks.Update(r.Context(), tank); err != nil {
		h.fail(w, err)

		return
	}

	writeJSON(w, http.StatusOK, schema.TankFromModel(tank))
}

// deleteTank: DELETE /tanks/{id}.
func (h *Handles) deleteTank(w http.ResponseWriter, r *http.Request, p *Principal) {
	tank, err := h.ownedTank(r, p)

	if err != nil {
		h.fail(w, err)

		return
	}

	if err := h.db.Tanks.Delete(r.Context(), tank.ID); err != nil {
		h.fail(w, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
