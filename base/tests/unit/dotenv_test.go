package unit

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
)

// useDotEnv escreve um .env de teste e faz o Config ler ele.
// O Config poe as variaveis do arquivo no ambiente do processo; no fim do teste, as que nasceram
// por causa dele sao tiradas, para nao vazar para os outros testes.
func useDotEnv(t *testing.T, content string) {
	t.Helper()

	before := make(map[string]bool)

	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")

		before[name] = true
	}

	t.Cleanup(func() {
		for _, entry := range os.Environ() {
			if name, _, _ := strings.Cut(entry, "="); !before[name] {
				os.Unsetenv(name)
			}
		}
	})

	path := filepath.Join(t.TempDir(), ".env")

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(backend.SetDotEnvPath(path))
}

func secrets(t *testing.T, names ...string) map[string]*string {
	t.Helper()

	config, err := backend.NewConfig(names, nil)

	if err != nil {
		t.Fatal(err)
	}

	return config.GetSecrets()
}

func TestDotEnvLoadsVariables(t *testing.T) {
	useDotEnv(t, "DOTENV_A=um\nDOTENV_B=dois\n")

	values := secrets(t, "DOTENV_A", "DOTENV_B", "DOTENV_C")

	if values["DOTENV_A"] == nil || *values["DOTENV_A"] != "um" || *values["DOTENV_B"] != "dois" {
		t.Fatalf("valores errados: %v", values)
	}

	if values["DOTENV_C"] != nil {
		t.Fatal("variavel que nao esta em lugar nenhum deveria ser nil")
	}
}

func TestDotEnvSatisfiesRequiredSecrets(t *testing.T) {
	useDotEnv(t, "DOTENV_REQUIRED=ok\n")

	if _, err := backend.NewConfig(nil, []string{"DOTENV_REQUIRED"}); err != nil {
		t.Fatalf("a variavel esta no .env, nao deveria faltar: %v", err)
	}

	_, err := backend.NewConfig(nil, []string{"DOTENV_REQUIRED", "DOTENV_MISSING"})

	var missing *backend.MissingSecretsError

	if !errors.As(err, &missing) || len(missing.Names) != 1 || missing.Names[0] != "DOTENV_MISSING" {
		t.Fatalf("esperava faltar so DOTENV_MISSING, veio %v", err)
	}
}

func TestEnvironmentWinsOverDotEnv(t *testing.T) {
	useDotEnv(t, "DOTENV_WIN=do-arquivo\nDOTENV_EMPTY=do-arquivo\n")

	t.Setenv("DOTENV_WIN", "do-ambiente")
	t.Setenv("DOTENV_EMPTY", "")

	values := secrets(t, "DOTENV_WIN", "DOTENV_EMPTY")

	if values["DOTENV_WIN"] == nil || *values["DOTENV_WIN"] != "do-ambiente" {
		t.Fatalf("o ambiente deveria ganhar do .env: %v", values["DOTENV_WIN"])
	}

	// existir no ambiente, mesmo vazia, vale mais que o .env
	if values["DOTENV_EMPTY"] != nil {
		t.Fatal("variavel vazia no ambiente deveria continuar nil, sem cair no .env")
	}
}

func TestMissingDotEnvFileIsNotAnError(t *testing.T) {
	t.Cleanup(backend.SetDotEnvPath(filepath.Join(t.TempDir(), "nao-existe", ".env")))
	t.Setenv("DOTENV_ONLY_ENV", "valor")

	values := secrets(t, "DOTENV_ONLY_ENV", "DOTENV_OTHER")

	if values["DOTENV_ONLY_ENV"] == nil || *values["DOTENV_ONLY_ENV"] != "valor" || values["DOTENV_OTHER"] != nil {
		t.Fatalf("sem .env, so o ambiente deveria valer: %v", values)
	}
}

func TestDotEnvFormats(t *testing.T) {
	useDotEnv(t, "\xef\xbb\xbf# comentario\r\n"+
		"\r\n"+
		"export DOTENV_EXPORT=exportada\r\n"+
		"DOTENV_SPACES = com espacos  \n"+
		"DOTENV_DOUBLE=\"aspas \\\"duplas\\\"\\ne linha\" # comentario\n"+
		"DOTENV_SINGLE='literal \\n # nao e comentario'\n"+
		"DOTENV_INLINE=valor # comentario\n"+
		"DOTENV_HASH=a#b\n"+
		"DOTENV_EMPTY=\n"+
		"DOTENV_URL=postgres://u:p@host/db?x=1&y=2\n"+
		"DOTENV_TWICE=primeiro\n"+
		"DOTENV_TWICE=segundo\n")

	values := secrets(t, "DOTENV_EXPORT", "DOTENV_SPACES", "DOTENV_DOUBLE", "DOTENV_SINGLE", "DOTENV_INLINE",
		"DOTENV_HASH", "DOTENV_EMPTY", "DOTENV_URL", "DOTENV_TWICE")

	want := map[string]string{
		"DOTENV_EXPORT": "exportada",
		"DOTENV_SPACES": "com espacos",
		"DOTENV_DOUBLE": "aspas \"duplas\"\ne linha",
		"DOTENV_SINGLE": `literal \n # nao e comentario`,
		"DOTENV_INLINE": "valor",
		"DOTENV_HASH":   "a#b",
		"DOTENV_URL":    "postgres://u:p@host/db?x=1&y=2",
		"DOTENV_TWICE":  "segundo",
	}

	for name, expected := range want {
		if values[name] == nil || *values[name] != expected {
			got := "<nil>"

			if values[name] != nil {
				got = *values[name]
			}

			t.Errorf("%s: esperava %q, veio %q", name, expected, got)
		}
	}

	if values["DOTENV_EMPTY"] != nil {
		t.Error("variavel vazia deveria ser nil")
	}
}

func TestBrokenDotEnvIsAnErrorWithoutTheValues(t *testing.T) {
	for _, content := range []string{
		"isso nao e uma variavel\n",
		"1ABC=x\n",
		"=sem-nome\n",
		"DOTENV_Q=\"sem fechar segredo123\n",
		"DOTENV_Q='fechada' texto-depois segredo123\n",
	} {
		useDotEnv(t, "OK=1\n"+content)

		_, err := backend.NewConfig([]string{"OK"}, nil)

		if err == nil {
			t.Errorf("%q deveria dar erro", content)

			continue
		}

		if strings.Contains(err.Error(), "segredo123") || !strings.Contains(err.Error(), "line 2") {
			t.Errorf("o erro deveria dizer a linha e nunca o valor: %v", err)
		}
	}
}

func TestDotEnvIsLoadedIntoTheProcessEnvironment(t *testing.T) {
	useDotEnv(t, "DOTENV_PROCESS=do-arquivo\nDOTENV_KEEP=do-arquivo\n")

	t.Setenv("DOTENV_KEEP", "do-ambiente")

	if _, err := backend.NewConfig(nil, nil); err != nil {
		t.Fatal(err)
	}

	// como o load_dotenv: depois do NewConfig, qualquer codigo (ou processo filho) ve as variaveis
	if got := os.Getenv("DOTENV_PROCESS"); got != "do-arquivo" {
		t.Fatalf("a variavel do .env deveria estar no ambiente, veio %q", got)
	}

	if got := os.Getenv("DOTENV_KEEP"); got != "do-ambiente" {
		t.Fatalf("o .env nao pode sobrescrever o ambiente, veio %q", got)
	}
}

func TestModuleRootDoesNotDependOnTheCurrentDirectory(t *testing.T) {
	want, err := filepath.Abs("../..")

	if err != nil {
		t.Fatal(err)
	}

	// a pasta de onde o comando roda nao importa: o fonte diz onde fica a raiz de base/
	for _, dir := range []string{".", t.TempDir(), filepath.Dir(want)} {
		t.Chdir(dir)

		got, found := backend.ModuleRoot()

		if !found || got != want {
			t.Errorf("rodando de %s: esperava %s, veio %s (%v)", dir, want, got, found)
		}
	}
}
