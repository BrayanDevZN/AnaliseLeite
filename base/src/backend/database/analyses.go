package database

import (
	"context"
	"fmt"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
)

func analysisKey(id int64) string { return fmt.Sprintf("analyses:%d", id) }

// analysisListKey e a chave da lista de analyses de um farm.
func analysisListKey(farmID int64) string { return fmt.Sprintf("analyses:farm:%d", farmID) }

// Analyses e a tabela analyses com cache aside.
type Analyses struct {
	db    *operations.Analyses
	cache *Cache
}

// NewAnalyses cria as operacoes de analyses com cache.
func NewAnalyses(db *operations.Analyses, cache *Cache) *Analyses {
	return &Analyses{db: db, cache: cache}
}

// Create cria no banco e no cache.
func (x *Analyses) Create(ctx context.Context, item *operations.Analysis) error {
	if err := x.db.Create(ctx, item); err != nil {
		return err
	}

	x.cache.set(ctx, "analysis", analysisKey(item.ID), item)
	x.cache.delete(ctx, analysisListKey(item.FarmID))

	return nil
}

// GetByID le do cache e, se nao achar, do banco (e guarda no cache).
func (x *Analyses) GetByID(ctx context.Context, id int64) (*operations.Analysis, error) {
	return readThrough(ctx, x.cache, "analysis", analysisKey(id), func() (*operations.Analysis, error) {
		return x.db.GetByID(ctx, id)
	})
}

// ListByFarm le a lista do cache e, se nao achar, do banco (e guarda no cache).
func (x *Analyses) ListByFarm(ctx context.Context, farmID int64) ([]operations.Analysis, error) {
	return readThrough(ctx, x.cache, "analyses", analysisListKey(farmID), func() ([]operations.Analysis, error) {
		return x.db.ListByFarm(ctx, farmID)
	})
}

// Update atualiza no banco e tira do cache (o item e as listas, inclusive a do pai antigo).
func (x *Analyses) Update(ctx context.Context, item *operations.Analysis) error {
	old, _ := x.GetByID(ctx, item.ID)

	if err := x.db.Update(ctx, item); err != nil {
		return err
	}

	keys := []string{analysisKey(item.ID), analysisListKey(item.FarmID)}

	if old != nil && old.FarmID != item.FarmID {
		keys = append(keys, analysisListKey(old.FarmID))
	}

	x.cache.delete(ctx, keys...)

	return nil
}

// Delete apaga no banco e tira do cache (o item e a lista do pai).
func (x *Analyses) Delete(ctx context.Context, id int64) error {
	old, _ := x.GetByID(ctx, id)

	if err := x.db.Delete(ctx, id); err != nil {
		return err
	}

	keys := []string{analysisKey(id)}

	if old != nil {
		keys = append(keys, analysisListKey(old.FarmID))
	}

	x.cache.delete(ctx, keys...)

	return nil
}
