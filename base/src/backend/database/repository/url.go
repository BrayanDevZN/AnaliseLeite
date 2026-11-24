package repository

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
)

// EnvName e o nome da variavel de ambiente com a URL do banco de dados.
const EnvName = "DATABASE_URL"

// URL devolve a URL do banco de dados.
// Le a DATABASE_URL (como secret nao obrigatoria do Config). Se ela nao existir,
// devolve a URL de um banco SQLite em <base>/data/data.db, onde <base> e a pasta do go.mod.
// Se logger for nil, os logs vao para o terminal.
// O valor de DATABASE_URL nunca vai para os logs, porque pode ter usuario e senha.
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

	path := filepath.Join(baseDir(), "data", "data.db")

	logger.Warning("%s is not set, using SQLite at %s", EnvName, path)

	return sqliteURL(path)
}

// baseDir sobe as pastas a partir da pasta atual ate achar o go.mod (a raiz de base/).
// Se nao achar, usa a pasta atual.
func baseDir() string {
	start, err := os.Getwd()

	if err != nil {
		return "."
	}

	for dir := start; ; {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)

		if parent == dir {
			return start
		}

		dir = parent
	}
}

// sqliteURL monta a URL sqlite:///caminho/do/arquivo.db a partir de um caminho de arquivo.
func sqliteURL(path string) string {
	urlPath := filepath.ToSlash(path)

	// no Windows o caminho comeca com a letra do disco (C:/...)
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}

	return (&url.URL{Scheme: "sqlite", Path: urlPath}).String()
}
