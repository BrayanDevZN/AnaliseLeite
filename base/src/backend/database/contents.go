package database

import (
	"context"
	"fmt"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
)

func contentKey(id int64) string { return fmt.Sprintf("contents:%d", id) }

// contentListKey e a chave da lista de contents de um analysis.
func contentListKey(analysisID int64) string { return fmt.Sprintf("contents:analysis:%d", analysisID) }

// Contents e a tabela contents com cache aside.
type Contents struct {
	db    *operations.Contents
	cache *Cache
}

// NewContents cria as operacoes de contents com cache.
func NewContents(db *operations.Contents, cache *Cache) *Contents {
	return &Contents{db: db, cache: cache}
}

// Create cria no banco e no cache.
func (x *Contents) Create(ctx context.Context, item *operations.Content) error {
	if err := x.db.Create(ctx, item); err != nil {
		return err
	}

	x.cache.set(ctx, "content", contentKey(item.ID), item)
	x.cache.delete(ctx, contentListKey(item.AnalysisID))

	return nil
}

// GetByID le do cache e, se nao achar, do banco (e guarda no cache).
func (x *Contents) GetByID(ctx context.Context, id int64) (*operations.Content, error) {
	return readThrough(ctx, x.cache, "content", contentKey(id), func() (*operations.Content, error) {
		return x.db.GetByID(ctx, id)
	})
}

// ListByAnalysis le a lista do cache e, se nao achar, do banco (e guarda no cache).
func (x *Contents) ListByAnalysis(ctx context.Context, analysisID int64) ([]operations.Content, error) {
	return readThrough(ctx, x.cache, "contents", contentListKey(analysisID), func() ([]operations.Content, error) {
		return x.db.ListByAnalysis(ctx, analysisID)
	})
}

// Update atualiza no banco e tira do cache (o item e as listas, inclusive a do pai antigo).
func (x *Contents) Update(ctx context.Context, item *operations.Content) error {
	old, _ := x.GetByID(ctx, item.ID)

	if err := x.db.Update(ctx, item); err != nil {
		return err
	}

	keys := []string{contentKey(item.ID), contentListKey(item.AnalysisID)}

	if old != nil && old.AnalysisID != item.AnalysisID {
		keys = append(keys, contentListKey(old.AnalysisID))
	}

	x.cache.delete(ctx, keys...)

	return nil
}

// Delete apaga no banco e tira do cache (o item e a lista do pai).
func (x *Contents) Delete(ctx context.Context, id int64) error {
	old, _ := x.GetByID(ctx, id)

	if err := x.db.Delete(ctx, id); err != nil {
		return err
	}

	keys := []string{contentKey(id)}

	if old != nil {
		keys = append(keys, contentListKey(old.AnalysisID))
	}

	x.cache.delete(ctx, keys...)

	return nil
}
