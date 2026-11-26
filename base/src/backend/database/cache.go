// Package database junta o banco de dados (repository) e o Redis num cache aside:
// ler olha o cache primeiro; criar grava no banco e no cache; atualizar e apagar
// tiram a chave do cache. Se o Redis nao existir ou falhar, tudo funciona so com o banco.
package database

import (
	"context"
	"encoding/json"
	"time"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/redis"
)

const (
	// CacheTTL e quanto tempo cada chave fica no cache.
	CacheTTL = 60 * time.Second

	// maxCacheSize e o maior valor (em bytes, ja em JSON) que vai para o cache.
	// Valores maiores (ex.: arquivos) ficam so no banco.
	maxCacheSize = 1 << 20
)

// Cache e o cache aside em cima do Redis. Se nao tiver Redis, todas as operacoes dele
// nao fazem nada, e quem usa o cache le e grava direto no banco.
// Erros do Redis nunca viram erro da operacao: o Redis ja loga e o banco segue valendo.
type Cache struct {
	redis  *redis.Operations
	logger *backend.Logger
}

// NewCache cria o cache. Com redisManage nil, o cache fica desligado.
func NewCache(redisManage *redis.Manage) *Cache {
	if redisManage == nil || redisManage.Connection == nil || redisManage.Operations == nil {
		return &Cache{}
	}

	return &Cache{redis: redisManage.Operations, logger: redisManage.Connection.Logger()}
}

// Enabled diz se o cache esta ligado (ou seja, se existe Redis).
func (c *Cache) Enabled() bool {
	return c != nil && c.redis != nil
}

// cacheGet le a chave do cache e decodifica o JSON. O bool diz se achou (hit).
func cacheGet[T any](ctx context.Context, c *Cache, name, key string) (T, bool) {
	var zero T

	if !c.Enabled() {
		return zero, false
	}

	data, err := c.redis.Get(ctx, key)

	if err != nil {
		c.logger.Info("Cache miss for %s", name)

		return zero, false
	}

	var value T

	if err := json.Unmarshal([]byte(data), &value); err != nil {
		c.logger.Warning("Ignoring invalid cache entry for %s: %v", name, err)

		return zero, false
	}

	c.logger.Info("Cache hit for %s", name)

	return value, true
}

// set grava o valor (em JSON) no cache com o CacheTTL.
// E sincrono de proposito: um set assincrono podia chegar depois de uma invalidacao e deixar dado velho.
func (c *Cache) set(ctx context.Context, name, key string, value any) {
	if !c.Enabled() {
		return
	}

	data, err := json.Marshal(value)

	if err != nil {
		c.logger.Warning("Could not encode %s for cache: %v", name, err)

		return
	}

	if len(data) > maxCacheSize {
		c.logger.Warning("Not caching %s: it is too big (%d bytes)", name, len(data))

		return
	}

	c.redis.Set(ctx, key, data, CacheTTL)
}

// delete tira as chaves do cache.
func (c *Cache) delete(ctx context.Context, keys ...string) {
	if !c.Enabled() || len(keys) == 0 {
		return
	}

	c.redis.Delete(ctx, keys...)
}

// readThrough le do cache; se nao achar, chama load (o banco) e guarda o resultado no cache.
func readThrough[T any](ctx context.Context, c *Cache, name, key string, load func() (T, error)) (T, error) {
	if value, ok := cacheGet[T](ctx, c, name, key); ok {
		return value, nil
	}

	value, err := load()

	if err != nil {
		var zero T

		return zero, err
	}

	c.set(ctx, name, key, value)

	return value, nil
}
