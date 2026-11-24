package unit

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/auth"
)

// o menor custo do bcrypt, para os testes nao ficarem lentos
const testCost = 4

// formato de um hash bcrypt com custo 04: $2a$04$ + 22 caracteres de salt + 31 de hash
var bcryptFormat = regexp.MustCompile(`^\$2[aby]\$04\$[./A-Za-z0-9]{53}$`)

func newTestHasher() (*auth.Hasher, func() string) {
	logger, buffer := newTestLogger()

	return auth.NewHasherWithCost(logger, testCost), buffer.String
}

func TestHashFormat(t *testing.T) {
	hasher, _ := newTestHasher()

	encoded, err := hasher.Hash("senha-secreta")

	if err != nil {
		t.Fatalf("nao esperava erro: %v", err)
	}

	if len(encoded) != 60 {
		t.Errorf("um hash bcrypt tem 60 caracteres, veio %d: %q", len(encoded), encoded)
	}

	if !bcryptFormat.MatchString(encoded) {
		t.Fatalf("hash fora do formato bcrypt com custo %d: %q", testCost, encoded)
	}
}

func TestHashUsesDefaultCost(t *testing.T) {
	logger, _ := newTestLogger()

	encoded, err := auth.NewHasher(logger).Hash("senha")

	if err != nil {
		t.Fatalf("nao esperava erro: %v", err)
	}

	if !strings.Contains(encoded, "$"+strconv.Itoa(auth.DefaultCost)+"$") {
		t.Fatalf("esperava o custo %d: %q", auth.DefaultCost, encoded)
	}
}

func TestHashInvalidCostFallsBackToDefault(t *testing.T) {
	logger, _ := newTestLogger()

	// um custo abaixo e outro acima do intervalo (cada hash no custo padrao leva alguns segundos)
	for _, cost := range []int{3, 99} {
		encoded, err := auth.NewHasherWithCost(logger, cost).Hash("senha")

		if err != nil {
			t.Fatalf("custo %d: nao esperava erro: %v", cost, err)
		}

		if !strings.Contains(encoded, "$"+strconv.Itoa(auth.DefaultCost)+"$") {
			t.Errorf("custo %d fora do intervalo deveria usar o padrao: %q", cost, encoded)
		}
	}
}

func TestHashDefaultCostIsNotWeak(t *testing.T) {
	if auth.DefaultCost < 10 {
		t.Fatalf("o custo padrao do bcrypt nao deveria ser menor que 10, veio %d", auth.DefaultCost)
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

func TestHashAcceptsUnicodeAndSymbolPasswords(t *testing.T) {
	hasher, _ := newTestHasher()

	for name, password := range map[string]string{
		"unicode":  "sénha-çom-açentos-日本語-🔐",
		"espacos":  "  com espacos  ",
		"simbolos": `!@#$%^&*()_+-=[]{}|;':",./<>?`,
		"um byte":  "a",
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

func TestHashPasswordLengthLimit(t *testing.T) {
	hasher, _ := newTestHasher()

	atLimit := strings.Repeat("a", 72)

	encoded, err := hasher.Hash(atLimit)

	if err != nil {
		t.Fatalf("72 bytes deveriam ser aceitos: %v", err)
	}

	if ok, _ := hasher.Verify(atLimit, encoded); !ok {
		t.Fatal("a senha de 72 bytes deveria conferir")
	}

	for name, password := range map[string]string{
		"73 bytes":                   strings.Repeat("a", 73),
		"muito longa":                strings.Repeat("a", 5000),
		"emojis (bytes, nao letras)": strings.Repeat("🔐", 19), // 19 x 4 bytes = 76
	} {
		hashed, err := hasher.Hash(password)

		if !errors.Is(err, auth.ErrPasswordTooLong) {
			t.Errorf("%s: esperava ErrPasswordTooLong, veio %v", name, err)
		}

		if hashed != "" {
			t.Errorf("%s: com erro, o hash deveria ser vazio", name)
		}
	}
}

func TestVerifyDoesNotIgnoreBytesAfter72(t *testing.T) {
	hasher, _ := newTestHasher()

	encoded, _ := hasher.Hash(strings.Repeat("a", 72))

	// o bcrypt puro ignoraria o "b" e diria que confere
	ok, err := hasher.Verify(strings.Repeat("a", 72)+"b", encoded)

	if err != nil || ok {
		t.Fatalf("uma senha maior que 72 bytes nao deveria conferir (ok=%v err=%v)", ok, err)
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

func TestVerifyUsesCostStoredInTheHash(t *testing.T) {
	logger, _ := newTestLogger()

	encoded, _ := auth.NewHasherWithCost(logger, 4).Hash("senha")

	// outro Hasher, com outro custo, ainda confere o hash antigo
	ok, err := auth.NewHasherWithCost(logger, 5).Verify("senha", encoded)

	if err != nil || !ok {
		t.Fatalf("esperava true, veio ok=%v err=%v", ok, err)
	}
}

func TestVerifyHashFromAnotherPasswordIsRejected(t *testing.T) {
	hasher, _ := newTestHasher()

	hashA, _ := hasher.Hash("a")
	hashB, _ := hasher.Hash("b")

	// o prefixo ($2a$04$) e os 22 caracteres de salt de A, com o resto do hash de B
	mixed := hashA[:29] + hashB[29:]

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

	cases := map[string]string{
		"vazio":                 "",
		"texto solto":           "isso-nao-e-um-hash",
		"senha em texto":        "senha",
		"muito curto":           "$2a$04$abc",
		"hash cortado":          valid[:40],
		"versao desconhecida":   "$9z$" + valid[4:],
		"sem prefixo":           valid[1:],
		"custo ilegivel":        "$2a$xx$" + valid[7:],
		"custo zero":            "$2a$00$" + valid[7:],
		"custo acima do limite": "$2a$16$" + valid[7:],
		"custo gigante":         "$2a$31$" + valid[7:],
		"salt com lixo":         valid[:10] + "!!!!" + valid[14:],
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

func TestHashLogsAnErrorForEmptyAndLongPasswords(t *testing.T) {
	for name, password := range map[string]string{"vazia": "", "longa": strings.Repeat("a", 100)} {
		hasher, logs := newTestHasher()

		hasher.Hash(password)

		output := logs()

		if !strings.Contains(output, "[ERROR] => ") || strings.Contains(output, "[SUCCESS]") {
			t.Errorf("%s: esperava um log de erro e nenhum de sucesso: %q", name, output)
		}
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
	hasher := auth.NewHasherWithCost(nil, testCost)

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
