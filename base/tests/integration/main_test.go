package integration

import (
	"os"
	"testing"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
)

// TestMain desliga a leitura do .env da raiz de base/, para os testes nao mudarem conforme o
// .env de quem roda. Os testes do proprio .env ligam um arquivo de teste com backend.SetDotEnvPath.
func TestMain(m *testing.M) {
	backend.SetDotEnvPath("")

	os.Exit(m.Run())
}
