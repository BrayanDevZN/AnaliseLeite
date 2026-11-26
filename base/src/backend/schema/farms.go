package schema

import (
	"net/http"
	"strings"
	"time"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
)

// FarmResponse e uma fazenda nas respostas.
type FarmResponse struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	City      string    `json:"city"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
}

// FarmFromModel converte a fazenda do banco para a resposta.
func FarmFromModel(farm *operations.Farm) FarmResponse {
	return FarmResponse{
		ID:        farm.ID,
		UserID:    farm.UserID,
		Name:      farm.Name,
		City:      farm.City,
		State:     farm.State,
		CreatedAt: farm.CreatedAt,
	}
}

// FarmsFromModel converte uma lista de fazendas do banco para a resposta (nunca nil).
func FarmsFromModel(farms []operations.Farm) []FarmResponse {
	responses := make([]FarmResponse, 0, len(farms))

	for i := range farms {
		responses = append(responses, FarmFromModel(&farms[i]))
	}

	return responses
}

// CreateFarmRequest e o corpo de POST /farms.
type CreateFarmRequest struct {
	Name  string `json:"name"`
	City  string `json:"city"`
	State string `json:"state"`
}

// Validate tira os espacos, poe o estado em maiusculas e valida os campos.
func (r *CreateFarmRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.City = strings.TrimSpace(r.City)
	r.State = strings.ToUpper(strings.TrimSpace(r.State))

	var v validator

	v.text("name", r.Name, maxNameLength)
	v.text("city", r.City, maxTextLength)
	v.state("state", r.State)

	return v.err()
}

// ToModel valida e monta a fazenda do banco, do usuario dono dela.
func (r *CreateFarmRequest) ToModel(userID int64) (*operations.Farm, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	return &operations.Farm{UserID: userID, Name: r.Name, City: r.City, State: r.State}, nil
}

// UpdateFarmRequest e o corpo de PATCH /farms/{id}. So os campos enviados mudam.
type UpdateFarmRequest struct {
	Name  *string `json:"name,omitempty"`
	City  *string `json:"city,omitempty"`
	State *string `json:"state,omitempty"`
}

// Validate tira os espacos, poe o estado em maiusculas e valida so os campos enviados.
func (r *UpdateFarmRequest) Validate() error {
	var v validator

	if r.Name != nil {
		trimmed := strings.TrimSpace(*r.Name)

		r.Name = &trimmed

		v.text("name", trimmed, maxNameLength)
	}

	if r.City != nil {
		trimmed := strings.TrimSpace(*r.City)

		r.City = &trimmed

		v.text("city", trimmed, maxTextLength)
	}

	if r.State != nil {
		upper := strings.ToUpper(strings.TrimSpace(*r.State))

		r.State = &upper

		v.state("state", upper)
	}

	if r.Name == nil && r.City == nil && r.State == nil {
		v.add("body", "must have at least one field")
	}

	return v.err()
}

// Apply valida e aplica os campos enviados na fazenda do banco.
func (r *UpdateFarmRequest) Apply(farm *operations.Farm) error {
	if err := r.Validate(); err != nil {
		return err
	}

	if r.Name != nil {
		farm.Name = *r.Name
	}

	if r.City != nil {
		farm.City = *r.City
	}

	if r.State != nil {
		farm.State = *r.State
	}

	return nil
}

// FarmRoutes sao as rotas de fazendas. Cada usuario ve so as dele (o admin ve todas).
var FarmRoutes = []Route{
	{Method: http.MethodPost, Path: "/farms", Description: "cria uma fazenda", Request: CreateFarmRequest{}, Response: FarmResponse{}},
	{Method: http.MethodGet, Path: "/farms", Description: "lista as fazendas do usuario", Response: []FarmResponse{}},
	{Method: http.MethodGet, Path: "/farms/{id}", Description: "busca uma fazenda", Response: FarmResponse{}},
	{Method: http.MethodPatch, Path: "/farms/{id}", Description: "atualiza uma fazenda", Request: UpdateFarmRequest{}, Response: FarmResponse{}},
	{Method: http.MethodDelete, Path: "/farms/{id}", Description: "apaga uma fazenda"},
}
