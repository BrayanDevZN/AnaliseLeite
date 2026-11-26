package midlleware

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ErrInvalidOrigin quando uma origem do CORS nao e do tipo http://host[:porta].
var ErrInvalidOrigin = fmt.Errorf("invalid cors origin: use http://host or https://host:port, with no path and no *")

// ParseOrigins le uma lista de origens separadas por virgula (ex.: "http://localhost:5173,https://app.exemplo.com").
// Cada uma precisa ser so esquema, host e porta. O "*" nao e aceito: o login vai em cookie, e o navegador
// recusa credenciais com "*".
func ParseOrigins(list string) ([]string, error) {
	var origins []string

	for _, item := range strings.Split(list, ",") {
		item = strings.TrimSpace(item)

		if item == "" {
			continue
		}

		parsed, err := url.Parse(item)

		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
			(parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
			return nil, ErrInvalidOrigin
		}

		origins = append(origins, strings.ToLower(parsed.Scheme+"://"+parsed.Host))
	}

	return origins, nil
}

// CORS deixa um frontend em outra origem (ex.: o Vite em localhost:5173) chamar a API com cookies.
// So as origens da lista recebem os headers; qualquer outra e tratada como antes (o navegador bloqueia).
type CORS struct {
	allowed map[string]bool
}

// NewCORS cria o CORS com as origens permitidas. Sem origens, devolve nil (o CORS fica desligado).
func NewCORS(origins []string) *CORS {
	if len(origins) == 0 {
		return nil
	}

	cors := &CORS{allowed: make(map[string]bool, len(origins))}

	for _, origin := range origins {
		cors.allowed[strings.ToLower(origin)] = true
	}

	return cors
}

// Handler poe os headers do CORS e responde sozinho os preflights (OPTIONS) das origens permitidas.
// Deve ficar por fora do rate limit, para ate o 429 chegar com os headers e o frontend conseguir le-lo.
func (c *CORS) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin == "" {
			next.ServeHTTP(w, r)

			return
		}

		// a resposta muda conforme a origem, entao os caches precisam saber
		w.Header().Add("Vary", "Origin")

		if !c.allowed[strings.ToLower(origin)] {
			next.ServeHTTP(w, r)

			return
		}

		header := w.Header()

		header.Set("Access-Control-Allow-Origin", origin)
		header.Set("Access-Control-Allow-Credentials", "true")
		header.Set("Access-Control-Expose-Headers", "Retry-After, X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset")

		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			header.Add("Vary", "Access-Control-Request-Method")
			header.Add("Vary", "Access-Control-Request-Headers")
			header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			header.Set("Access-Control-Max-Age", "600")

			w.WriteHeader(http.StatusNoContent)

			return
		}

		next.ServeHTTP(w, r)
	})
}
