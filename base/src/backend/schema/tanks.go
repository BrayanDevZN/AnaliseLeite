package schema

import (
	"net/http"
	"strings"
	"time"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
)

// maxCapacityLiters e a maior capacidade aceita para um tanque.
const maxCapacityLiters = 1_000_000

// TankResponse e um tanque nas respostas.
type TankResponse struct {
	ID             int64     `json:"id"`
	FarmID         int64     `json:"farm_id"`
	Type           string    `json:"type"`
	CapacityLiters float64   `json:"capacity_liters"`
	CreatedAt      time.Time `json:"created_at"`
}

// TankFromModel converte o tanque do banco para a resposta.
func TankFromModel(tank *operations.Tank) TankResponse {
	return TankResponse{
		ID:             tank.ID,
		FarmID:         tank.FarmID,
		Type:           tank.Type,
		CapacityLiters: tank.CapacityLiters,
		CreatedAt:      tank.CreatedAt,
	}
}

// TanksFromModel converte uma lista de tanques do banco para a resposta (nunca nil).
func TanksFromModel(tanks []operations.Tank) []TankResponse {
	responses := make([]TankResponse, 0, len(tanks))

	for i := range tanks {
		responses = append(responses, TankFromModel(&tanks[i]))
	}

	return responses
}

// CreateTankRequest e o corpo de POST /farms/{id}/tanks.
type CreateTankRequest struct {
	Type           string  `json:"type"`
	CapacityLiters float64 `json:"capacity_liters"`
}

// Validate tira os espacos e valida os campos.
func (r *CreateTankRequest) Validate() error {
	r.Type = strings.TrimSpace(r.Type)

	var v validator

	v.text("type", r.Type, maxTextLength)
	v.number("capacity_liters", r.CapacityLiters, 1, maxCapacityLiters)

	return v.err()
}

// ToModel valida e monta o tanque do banco, da fazenda dele.
func (r *CreateTankRequest) ToModel(farmID int64) (*operations.Tank, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	return &operations.Tank{FarmID: farmID, Type: r.Type, CapacityLiters: r.CapacityLiters}, nil
}

// UpdateTankRequest e o corpo de PATCH /tanks/{id}. So os campos enviados mudam.
type UpdateTankRequest struct {
	Type           *string  `json:"type,omitempty"`
	CapacityLiters *float64 `json:"capacity_liters,omitempty"`
}

// Validate tira os espacos e valida so os campos enviados.
func (r *UpdateTankRequest) Validate() error {
	var v validator

	if r.Type != nil {
		trimmed := strings.TrimSpace(*r.Type)

		r.Type = &trimmed

		v.text("type", trimmed, maxTextLength)
	}

	if r.CapacityLiters != nil {
		v.number("capacity_liters", *r.CapacityLiters, 1, maxCapacityLiters)
	}

	if r.Type == nil && r.CapacityLiters == nil {
		v.add("body", "must have at least one field")
	}

	return v.err()
}

// Apply valida e aplica os campos enviados no tanque do banco.
func (r *UpdateTankRequest) Apply(tank *operations.Tank) error {
	if err := r.Validate(); err != nil {
		return err
	}

	if r.Type != nil {
		tank.Type = *r.Type
	}

	if r.CapacityLiters != nil {
		tank.CapacityLiters = *r.CapacityLiters
	}

	return nil
}

// TankRoutes sao as rotas de tanques (so de fazendas que o usuario pode ver).
var TankRoutes = []Route{
	{Method: http.MethodPost, Path: "/farms/{id}/tanks", Description: "cria um tanque na fazenda", Request: CreateTankRequest{}, Response: TankResponse{}},
	{Method: http.MethodGet, Path: "/farms/{id}/tanks", Description: "lista os tanques da fazenda", Response: []TankResponse{}},
	{Method: http.MethodGet, Path: "/tanks/{id}", Description: "busca um tanque", Response: TankResponse{}},
	{Method: http.MethodPatch, Path: "/tanks/{id}", Description: "atualiza um tanque", Request: UpdateTankRequest{}, Response: TankResponse{}},
	{Method: http.MethodDelete, Path: "/tanks/{id}", Description: "apaga um tanque"},
}
