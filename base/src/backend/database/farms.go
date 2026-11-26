package database

import (
	"context"
	"fmt"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
)

func farmKey(id int64) string { return fmt.Sprintf("farms:%d", id) }

// farmListKey e a chave da lista de farms de um user.
func farmListKey(userID int64) string { return fmt.Sprintf("farms:user:%d", userID) }

// Farms e a tabela farms com cache aside.
type Farms struct {
	db    *operations.Farms
	cache *Cache
}

// NewFarms cria as operacoes de farms com cache.
func NewFarms(db *operations.Farms, cache *Cache) *Farms {
	return &Farms{db: db, cache: cache}
}

// Create cria no banco e no cache.
func (x *Farms) Create(ctx context.Context, item *operations.Farm) error {
	if err := x.db.Create(ctx, item); err != nil {
		return err
	}

	x.cache.set(ctx, "farm", farmKey(item.ID), item)
	x.cache.delete(ctx, farmListKey(item.UserID))

	return nil
}

// GetByID le do cache e, se nao achar, do banco (e guarda no cache).
func (x *Farms) GetByID(ctx context.Context, id int64) (*operations.Farm, error) {
	return readThrough(ctx, x.cache, "farm", farmKey(id), func() (*operations.Farm, error) {
		return x.db.GetByID(ctx, id)
	})
}

// ListByUser le a lista do cache e, se nao achar, do banco (e guarda no cache).
func (x *Farms) ListByUser(ctx context.Context, userID int64) ([]operations.Farm, error) {
	return readThrough(ctx, x.cache, "farms", farmListKey(userID), func() ([]operations.Farm, error) {
		return x.db.ListByUser(ctx, userID)
	})
}

// Update atualiza no banco e tira do cache (o item e as listas, inclusive a do pai antigo).
func (x *Farms) Update(ctx context.Context, item *operations.Farm) error {
	old, _ := x.GetByID(ctx, item.ID)

	if err := x.db.Update(ctx, item); err != nil {
		return err
	}

	keys := []string{farmKey(item.ID), farmListKey(item.UserID)}

	if old != nil && old.UserID != item.UserID {
		keys = append(keys, farmListKey(old.UserID))
	}

	x.cache.delete(ctx, keys...)

	return nil
}

// Delete apaga no banco e tira do cache (o item e a lista do pai).
func (x *Farms) Delete(ctx context.Context, id int64) error {
	old, _ := x.GetByID(ctx, id)

	if err := x.db.Delete(ctx, id); err != nil {
		return err
	}

	keys := []string{farmKey(id)}

	if old != nil {
		keys = append(keys, farmListKey(old.UserID))
	}

	x.cache.delete(ctx, keys...)

	return nil
}

// GetPhoto le a foto da fazenda direto do banco. Ela nao passa pelo cache: e grande e e pedida de vez em quando.
func (x *Farms) GetPhoto(ctx context.Context, id int64) ([]byte, error) {
	return x.db.GetPhoto(ctx, id)
}

// SetPhoto grava (ou, com nil, remove) a foto e tira a fazenda do cache, porque o has_photo dela mudou.
func (x *Farms) SetPhoto(ctx context.Context, id int64, photo []byte) error {
	old, _ := x.GetByID(ctx, id)

	if err := x.db.SetPhoto(ctx, id, photo); err != nil {
		return err
	}

	keys := []string{farmKey(id)}

	if old != nil {
		keys = append(keys, farmListKey(old.UserID))
	}

	x.cache.delete(ctx, keys...)

	return nil
}
