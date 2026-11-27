package database

import (
	"context"
	"fmt"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
)

const usersAllKey = "users:all"

func userKey(id int64) string { return fmt.Sprintf("users:%d", id) }

// userEmailKey guarda so o id do usuario; o usuario inteiro fica na chave do id.
func userEmailKey(email string) string { return "users:email:" + email }

// Users e a tabela users com cache aside.
type Users struct {
	db    *operations.Users
	cache *Cache
}

// NewUsers cria as operacoes de users com cache.
func NewUsers(db *operations.Users, cache *Cache) *Users {
	return &Users{db: db, cache: cache}
}

// Create cria no banco e no cache.
func (u *Users) Create(ctx context.Context, user *operations.User) error {
	if err := u.db.Create(ctx, user); err != nil {
		return err
	}

	u.cache.set(ctx, "user", userKey(user.ID), user)
	u.cache.delete(ctx, usersAllKey)

	return nil
}

// GetByID le do cache e, se nao achar, do banco (e guarda no cache).
func (u *Users) GetByID(ctx context.Context, id int64) (*operations.User, error) {
	return readThrough(ctx, u.cache, "user", userKey(id), func() (*operations.User, error) {
		return u.db.GetByID(ctx, id)
	})
}

// GetByEmail acha o id pelo email no cache e le o usuario pelo id. Se o cache nao tiver
// ou estiver velho (o email do usuario mudou), le do banco e guarda no cache.
func (u *Users) GetByEmail(ctx context.Context, email string) (*operations.User, error) {
	if id, ok := cacheGet[int64](ctx, u.cache, "user email", userEmailKey(email)); ok {
		if user, err := u.GetByID(ctx, id); err == nil && user.Email == email {
			return user, nil
		}
	}

	user, err := u.db.GetByEmail(ctx, email)

	if err != nil {
		return nil, err
	}

	u.cache.set(ctx, "user", userKey(user.ID), user)
	u.cache.set(ctx, "user email", userEmailKey(email), user.ID)

	return user, nil
}

// List le a lista de usuarios do cache e, se nao achar, do banco.
func (u *Users) List(ctx context.Context) ([]operations.User, error) {
	return readThrough(ctx, u.cache, "users", usersAllKey, func() ([]operations.User, error) {
		return u.db.List(ctx)
	})
}

// Update atualiza no banco e tira do cache.
func (u *Users) Update(ctx context.Context, user *operations.User) error {
	if err := u.db.Update(ctx, user); err != nil {
		return err
	}

	u.cache.delete(ctx, userKey(user.ID), userEmailKey(user.Email), usersAllKey)

	return nil
}

// Delete apaga no banco e tira do cache.
func (u *Users) Delete(ctx context.Context, id int64) error {
	if err := u.db.Delete(ctx, id); err != nil {
		return err
	}

	u.cache.delete(ctx, userKey(id), usersAllKey)

	return nil
}
