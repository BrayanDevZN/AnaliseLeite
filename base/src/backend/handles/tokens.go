package handles

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/redis"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
)

// refreshPrefix marca o subject do refresh token: "refresh:<id do usuario>:<id do token>".
// Assim ele nunca e aceito como token de acesso (que tem so o id do usuario no subject).
const refreshPrefix = "refresh:"

var errInvalidRefresh = errors.New("invalid refresh token")

// newTokenID gera o id aleatorio de um refresh token.
func newTokenID() (string, error) {
	data := make([]byte, 16)

	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("generate token id: %w", err)
	}

	return hex.EncodeToString(data), nil
}

// issueSession cria os dois tokens do usuario e poe os dois nos cookies HttpOnly da resposta.
func (h *Handles) issueSession(w http.ResponseWriter, user *operations.User) error {
	id := strconv.FormatInt(user.ID, 10)

	access, err := h.access.Create(id)

	if err != nil {
		return err
	}

	tokenID, err := newTokenID()

	if err != nil {
		return err
	}

	refresh, err := h.refresh.Create(refreshPrefix + id + ":" + tokenID)

	if err != nil {
		return err
	}

	h.setCookie(w, AccessCookie, access, "/", h.access.TTL())
	h.setCookie(w, RefreshCookie, refresh, refreshCookiePath, h.refresh.TTL())

	return nil
}

// refreshToken e o que se le de um refresh token valido.
type refreshToken struct {
	userID    int64
	tokenID   string
	expiresAt time.Time
}

// readRefresh le o refresh token do cookie, confere a assinatura, a validade e o formato.
func (h *Handles) readRefresh(r *http.Request) (*refreshToken, error) {
	cookie, err := r.Cookie(RefreshCookie)

	if err != nil || cookie.Value == "" {
		return nil, errInvalidRefresh
	}

	claims, err := h.refresh.Read(cookie.Value)

	if err != nil {
		return nil, errInvalidRefresh
	}

	parts := strings.Split(strings.TrimPrefix(claims.Subject, refreshPrefix), ":")

	if !strings.HasPrefix(claims.Subject, refreshPrefix) || len(parts) != 2 || parts[1] == "" {
		return nil, errInvalidRefresh
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)

	if err != nil || id <= 0 {
		return nil, errInvalidRefresh
	}

	return &refreshToken{userID: id, tokenID: parts[1], expiresAt: claims.ExpiresAt}, nil
}

const (
	refreshCookiePath = "/auth"
	revokedPrefix     = "revoked:"
)

// revocations guarda os refresh tokens ja usados ou deslogados, ate eles vencerem.
// Grava na memoria e, se houver Redis, nele tambem (assim varias instancias do backend concordam).
// A leitura olha a memoria e depois o Redis. Se o Redis falhar, a memoria continua valendo.
type revocations struct {
	redis *redis.Operations

	mu        sync.Mutex
	items     map[string]time.Time
	nextSweep time.Time
}

func newRevocations(redisManage *redis.Manage) *revocations {
	store := &revocations{items: make(map[string]time.Time)}

	if redisManage != nil && redisManage.Connection != nil && redisManage.Operations != nil {
		store.redis = redisManage.Operations
	}

	return store
}

// revoke marca o token como invalido ate o instante until (quando ele venceria).
func (s *revocations) revoke(ctx context.Context, tokenID string, until time.Time) {
	ttl := time.Until(until)

	if ttl <= 0 {
		return
	}

	s.mu.Lock()

	s.sweep(time.Now())
	s.items[tokenID] = until

	s.mu.Unlock()

	if s.redis != nil {
		s.redis.Set(ctx, revokedPrefix+tokenID, "1", ttl)
	}
}

// revoked diz se o token ja foi usado ou deslogado.
func (s *revocations) revoked(ctx context.Context, tokenID string) bool {
	s.mu.Lock()

	until, found := s.items[tokenID]

	s.mu.Unlock()

	if found && time.Now().Before(until) {
		return true
	}

	if s.redis != nil {
		count, err := s.redis.Exists(ctx, revokedPrefix+tokenID)

		return err == nil && count > 0
	}

	return false
}

// sweep apaga da memoria os tokens que ja venceriam de qualquer jeito (no maximo uma vez por minuto).
// Quem chama ja segura o mutex.
func (s *revocations) sweep(now time.Time) {
	if now.Before(s.nextSweep) {
		return
	}

	s.nextSweep = now.Add(time.Minute)

	for id, until := range s.items {
		if !now.Before(until) {
			delete(s.items, id)
		}
	}
}
