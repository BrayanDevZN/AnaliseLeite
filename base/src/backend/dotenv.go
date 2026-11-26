package backend

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// DotEnvFile e o nome do arquivo de variaveis, que fica na raiz de base/ (a pasta do go.mod).
const DotEnvFile = ".env"

// dotEnvOverride troca o arquivo que o Config le (usado em testes). Sem ele, o arquivo e o <raiz de base/>/.env.
var dotEnvOverride struct {
	sync.RWMutex
	set  bool
	path string
}

// SetDotEnvPath faz o Config ler esse arquivo no lugar do .env da raiz de base/. Um caminho vazio
// desliga a leitura do arquivo (so as variaveis de ambiente valem). Devolve a funcao que desfaz a troca.
func SetDotEnvPath(path string) (restore func()) {
	dotEnvOverride.Lock()
	defer dotEnvOverride.Unlock()

	previousSet, previousPath := dotEnvOverride.set, dotEnvOverride.path

	dotEnvOverride.set, dotEnvOverride.path = true, path

	return func() {
		dotEnvOverride.Lock()
		defer dotEnvOverride.Unlock()

		dotEnvOverride.set, dotEnvOverride.path = previousSet, previousPath
	}
}

// ModuleRoot devolve a raiz de base/ (a pasta do go.mod) e se achou.
// Igual ao Path(__file__) do config.py do pipeline, o primeiro lugar que olha e o do proprio codigo-fonte,
// entao nao depende da pasta de onde o comando foi rodado. Se o fonte nao esta mais la (binario
// copiado para outra maquina), sobe as pastas a partir da pasta atual ate achar o go.mod.
func ModuleRoot() (string, bool) {
	if _, file, _, ok := runtime.Caller(0); ok {
		root := filepath.Join(filepath.Dir(file), "..", "..")

		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			return filepath.Clean(root), true
		}
	}

	dir, err := os.Getwd()

	if err != nil {
		return "", false
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, true
		}

		parent := filepath.Dir(dir)

		if parent == dir {
			return "", false
		}

		dir = parent
	}
}

// dotEnvPath devolve o arquivo a ler, ou "" se nao ha arquivo para ler.
func dotEnvPath() string {
	dotEnvOverride.RLock()
	defer dotEnvOverride.RUnlock()

	if dotEnvOverride.set {
		return dotEnvOverride.path
	}

	root, found := ModuleRoot()

	if !found {
		return ""
	}

	return filepath.Join(root, DotEnvFile)
}

// loadDotEnv carrega o .env da raiz de base/ no ambiente do processo, como o load_dotenv do config.py:
// so poe as variaveis que ainda nao existem no ambiente, entao o que veio de fora nunca e atropelado
// (uma variavel que existe no ambiente, mesmo vazia, fica como esta). Se o arquivo nao existir, nao e erro
// e so as variaveis de ambiente valem. Se existir e estiver quebrado, devolve o erro (sem os valores).
func loadDotEnv() error {
	path := dotEnvPath()

	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)

	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("read %s: %w", DotEnvFile, err)
	}

	values, err := parseDotEnv(data)

	if err != nil {
		return err
	}

	for name, value := range values {
		if _, exists := os.LookupEnv(name); exists {
			continue
		}

		if err := os.Setenv(name, value); err != nil {
			return fmt.Errorf("load %s: %w", DotEnvFile, err)
		}
	}

	return nil
}

// parseDotEnv le o formato NOME=valor, uma por linha. Aceita linhas em branco, comentarios (#),
// "export NOME=valor", valores entre aspas simples (literais) ou duplas (com \n, \t, \" e \\),
// e comentario no fim da linha (` # ...`). Se um nome se repete, vale o ultimo.
// Os erros dizem so o numero da linha, nunca o conteudo, porque ele pode ser uma senha.
func parseDotEnv(data []byte) (map[string]string, error) {
	text := strings.TrimPrefix(string(data), "\xef\xbb\xbf")
	values := make(map[string]string)

	for number, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if rest, found := strings.CutPrefix(line, "export "); found {
			line = strings.TrimSpace(rest)
		}

		name, raw, found := strings.Cut(line, "=")
		name = strings.TrimSpace(name)

		if !found || !validName(name) {
			return nil, fmt.Errorf("%s line %d is not NAME=value", DotEnvFile, number+1)
		}

		value, err := parseValue(strings.TrimSpace(raw))

		if err != nil {
			return nil, fmt.Errorf("%s line %d: %w", DotEnvFile, number+1, err)
		}

		values[name] = value
	}

	return values, nil
}

func validName(name string) bool {
	if name == "" {
		return false
	}

	for i, char := range name {
		letter := char == '_' || (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')

		if !letter && (i == 0 || char < '0' || char > '9') {
			return false
		}
	}

	return true
}

// parseValue le o valor de uma linha: com aspas, sem aspas, ou vazio.
func parseValue(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}

	quote := raw[0]

	if quote != '"' && quote != '\'' {
		value, _, _ := strings.Cut(raw, " #")

		return strings.TrimSpace(value), nil
	}

	var value strings.Builder

	for i := 1; i < len(raw); i++ {
		char := raw[i]

		switch {
		case quote == '"' && char == '\\' && i+1 < len(raw):
			i++

			switch raw[i] {
			case 'n':
				value.WriteByte('\n')
			case 't':
				value.WriteByte('\t')
			default:
				value.WriteByte(raw[i])
			}

		case char == quote:
			rest := strings.TrimSpace(raw[i+1:])

			if rest != "" && !strings.HasPrefix(rest, "#") {
				return "", errors.New("unexpected text after the closing quote")
			}

			return value.String(), nil

		default:
			value.WriteByte(char)
		}
	}

	return "", errors.New("quote is not closed")
}
