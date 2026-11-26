// Comando principal do backend: le a configuracao do ambiente, conecta no banco, no Redis e no
// pipeline, monta as rotas com o rate limit e sobe o servidor HTTP. Para rodar: go run .
//
// As variaveis podem vir do ambiente ou de um arquivo .env na raiz de base/ (a pasta do go.mod).
// Se o arquivo nao existir, so o ambiente vale; se a variavel existir nos dois, vale a do ambiente.
//
// Variaveis:
//
//	JWT_SECRET        obrigatoria: chave que assina os tokens (minimo de 32 caracteres)
//	DATABASE_URL      URL do banco (padrao: SQLite em data/data.db)
//	REDIS_URL         URL do Redis; sem ela nao ha cache e o rate limit conta na memoria
//	MODEL_URL         URL do pipeline; junto com ML_KEY liga as rotas do modelo
//	ML_KEY            chave de API do pipeline
//	PORT              porta do servidor (padrao 8080)
//	COOKIE_INSECURE   "true" tira o Secure dos cookies (so para desenvolvimento local em http)
//	COOKIE_DOMAIN     dominio dos cookies (padrao: o do proprio backend)
//	TRUST_PROXY       "true" confia no X-Forwarded-For (so atras de um proxy confiavel)
//	CORS_ORIGINS      origens do frontend que podem chamar a API com cookies, separadas por virgula
//	                  (ex.: http://localhost:5173). Sem ela, o CORS fica desligado
//	ADMIN_EMAIL, ADMIN_PASSWORD, ADMIN_NAME
//	                  se definidas, criam esse admin na primeira vez que o backend sobe
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/auth"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/redis"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/repository/operations"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/handles"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/midlleware"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/schema"
)

const (
	defaultPort = "8080"

	accessTTL  = 15 * time.Minute
	refreshTTL = 7 * 24 * time.Hour

	shutdownTimeout = 30 * time.Second
)

// settings sao as variaveis de ambiente opcionais.
type settings struct {
	port           string
	cookieInsecure bool
	cookieDomain   string
	trustProxy     bool
	corsOrigins    []string
	adminEmail     string
	adminPassword  string
	adminName      string
}

func main() {
	logger := backend.NewLogger()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	err := run(ctx, logger)

	stop()

	if err != nil {
		logger.Error("Backend stopped: %v", err)

		os.Exit(1)
	}
}

func run(ctx context.Context, logger *backend.Logger) error {
	config, err := readSettings()

	if err != nil {
		return err
	}

	// banco de dados (a URL vem da DATABASE_URL) e as tabelas
	connection := repository.Connect(ctx, logger)

	if err := connection.Wait(); err != nil {
		return fmt.Errorf("database: %w", err)
	}

	repositoryManage := repository.NewManage(connection)

	// Redis e opcional: sem a REDIS_URL, nao ha cache e o rate limit conta na memoria
	var redisManage *redis.Manage

	if redisConnection := redis.ConnectIfConfigured(ctx, logger); redisConnection != nil {
		redisManage = redis.NewManage(redisConnection)
	}

	db := database.NewManage(repositoryManage, redisManage)

	defer db.Close()

	if err := repositoryManage.Migrate.CreateTables(ctx); err != nil {
		return err
	}

	hasher := auth.NewHasher(logger)

	access, err := auth.NewJWT(logger, accessTTL)

	if err != nil {
		return fmt.Errorf("access token: %w", err)
	}

	refresh, err := auth.NewJWT(logger, refreshTTL)

	if err != nil {
		return fmt.Errorf("refresh token: %w", err)
	}

	if err := createAdmin(ctx, logger, db, hasher, config); err != nil {
		return err
	}

	// o pipeline e opcional: sem MODEL_URL e ML_KEY, as rotas do modelo respondem 503
	requests, err := newRequests(logger)

	if err != nil {
		return err
	}

	routes, err := handles.New(logger, db, requests, hasher, access, refresh, handles.Config{
		InsecureCookies: config.cookieInsecure,
		CookieDomain:    config.cookieDomain,
	})

	if err != nil {
		return err
	}

	handler, err := routes.Handler()

	if err != nil {
		return err
	}

	limiter, err := midlleware.New(logger, redisManage, midlleware.Config{
		User:       midlleware.Rule{Limit: 120, Window: time.Minute},
		Public:     midlleware.Rule{Limit: 30, Window: time.Minute},
		Global:     midlleware.Rule{Limit: 3000, Window: time.Minute},
		Identify:   routes.Depends().Identify,
		TrustProxy: config.trustProxy,
	})

	if err != nil {
		return err
	}

	// o CORS fica por fora do rate limit, para ate a resposta 429 chegar ao frontend com os headers certos
	var final http.Handler = limiter.Handler(handler)

	if cors := midlleware.NewCORS(config.corsOrigins); cors != nil {
		logger.Success("CORS enabled for %d origin(s)", len(config.corsOrigins))

		final = cors.Handler(final)
	}

	return serve(ctx, logger, config.port, final)
}

// serve sobe o servidor e, quando o ctx e cancelado (Ctrl+C ou SIGTERM), termina as requisicoes em andamento.
func serve(ctx context.Context, logger *backend.Logger, port string, handler http.Handler) error {
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,  // o upload de arquivo pode ser grande
		WriteTimeout:      120 * time.Second, // a resposta pode esperar o modelo
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}

	failed := make(chan error, 1)

	go func() {
		logger.Success("Listening on port %s", port)

		failed <- server.ListenAndServe()
	}()

	select {
	case err := <-failed:
		return fmt.Errorf("server: %w", err)

	case <-ctx.Done():
	}

	logger.Info("Shutting down...")

	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)

	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	logger.Success("Server stopped with success")

	return nil
}

// newRequests cria o cliente do pipeline. Se MODEL_URL ou ML_KEY faltarem, devolve nil (sem erro).
func newRequests(logger *backend.Logger) (*backend.Requests, error) {
	requests, err := backend.NewRequests(logger)

	var missing *backend.MissingSecretsError

	switch {
	case errors.As(err, &missing):
		logger.Warning("The model routes are disabled: %v", err)

		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("model service: %w", err)
	}

	return requests, nil
}

// createAdmin cria o admin das variaveis ADMIN_EMAIL e ADMIN_PASSWORD, se ainda nao existir.
// Se a conta ja existe, nao mexe nela (nem na senha).
func createAdmin(ctx context.Context, logger *backend.Logger, db *database.Manage, hasher *auth.Hasher, config settings) error {
	if config.adminEmail == "" && config.adminPassword == "" {
		return nil
	}

	// O admin da inicializacao vem das variaveis de ambiente (quem opera o servidor), nao de um usuario: por isso
	// fica fora das regras de cadastro (email @gmail.com e senha forte). So confere o basico.
	email := strings.ToLower(strings.TrimSpace(config.adminEmail))
	name := strings.TrimSpace(config.adminName)

	if !strings.Contains(email, "@") || len(config.adminPassword) < 8 || len(config.adminPassword) > 72 {
		return errors.New("ADMIN_EMAIL and ADMIN_PASSWORD: need a valid email and a password of 8 to 72 characters")
	}

	hashed, err := hasher.Hash(config.adminPassword)

	if err != nil {
		return fmt.Errorf("ADMIN_PASSWORD: %w", err)
	}

	admin := &operations.User{Name: name, Email: email, Password: &hashed, Role: schema.RoleAdmin}

	if _, err := db.Users.GetByEmail(ctx, admin.Email); err == nil {
		logger.Info("Admin already exists")

		return nil
	} else if !errors.Is(err, operations.ErrNotFound) {
		return err
	}

	if err := db.Users.Create(ctx, admin); err != nil {
		return fmt.Errorf("create admin: %w", err)
	}

	logger.Success("Admin created with success")

	return nil
}

// readSettings le as variaveis de ambiente opcionais.
func readSettings() (settings, error) {
	names := []string{"PORT", "COOKIE_INSECURE", "COOKIE_DOMAIN", "TRUST_PROXY", "CORS_ORIGINS", "ADMIN_EMAIL", "ADMIN_PASSWORD", "ADMIN_NAME"}

	config, err := backend.NewConfig(names, nil)

	if err != nil {
		return settings{}, err
	}

	values := config.GetSecrets()

	text := func(name, fallback string) string {
		if value := values[name]; value != nil {
			return *value
		}

		return fallback
	}

	flag := func(name string) (bool, error) {
		value := values[name]

		if value == nil {
			return false, nil
		}

		parsed, err := strconv.ParseBool(*value)

		if err != nil {
			return false, fmt.Errorf("%s must be true or false", name)
		}

		return parsed, nil
	}

	result := settings{
		port:          text("PORT", defaultPort),
		cookieDomain:  text("COOKIE_DOMAIN", ""),
		adminEmail:    text("ADMIN_EMAIL", ""),
		adminPassword: text("ADMIN_PASSWORD", ""),
		adminName:     text("ADMIN_NAME", "Admin"),
	}

	if port, err := strconv.Atoi(result.port); err != nil || port < 1 || port > 65535 {
		return settings{}, errors.New("PORT must be a number between 1 and 65535")
	}

	if result.cookieInsecure, err = flag("COOKIE_INSECURE"); err != nil {
		return settings{}, err
	}

	if result.trustProxy, err = flag("TRUST_PROXY"); err != nil {
		return settings{}, err
	}

	if result.corsOrigins, err = midlleware.ParseOrigins(text("CORS_ORIGINS", "")); err != nil {
		return settings{}, fmt.Errorf("CORS_ORIGINS: %w", err)
	}

	return result, nil
}
