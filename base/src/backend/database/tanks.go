package database

import (
	"context"
	"fmt"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
)

func tankKey(id int64) string { return fmt.Sprintf("tanks:%d", id) }

// tankListKey e a chave da lista de tanks de um farm.
func tankListKey(farmID int64) string { return fmt.Sprintf("tanks:farm:%d", farmID) }

// Tanks e a tabela tanks com cache aside.
type Tanks struct {
	db    *operations.Tanks
	cache *Cache
}

// NewTanks cria as operacoes de tanks com cache.
func NewTanks(db *operations.Tanks, cache *Cache) *Tanks {
	return &Tanks{db: db, cache: cache}
}

// Create cria no banco e no cache.
func (x *Tanks) Create(ctx context.Context, item *operations.Tank) error {
	if err := x.db.Create(ctx, item); err != nil {
		return err
	}

	x.cache.set(ctx, "tank", tankKey(item.ID), item)
	x.cache.delete(ctx, tankListKey(item.FarmID))

	return nil
}

// GetByID le do cache e, se nao achar, do banco (e guarda no cache).
func (x *Tanks) GetByID(ctx context.Context, id int64) (*operations.Tank, error) {
	return readThrough(ctx, x.cache, "tank", tankKey(id), func() (*operations.Tank, error) {
		return x.db.GetByID(ctx, id)
	})
}

// ListByFarm le a lista do cache e, se nao achar, do banco (e guarda no cache).
func (x *Tanks) ListByFarm(ctx context.Context, farmID int64) ([]operations.Tank, error) {
	return readThrough(ctx, x.cache, "tanks", tankListKey(farmID), func() ([]operations.Tank, error) {
		return x.db.ListByFarm(ctx, farmID)
	})
}

// Update atualiza no banco e tira do cache (o item e as listas, inclusive a do pai antigo).
func (x *Tanks) Update(ctx context.Context, item *operations.Tank) error {
	old, _ := x.GetByID(ctx, item.ID)

	if err := x.db.Update(ctx, item); err != nil {
		return err
	}

	keys := []string{tankKey(item.ID), tankListKey(item.FarmID)}

	if old != nil && old.FarmID != item.FarmID {
		keys = append(keys, tankListKey(old.FarmID))
	}

	x.cache.delete(ctx, keys...)

	return nil
}

// Delete apaga no banco e tira do cache (o item e a lista do pai).
func (x *Tanks) Delete(ctx context.Context, id int64) error {
	old, _ := x.GetByID(ctx, id)

	if err := x.db.Delete(ctx, id); err != nil {
		return err
	}

	keys := []string{tankKey(id)}

	if old != nil {
		keys = append(keys, tankListKey(old.FarmID))
	}

	x.cache.delete(ctx, keys...)

	return nil
}
