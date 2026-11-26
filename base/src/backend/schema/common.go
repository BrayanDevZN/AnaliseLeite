// Package schema tem o formato dos dados das rotas do backend: o que cada rota recebe e devolve,
// a validacao desses dados e a conversao de/para as structs do banco. Cada arquivo cuida de um grupo
// de rotas e tem a tabela dessas rotas (metodo HTTP, caminho, quem pode usar e os schemas).
package schema

import (
	"errors"
	"math"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// AllowedEmailDomain e o unico dominio de email aceito ao criar uma conta ou trocar o email.
const AllowedEmailDomain = "gmail.com"

// IsAllowedEmail diz se o email (ja em minusculas) e do dominio aceito.
func IsAllowedEmail(email string) bool {
	return strings.HasSuffix(email, "@"+AllowedEmailDomain)
}

// Papeis dos usuarios.
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

const (
	maxNameLength  = 120
	maxEmailLength = 254
	maxTextLength  = 100
)

// states sao as siglas dos estados (as mesmas do schema do pipeline).
var states = map[string]bool{
	"AC": true, "AL": true, "AP": true, "AM": true, "BA": true, "CE": true, "DF": true, "ES": true,
	"GO": true, "MA": true, "MT": true, "MS": true, "MG": true, "PA": true, "PB": true, "PR": true,
	"PE": true, "PI": true, "RJ": true, "RN": true, "RS": true, "RO": true, "RR": true, "SC": true,
	"SP": true, "SE": true, "TO": true,
}

// modelName so aceita letras, numeros, _, - e . (o nome vira parte de um caminho no pipeline).
var modelName = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,100}$`)

// Route descreve uma rota: o metodo HTTP, o caminho, quem pode usar e os schemas dela.
type Route struct {
	Method      string
	Path        string
	Description string

	// Public e true para rotas sem login (o rate limit conta essas por IP).
	Public bool

	// AdminOnly e true para rotas que so o admin pode usar.
	AdminOnly bool

	// Request e Response sao valores vazios dos schemas (nil quando a rota nao recebe ou nao devolve corpo).
	Request  any
	Response any
}

// Pattern devolve o padrao do http.ServeMux, ex.: "POST /auth/login".
func (r Route) Pattern() string {
	return r.Method + " " + r.Path
}

// Routes devolve todas as rotas do backend.
func Routes() []Route {
	var all []Route

	for _, group := range [][]Route{HealthRoutes, AuthRoutes, UserRoutes, FarmRoutes, TankRoutes, AnalysisRoutes, ModelRoutes, PhotoRoutes} {
		all = append(all, group...)
	}

	return all
}

// ErrorResponse e o corpo de qualquer resposta de erro (mesmo formato do pipeline: {"detail": ...}).
type ErrorResponse struct {
	Detail string       `json:"detail"`
	Errors []FieldError `json:"errors,omitempty"`
}

// HealthResponse e a resposta de GET /health.
type HealthResponse struct {
	Status bool `json:"status"`
}

// HealthRoutes sao as rotas de saude.
var HealthRoutes = []Route{
	{Method: http.MethodGet, Path: "/health", Description: "o backend esta de pe", Public: true, Response: HealthResponse{}},
}

// FieldError e um erro de validacao de um campo.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError agrupa todos os erros de validacao de um corpo (a rota responde 422).
type ValidationError struct {
	Errors []FieldError
}

func (e *ValidationError) Error() string {
	messages := make([]string, 0, len(e.Errors))

	for _, fieldError := range e.Errors {
		messages = append(messages, fieldError.Field+": "+fieldError.Message)
	}

	return "invalid request: " + strings.Join(messages, "; ")
}

// Response monta o corpo de erro desta validacao.
func (e *ValidationError) Response() ErrorResponse {
	return ErrorResponse{Detail: "invalid request", Errors: e.Errors}
}

// AsValidationError diz se o erro e um *ValidationError e o devolve.
func AsValidationError(err error) (*ValidationError, bool) {
	var validation *ValidationError

	return validation, errors.As(err, &validation)
}

// validator junta os erros de um corpo, para devolver todos de uma vez.
type validator struct {
	errors []FieldError
}

func (v *validator) add(field, message string) {
	v.errors = append(v.errors, FieldError{Field: field, Message: message})
}

// err devolve nil se nao houve erro.
func (v *validator) err() error {
	if len(v.errors) == 0 {
		return nil
	}

	return &ValidationError{Errors: v.errors}
}

// text valida um texto obrigatorio (ja sem espacos nas pontas) e o tamanho dele.
func (v *validator) text(field, value string, max int) {
	switch {
	case value == "":
		v.add(field, "is required")

	case utf8.RuneCountInString(value) > max:
		v.add(field, "is too long")
	}
}

// email valida o formato do email.
func (v *validator) email(field, value string) {
	if value == "" {
		v.add(field, "is required")

		return
	}

	address, err := mail.ParseAddress(value)

	// o dominio precisa ter um ponto (a@b e valido pela RFC, mas nao serve para ninguem)
	if err != nil || address.Address != value || len(value) > maxEmailLength || !strings.Contains(value[strings.LastIndex(value, "@"):], ".") {
		v.add(field, "is not a valid email")
	}
}

// gmail confere que o email e do dominio aceito (so ao CRIAR uma conta ou TROCAR o email: quem ja tem conta
// com outro dominio continua entrando e editando o perfil normalmente).
func (v *validator) gmail(field, value string) {
	if value != "" && !IsAllowedEmail(value) {
		v.add(field, "must be a gmail.com address")
	}
}

// newPassword valida uma senha NOVA: de 8 a 72 caracteres, com letra maiuscula, letra minuscula e numero.
// O login nao usa esta regra (so confere a senha), para as contas antigas continuarem entrando.
func (v *validator) newPassword(field, value string) {
	v.password(field, value)

	var upper, lower, digit bool

	for _, char := range value {
		switch {
		case unicode.IsUpper(char):
			upper = true

		case unicode.IsLower(char):
			lower = true

		case unicode.IsDigit(char):
			digit = true
		}
	}

	if !upper {
		v.add(field, "must have an uppercase letter")
	}

	if !lower {
		v.add(field, "must have a lowercase letter")
	}

	if !digit {
		v.add(field, "must have a number")
	}
}

// password valida o tamanho da senha (o bcrypt so le os 72 primeiros bytes).
func (v *validator) password(field, value string) {
	switch {
	case len(value) < 8:
		v.add(field, "must have at least 8 characters")

	case len(value) > 72:
		v.add(field, "must have at most 72 bytes")
	}
}

func (v *validator) role(field, value string) {
	if value != RoleAdmin && value != RoleUser {
		v.add(field, "must be admin or user")
	}
}

func (v *validator) state(field, value string) {
	if !states[value] {
		v.add(field, "must be a brazilian state abbreviation, like MG")
	}
}

// number valida que o numero e finito e esta entre min e max.
func (v *validator) number(field string, value, min, max float64) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < min || value > max {
		v.add(field, "is out of the valid range")
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
