package unit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/redis"
)

// startRedisServer sobe um redis-server (ou valkey-server) numa porta livre e devolve a URL dele.
// Se nenhum dos dois estiver instalado, o teste e pulado.
func startRedisServer(t *testing.T) string {
	t.Helper()

	// no CI o Redis vem de um service: TEST_REDIS_URL aponta para ele.
	// Cuidado: o banco 0 dessa URL e limpo (FLUSHDB) a cada uso, entao nunca aponte para um Redis com dados.
	if url := os.Getenv("TEST_REDIS_URL"); url != "" {
		options, err := goredis.ParseURL(url)

		if err != nil {
			t.Fatalf("TEST_REDIS_URL invalida: %v", err)
		}

		client := goredis.NewClient(options)

		defer client.Close()

		if err := client.FlushDB(context.Background()).Err(); err != nil {
			t.Fatalf("limpar o Redis de teste: %v", err)
		}

		return url
	}

	binary, err := exec.LookPath("redis-server")

	if err != nil {
		binary, err = exec.LookPath("valkey-server")
	}

	if err != nil {
		t.Skip("redis-server nao esta instalado")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")

	if err != nil {
		t.Fatal(err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	listener.Close()

	command := exec.Command(binary, "--port", fmt.Sprint(port), "--bind", "127.0.0.1", "--save", "", "--appendonly", "no", "--dir", t.TempDir())

	if err := command.Start(); err != nil {
		t.Fatalf("subir o redis-server: %v", err)
	}

	t.Cleanup(func() {
		command.Process.Kill()
		command.Wait()
	})

	return fmt.Sprintf("redis://127.0.0.1:%d/0", port)
}

func newTestRedis(t *testing.T, ctx context.Context) *redis.Manage {
	t.Helper()

	connection := redis.Connect(ctx, backend.NewLoggerWriter(io.Discard),
		redis.WithURL(startRedisServer(t)), redis.WithRetryDelay(100*time.Millisecond))

	t.Cleanup(func() { connection.Close() })

	if err := connection.Wait(); err != nil {
		t.Fatalf("conectar no redis: %v", err)
	}

	return redis.NewManage(connection)
}

func TestRedis(t *testing.T) {
	ctx := context.Background()
	ops := newTestRedis(t, ctx).Operations

	t.Run("ping", func(t *testing.T) {
		must(t, ops.Ping(ctx))
	})

	t.Run("strings", func(t *testing.T) {
		must(t, ops.Set(ctx, "name", "ana", 0))

		got, err := ops.Get(ctx, "name")
		must(t, err)

		if got != "ana" {
			t.Fatalf("esperava ana, veio %q", got)
		}

		_, err = ops.Get(ctx, "missing")
		wantErr(t, err, redis.ErrNotFound)

		// SetNX so grava se a chave nao existe
		ok, err := ops.SetNX(ctx, "lock", "a", time.Minute)
		must(t, err)

		if !ok {
			t.Fatal("SetNX deveria gravar na primeira vez")
		}

		ok, err = ops.SetNX(ctx, "lock", "b", time.Minute)
		must(t, err)

		if ok {
			t.Fatal("SetNX nao deveria gravar numa chave que existe")
		}

		got, err = ops.GetDel(ctx, "lock")
		must(t, err)

		if got != "a" {
			t.Fatalf("GetDel devolveu %q, esperava a", got)
		}

		_, err = ops.GetDel(ctx, "lock")
		wantErr(t, err, redis.ErrNotFound)
	})

	t.Run("expiration", func(t *testing.T) {
		must(t, ops.Set(ctx, "temp", "x", time.Minute))

		ttl, err := ops.TTL(ctx, "temp")
		must(t, err)

		if ttl <= 0 || ttl > time.Minute {
			t.Fatalf("TTL errado: %v", ttl)
		}

		// chave sem expiracao devolve 0
		ttl, err = ops.TTL(ctx, "name")
		must(t, err)

		if ttl != 0 {
			t.Fatalf("chave sem expiracao deveria dar 0, veio %v", ttl)
		}

		ok, err := ops.Expire(ctx, "name", time.Hour)
		must(t, err)

		if !ok {
			t.Fatal("Expire deveria achar a chave")
		}

		ok, err = ops.Expire(ctx, "missing", time.Hour)
		must(t, err)

		if ok {
			t.Fatal("Expire nao deveria achar uma chave que nao existe")
		}

		_, err = ops.TTL(ctx, "missing")
		wantErr(t, err, redis.ErrNotFound)
	})

	t.Run("counters", func(t *testing.T) {
		value, err := ops.Incr(ctx, "counter")
		must(t, err)

		if value != 1 {
			t.Fatalf("Incr deveria comecar em 1, veio %d", value)
		}

		value, err = ops.IncrBy(ctx, "counter", 10)
		must(t, err)

		if value != 11 {
			t.Fatalf("esperava 11, veio %d", value)
		}

		value, err = ops.Decr(ctx, "counter")
		must(t, err)

		if value != 10 {
			t.Fatalf("esperava 10, veio %d", value)
		}

		value, err = ops.DecrBy(ctx, "counter", 4)
		must(t, err)

		if value != 6 {
			t.Fatalf("esperava 6, veio %d", value)
		}
	})

	t.Run("concurrent increments", func(t *testing.T) {
		const workers = 50

		var wait sync.WaitGroup

		errs := make(chan error, workers)

		for range workers {
			wait.Add(1)

			go func() {
				defer wait.Done()

				_, err := ops.Incr(ctx, "race")

				errs <- err
			}()
		}

		wait.Wait()
		close(errs)

		for err := range errs {
			must(t, err)
		}

		got, err := ops.Get(ctx, "race")
		must(t, err)

		if got != fmt.Sprint(workers) {
			t.Fatalf("esperava %d, veio %s", workers, got)
		}
	})

	t.Run("hash", func(t *testing.T) {
		added, err := ops.HSet(ctx, "farm:1", map[string]any{"name": "Boa", "liters": 100})
		must(t, err)

		if added != 2 {
			t.Fatalf("esperava 2 campos novos, veio %d", added)
		}

		got, err := ops.HGet(ctx, "farm:1", "name")
		must(t, err)

		if got != "Boa" {
			t.Fatalf("esperava Boa, veio %q", got)
		}

		_, err = ops.HGet(ctx, "farm:1", "missing")
		wantErr(t, err, redis.ErrNotFound)

		all, err := ops.HGetAll(ctx, "farm:1")
		must(t, err)

		if len(all) != 2 || all["liters"] != "100" {
			t.Fatalf("HGetAll errado: %v", all)
		}

		_, err = ops.HGetAll(ctx, "farm:missing")
		wantErr(t, err, redis.ErrNotFound)

		value, err := ops.HIncrBy(ctx, "farm:1", "liters", 5)
		must(t, err)

		if value != 105 {
			t.Fatalf("esperava 105, veio %d", value)
		}

		count, err := ops.HDel(ctx, "farm:1", "name", "missing")
		must(t, err)

		if count != 1 {
			t.Fatalf("HDel deveria apagar 1 campo, apagou %d", count)
		}
	})

	t.Run("list", func(t *testing.T) {
		length, err := ops.RPush(ctx, "queue", "b", "c")
		must(t, err)

		if length != 2 {
			t.Fatalf("esperava tamanho 2, veio %d", length)
		}

		length, err = ops.LPush(ctx, "queue", "a")
		must(t, err)

		if length != 3 {
			t.Fatalf("esperava tamanho 3, veio %d", length)
		}

		items, err := ops.LRange(ctx, "queue", 0, -1)
		must(t, err)

		if fmt.Sprint(items) != "[a b c]" {
			t.Fatalf("LRange errado: %v", items)
		}

		first, err := ops.LPop(ctx, "queue")
		must(t, err)

		last, err := ops.RPop(ctx, "queue")
		must(t, err)

		if first != "a" || last != "c" {
			t.Fatalf("esperava a e c, veio %q e %q", first, last)
		}

		size, err := ops.LLen(ctx, "queue")
		must(t, err)

		if size != 1 {
			t.Fatalf("esperava tamanho 1, veio %d", size)
		}

		_, err = ops.LPop(ctx, "queue:missing")
		wantErr(t, err, redis.ErrNotFound)

		_, err = ops.RPop(ctx, "queue:missing")
		wantErr(t, err, redis.ErrNotFound)
	})

	t.Run("set", func(t *testing.T) {
		added, err := ops.SAdd(ctx, "tags", "x", "y", "x")
		must(t, err)

		if added != 2 {
			t.Fatalf("esperava 2 valores novos, veio %d", added)
		}

		ok, err := ops.SIsMember(ctx, "tags", "x")
		must(t, err)

		if !ok {
			t.Fatal("x deveria estar no conjunto")
		}

		members, err := ops.SMembers(ctx, "tags")
		must(t, err)

		if len(members) != 2 {
			t.Fatalf("esperava 2 valores, veio %v", members)
		}

		removed, err := ops.SRem(ctx, "tags", "x", "z")
		must(t, err)

		if removed != 1 {
			t.Fatalf("SRem deveria tirar 1 valor, tirou %d", removed)
		}
	})

	t.Run("keys", func(t *testing.T) {
		count, err := ops.Exists(ctx, "name", "counter", "missing")
		must(t, err)

		if count != 2 {
			t.Fatalf("Exists deveria achar 2 chaves, achou %d", count)
		}

		count, err = ops.Delete(ctx, "name", "counter", "missing")
		must(t, err)

		if count != 2 {
			t.Fatalf("Delete deveria apagar 2 chaves, apagou %d", count)
		}

		_, err = ops.Get(ctx, "name")
		wantErr(t, err, redis.ErrNotFound)
	})
}

func TestRedisAsync(t *testing.T) {
	ctx := context.Background()
	manage := newTestRedis(t, ctx)
	ops := manage.Operations

	// 50 incrementos assincronos ao mesmo tempo nao podem perder nenhum
	const workers = 50

	for range workers {
		ops.IncrAsync(ctx, "async:counter")
	}

	ops.SetAsync(ctx, "async:key", "value", 0)

	// o ctx cancelado de quem chamou nao cancela a operacao
	canceled, cancel := context.WithCancel(ctx)

	cancel()

	ops.SetAsync(canceled, "async:canceled", "ok", 0)
	ops.Wait()

	for key, want := range map[string]string{"async:counter": fmt.Sprint(workers), "async:key": "value", "async:canceled": "ok"} {
		got, err := ops.Get(ctx, key)
		must(t, err)

		if got != want {
			t.Fatalf("%s: esperava %q, veio %q", key, want, got)
		}
	}

	ops.DeleteAsync(ctx, "async:key", "async:canceled", "async:missing")
	ops.Wait()

	count, err := ops.Exists(ctx, "async:key", "async:canceled")
	must(t, err)

	if count != 0 {
		t.Fatalf("DeleteAsync deveria ter apagado as chaves, sobraram %d", count)
	}

	// sem conexao nao pode travar nem dar panic
	none := redis.NewManage(nil).Operations

	none.IncrAsync(ctx, "x")
	none.Wait()

	must(t, manage.Close())
}

func TestRedisInvalidURLDoesNotRetry(t *testing.T) {
	calls := 0

	connection := redis.Connect(context.Background(), backend.NewLoggerWriter(io.Discard),
		redis.WithURL("http://localhost:6379"),
		redis.WithRetryDelay(time.Millisecond),
		redis.WithOpener(func(ctx context.Context, url string) (*goredis.Client, error) {
			calls++

			return nil, redis.ErrInvalidURL
		}),
	)

	wantErr(t, connection.Wait(), redis.ErrInvalidURL)

	if calls != 1 {
		t.Fatalf("URL invalida nao deveria ter nova tentativa, mas houve %d chamadas", calls)
	}

	if connection.Client() != nil {
		t.Fatal("Client deveria ser nil")
	}

	// URL invalida de verdade, sem o Opener trocado
	real := redis.Connect(context.Background(), backend.NewLoggerWriter(io.Discard), redis.WithURL("http://localhost:6379"))

	wantErr(t, real.Wait(), redis.ErrInvalidURL)
}

func TestRedisRetriesUntilItConnects(t *testing.T) {
	calls := 0

	connection := redis.Connect(context.Background(), backend.NewLoggerWriter(io.Discard),
		redis.WithURL("redis://localhost:6379/0"),
		redis.WithRetryDelay(time.Millisecond),
		redis.WithOpener(func(ctx context.Context, url string) (*goredis.Client, error) {
			calls++

			if calls < 3 {
				return nil, errors.New("down")
			}

			return goredis.NewClient(&goredis.Options{Addr: "localhost:6379"}), nil
		}),
	)

	must(t, connection.Wait())

	if calls != 3 || connection.Client() == nil {
		t.Fatalf("esperava conectar na 3a tentativa, foram %d chamadas", calls)
	}

	must(t, connection.Close())
}

func TestRedisGivesUpAfterRetries(t *testing.T) {
	calls := 0

	connection := redis.Connect(context.Background(), backend.NewLoggerWriter(io.Discard),
		redis.WithURL("redis://localhost:6379/0"),
		redis.WithRetryDelay(time.Millisecond),
		redis.WithOpener(func(ctx context.Context, url string) (*goredis.Client, error) {
			calls++

			return nil, errors.New("down")
		}),
	)

	if connection.Wait() == nil {
		t.Fatal("deveria falhar")
	}

	if calls != redis.Retries+1 {
		t.Fatalf("esperava %d tentativas, foram %d", redis.Retries+1, calls)
	}

	// operacoes sem conexao devolvem ErrNoConnection
	ops := redis.NewManage(connection).Operations

	wantErr(t, ops.Ping(context.Background()), redis.ErrNoConnection)
	must(t, connection.Close())
}

func TestRedisCanceledConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	connection := redis.Connect(ctx, backend.NewLoggerWriter(io.Discard),
		redis.WithURL("redis://localhost:6379/0"),
		redis.WithRetryDelay(time.Hour),
		redis.WithOpener(func(ctx context.Context, url string) (*goredis.Client, error) {
			cancel()

			return nil, errors.New("down")
		}),
	)

	wantErr(t, connection.Wait(), context.Canceled)
}

func TestRedisWithoutConnection(t *testing.T) {
	ops := redis.NewManage(nil).Operations

	_, err := ops.Get(context.Background(), "key")
	wantErr(t, err, redis.ErrNoConnection)
}
