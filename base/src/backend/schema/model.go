package schema

import "net/http"

// ModelsResponse e a resposta de GET /models: os modelos que o pipeline tem.
type ModelsResponse struct {
	Models []string `json:"models"`
}

// ModelRoutes sao as rotas que repassam para o pipeline.
var ModelRoutes = []Route{
	{Method: http.MethodGet, Path: "/models", Description: "lista os modelos do pipeline", Response: ModelsResponse{}},
}
