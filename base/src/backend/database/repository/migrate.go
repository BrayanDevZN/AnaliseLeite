package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
)

// ErrNoConnection quando nao ha conexao com o banco para usar (e o mesmo erro do operations).
var ErrNoConnection = operations.ErrNoConnection

// tables sao as tabelas na ordem de criacao (quem tem FK vem depois da tabela que ela aponta).
// Para apagar, a ordem e a inversa.
var tables = []struct {
	name   string
	create string
}{
	{"users", `CREATE TABLE IF NOT EXISTS users (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		name          TEXT NOT NULL,
		email         TEXT NOT NULL UNIQUE,
		password      TEXT,
		role          TEXT NOT NULL DEFAULT 'user',
		profile_photo BLOB,
		created_at    DATETIME NOT NULL
	)`},
	{"farms", `CREATE TABLE IF NOT EXISTS farms (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
		name       TEXT NOT NULL,
		city       TEXT NOT NULL,
		state      TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		photo      BLOB
	)`},
	{"tanks", `CREATE TABLE IF NOT EXISTS tanks (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		farm_id         INTEGER NOT NULL REFERENCES farms (id) ON DELETE CASCADE,
		type            TEXT NOT NULL,
		capacity_liters REAL NOT NULL,
		created_at      DATETIME NOT NULL
	)`},
	{"analyses", `CREATE TABLE IF NOT EXISTS analyses (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		farm_id    INTEGER NOT NULL REFERENCES farms (id) ON DELETE CASCADE,
		tank_id    INTEGER REFERENCES tanks (id) ON DELETE SET NULL,
		created_at DATETIME NOT NULL
	)`},
	{"contents", `CREATE TABLE IF NOT EXISTS contents (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		analysis_id INTEGER NOT NULL REFERENCES analyses (id) ON DELETE CASCADE,
		content     TEXT,
		file        BLOB,
		created_at  DATETIME NOT NULL,
		CHECK ((content IS NULL) <> (file IS NULL))
	)`},
}

// addedColumns sao as colunas que foram acrescentadas depois da primeira versao das tabelas.
var addedColumns = []string{
	"ALTER TABLE farms ADD COLUMN photo BLOB",
}

// Migrate cria e apaga as tabelas do banco.
type Migrate struct {
	connection *Connection
}

// NewMigrate cria o Migrate usando a conexao.
func NewMigrate(connection *Connection) *Migrate {
	return &Migrate{connection: connection}
}

// CreateTables cria as tabelas que ainda nao existem, tudo numa transacao.
func (m *Migrate) CreateTables(ctx context.Context) error {
	return m.run(ctx, "Creating tables", "Tables created with success", "create", func(exec execer) error {
		for _, table := range tables {
			m.connection.logger.Info("Creating table %s...", table.name)

			if err := exec(table.create); err != nil {
				return fmt.Errorf("table %s: %w", table.name, err)
			}
		}

		// bancos criados antes de uma coluna existir nao a ganham pelo CREATE TABLE IF NOT EXISTS: cada coluna
		// nova entra aqui (o SQLite nao tem "ADD COLUMN IF NOT EXISTS", entao o erro de coluna repetida e esperado)
		for _, column := range addedColumns {
			if err := exec(column); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
				return fmt.Errorf("add column: %w", err)
			}
		}

		return nil
	})
}

// DropTables apaga todas as tabelas (e os dados delas), tudo numa transacao.
func (m *Migrate) DropTables(ctx context.Context) error {
	return m.run(ctx, "Dropping tables", "Tables dropped with success", "drop", func(exec execer) error {
		for i := len(tables) - 1; i >= 0; i-- {
			m.connection.logger.Info("Dropping table %s...", tables[i].name)

			if err := exec("DROP TABLE IF EXISTS " + tables[i].name); err != nil {
				return fmt.Errorf("table %s: %w", tables[i].name, err)
			}
		}

		return nil
	})
}

// execer roda um comando SQL dentro da transacao.
type execer func(query string) error

// run espera a conexao, abre a transacao, roda os comandos e confirma (ou desfaz se algo falhar).
func (m *Migrate) run(ctx context.Context, start, success, action string, steps func(execer) error) error {
	if m.connection == nil {
		return ErrNoConnection
	}

	db := m.connection.DB()

	if db == nil {
		if err := m.connection.Wait(); err != nil {
			return fmt.Errorf("%w: %w", ErrNoConnection, err)
		}

		return ErrNoConnection
	}

	logger := m.connection.logger

	logger.Info("%s...", start)

	tx, err := db.BeginTx(ctx, nil)

	if err != nil {
		logger.Error("Could not %s tables: %v", action, err)

		return fmt.Errorf("%s tables: %w", action, err)
	}

	err = steps(func(query string) error {
		_, err := tx.ExecContext(ctx, query)

		return err
	})

	if err == nil {
		err = tx.Commit()
	}

	if err != nil {
		tx.Rollback()

		logger.Error("Could not %s tables: %v", action, err)

		return fmt.Errorf("%s tables: %w", action, err)
	}

	logger.Success("%s", success)

	return nil
}
