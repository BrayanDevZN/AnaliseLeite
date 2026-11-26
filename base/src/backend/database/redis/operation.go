package redis

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// ErrNotFound quando a chave (ou o campo) nao existe.
var ErrNotFound = errors.New("key not found")

// asyncTimeout e o tempo maximo de uma operacao assincrona.
const asyncTimeout = 5 * time.Second

// Operations tem as operacoes do Redis, todas usando a mesma conexao.
type Operations struct {
	connection *Connection
	pending    sync.WaitGroup
}

// NewOperations cria as operacoes usando a conexao.
func NewOperations(connection *Connection) *Operations {
	return &Operations{connection: connection}
}

// begin loga o comeco da operacao e devolve a funcao que loga o fim.
// Use assim, com o erro de retorno nomeado: defer begin(connection, "set key")(&err)
// Nunca ponha chaves nem valores na descricao: eles podem ter dados pessoais.
func begin(connection *Connection, format string, args ...any) func(*error) {
	logger := connection.Logger()
	action := fmt.Sprintf(format, args...)

	logger.Info("Starting to %s...", action)

	return func(err *error) {
		switch {
		case *err == nil:
			logger.Success("Finished to %s with success", action)

		case errors.Is(*err, ErrNotFound):
			logger.Warning("Could not %s: %v", action, *err)

		default:
			logger.Error("Could not %s: %v", action, *err)
		}
	}
}

// Wait espera todas as operacoes assincronas (SetAsync, IncrAsync, DeleteAsync) que ainda estao rodando.
func (o *Operations) Wait() {
	o.pending.Wait()
}

// async roda a operacao numa goroutine e volta na hora, sem devolver nada.
// O erro nao se perde: as operacoes ja logam o comeco e o fim (com o erro, se houver).
// A operacao nao e cancelada se o ctx de quem chamou for cancelado (ex.: o fim de uma
// requisicao), mas tem um tempo maximo de asyncTimeout. Use Wait para esperar as que ainda rodam.
func (o *Operations) async(ctx context.Context, run func(context.Context)) {
	ctx = context.WithoutCancel(ctx)

	o.pending.Add(1)

	go func() {
		defer o.pending.Done()

		ctx, cancel := context.WithTimeout(ctx, asyncTimeout)

		defer cancel()

		run(ctx)
	}()
}

// client espera a conexao terminar e devolve o cliente.
func (o *Operations) client() (*goredis.Client, error) {
	if o.connection == nil {
		return nil, ErrNoConnection
	}

	client := o.connection.Client()

	if client == nil {
		if err := o.connection.Wait(); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrNoConnection, err)
		}

		return nil, ErrNoConnection
	}

	return client, nil
}

// notFound troca o goredis.Nil (chave que nao existe) pelo ErrNotFound.
func notFound(err error) error {
	if errors.Is(err, goredis.Nil) {
		return ErrNotFound
	}

	return err
}

// Ping testa se o Redis responde.
func (o *Operations) Ping(ctx context.Context) (err error) {
	defer begin(o.connection, "ping")(&err)

	client, err := o.client()

	if err != nil {
		return err
	}

	return client.Ping(ctx).Err()
}

// ---- strings ----

// Set grava o valor na chave. Com ttl maior que zero a chave expira depois desse tempo.
func (o *Operations) Set(ctx context.Context, key string, value any, ttl time.Duration) (err error) {
	defer begin(o.connection, "set key")(&err)

	client, err := o.client()

	if err != nil {
		return err
	}

	return client.Set(ctx, key, value, ttl).Err()
}

// SetNX grava so se a chave ainda nao existir (serve de lock). Devolve true se gravou.
func (o *Operations) SetNX(ctx context.Context, key string, value any, ttl time.Duration) (ok bool, err error) {
	defer begin(o.connection, "set key if not exists")(&err)

	client, err := o.client()

	if err != nil {
		return false, err
	}

	return client.SetNX(ctx, key, value, ttl).Result()
}

// Get devolve o valor da chave, ou ErrNotFound se ela nao existir.
func (o *Operations) Get(ctx context.Context, key string) (value string, err error) {
	defer begin(o.connection, "get key")(&err)

	client, err := o.client()

	if err != nil {
		return "", err
	}

	value, err = client.Get(ctx, key).Result()

	return value, notFound(err)
}

// GetDel devolve o valor da chave e apaga ela, tudo de uma vez. ErrNotFound se ela nao existir.
func (o *Operations) GetDel(ctx context.Context, key string) (value string, err error) {
	defer begin(o.connection, "get and delete key")(&err)

	client, err := o.client()

	if err != nil {
		return "", err
	}

	value, err = client.GetDel(ctx, key).Result()

	return value, notFound(err)
}

// SetAsync e o Set numa goroutine: volta na hora e nao devolve nada.
func (o *Operations) SetAsync(ctx context.Context, key string, value any, ttl time.Duration) {
	o.async(ctx, func(ctx context.Context) {
		o.Set(ctx, key, value, ttl)
	})
}

// ---- contadores ----

// Incr soma 1 ao numero da chave (comeca em 0 se nao existir) e devolve o novo valor.
func (o *Operations) Incr(ctx context.Context, key string) (value int64, err error) {
	defer begin(o.connection, "increment key")(&err)

	client, err := o.client()

	if err != nil {
		return 0, err
	}

	return client.Incr(ctx, key).Result()
}

// IncrAsync e o Incr numa goroutine: volta na hora e nao devolve nada.
func (o *Operations) IncrAsync(ctx context.Context, key string) {
	o.async(ctx, func(ctx context.Context) {
		o.Incr(ctx, key)
	})
}

// IncrBy soma amount ao numero da chave e devolve o novo valor.
func (o *Operations) IncrBy(ctx context.Context, key string, amount int64) (value int64, err error) {
	defer begin(o.connection, "increment key by amount")(&err)

	client, err := o.client()

	if err != nil {
		return 0, err
	}

	return client.IncrBy(ctx, key, amount).Result()
}

// Decr tira 1 do numero da chave e devolve o novo valor.
func (o *Operations) Decr(ctx context.Context, key string) (value int64, err error) {
	defer begin(o.connection, "decrement key")(&err)

	client, err := o.client()

	if err != nil {
		return 0, err
	}

	return client.Decr(ctx, key).Result()
}

// DecrBy tira amount do numero da chave e devolve o novo valor.
func (o *Operations) DecrBy(ctx context.Context, key string, amount int64) (value int64, err error) {
	defer begin(o.connection, "decrement key by amount")(&err)

	client, err := o.client()

	if err != nil {
		return 0, err
	}

	return client.DecrBy(ctx, key, amount).Result()
}

// ---- chaves ----

// Delete apaga as chaves e devolve quantas existiam.
func (o *Operations) Delete(ctx context.Context, keys ...string) (count int64, err error) {
	defer begin(o.connection, "delete %d keys", len(keys))(&err)

	client, err := o.client()

	if err != nil {
		return 0, err
	}

	return client.Del(ctx, keys...).Result()
}

// DeleteAsync e o Delete numa goroutine: volta na hora e nao devolve nada.
func (o *Operations) DeleteAsync(ctx context.Context, keys ...string) {
	o.async(ctx, func(ctx context.Context) {
		o.Delete(ctx, keys...)
	})
}

// Exists devolve quantas das chaves existem.
func (o *Operations) Exists(ctx context.Context, keys ...string) (count int64, err error) {
	defer begin(o.connection, "check %d keys", len(keys))(&err)

	client, err := o.client()

	if err != nil {
		return 0, err
	}

	return client.Exists(ctx, keys...).Result()
}

// Expire faz a chave expirar depois do ttl. Devolve false se a chave nao existe.
func (o *Operations) Expire(ctx context.Context, key string, ttl time.Duration) (ok bool, err error) {
	defer begin(o.connection, "set key expiration")(&err)

	client, err := o.client()

	if err != nil {
		return false, err
	}

	return client.Expire(ctx, key, ttl).Result()
}

// TTL devolve quanto falta para a chave expirar.
// Devolve ErrNotFound se a chave nao existe e 0 se ela existe mas nao expira.
func (o *Operations) TTL(ctx context.Context, key string) (ttl time.Duration, err error) {
	defer begin(o.connection, "get key expiration")(&err)

	client, err := o.client()

	if err != nil {
		return 0, err
	}

	ttl, err = client.TTL(ctx, key).Result()

	if err != nil {
		return 0, err
	}

	switch ttl {
	case -2:
		return 0, ErrNotFound

	case -1:
		return 0, nil
	}

	return ttl, nil
}

// ---- hash ----

// HSet grava os campos (campo -> valor) no hash e devolve quantos campos eram novos.
func (o *Operations) HSet(ctx context.Context, key string, values map[string]any) (added int64, err error) {
	defer begin(o.connection, "set %d hash fields", len(values))(&err)

	client, err := o.client()

	if err != nil {
		return 0, err
	}

	return client.HSet(ctx, key, values).Result()
}

// HGet devolve um campo do hash, ou ErrNotFound se ele nao existir.
func (o *Operations) HGet(ctx context.Context, key, field string) (value string, err error) {
	defer begin(o.connection, "get hash field")(&err)

	client, err := o.client()

	if err != nil {
		return "", err
	}

	value, err = client.HGet(ctx, key, field).Result()

	return value, notFound(err)
}

// HGetAll devolve todos os campos do hash. Se o hash nao existe, devolve ErrNotFound.
func (o *Operations) HGetAll(ctx context.Context, key string) (values map[string]string, err error) {
	defer begin(o.connection, "get all hash fields")(&err)

	client, err := o.client()

	if err != nil {
		return nil, err
	}

	values, err = client.HGetAll(ctx, key).Result()

	if err != nil {
		return nil, err
	}

	if len(values) == 0 {
		return nil, ErrNotFound
	}

	return values, nil
}

// HDel apaga campos do hash e devolve quantos existiam.
func (o *Operations) HDel(ctx context.Context, key string, fields ...string) (count int64, err error) {
	defer begin(o.connection, "delete %d hash fields", len(fields))(&err)

	client, err := o.client()

	if err != nil {
		return 0, err
	}

	return client.HDel(ctx, key, fields...).Result()
}

// HIncrBy soma amount ao numero de um campo do hash e devolve o novo valor.
func (o *Operations) HIncrBy(ctx context.Context, key, field string, amount int64) (value int64, err error) {
	defer begin(o.connection, "increment hash field")(&err)

	client, err := o.client()

	if err != nil {
		return 0, err
	}

	return client.HIncrBy(ctx, key, field, amount).Result()
}

// ---- lista ----

// LPush poe os valores no comeco da lista e devolve o tamanho dela.
func (o *Operations) LPush(ctx context.Context, key string, values ...any) (length int64, err error) {
	defer begin(o.connection, "push %d values to list head", len(values))(&err)

	client, err := o.client()

	if err != nil {
		return 0, err
	}

	return client.LPush(ctx, key, values...).Result()
}

// RPush poe os valores no fim da lista e devolve o tamanho dela.
func (o *Operations) RPush(ctx context.Context, key string, values ...any) (length int64, err error) {
	defer begin(o.connection, "push %d values to list tail", len(values))(&err)

	client, err := o.client()

	if err != nil {
		return 0, err
	}

	return client.RPush(ctx, key, values...).Result()
}

// LPop tira e devolve o primeiro item da lista, ou ErrNotFound se ela estiver vazia.
func (o *Operations) LPop(ctx context.Context, key string) (value string, err error) {
	defer begin(o.connection, "pop list head")(&err)

	client, err := o.client()

	if err != nil {
		return "", err
	}

	value, err = client.LPop(ctx, key).Result()

	return value, notFound(err)
}

// RPop tira e devolve o ultimo item da lista, ou ErrNotFound se ela estiver vazia.
func (o *Operations) RPop(ctx context.Context, key string) (value string, err error) {
	defer begin(o.connection, "pop list tail")(&err)

	client, err := o.client()

	if err != nil {
		return "", err
	}

	value, err = client.RPop(ctx, key).Result()

	return value, notFound(err)
}

// LRange devolve os itens da lista de start ate stop (-1 e o ultimo).
func (o *Operations) LRange(ctx context.Context, key string, start, stop int64) (values []string, err error) {
	defer begin(o.connection, "get list range")(&err)

	client, err := o.client()

	if err != nil {
		return nil, err
	}

	return client.LRange(ctx, key, start, stop).Result()
}

// LLen devolve o tamanho da lista.
func (o *Operations) LLen(ctx context.Context, key string) (length int64, err error) {
	defer begin(o.connection, "get list length")(&err)

	client, err := o.client()

	if err != nil {
		return 0, err
	}

	return client.LLen(ctx, key).Result()
}

// ---- conjunto (set) ----

// SAdd poe os valores no conjunto e devolve quantos eram novos.
func (o *Operations) SAdd(ctx context.Context, key string, values ...any) (added int64, err error) {
	defer begin(o.connection, "add %d values to set", len(values))(&err)

	client, err := o.client()

	if err != nil {
		return 0, err
	}

	return client.SAdd(ctx, key, values...).Result()
}

// SRem tira os valores do conjunto e devolve quantos existiam.
func (o *Operations) SRem(ctx context.Context, key string, values ...any) (removed int64, err error) {
	defer begin(o.connection, "remove %d values from set", len(values))(&err)

	client, err := o.client()

	if err != nil {
		return 0, err
	}

	return client.SRem(ctx, key, values...).Result()
}

// SMembers devolve todos os valores do conjunto.
func (o *Operations) SMembers(ctx context.Context, key string) (values []string, err error) {
	defer begin(o.connection, "get set members")(&err)

	client, err := o.client()

	if err != nil {
		return nil, err
	}

	return client.SMembers(ctx, key).Result()
}

// SIsMember diz se o valor esta no conjunto.
func (o *Operations) SIsMember(ctx context.Context, key string, value any) (ok bool, err error) {
	defer begin(o.connection, "check set member")(&err)

	client, err := o.client()

	if err != nil {
		return false, err
	}

	return client.SIsMember(ctx, key, value).Result()
}
