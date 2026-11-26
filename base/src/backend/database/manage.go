package database

import (
	"errors"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/redis"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository"
)

// Manage junta as operacoes com cache aside (Users, Farms, Tanks, Analyses e Contents)
// e, separados, o banco de dados (Repository) e o Redis (Redis).
type Manage struct {
	Repository *repository.Manage

	// Redis fica nil quando nao ha Redis. Ai as operacoes usam so o banco.
	Redis *redis.Manage

	Cache    *Cache
	Users    *Users
	Farms    *Farms
	Tanks    *Tanks
	Analyses *Analyses
	Contents *Contents
}

// NewManage junta o banco e o Redis. Passe redisManage nil para rodar sem cache.
func NewManage(repositoryManage *repository.Manage, redisManage *redis.Manage) *Manage {
	cache := NewCache(redisManage)
	ops := repositoryManage.Operations

	return &Manage{
		Repository: repositoryManage,
		Redis:      redisManage,
		Cache:      cache,
		Users:      NewUsers(ops.Users, cache),
		Farms:      NewFarms(ops.Farms, cache),
		Tanks:      NewTanks(ops.Tanks, cache),
		Analyses:   NewAnalyses(ops.Analyses, cache),
		Contents:   NewContents(ops.Contents, cache),
	}
}

// Close fecha o Redis (se houver) e o banco de dados.
func (m *Manage) Close() error {
	var errs []error

	if m.Redis != nil {
		errs = append(errs, m.Redis.Close())
	}

	errs = append(errs, m.Repository.Connection.Close())

	return errors.Join(errs...)
}
