package handles

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/auth"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/schema"
)

// maxBodySize e o maior corpo JSON aceito (em bytes).
const maxBodySize = 1 << 20

// Config e a configuracao dos handlers.
type Config struct {
	// InsecureCookies tira o atributo Secure dos cookies. Use so em desenvolvimento local, em http.
	InsecureCookies bool

	// CookieDomain e o dominio dos cookies (vazio usa o do proprio backend).
	CookieDomain string

	// SameSite e o SameSite dos cookies (padrao Lax). Ele e a protecao contra CSRF, ja que o login vai em cookie.
	SameSite http.SameSite
}

// handler e o formato de todo handler. O Principal e nil nas rotas publicas.
type handler func(w http.ResponseWriter, r *http.Request, p *Principal)

// Handles junta tudo que os handlers usam.
type Handles struct {
	logger   *backend.Logger
	config   Config
	db       *database.Manage
	requests *backend.Requests
	hasher   *auth.Hasher
	access   *auth.JWT
	refresh  *auth.JWT
	depends  *Depends
	revoked  *revocations

	// google e nil se o login com o Google nao esta configurado (veja google.go).
	google *googleAuth

	// dummyHash e verificado no login de um email que nao existe, para o tempo de resposta
	// nao revelar quais emails tem conta.
	dummyHash string
}

// New cria os handlers. O requests e opcional: sem ele, as rotas do modelo respondem 503.
// O access e o refresh sao dois auth.JWT, com tempos de vida diferentes (ex.: 15 minutos e 7 dias).
// Se logger for nil, os logs vao para o terminal.
func New(logger *backend.Logger, db *database.Manage, requests *backend.Requests, hasher *auth.Hasher, access, refresh *auth.JWT, config Config) (*Handles, error) {
	if db == nil || hasher == nil || access == nil || refresh == nil {
		return nil, errors.New("database, hasher and both jwt are required")
	}

	if logger == nil {
		logger = backend.NewLogger()
	}

	if config.SameSite == 0 {
		config.SameSite = http.SameSiteLaxMode
	}

	dummy, err := hasher.Hash("senha-so-para-igualar-o-tempo")

	if err != nil {
		return nil, fmt.Errorf("prepare password timing: %w", err)
	}

	google, err := newGoogleAuth(logger)

	if err != nil {
		return nil, err
	}

	return &Handles{
		logger:    logger,
		google:    google,
		config:    config,
		db:        db,
		requests:  requests,
		hasher:    hasher,
		access:    access,
		refresh:   refresh,
		depends:   NewDepends(access, db.Users),
		revoked:   newRevocations(db.Redis),
		dummyHash: dummy,
	}, nil
}

// Depends devolve o Depends dos handlers (o Identify dele serve para o rate limit).
func (h *Handles) Depends() *Depends {
	return h.depends
}

// Handler monta o roteador com todas as rotas do schema. Devolve erro se alguma rota nao tiver
// handler, ou se existir handler de uma rota que nao esta no schema.
func (h *Handles) Handler() (http.Handler, error) {
	handlers := h.handlers()
	mux := http.NewServeMux()

	for _, route := range schema.Routes() {
		fn, found := handlers[route.Pattern()]

		if !found {
			return nil, fmt.Errorf("no handler for route %s", route.Pattern())
		}

		delete(handlers, route.Pattern())

		mux.Handle(route.Pattern(), h.guard(route, fn))
	}

	for pattern := range handlers {
		return nil, fmt.Errorf("handler %s is not in the schema", pattern)
	}

	return mux, nil
}

// handlers liga o padrao de cada rota ao handler dela.
func (h *Handles) handlers() map[string]handler {
	return map[string]handler{
		"GET /health": h.health,

		"POST /auth/register":       h.register,
		"POST /auth/login":          h.login,
		"POST /auth/refresh":        h.refreshSession,
		"POST /auth/logout":         h.logout,
		"GET /auth/google":          h.googleStart,
		"GET /auth/google/callback": h.googleCallback,
		"GET /me":                   h.me,
		"PATCH /me":                 h.updateMe,

		"POST /users":        h.createUser,
		"GET /users":         h.listUsers,
		"GET /users/{id}":    h.getUser,
		"PATCH /users/{id}":  h.updateUser,
		"DELETE /users/{id}": h.deleteUser,

		"POST /farms":        h.createFarm,
		"GET /farms":         h.listFarms,
		"GET /farms/{id}":    h.getFarm,
		"PATCH /farms/{id}":  h.updateFarm,
		"DELETE /farms/{id}": h.deleteFarm,

		"POST /farms/{id}/tanks": h.createTank,
		"GET /farms/{id}/tanks":  h.listTanks,
		"GET /tanks/{id}":        h.getTank,
		"PATCH /tanks/{id}":      h.updateTank,
		"DELETE /tanks/{id}":     h.deleteTank,

		"POST /farms/{id}/analyses":   h.createAnalysis,
		"GET /farms/{id}/analyses":    h.listAnalyses,
		"GET /analyses":               h.listMyAnalyses,
		"GET /stats":                  h.stats,
		"GET /analyses/{id}":          h.getAnalysis,
		"GET /analyses/{id}/contents": h.listContents,
		"DELETE /analyses/{id}":       h.deleteAnalysis,

		"GET /models": h.models,

		"GET /users/{id}/photo":    h.getUserPhoto,
		"PUT /me/photo":            h.putMyPhoto,
		"DELETE /me/photo":         h.deleteMyPhoto,
		"GET /farms/{id}/photo":    h.getFarmPhoto,
		"PUT /farms/{id}/photo":    h.putFarmPhoto,
		"DELETE /farms/{id}/photo": h.deleteFarmPhoto,
	}
}

// guard aplica o login e o papel da rota (Public e AdminOnly do schema) antes do handler.
func (h *Handles) guard(route schema.Route, fn handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var principal *Principal

		if !route.Public {
			var err error

			principal, err = h.depends.Principal(r.Context(), r)

			if err != nil {
				h.fail(w, err)

				return
			}

			if route.AdminOnly && !principal.IsAdmin() {
				h.fail(w, errForbidden)

				return
			}
		}

		fn(w, r, principal)
	})
}

// httpError e um erro com o status HTTP que a resposta deve ter.
type httpError struct {
	status int
	detail string
}

func (e *httpError) Error() string { return e.detail }

var (
	errForbidden = &httpError{http.StatusForbidden, "forbidden"}
	errNotFound  = &httpError{http.StatusNotFound, "not found"}
	errConflict  = &httpError{http.StatusConflict, "already exists"}
)

// fail responde o erro com o status certo. Erros inesperados vao para o log e voltam como 500 sem detalhes.
func (h *Handles) fail(w http.ResponseWriter, err error) {
	var failure *httpError

	if validation, ok := schema.AsValidationError(err); ok {
		writeJSON(w, http.StatusUnprocessableEntity, validation.Response())

		return
	}

	switch {
	case errors.Is(err, ErrUnauthorized), errors.Is(err, errInvalidRefresh):
		w.Header().Set("WWW-Authenticate", "Bearer")
		writeJSON(w, http.StatusUnauthorized, schema.ErrorResponse{Detail: "unauthorized"})

	case errors.Is(err, operations.ErrNotFound):
		writeJSON(w, http.StatusNotFound, schema.ErrorResponse{Detail: errNotFound.detail})

	case errors.As(err, &failure):
		writeJSON(w, failure.status, schema.ErrorResponse{Detail: failure.detail})

	case isConflict(err):
		response := schema.ErrorResponse{Detail: errConflict.detail}

		if field := conflictField(err); field != "" {
			response.Errors = []schema.FieldError{{Field: field, Message: "is already registered"}}
		}

		writeJSON(w, errConflict.status, response)

	default:
		h.logger.Error("Request failed: %v", err)

		writeJSON(w, http.StatusInternalServerError, schema.ErrorResponse{Detail: "internal error"})
	}
}

// conflictField acha o nome do campo de um erro de valor unico do SQLite
// ("UNIQUE constraint failed: users.email" -> "email"), ou "" se nao conseguir.
func conflictField(err error) string {
	_, after, found := strings.Cut(err.Error(), "UNIQUE constraint failed: ")

	if !found {
		return ""
	}

	column, _, _ := strings.Cut(after, " ")
	column = strings.TrimRight(column, ")")

	if _, name, ok := strings.Cut(column, "."); ok {
		return name
	}

	return ""
}

// isConflict diz se o erro do banco e de um valor unico repetido (ex.: email).
func isConflict(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if body != nil {
		json.NewEncoder(w).Encode(body)
	}
}

// decodeJSON le o corpo da requisicao como JSON (no maximo maxBodySize, sem campos desconhecidos).
// O erro e sempre um *schema.ValidationError, que vira 422.
func decodeJSON(w http.ResponseWriter, r *http.Request, out any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodySize))

	decoder.DisallowUnknownFields()

	if err := decoder.Decode(out); err != nil {
		return bodyError("body", "is not valid JSON for this route")
	}

	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return bodyError("body", "must have only one JSON value")
	}

	return nil
}

func bodyError(field, message string) error {
	return &schema.ValidationError{Errors: []schema.FieldError{{Field: field, Message: message}}}
}

// pathID le o {id} da rota. Um id que nao e um numero positivo e tratado como "nao achou".
func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)

	if err != nil || id <= 0 {
		return 0, errNotFound
	}

	return id, nil
}

// setCookie poe um cookie HttpOnly na resposta.
func (h *Handles) setCookie(w http.ResponseWriter, name, value, path string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Domain:   h.config.CookieDomain,
		MaxAge:   int(ttl.Seconds()),
		Expires:  time.Now().Add(ttl),
		HttpOnly: true,
		Secure:   !h.config.InsecureCookies,
		SameSite: h.config.SameSite,
	})
}

// clearCookies apaga os cookies dos tokens no navegador.
func (h *Handles) clearCookies(w http.ResponseWriter) {
	for name, path := range map[string]string{AccessCookie: "/", RefreshCookie: refreshCookiePath} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Path:     path,
			Domain:   h.config.CookieDomain,
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   !h.config.InsecureCookies,
			SameSite: h.config.SameSite,
		})
	}
}
