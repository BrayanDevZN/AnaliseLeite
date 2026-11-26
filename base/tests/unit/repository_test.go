package unit

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
)

// newTestManage abre um SQLite novo numa pasta temporaria e devolve o Manage dele.
func newTestManage(t *testing.T, ctx context.Context) *repository.Manage {
	t.Helper()

	url := "sqlite://" + filepath.ToSlash(filepath.Join(t.TempDir(), "test.db"))

	connection := repository.Connect(ctx, backend.NewLoggerWriter(io.Discard), repository.WithURL(url))

	t.Cleanup(func() { connection.Close() })

	if err := connection.Wait(); err != nil {
		t.Fatalf("conectar no banco: %v", err)
	}

	return repository.NewManage(connection)
}

func tableCount(t *testing.T, manage *repository.Manage) int {
	t.Helper()

	var count int

	err := manage.Connection.DB().QueryRow(
		"SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'",
	).Scan(&count)

	if err != nil {
		t.Fatalf("contar tabelas: %v", err)
	}

	return count
}

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

func TestRepository(t *testing.T) {
	ctx := context.Background()
	manage := newTestManage(t, ctx)
	ops := manage.Operations

	t.Run("create tables", func(t *testing.T) {
		must(t, manage.Migrate.CreateTables(ctx))

		if got := tableCount(t, manage); got != 5 {
			t.Fatalf("esperava 5 tabelas, veio %d", got)
		}

		// rodar de novo nao pode dar erro (IF NOT EXISTS)
		must(t, manage.Migrate.CreateTables(ctx))
	})

	var (
		user     = &operations.User{Name: "Ana", Email: "ana@example.com", Role: "admin"}
		farm     = &operations.Farm{Name: "Fazenda Boa", City: "Uberaba", State: "MG"}
		tank     = &operations.Tank{Type: "expansao", CapacityLiters: 2000.5}
		analysis = &operations.Analysis{}
		noTank   = &operations.Analysis{}
		jsonData = &operations.Content{Content: []byte(`{"temperature":4.2}`)}
		fileData = &operations.Content{File: []byte{0x00, 0x01, 0xff}}
	)

	t.Run("users", func(t *testing.T) {
		must(t, ops.Users.Create(ctx, user))

		if user.ID == 0 || user.CreatedAt.IsZero() {
			t.Fatalf("Create deveria preencher ID e CreatedAt: %+v", user)
		}

		duplicate := &operations.User{Name: "Outra", Email: user.Email, Role: "user"}

		if err := ops.Users.Create(ctx, duplicate); err == nil {
			t.Fatal("email repetido deveria dar erro")
		}

		got, err := ops.Users.GetByID(ctx, user.ID)
		must(t, err)

		if got.Name != "Ana" || got.Role != "admin" || got.Password != nil || got.ProfilePhoto != nil {
			t.Fatalf("usuario errado (password e foto deveriam ser nil): %+v", got)
		}

		byEmail, err := ops.Users.GetByEmail(ctx, user.Email)
		must(t, err)

		if byEmail.ID != user.ID {
			t.Fatalf("GetByEmail devolveu o usuario %d, esperava %d", byEmail.ID, user.ID)
		}

		password := "hash"
		user.Name = "Ana Maria"
		user.Password = &password
		user.ProfilePhoto = []byte("foto")

		must(t, ops.Users.Update(ctx, user))

		got, err = ops.Users.GetByID(ctx, user.ID)
		must(t, err)

		if got.Name != "Ana Maria" || got.Password == nil || *got.Password != "hash" || !bytes.Equal(got.ProfilePhoto, []byte("foto")) {
			t.Fatalf("Update nao gravou: %+v", got)
		}

		list, err := ops.Users.List(ctx)
		must(t, err)

		if len(list) != 1 || list[0].ID != user.ID {
			t.Fatalf("List errado: %+v", list)
		}
	})

	t.Run("farms", func(t *testing.T) {
		farm.UserID = user.ID

		must(t, ops.Farms.Create(ctx, farm))

		if farm.ID == 0 || farm.CreatedAt.IsZero() {
			t.Fatalf("Create deveria preencher ID e CreatedAt: %+v", farm)
		}

		got, err := ops.Farms.GetByID(ctx, farm.ID)
		must(t, err)

		if got.Name != "Fazenda Boa" || got.UserID != user.ID || got.City != "Uberaba" || got.State != "MG" {
			t.Fatalf("fazenda errada: %+v", got)
		}

		farm.Name = "Fazenda Nova"

		must(t, ops.Farms.Update(ctx, farm))

		list, err := ops.Farms.ListByUser(ctx, user.ID)
		must(t, err)

		if len(list) != 1 || list[0].Name != "Fazenda Nova" {
			t.Fatalf("ListByUser errado: %+v", list)
		}

		other, err := ops.Farms.ListByUser(ctx, user.ID+100)
		must(t, err)

		if len(other) != 0 {
			t.Fatalf("outro usuario nao deveria ter fazendas: %+v", other)
		}
	})

	t.Run("tanks", func(t *testing.T) {
		tank.FarmID = farm.ID

		must(t, ops.Tanks.Create(ctx, tank))

		if tank.ID == 0 || tank.CreatedAt.IsZero() {
			t.Fatalf("Create deveria preencher ID e CreatedAt: %+v", tank)
		}

		got, err := ops.Tanks.GetByID(ctx, tank.ID)
		must(t, err)

		if got.Type != "expansao" || got.CapacityLiters != 2000.5 || got.FarmID != farm.ID {
			t.Fatalf("tanque errado: %+v", got)
		}

		tank.CapacityLiters = 3000

		must(t, ops.Tanks.Update(ctx, tank))

		list, err := ops.Tanks.ListByFarm(ctx, farm.ID)
		must(t, err)

		if len(list) != 1 || list[0].CapacityLiters != 3000 {
			t.Fatalf("ListByFarm errado: %+v", list)
		}
	})

	t.Run("analyses", func(t *testing.T) {
		analysis.FarmID = farm.ID
		analysis.TankID = &tank.ID
		noTank.FarmID = farm.ID

		must(t, ops.Analyses.Create(ctx, analysis))
		must(t, ops.Analyses.Create(ctx, noTank))

		if analysis.ID == 0 || noTank.ID == 0 || analysis.CreatedAt.IsZero() {
			t.Fatalf("Create deveria preencher ID e CreatedAt: %+v %+v", analysis, noTank)
		}

		got, err := ops.Analyses.GetByID(ctx, analysis.ID)
		must(t, err)

		if got.TankID == nil || *got.TankID != tank.ID {
			t.Fatalf("a analise deveria ter o tanque %d: %+v", tank.ID, got)
		}

		got, err = ops.Analyses.GetByID(ctx, noTank.ID)
		must(t, err)

		if got.TankID != nil {
			t.Fatalf("a analise sem tanque deveria ter TankID nil: %+v", got)
		}

		noTank.TankID = &tank.ID

		must(t, ops.Analyses.Update(ctx, noTank))

		got, err = ops.Analyses.GetByID(ctx, noTank.ID)
		must(t, err)

		if got.TankID == nil {
			t.Fatal("Update deveria ter gravado o tanque")
		}

		list, err := ops.Analyses.ListByFarm(ctx, farm.ID)
		must(t, err)

		// da mais nova para a mais antiga
		if len(list) != 2 || list[0].ID != noTank.ID || list[1].ID != analysis.ID {
			t.Fatalf("ListByFarm errado: %+v", list)
		}
	})

	t.Run("contents", func(t *testing.T) {
		jsonData.AnalysisID = analysis.ID
		fileData.AnalysisID = analysis.ID

		must(t, ops.Contents.Create(ctx, jsonData))
		must(t, ops.Contents.Create(ctx, fileData))

		if jsonData.ID == 0 || fileData.ID == 0 || jsonData.CreatedAt.IsZero() {
			t.Fatalf("Create deveria preencher ID e CreatedAt: %+v %+v", jsonData, fileData)
		}

		// os dois preenchidos ou os dois vazios nao pode
		both := &operations.Content{AnalysisID: analysis.ID, Content: []byte(`{}`), File: []byte("x")}
		neither := &operations.Content{AnalysisID: analysis.ID}

		wantErr(t, ops.Contents.Create(ctx, both), operations.ErrInvalidContent)
		wantErr(t, ops.Contents.Create(ctx, neither), operations.ErrInvalidContent)
		wantErr(t, ops.Contents.Update(ctx, both), operations.ErrInvalidContent)

		got, err := ops.Contents.GetByID(ctx, jsonData.ID)
		must(t, err)

		if string(got.Content) != `{"temperature":4.2}` || got.File != nil {
			t.Fatalf("o conteudo JSON deveria ter so Content: %+v", got)
		}

		got, err = ops.Contents.GetByID(ctx, fileData.ID)
		must(t, err)

		if !bytes.Equal(got.File, []byte{0x00, 0x01, 0xff}) || got.Content != nil {
			t.Fatalf("o conteudo de arquivo deveria ter so File: %+v", got)
		}

		jsonData.Content = []byte(`{"temperature":5}`)

		must(t, ops.Contents.Update(ctx, jsonData))

		got, err = ops.Contents.GetByID(ctx, jsonData.ID)
		must(t, err)

		if string(got.Content) != `{"temperature":5}` {
			t.Fatalf("Update nao gravou: %+v", got)
		}

		list, err := ops.Contents.ListByAnalysis(ctx, analysis.ID)
		must(t, err)

		if len(list) != 2 {
			t.Fatalf("esperava 2 conteudos, veio %d", len(list))
		}
	})

	t.Run("not found", func(t *testing.T) {
		const missing = 9999

		_, err := ops.Users.GetByID(ctx, missing)
		wantErr(t, err, operations.ErrNotFound)

		_, err = ops.Users.GetByEmail(ctx, "ninguem@example.com")
		wantErr(t, err, operations.ErrNotFound)

		_, err = ops.Farms.GetByID(ctx, missing)
		wantErr(t, err, operations.ErrNotFound)

		_, err = ops.Tanks.GetByID(ctx, missing)
		wantErr(t, err, operations.ErrNotFound)

		_, err = ops.Analyses.GetByID(ctx, missing)
		wantErr(t, err, operations.ErrNotFound)

		_, err = ops.Contents.GetByID(ctx, missing)
		wantErr(t, err, operations.ErrNotFound)

		wantErr(t, ops.Users.Update(ctx, &operations.User{ID: missing, Name: "x", Email: "x", Role: "user"}), operations.ErrNotFound)
		wantErr(t, ops.Users.Delete(ctx, missing), operations.ErrNotFound)
		wantErr(t, ops.Farms.Delete(ctx, missing), operations.ErrNotFound)
		wantErr(t, ops.Tanks.Delete(ctx, missing), operations.ErrNotFound)
		wantErr(t, ops.Analyses.Delete(ctx, missing), operations.ErrNotFound)
		wantErr(t, ops.Contents.Delete(ctx, missing), operations.ErrNotFound)
	})

	t.Run("delete", func(t *testing.T) {
		must(t, ops.Contents.Delete(ctx, jsonData.ID))
		must(t, ops.Contents.Delete(ctx, fileData.ID))
		must(t, ops.Analyses.Delete(ctx, analysis.ID))
		must(t, ops.Analyses.Delete(ctx, noTank.ID))
		must(t, ops.Tanks.Delete(ctx, tank.ID))
		must(t, ops.Farms.Delete(ctx, farm.ID))
		must(t, ops.Users.Delete(ctx, user.ID))

		_, err := ops.Users.GetByID(ctx, user.ID)
		wantErr(t, err, operations.ErrNotFound)

		list, err := ops.Users.List(ctx)
		must(t, err)

		if len(list) != 0 {
			t.Fatalf("nao deveria sobrar usuario: %+v", list)
		}
	})

	t.Run("drop tables", func(t *testing.T) {
		must(t, manage.Migrate.DropTables(ctx))

		if got := tableCount(t, manage); got != 0 {
			t.Fatalf("esperava 0 tabelas, veio %d", got)
		}

		// rodar de novo nao pode dar erro (IF EXISTS)
		must(t, manage.Migrate.DropTables(ctx))

		// sem as tabelas, as operacoes tem que falhar
		if _, err := ops.Users.List(ctx); err == nil {
			t.Fatal("List sem a tabela deveria dar erro")
		}
	})
}

func TestRepositoryWithoutConnection(t *testing.T) {
	ctx := context.Background()
	manage := repository.NewManage(nil)

	wantErr(t, manage.Migrate.CreateTables(ctx), repository.ErrNoConnection)
	wantErr(t, manage.Migrate.DropTables(ctx), repository.ErrNoConnection)

	_, err := manage.Operations.Users.List(ctx)
	wantErr(t, err, operations.ErrNoConnection)
}
