package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"

	// driver do SQLite escrito em Go (nao precisa de compilador C)
	_ "modernc.org/sqlite"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
)

const (
	// Retries e quantas vezes tenta de novo depois que a primeira tentativa falha.
	Retries = 3

	// espera antes da primeira nova tentativa; dobra a cada tentativa (500ms, 1s, 2s)
	defaultRetryDelay = 500 * time.Millisecond

	// tempo maximo de cada tentativa de conexao
	attemptTimeout = 5 * time.Second
)

var (
	ErrUnsupportedScheme = errors.New("unsupported database scheme")
	ErrInvalidURL        = errors.New("invalid database url")
)

// no Windows o caminho de uma URL file vem como /C:/pasta/arquivo.db
var windowsDrive = regexp.MustCompile(`^/[A-Za-z]:`)

// Opener abre uma conexao com o banco a partir de uma URL.
type Opener func(ctx context.Context, databaseURL string) (*sql.DB, error)

// Option muda o comportamento de Connect.
type Option func(*Connection)

// WithURL usa essa URL em vez de ler a DATABASE_URL.
func WithURL(databaseURL string) Option {
	return func(c *Connection) {
		c.url = databaseURL
		c.hasURL = true
	}
}

// WithRetryDelay muda a espera antes da primeira nova tentativa (ela dobra a cada tentativa).
func WithRetryDelay(delay time.Duration) Option {
	return func(c *Connection) {
		c.retryDelay = delay
	}
}

// WithOpener troca a funcao que abre a conexao (util em testes ou para outros bancos).
func WithOpener(opener Opener) Option {
	return func(c *Connection) {
		c.opener = opener
	}
}

// Connection e a conexao com o banco de dados, aberta em segundo plano.
type Connection struct {
	logger     *backend.Logger
	url        string
	hasURL     bool
	retryDelay time.Duration
	opener     Opener

	done chan struct{}
	db   *sql.DB
	err  error
}

// Connect comeca a conexao com o banco em segundo plano e devolve na hora o objeto da conexao.
// Se a conexao falhar, tenta de novo ate Retries vezes, esperando mais a cada tentativa.
// Use Wait ou DB para esperar o resultado. Se ctx for cancelado, as tentativas param.
// Se logger for nil, os logs vao para o terminal.
func Connect(ctx context.Context, logger *backend.Logger, options ...Option) *Connection {
	if logger == nil {
		logger = backend.NewLogger()
	}

	connection := &Connection{
		logger:     logger,
		retryDelay: defaultRetryDelay,
		done:       make(chan struct{}),
	}

	connection.opener = connection.open

	for _, option := range options {
		option(connection)
	}

	go connection.run(ctx)

	return connection
}

// Logger devolve o logger da conexao (ou um logger novo se a conexao for nil).
func (c *Connection) Logger() *backend.Logger {
	if c == nil || c.logger == nil {
		return backend.NewLogger()
	}

	return c.logger
}

// Wait espera a conexao terminar (com sucesso ou nao) e devolve o erro, se houve.
func (c *Connection) Wait() error {
	<-c.done

	return c.err
}

// DB espera a conexao terminar e devolve o *sql.DB, ou nil se a conexao falhou.
func (c *Connection) DB() *sql.DB {
	<-c.done

	return c.db
}

// Done devolve um canal que fecha quando a conexao termina (com sucesso ou nao).
func (c *Connection) Done() <-chan struct{} {
	return c.done
}

// Close espera a conexao terminar e fecha o banco. Nao faz nada se a conexao falhou.
func (c *Connection) Close() error {
	<-c.done

	if c.db == nil {
		return nil
	}

	c.logger.Info("Closing database connection...")

	if err := c.db.Close(); err != nil {
		c.logger.Error("Could not close database connection: %v", err)

		return fmt.Errorf("close database: %w", err)
	}

	c.logger.Success("Database connection closed with success")

	return nil
}

// run faz as tentativas de conexao e guarda o resultado.
func (c *Connection) run(ctx context.Context) {
	defer close(c.done)

	if !c.hasURL {
		c.url = URL(c.logger)
	}

	attempts := Retries + 1
	delay := c.retryDelay

	for attempt := 1; attempt <= attempts; attempt++ {
		c.logger.Info("Connecting to database (attempt %d/%d)...", attempt, attempts)

		db, err := c.opener(ctx, c.url)

		if err == nil {
			c.db = db

			c.logger.Success("Connected to database with success")

			return
		}

		// uma URL ou um banco que nao existe no sistema nao melhora tentando de novo
		if errors.Is(err, ErrUnsupportedScheme) || errors.Is(err, ErrInvalidURL) {
			c.logger.Error("Could not connect to database: %v", err)

			c.err = err

			return
		}

		if attempt == attempts {
			c.logger.Error("Could not connect to database after %d attempts: %v", attempts, err)

			c.err = fmt.Errorf("connect to database after %d attempts: %w", attempts, err)

			return
		}

		c.logger.Warning("Attempt %d/%d failed: %v. Trying again in %s...", attempt, attempts, err, delay)

		select {
		case <-ctx.Done():
			c.logger.Warning("Connection canceled")

			c.err = ctx.Err()

			return

		case <-time.After(delay):
		}

		delay *= 2
	}
}

// open abre e testa uma conexao, conforme o tipo de banco da URL.
func (c *Connection) open(ctx context.Context, databaseURL string) (*sql.DB, error) {
	parsed, err := url.Parse(databaseURL)

	// o erro do url.Parse repete a URL inteira, que pode ter senha, entao ele nao e repassado
	if err != nil || parsed.Scheme == "" {
		return nil, ErrInvalidURL
	}

	switch parsed.Scheme {
	case "sqlite":
		return c.openSQLite(ctx, parsed)

	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedScheme, parsed.Scheme)
	}
}

// openSQLite abre um banco SQLite, criando a pasta do arquivo se ela nao existir.
func (c *Connection) openSQLite(ctx context.Context, parsed *url.URL) (*sql.DB, error) {
	path := windowsDrive.ReplaceAllStringFunc(parsed.Path, func(match string) string { return match[1:] })

	if path == "" || path == "/" {
		return nil, ErrInvalidURL
	}

	path = filepath.FromSlash(path)
	dir := filepath.Dir(path)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		c.logger.Info("Creating database directory %s...", dir)

		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}

	dsn := "file:" + filepath.ToSlash(path)

	if parsed.RawQuery != "" {
		dsn += "?" + parsed.RawQuery
	}

	c.logger.Info("Opening SQLite database at %s...", path)

	db, err := sql.Open("sqlite", dsn)

	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// o sql.Open so prepara o objeto; o Ping e que confirma que o banco responde
	pingContext, cancel := context.WithTimeout(ctx, attemptTimeout)

	defer cancel()

	if err := db.PingContext(pingContext); err != nil {
		db.Close()

		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	return db, nil
}
