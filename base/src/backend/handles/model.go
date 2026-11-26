package handles

import (
	"errors"
	"net/http"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/schema"
)

// errNoModel quando o backend foi iniciado sem o cliente do pipeline.
var errNoModel = &httpError{http.StatusServiceUnavailable, "model service is not configured"}

func asRequestError(err error) (*backend.RequestError, bool) {
	var requestError *backend.RequestError

	return requestError, errors.As(err, &requestError)
}

// modelError traduz um erro do pipeline: o que ele respondeu vira 502, e se ele nao respondeu, 503.
func (h *Handles) modelError(err error) error {
	h.logger.Error("Model service failed: %v", err)

	if _, ok := asRequestError(err); ok {
		return &httpError{http.StatusBadGateway, "model service rejected the request"}
	}

	return &httpError{http.StatusServiceUnavailable, "model service is unavailable"}
}

// health: GET /health.
func (h *Handles) health(w http.ResponseWriter, r *http.Request, _ *Principal) {
	writeJSON(w, http.StatusOK, schema.HealthResponse{Status: true})
}

// models: GET /models. Lista os modelos do pipeline.
func (h *Handles) models(w http.ResponseWriter, r *http.Request, _ *Principal) {
	if h.requests == nil {
		h.fail(w, errNoModel)

		return
	}

	models, err := h.requests.Models(r.Context())

	if err != nil {
		h.fail(w, h.modelError(err))

		return
	}

	writeJSON(w, http.StatusOK, schema.ModelsResponse{Models: models})
}
