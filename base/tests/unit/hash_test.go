package unit

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/auth"
)

// poucas iteracoes, para os testes nao ficarem lentos
const testIterations = 1000

func newTestHasher() (*auth.Hasher, func() string) {
	logger, buffer := newTestLogger()

	return auth.NewHasherWithIterations(logger, testIterations), buffer.String
}

func TestHashFormat(t *testing.T) {
	hasher, _ := newTestHasher()

	encoded, err := hasher.Hash("senha-secreta")

	if err != nil {
		t.Fatalf("nao esperava erro: %v", err)
	}

	parts := strings.Split(encoded, "$")

	if len(parts) != 4 {
		t.Fatalf("esperava 4 partes separadas por $, veio %d: %q", len(parts), encoded)
	}

	if parts[0] != "pbkdf2-sha256" {
		t.Errorf("algoritmo: esperava pbkdf2-sha256, veio %q", parts[0])
	}

	if parts[1] != strconv.Itoa(testIterations) {
		t.Errorf("iteracoes: esperava %d, veio %q", testIterations, parts[1])
	}

	if parts[2] == "" || parts[3] == "" {
		t.Errorf("salt e hash nao deveriam estar vazios: %q", encoded)
	}
}

func TestHashUsesDefaultIterations(t *testing.T) {
	logger, _ := newTestLogger()

	encoded, err := auth.NewHasher(logger).Hash("senha")

	if err != nil {
		t.Fatalf("nao esperava erro: %v", err)
	}

	if !strings.Contains(encoded, "$"+strconv.Itoa(auth.DefaultIterations)+"$") {
		t.Fatalf("esperava %d iteracoes: %q", auth.DefaultIterations, encoded)
	}
}

func TestHashNonPositiveIterationsFallBackToDefault(t *testing.T) {
	logger, _ := newTestLogger()

	encoded, err := auth.NewHasherWithIterations(logger, 0).Hash("senha")

	if err != nil {
		t.Fatalf("nao esperava erro: %v", err)
	}

	if !strings.Contains(encoded, "$"+strconv.Itoa(auth.DefaultIterations)+"$") {
		t.Fatalf("0 iteracoes deveria usar o padrao: %q", encoded)
	}
}

func TestHashSamePasswordGivesDifferentHashes(t *testing.T) {
	hasher, _ := newTestHasher()

	first, _ := hasher.Hash("mesma-senha")
	second, _ := hasher.Hash("mesma-senha")

	if first == second {
		t.Fatal("o salt aleatorio deveria gerar hashes diferentes para a mesma senha")
	}
}

func TestHashDoesNotContainThePassword(t *testing.T) {
	hasher, _ := newTestHasher()

	encoded, _ := hasher.Hash("senha-que-nao-pode-vazar")

	if strings.Contains(encoded, "senha-que-nao-pode-vazar") {
		t.Fatalf("o hash nao pode conter a senha: %q", encoded)
	}
}

func TestHashEmptyPassword(t *testing.T) {
	hasher, _ := newTestHasher()

	encoded, err := hasher.Hash("")

	if !errors.Is(err, auth.ErrEmptyPassword) {
		t.Fatalf("esperava ErrEmptyPassword, veio %v", err)
	}

	if encoded != "" {
		t.Fatalf("com erro, o hash deveria ser vazio, veio %q", encoded)
	}
}

func TestHashAcceptsUnicodeAndLongPasswords(t *testing.T) {
	hasher, _ := newTestHasher()

	for name, password := range map[string]string{
		"unicode":  "sénha-çom-açentos-日本語-🔐",
		"longa":    strings.Repeat("a", 5000),
		"espacos":  "  com espacos  ",
		"simbolos": `!@#$%^&*()_+-=[]{}|;':",./<>?`,
	} {
		encoded, err := hasher.Hash(password)

		if err != nil {
			t.Fatalf("%s: nao esperava erro: %v", name, err)
		}

		if ok, err := hasher.Verify(password, encoded); err != nil || !ok {
			t.Errorf("%s: a senha deveria conferir (ok=%v, err=%v)", name, ok, err)
		}
	}
}

func TestVerifyCorrectPassword(t *testing.T) {
	hasher, _ := newTestHasher()

	encoded, _ := hasher.Hash("senha-correta")

	ok, err := hasher.Verify("senha-correta", encoded)

	if err != nil || !ok {
		t.Fatalf("esperava true e nenhum erro, veio ok=%v err=%v", ok, err)
	}
}

func TestVerifyWrongPassword(t *testing.T) {
	hasher, _ := newTestHasher()

	encoded, _ := hasher.Hash("senha-correta")

	for _, wrong := range []string{"senha-errada", "SENHA-CORRETA", "senha-correta ", "", "senha-corret"} {
		ok, err := hasher.Verify(wrong, encoded)

		if err != nil {
			t.Errorf("%q: senha errada nao e um erro, veio %v", wrong, err)
		}

		if ok {
			t.Errorf("%q: nao deveria conferir", wrong)
		}
	}
}

func TestVerifyUsesIterationsStoredInTheHash(t *testing.T) {
	logger, _ := newTestLogger()

	encoded, _ := auth.NewHasherWithIterations(logger, 1000).Hash("senha")

	// outro Hasher, com outro numero de iteracoes, ainda confere o hash antigo
	ok, err := auth.NewHasherWithIterations(logger, 2000).Verify("senha", encoded)

	if err != nil || !ok {
		t.Fatalf("esperava true, veio ok=%v err=%v", ok, err)
	}
}

func TestVerifyHashFromAnotherPasswordIsRejected(t *testing.T) {
	hasher, _ := newTestHasher()

	hashA, _ := hasher.Hash("a")
	hashB, _ := hasher.Hash("b")

	partsA := strings.Split(hashA, "$")
	partsB := strings.Split(hashB, "$")

	// salt de A com a chave de B
	mixed := strings.Join([]string{partsA[0], partsA[1], partsA[2], partsB[3]}, "$")

	if ok, _ := hasher.Verify("a", mixed); ok {
		t.Fatal("um hash adulterado nao deveria conferir")
	}

	if ok, _ := hasher.Verify("b", mixed); ok {
		t.Fatal("um hash adulterado nao deveria conferir")
	}
}

func TestVerifyInvalidFormats(t *testing.T) {
	hasher, _ := newTestHasher()

	valid, _ := hasher.Hash("senha")
	parts := strings.Split(valid, "$")

	cases := map[string]string{
		"vazio":              "",
		"texto solto":        "isso-nao-e-um-hash",
		"senha em texto":     "senha",
		"poucas partes":      "pbkdf2-sha256$1000$abc",
		"partes demais":      valid + "$extra",
		"algoritmo errado":   "bcrypt$" + strings.Join(parts[1:], "$"),
		"iteracoes texto":    strings.Join([]string{parts[0], "muitas", parts[2], parts[3]}, "$"),
		"iteracoes zero":     strings.Join([]string{parts[0], "0", parts[2], parts[3]}, "$"),
		"iteracoes negativa": strings.Join([]string{parts[0], "-5", parts[2], parts[3]}, "$"),
		"iteracoes enormes":  strings.Join([]string{parts[0], "99999999", parts[2], parts[3]}, "$"),
		"salt invalido":      strings.Join([]string{parts[0], parts[1], "!!!", parts[3]}, "$"),
		"salt curto":         strings.Join([]string{parts[0], parts[1], "YWJj", parts[3]}, "$"),
		"chave invalida":     strings.Join([]string{parts[0], parts[1], parts[2], "!!!"}, "$"),
		"chave curta":        strings.Join([]string{parts[0], parts[1], parts[2], "YWJj"}, "$"),
	}

	for name, encoded := range cases {
		ok, err := hasher.Verify("senha", encoded)

		if !errors.Is(err, auth.ErrInvalidHash) {
			t.Errorf("%s: esperava ErrInvalidHash, veio %v", name, err)
		}

		if ok {
			t.Errorf("%s: nao deveria conferir", name)
		}
	}
}

func TestHashLogsGenerationWithoutLeakingSecrets(t *testing.T) {
	hasher, logs := newTestHasher()

	encoded, _ := hasher.Hash("senha-que-nao-pode-vazar")

	output := logs()

	for _, want := range []string{"[INFO] => ", "Generating password hash", "[SUCCESS] => ", "Password hash generated with success"} {
		if !strings.Contains(output, want) {
			t.Errorf("faltou %q nos logs: %q", want, output)
		}
	}

	if strings.Contains(output, "senha-que-nao-pode-vazar") || strings.Contains(output, encoded) {
		t.Fatalf("os logs nao podem conter a senha nem o hash: %q", output)
	}
}

func TestHashLogsAnErrorForEmptyPassword(t *testing.T) {
	hasher, logs := newTestHasher()

	hasher.Hash("")

	output := logs()

	if !strings.Contains(output, "[ERROR] => ") || strings.Contains(output, "[SUCCESS]") {
		t.Fatalf("esperava um log de erro e nenhum de sucesso: %q", output)
	}
}

func TestVerifyLogs(t *testing.T) {
	hasher, logs := newTestHasher()

	encoded, _ := hasher.Hash("certa")

	hasher.Verify("certa", encoded)
	hasher.Verify("errada", encoded)
	hasher.Verify("certa", "lixo")

	output := logs()

	for _, want := range []string{"Verifying password", "Password verified with success", "[WARNING] => ", "Password does not match", "[ERROR] => ", "Invalid password hash format"} {
		if !strings.Contains(output, want) {
			t.Errorf("faltou %q nos logs: %q", want, output)
		}
	}

	for _, secret := range []string{"certa", "errada", encoded} {
		if strings.Contains(output, secret) {
			t.Errorf("os logs nao podem conter %q: %q", secret, output)
		}
	}
}

func TestNewHasherWithNilLoggerDoesNotPanic(t *testing.T) {
	hasher := auth.NewHasherWithIterations(nil, testIterations)

	encoded, err := hasher.Hash("senha")

	if err != nil {
		t.Fatalf("nao esperava erro: %v", err)
	}

	if ok, _ := hasher.Verify("senha", encoded); !ok {
		t.Fatal("deveria conferir")
	}
}

func TestHashConcurrentUse(t *testing.T) {
	hasher, _ := newTestHasher()

	var wait sync.WaitGroup

	for i := 0; i < 20; i++ {
		wait.Add(1)

		go func(i int) {
			defer wait.Done()

			password := "senha-" + strconv.Itoa(i)
			encoded, err := hasher.Hash(password)

			if err != nil {
				t.Errorf("erro ao gerar: %v", err)

				return
			}

			if ok, err := hasher.Verify(password, encoded); err != nil || !ok {
				t.Errorf("a senha %d deveria conferir (ok=%v err=%v)", i, ok, err)
			}
		}(i)
	}

	wait.Wait()
}
