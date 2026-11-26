package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/redis"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
)

func quiet() *backend.Logger { return backend.NewLoggerWriter(io.Discard) }

func must(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}

func wantErr(t *testing.T, got, want error) {
	t.Helper()

	if !errors.Is(got, want) {
		t.Fatalf("esperava o erro %v, veio %v", want, got)
	}
}

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

// newRepository abre um SQLite novo e cria as tabelas.
func newRepository(t *testing.T, ctx context.Context) *repository.Manage {
	t.Helper()

	url := "sqlite://" + filepath.ToSlash(filepath.Join(t.TempDir(), "test.db"))
	connection := repository.Connect(ctx, quiet(), repository.WithURL(url))

	must(t, connection.Wait())

	manage := repository.NewManage(connection)

	must(t, manage.Migrate.CreateTables(ctx))

	return manage
}

func newRedis(t *testing.T, ctx context.Context, url string) *redis.Manage {
	t.Helper()

	connection := redis.Connect(ctx, quiet(), redis.WithURL(url), redis.WithRetryDelay(100*time.Millisecond))

	must(t, connection.Wait())

	return redis.NewManage(connection)
}

// newManage monta o Manage com banco SQLite e, se withRedis, com um Redis de verdade.
func newManage(t *testing.T, ctx context.Context, withRedis bool) *database.Manage {
	t.Helper()

	var redisManage *redis.Manage

	if withRedis {
		redisManage = newRedis(t, ctx, startRedisServer(t))
	}

	manage := database.NewManage(newRepository(t, ctx), redisManage)

	t.Cleanup(func() { manage.Close() })

	return manage
}

// cached confere se a chave esta no Redis e se o TTL dela e de no maximo 60 segundos.
func cached(t *testing.T, ctx context.Context, manage *database.Manage, key string) {
	t.Helper()

	count, err := manage.Redis.Operations.Exists(ctx, key)
	must(t, err)

	if count != 1 {
		t.Fatalf("a chave %s deveria estar no cache", key)
	}

	ttl, err := manage.Redis.Operations.TTL(ctx, key)
	must(t, err)

	if ttl <= 0 || ttl > database.CacheTTL {
		t.Fatalf("o TTL de %s deveria ficar entre 0 e %v, veio %v", key, database.CacheTTL, ttl)
	}
}

func notCached(t *testing.T, ctx context.Context, manage *database.Manage, key string) {
	t.Helper()

	count, err := manage.Redis.Operations.Exists(ctx, key)
	must(t, err)

	if count != 0 {
		t.Fatalf("a chave %s nao deveria estar no cache", key)
	}
}

func TestCacheAsideWithRedis(t *testing.T) {
	ctx := context.Background()
	manage := newManage(t, ctx, true)
	direct := manage.Repository.Operations // escreve direto no banco, sem passar pelo cache

	user := &operations.User{Name: "Ana", Email: "ana@example.com", Role: "admin"}
	farm := &operations.Farm{Name: "Boa", City: "Uberaba", State: "MG"}
	tank := &operations.Tank{Type: "expansao", CapacityLiters: 1000}
	analysis := &operations.Analysis{}
	content := &operations.Content{Content: []byte(`{"temperature":4.2}`)}

	t.Run("users", func(t *testing.T) {
		must(t, manage.Users.Create(ctx, user))
		cached(t, ctx, manage, fmt.Sprintf("users:%d", user.ID))

		// alterar direto no banco: a leitura ainda vem do cache (dado velho)
		changed := *user
		changed.Name = "Banco"

		must(t, direct.Users.Update(ctx, &changed))

		got, err := manage.Users.GetByID(ctx, user.ID)
		must(t, err)

		if got.Name != "Ana" {
			t.Fatalf("a leitura deveria vir do cache (Ana), veio %q", got.Name)
		}

		// atualizar pelo Manage tira do cache; a leitura seguinte vai ao banco e guarda de novo
		user.Name = "Ana Maria"

		must(t, manage.Users.Update(ctx, user))
		notCached(t, ctx, manage, fmt.Sprintf("users:%d", user.ID))

		got, err = manage.Users.GetByID(ctx, user.ID)
		must(t, err)

		if got.Name != "Ana Maria" {
			t.Fatalf("esperava Ana Maria, veio %q", got.Name)
		}

		cached(t, ctx, manage, fmt.Sprintf("users:%d", user.ID))

		// por email: acha pelo cache depois da primeira vez
		byEmail, err := manage.Users.GetByEmail(ctx, user.Email)
		must(t, err)

		if byEmail.ID != user.ID {
			t.Fatalf("GetByEmail devolveu o usuario %d, esperava %d", byEmail.ID, user.ID)
		}

		cached(t, ctx, manage, "users:email:"+user.Email)

		byEmail, err = manage.Users.GetByEmail(ctx, user.Email)
		must(t, err)

		if byEmail.Name != "Ana Maria" {
			t.Fatalf("GetByEmail pelo cache: esperava Ana Maria, veio %q", byEmail.Name)
		}

		// o email mudou: o indice velho nao pode devolver o usuario errado
		old := user.Email
		user.Email = "novo@example.com"

		must(t, manage.Users.Update(ctx, user))

		_, err = manage.Users.GetByEmail(ctx, old)
		wantErr(t, err, operations.ErrNotFound)

		byEmail, err = manage.Users.GetByEmail(ctx, user.Email)
		must(t, err)

		if byEmail.ID != user.ID {
			t.Fatal("GetByEmail com o email novo deveria achar o usuario")
		}

		// lista: guarda no cache, e criar outro usuario invalida
		list, err := manage.Users.List(ctx)
		must(t, err)

		if len(list) != 1 {
			t.Fatalf("esperava 1 usuario, veio %d", len(list))
		}

		cached(t, ctx, manage, "users:all")

		other := &operations.User{Name: "Beto", Email: "beto@example.com", Role: "user"}

		must(t, manage.Users.Create(ctx, other))
		notCached(t, ctx, manage, "users:all")

		list, err = manage.Users.List(ctx)
		must(t, err)

		if len(list) != 2 {
			t.Fatalf("esperava 2 usuarios, veio %d", len(list))
		}

		// apagar tira do cache
		must(t, manage.Users.Delete(ctx, other.ID))
		notCached(t, ctx, manage, fmt.Sprintf("users:%d", other.ID))

		_, err = manage.Users.GetByID(ctx, other.ID)
		wantErr(t, err, operations.ErrNotFound)
	})

	t.Run("farms", func(t *testing.T) {
		farm.UserID = user.ID

		must(t, manage.Farms.Create(ctx, farm))
		cached(t, ctx, manage, fmt.Sprintf("farms:%d", farm.ID))

		changed := *farm
		changed.Name = "Banco"

		must(t, direct.Farms.Update(ctx, &changed))

		got, err := manage.Farms.GetByID(ctx, farm.ID)
		must(t, err)

		if got.Name != "Boa" {
			t.Fatalf("a leitura deveria vir do cache (Boa), veio %q", got.Name)
		}

		list, err := manage.Farms.ListByUser(ctx, user.ID)
		must(t, err)

		if len(list) != 1 {
			t.Fatalf("esperava 1 fazenda, veio %d", len(list))
		}

		cached(t, ctx, manage, fmt.Sprintf("farms:user:%d", user.ID))

		farm.Name = "Nova"

		must(t, manage.Farms.Update(ctx, farm))
		notCached(t, ctx, manage, fmt.Sprintf("farms:%d", farm.ID))
		notCached(t, ctx, manage, fmt.Sprintf("farms:user:%d", user.ID))

		got, err = manage.Farms.GetByID(ctx, farm.ID)
		must(t, err)

		if got.Name != "Nova" {
			t.Fatalf("esperava Nova, veio %q", got.Name)
		}
	})

	t.Run("tanks", func(t *testing.T) {
		tank.FarmID = farm.ID

		must(t, manage.Tanks.Create(ctx, tank))
		cached(t, ctx, manage, fmt.Sprintf("tanks:%d", tank.ID))

		changed := *tank
		changed.CapacityLiters = 5

		must(t, direct.Tanks.Update(ctx, &changed))

		got, err := manage.Tanks.GetByID(ctx, tank.ID)
		must(t, err)

		if got.CapacityLiters != 1000 {
			t.Fatalf("a leitura deveria vir do cache (1000), veio %v", got.CapacityLiters)
		}

		list, err := manage.Tanks.ListByFarm(ctx, farm.ID)
		must(t, err)

		if len(list) != 1 {
			t.Fatalf("esperava 1 tanque, veio %d", len(list))
		}

		cached(t, ctx, manage, fmt.Sprintf("tanks:farm:%d", farm.ID))

		tank.CapacityLiters = 2000

		must(t, manage.Tanks.Update(ctx, tank))
		notCached(t, ctx, manage, fmt.Sprintf("tanks:%d", tank.ID))
		notCached(t, ctx, manage, fmt.Sprintf("tanks:farm:%d", farm.ID))

		got, err = manage.Tanks.GetByID(ctx, tank.ID)
		must(t, err)

		if got.CapacityLiters != 2000 {
			t.Fatalf("esperava 2000, veio %v", got.CapacityLiters)
		}
	})

	t.Run("analyses", func(t *testing.T) {
		analysis.FarmID = farm.ID
		analysis.TankID = &tank.ID

		must(t, manage.Analyses.Create(ctx, analysis))
		cached(t, ctx, manage, fmt.Sprintf("analyses:%d", analysis.ID))

		got, err := manage.Analyses.GetByID(ctx, analysis.ID)
		must(t, err)

		if got.TankID == nil || *got.TankID != tank.ID {
			t.Fatalf("o tanque deveria sobreviver ao cache: %+v", got)
		}

		// sem tanque (NULL) tambem passa pelo cache
		noTank := &operations.Analysis{FarmID: farm.ID}

		must(t, manage.Analyses.Create(ctx, noTank))

		got, err = manage.Analyses.GetByID(ctx, noTank.ID)
		must(t, err)

		if got.TankID != nil {
			t.Fatalf("TankID deveria ser nil: %+v", got)
		}

		list, err := manage.Analyses.ListByFarm(ctx, farm.ID)
		must(t, err)

		if len(list) != 2 {
			t.Fatalf("esperava 2 analises, veio %d", len(list))
		}

		cached(t, ctx, manage, fmt.Sprintf("analyses:farm:%d", farm.ID))

		must(t, manage.Analyses.Delete(ctx, noTank.ID))
		notCached(t, ctx, manage, fmt.Sprintf("analyses:%d", noTank.ID))
		notCached(t, ctx, manage, fmt.Sprintf("analyses:farm:%d", farm.ID))

		_, err = manage.Analyses.GetByID(ctx, noTank.ID)
		wantErr(t, err, operations.ErrNotFound)
	})

	t.Run("contents", func(t *testing.T) {
		content.AnalysisID = analysis.ID

		must(t, manage.Contents.Create(ctx, content))
		cached(t, ctx, manage, fmt.Sprintf("contents:%d", content.ID))

		got, err := manage.Contents.GetByID(ctx, content.ID)
		must(t, err)

		if string(got.Content) != `{"temperature":4.2}` || got.File != nil {
			t.Fatalf("o conteudo deveria sobreviver ao cache: %+v", got)
		}

		changed := *content
		changed.Content = []byte(`{"temperature":9}`)

		must(t, direct.Contents.Update(ctx, &changed))

		got, err = manage.Contents.GetByID(ctx, content.ID)
		must(t, err)

		if string(got.Content) != `{"temperature":4.2}` {
			t.Fatalf("a leitura deveria vir do cache, veio %s", got.Content)
		}

		list, err := manage.Contents.ListByAnalysis(ctx, analysis.ID)
		must(t, err)

		if len(list) != 1 {
			t.Fatalf("esperava 1 conteudo, veio %d", len(list))
		}

		cached(t, ctx, manage, fmt.Sprintf("contents:analysis:%d", analysis.ID))

		// arquivo grande demais nao vai para o cache, mas fica no banco
		big := &operations.Content{AnalysisID: analysis.ID, File: make([]byte, 2<<20)}

		must(t, manage.Contents.Create(ctx, big))
		notCached(t, ctx, manage, fmt.Sprintf("contents:%d", big.ID))

		got, err = manage.Contents.GetByID(ctx, big.ID)
		must(t, err)

		if len(got.File) != 2<<20 {
			t.Fatalf("o arquivo deveria vir do banco inteiro, veio %d bytes", len(got.File))
		}

		content.Content = []byte(`{"temperature":7}`)

		must(t, manage.Contents.Update(ctx, content))
		notCached(t, ctx, manage, fmt.Sprintf("contents:%d", content.ID))

		must(t, manage.Contents.Delete(ctx, content.ID))
		must(t, manage.Contents.Delete(ctx, big.ID))

		_, err = manage.Contents.GetByID(ctx, content.ID)
		wantErr(t, err, operations.ErrNotFound)
	})

	t.Run("delete the rest", func(t *testing.T) {
		must(t, manage.Analyses.Delete(ctx, analysis.ID))
		must(t, manage.Tanks.Delete(ctx, tank.ID))

		must(t, manage.Farms.Delete(ctx, farm.ID))
		notCached(t, ctx, manage, fmt.Sprintf("farms:%d", farm.ID))
		notCached(t, ctx, manage, fmt.Sprintf("farms:user:%d", user.ID))

		_, err := manage.Farms.GetByID(ctx, farm.ID)
		wantErr(t, err, operations.ErrNotFound)

		// erro de "nao achou" nao vai para o cache
		notCached(t, ctx, manage, fmt.Sprintf("farms:%d", farm.ID))
	})
}

func TestWithoutRedis(t *testing.T) {
	ctx := context.Background()
	manage := newManage(t, ctx, false)

	if manage.Redis != nil || manage.Cache.Enabled() {
		t.Fatal("sem Redis, o cache deveria ficar desligado")
	}

	user := &operations.User{Name: "Ana", Email: "ana@example.com", Role: "admin"}

	must(t, manage.Users.Create(ctx, user))

	// sem cache, a leitura sempre mostra o que esta no banco
	changed := *user
	changed.Name = "Banco"

	must(t, manage.Repository.Operations.Users.Update(ctx, &changed))

	got, err := manage.Users.GetByID(ctx, user.ID)
	must(t, err)

	if got.Name != "Banco" {
		t.Fatalf("sem cache deveria ler do banco (Banco), veio %q", got.Name)
	}

	byEmail, err := manage.Users.GetByEmail(ctx, user.Email)
	must(t, err)

	if byEmail.ID != user.ID {
		t.Fatal("GetByEmail deveria funcionar sem Redis")
	}

	farm := &operations.Farm{UserID: user.ID, Name: "Boa", City: "c", State: "s"}

	must(t, manage.Farms.Create(ctx, farm))

	list, err := manage.Farms.ListByUser(ctx, user.ID)
	must(t, err)

	if len(list) != 1 {
		t.Fatalf("esperava 1 fazenda, veio %d", len(list))
	}

	farm.Name = "Nova"

	must(t, manage.Farms.Update(ctx, farm))
	must(t, manage.Farms.Delete(ctx, farm.ID))
	must(t, manage.Users.Delete(ctx, user.ID))

	_, err = manage.Users.GetByID(ctx, user.ID)
	wantErr(t, err, operations.ErrNotFound)
}

func TestRedisDownFallsBackToDatabase(t *testing.T) {
	ctx := context.Background()

	// o Redis esta configurado, mas nao conecta
	connection := redis.Connect(ctx, quiet(),
		redis.WithURL("redis://localhost:6379/0"),
		redis.WithRetryDelay(time.Millisecond),
		redis.WithOpener(func(ctx context.Context, url string) (*goredis.Client, error) {
			return nil, errors.New("down")
		}),
	)

	manage := database.NewManage(newRepository(t, ctx), redis.NewManage(connection))

	t.Cleanup(func() { manage.Close() })

	user := &operations.User{Name: "Ana", Email: "ana@example.com", Role: "admin"}

	must(t, manage.Users.Create(ctx, user))

	got, err := manage.Users.GetByID(ctx, user.ID)
	must(t, err)

	if got.Name != "Ana" {
		t.Fatalf("esperava Ana, veio %q", got.Name)
	}

	user.Name = "Ana Maria"

	must(t, manage.Users.Update(ctx, user))
	must(t, manage.Users.Delete(ctx, user.ID))
}

func TestRedisOnlyWhenConfigured(t *testing.T) {
	ctx := context.Background()

	previous, had := os.LookupEnv(redis.EnvName)

	t.Cleanup(func() {
		if had {
			os.Setenv(redis.EnvName, previous)
		} else {
			os.Unsetenv(redis.EnvName)
		}
	})

	os.Unsetenv(redis.EnvName)

	if redis.Configured() {
		t.Fatal("sem a variavel, Configured deveria ser false")
	}

	if connection := redis.ConnectIfConfigured(ctx, quiet()); connection != nil {
		t.Fatal("sem a variavel, nao deveria conectar no Redis")
	}

	// sem Redis, o Manage funciona so com o banco
	manage := database.NewManage(newRepository(t, ctx), redis.NewManage(nil))

	if manage.Cache.Enabled() {
		t.Fatal("conexao nil deveria deixar o cache desligado")
	}

	os.Setenv(redis.EnvName, startRedisServer(t))

	if !redis.Configured() {
		t.Fatal("com a variavel, Configured deveria ser true")
	}

	connection := redis.ConnectIfConfigured(ctx, quiet())

	if connection == nil {
		t.Fatal("com a variavel, deveria conectar no Redis")
	}

	must(t, connection.Wait())
	must(t, connection.Close())
}
