// Package operations tem as operacoes de cada tabela do banco (users, farms, tanks, analyses e contents).
package operations

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
)

var (
	// ErrNotFound quando o registro nao existe.
	ErrNotFound = errors.New("record not found")

	// ErrNoConnection quando nao ha conexao com o banco para usar.
	ErrNoConnection = errors.New("no database connection")
)

// Connection e o que as operacoes precisam da conexao com o banco.
// O *repository.Connection ja tem esses metodos. Foi feito como interface para o
// repository poder importar este pacote (sem import ciclico) e juntar tudo no Manage.
type Connection interface {
	DB() *sql.DB
	Wait() error
	Logger() *backend.Logger
}

// scanner e o que *sql.Row e *sql.Rows tem em comum, para o mesmo scan servir nos dois.
type scanner interface {
	Scan(dest ...any) error
}

// begin loga o comeco da operacao e devolve a funcao que loga o fim.
// Use assim, com o erro de retorno nomeado: defer begin(connection, "create user")(&err)
// Nunca ponha dados pessoais (email, senha) na descricao, so ids.
func begin(connection Connection, format string, args ...any) func(*error) {
	logger := backend.NewLogger()

	if connection != nil {
		logger = connection.Logger()
	}

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

// database espera a conexao terminar e devolve o *sql.DB dela.
func database(connection Connection) (*sql.DB, error) {
	if connection == nil {
		return nil, ErrNoConnection
	}

	db := connection.DB()

	if db == nil {
		if err := connection.Wait(); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrNoConnection, err)
		}

		return nil, ErrNoConnection
	}

	return db, nil
}

// affected devolve ErrNotFound se o Update ou Delete nao mexeu em nenhuma linha.
func affected(result sql.Result) error {
	count, err := result.RowsAffected()

	if err != nil {
		return err
	}

	if count == 0 {
		return ErrNotFound
	}

	return nil
}

// notFound troca o sql.ErrNoRows pelo ErrNotFound.
func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}

	return err
}
