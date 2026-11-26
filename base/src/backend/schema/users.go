package schema

import (
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
)

// maxPhotoSize e o maior tamanho da foto de perfil (em bytes, ja decodificada).
const maxPhotoSize = 2 << 20

// HashFunc transforma a senha no hash que vai para o banco (o auth.Hasher.Hash).
type HashFunc func(password string) (string, error)

// UserResponse e um usuario nas respostas. Nunca tem a senha nem o hash dela.
type UserResponse struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Email           string    `json:"email"`
	Role            string    `json:"role"`
	HasPassword     bool      `json:"has_password"`
	HasProfilePhoto bool      `json:"has_profile_photo"`
	CreatedAt       time.Time `json:"created_at"`
}

// UserFromModel converte o usuario do banco para a resposta.
func UserFromModel(user *operations.User) UserResponse {
	return UserResponse{
		ID:              user.ID,
		Name:            user.Name,
		Email:           user.Email,
		Role:            user.Role,
		HasPassword:     user.Password != nil,
		HasProfilePhoto: len(user.ProfilePhoto) > 0,
		CreatedAt:       user.CreatedAt,
	}
}

// UsersFromModel converte uma lista de usuarios do banco para a resposta (nunca nil).
func UsersFromModel(users []operations.User) []UserResponse {
	responses := make([]UserResponse, 0, len(users))

	for i := range users {
		responses = append(responses, UserFromModel(&users[i]))
	}

	return responses
}

// CreateUserRequest e o corpo de POST /users (so admin). A senha e opcional (ex.: conta so do Google).
type CreateUserRequest struct {
	Name     string  `json:"name"`
	Email    string  `json:"email"`
	Password *string `json:"password,omitempty"`
	Role     string  `json:"role"`
}

// Validate tira os espacos, poe o email em minusculas e valida os campos. Sem papel, usa "user".
func (r *CreateUserRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.Email = normalizeEmail(r.Email)

	if r.Role == "" {
		r.Role = RoleUser
	}

	var v validator

	v.text("name", r.Name, maxNameLength)
	v.email("email", r.Email)
	v.role("role", r.Role)

	if r.Password != nil {
		v.password("password", *r.Password)
	}

	return v.err()
}

// ToModel valida e monta o usuario do banco, com o hash da senha (se tiver).
func (r *CreateUserRequest) ToModel(hash HashFunc) (*operations.User, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	user := &operations.User{Name: r.Name, Email: r.Email, Role: r.Role}

	if r.Password != nil {
		hashed, err := hash(*r.Password)

		if err != nil {
			return nil, err
		}

		user.Password = &hashed
	}

	return user, nil
}

// ToUser monta o usuario do banco a partir do cadastro publico: sempre com o papel "user".
func (r *RegisterRequest) ToUser(hash HashFunc) (*operations.User, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	hashed, err := hash(r.Password)

	if err != nil {
		return nil, err
	}

	return &operations.User{Name: r.Name, Email: r.Email, Password: &hashed, Role: RoleUser}, nil
}

// UpdateUserRequest e o corpo de PATCH /users/{id}. So os campos enviados mudam.
// ProfilePhoto e a foto em base64; uma string vazia remove a foto.
type UpdateUserRequest struct {
	Name         *string `json:"name,omitempty"`
	Email        *string `json:"email,omitempty"`
	Password     *string `json:"password,omitempty"`
	Role         *string `json:"role,omitempty"`
	ProfilePhoto *string `json:"profile_photo,omitempty"`

	photo []byte
}

// Validate tira os espacos, poe o email em minusculas e valida so os campos enviados.
func (r *UpdateUserRequest) Validate() error {
	var v validator

	if r.Name != nil {
		trimmed := strings.TrimSpace(*r.Name)

		r.Name = &trimmed

		v.text("name", trimmed, maxNameLength)
	}

	if r.Email != nil {
		normalized := normalizeEmail(*r.Email)

		r.Email = &normalized

		v.email("email", normalized)
	}

	if r.Password != nil {
		v.password("password", *r.Password)
	}

	if r.Role != nil {
		v.role("role", *r.Role)
	}

	r.photo = nil

	if r.ProfilePhoto != nil && *r.ProfilePhoto != "" {
		photo, err := base64.StdEncoding.DecodeString(*r.ProfilePhoto)

		switch {
		case err != nil:
			v.add("profile_photo", "must be base64")

		case len(photo) > maxPhotoSize:
			v.add("profile_photo", "is too big")

		default:
			r.photo = photo
		}
	}

	if r.Name == nil && r.Email == nil && r.Password == nil && r.Role == nil && r.ProfilePhoto == nil {
		v.add("body", "must have at least one field")
	}

	return v.err()
}

// Apply valida e aplica os campos enviados no usuario do banco (a senha vira hash).
func (r *UpdateUserRequest) Apply(user *operations.User, hash HashFunc) error {
	if err := r.Validate(); err != nil {
		return err
	}

	if r.Password != nil {
		hashed, err := hash(*r.Password)

		if err != nil {
			return err
		}

		user.Password = &hashed
	}

	if r.Name != nil {
		user.Name = *r.Name
	}

	if r.Email != nil {
		user.Email = *r.Email
	}

	if r.Role != nil {
		user.Role = *r.Role
	}

	if r.ProfilePhoto != nil {
		user.ProfilePhoto = r.photo
	}

	return nil
}

// UserRoutes sao as rotas de usuarios (so admin).
var UserRoutes = []Route{
	{Method: http.MethodPost, Path: "/users", Description: "cria um usuario", AdminOnly: true, Request: CreateUserRequest{}, Response: UserResponse{}},
	{Method: http.MethodGet, Path: "/users", Description: "lista os usuarios", AdminOnly: true, Response: []UserResponse{}},
	{Method: http.MethodGet, Path: "/users/{id}", Description: "busca um usuario", AdminOnly: true, Response: UserResponse{}},
	{Method: http.MethodPatch, Path: "/users/{id}", Description: "atualiza um usuario", AdminOnly: true, Request: UpdateUserRequest{}, Response: UserResponse{}},
	{Method: http.MethodDelete, Path: "/users/{id}", Description: "apaga um usuario", AdminOnly: true},
}
