// Package redis tem a conexao com o Redis, as operacoes dele e o Manage que junta as duas.
package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
)

const (
	// EnvName e o nome da variavel de ambiente com a URL do Redis.
	EnvName = "REDIS_URL"

	// defaultURL e usada quando a REDIS_URL nao existe.
	defaultURL = "redis://localhost:6379/0"

	// Retries e quantas vezes tenta de novo depois que a primeira tentativa falha.
	Retries = 3

	// espera antes da primeira nova tentativa; dobra a cada tentativa (500ms, 1s, 2s)
	defaultRetryDelay = 500 * time.Millisecond

	// tempo maximo de cada tentativa de conexao
	attemptTimeout = 5 * time.Second
)

var (
	ErrInvalidURL   = errors.New("invalid redis url")
	ErrNoConnection = errors.New("no redis connection")
)

// Opener abre uma conexao com o Redis a partir de uma URL.
type Opener func(ctx context.Context, redisURL string) (*goredis.Client, error)

// Option muda o comportamento de Connect.
type Option func(*Connection)

// WithURL usa essa URL em vez de ler a REDIS_URL.
func WithURL(redisURL string) Option {
	return func(c *Connection) {
		c.url = redisURL
		c.hasURL = true
	}
}

// WithRetryDelay muda a espera antes da primeira nova tentativa (ela dobra a cada tentativa).
func WithRetryDelay(delay time.Duration) Option {
	return func(c *Connection) {
		c.retryDelay = delay
	}
}

// WithOpener troca a funcao que abre a conexao (util em testes).
func WithOpener(opener Opener) Option {
	return func(c *Connection) {
		c.opener = opener
	}
}

// Connection e a conexao com o Redis, aberta em segundo plano.
type Connection struct {
	logger     *backend.Logger
	url        string
	hasURL     bool
	retryDelay time.Duration
	opener     Opener

	done   chan struct{}
	client *goredis.Client
	err    error
}

// Connect comeca a conexao com o Redis em segundo plano e devolve na hora o objeto da conexao.
// Se a conexao falhar, tenta de novo ate Retries vezes, esperando mais a cada tentativa.
// Use Wait ou Client para esperar o resultado. Se ctx for cancelado, as tentativas param.
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

// Client espera a conexao terminar e devolve o cliente, ou nil se a conexao falhou.
// O cliente pode ser usado por varias goroutines ao mesmo tempo.
func (c *Connection) Client() *goredis.Client {
	<-c.done

	return c.client
}

// Done devolve um canal que fecha quando a conexao termina (com sucesso ou nao).
func (c *Connection) Done() <-chan struct{} {
	return c.done
}

// Close espera a conexao terminar e fecha o Redis. Nao faz nada se a conexao falhou.
func (c *Connection) Close() error {
	<-c.done

	if c.client == nil {
		return nil
	}

	c.logger.Info("Closing redis connection...")

	if err := c.client.Close(); err != nil {
		c.logger.Error("Could not close redis connection: %v", err)

		return fmt.Errorf("close redis: %w", err)
	}

	c.logger.Success("Redis connection closed with success")

	return nil
}

// URL devolve a URL do Redis.
// Le a REDIS_URL (como secret nao obrigatoria do Config). Se ela nao existir, usa redis://localhost:6379/0.
// O valor de REDIS_URL nunca vai para os logs, porque pode ter usuario e senha.
func URL(logger *backend.Logger) string {
	if logger == nil {
		logger = backend.NewLogger()
	}

	logger.Info("Reading %s...", EnvName)

	config, err := backend.NewConfig([]string{EnvName}, nil)

	if err != nil {
		logger.Error("%v", err)
	} else if value := config.GetSecrets()[EnvName]; value != nil {
		logger.Success("%s loaded with success", EnvName)

		return *value
	}

	logger.Warning("%s is not set, using %s", EnvName, defaultURL)

	return defaultURL
}

// Configured diz se a REDIS_URL existe no ambiente.
func Configured() bool {
	config, err := backend.NewConfig([]string{EnvName}, nil)

	return err == nil && config.GetSecrets()[EnvName] != nil
}

// ConnectIfConfigured faz o Connect so se a REDIS_URL existir. Se ela nao existir,
// devolve nil (sem Redis, quem usa o cache fica so com o banco de dados).
func ConnectIfConfigured(ctx context.Context, logger *backend.Logger, options ...Option) *Connection {
	if logger == nil {
		logger = backend.NewLogger()
	}

	if !Configured() {
		logger.Warning("%s is not set, redis is disabled", EnvName)

		return nil
	}

	return Connect(ctx, logger, options...)
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
		c.logger.Info("Connecting to redis (attempt %d/%d)...", attempt, attempts)

		client, err := c.opener(ctx, c.url)

		if err == nil {
			c.client = client

			c.logger.Success("Connected to redis with success")

			return
		}

		// uma URL invalida nao melhora tentando de novo
		if errors.Is(err, ErrInvalidURL) {
			c.logger.Error("Could not connect to redis: %v", err)

			c.err = err

			return
		}

		if attempt == attempts {
			c.logger.Error("Could not connect to redis after %d attempts: %v", attempts, err)

			c.err = fmt.Errorf("connect to redis after %d attempts: %w", attempts, err)

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

// open abre e testa uma conexao com o Redis.
func (c *Connection) open(ctx context.Context, redisURL string) (*goredis.Client, error) {
	options, err := goredis.ParseURL(redisURL)

	// o erro do ParseURL pode repetir a URL inteira, que pode ter senha, entao ele nao e repassado
	if err != nil {
		return nil, ErrInvalidURL
	}

	c.logger.Info("Opening redis at %s...", options.Addr)

	client := goredis.NewClient(options)

	// o NewClient so prepara o objeto; o Ping e que confirma que o Redis responde
	pingContext, cancel := context.WithTimeout(ctx, attemptTimeout)

	defer cancel()

	if err := client.Ping(pingContext).Err(); err != nil {
		client.Close()

		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}
