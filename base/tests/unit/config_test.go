package unit

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
)

func TestNewConfigWithRequiredSecretsPresent(t *testing.T) {
	t.Setenv("ML_KEY", "abc")

	config, err := backend.NewConfig(nil, []string{"ML_KEY"})

	if err != nil {
		t.Fatalf("nao esperava erro, veio: %v", err)
	}

	if config == nil {
		t.Fatal("esperava uma Config, veio nil")
	}
}

func TestNewConfigWithoutRequiredSecretsNeverFails(t *testing.T) {
	config, err := backend.NewConfig([]string{"NAO_EXISTE"}, nil)

	if err != nil {
		t.Fatalf("nao esperava erro sem secrets obrigatorias, veio: %v", err)
	}

	if config == nil {
		t.Fatal("esperava uma Config, veio nil")
	}
}

func TestNewConfigMissingRequiredSecret(t *testing.T) {
	config, err := backend.NewConfig(nil, []string{"SECRET_QUE_NAO_EXISTE"})

	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}

	if config != nil {
		t.Fatal("com erro, a Config deveria ser nil")
	}

	var missing *backend.MissingSecretsError

	if !errors.As(err, &missing) {
		t.Fatalf("esperava *MissingSecretsError, veio %T", err)
	}
}

func TestNewConfigReportsEveryMissingSecret(t *testing.T) {
	t.Setenv("PRESENTE", "ok")

	_, err := backend.NewConfig(nil, []string{"FALTA_1", "PRESENTE", "FALTA_2"})

	var missing *backend.MissingSecretsError

	if !errors.As(err, &missing) {
		t.Fatalf("esperava *MissingSecretsError, veio %v", err)
	}

	want := []string{"FALTA_1", "FALTA_2"}

	if !reflect.DeepEqual(missing.Names, want) {
		t.Fatalf("nomes que faltam: esperava %v, veio %v", want, missing.Names)
	}
}

func TestMissingSecretsErrorMessage(t *testing.T) {
	err := &backend.MissingSecretsError{Names: []string{"A", "B"}}

	want := "missing required secrets: A, B"

	if err.Error() != want {
		t.Fatalf("mensagem: esperava %q, veio %q", want, err.Error())
	}
}

func TestNewConfigErrorMessageListsNames(t *testing.T) {
	_, err := backend.NewConfig(nil, []string{"UMA_QUE_FALTA", "OUTRA_QUE_FALTA"})

	if err == nil || !strings.Contains(err.Error(), "UMA_QUE_FALTA, OUTRA_QUE_FALTA") {
		t.Fatalf("a mensagem deveria listar as secrets que faltam, veio %v", err)
	}
}

func TestEmptyRequiredSecretCountsAsMissing(t *testing.T) {
	t.Setenv("VAZIA", "")

	_, err := backend.NewConfig(nil, []string{"VAZIA"})

	var missing *backend.MissingSecretsError

	if !errors.As(err, &missing) {
		t.Fatalf("uma secret vazia deveria contar como ausente, veio %v", err)
	}
}

func TestGetSecretsReturnsNameAndValue(t *testing.T) {
	t.Setenv("PORTA", "8080")
	t.Setenv("HOST", "localhost")

	config, _ := backend.NewConfig([]string{"PORTA", "HOST"}, nil)
	secrets := config.GetSecrets()

	if len(secrets) != 2 {
		t.Fatalf("esperava 2 chaves, veio %d: %v", len(secrets), secrets)
	}

	if secrets["PORTA"] == nil || *secrets["PORTA"] != "8080" {
		t.Fatalf("PORTA errada: %v", secrets["PORTA"])
	}

	if secrets["HOST"] == nil || *secrets["HOST"] != "localhost" {
		t.Fatalf("HOST errado: %v", secrets["HOST"])
	}
}

func TestGetSecretsMissingVariableIsNil(t *testing.T) {
	config, _ := backend.NewConfig([]string{"VARIAVEL_QUE_NAO_EXISTE"}, nil)
	secrets := config.GetSecrets()

	value, exists := secrets["VARIAVEL_QUE_NAO_EXISTE"]

	if !exists {
		t.Fatal("a variavel deveria aparecer no mapa, com valor nil")
	}

	if value != nil {
		t.Fatalf("esperava nil, veio %q", *value)
	}
}

func TestGetSecretsEmptyVariableIsNil(t *testing.T) {
	t.Setenv("VAZIA", "")

	config, _ := backend.NewConfig([]string{"VAZIA"}, nil)

	if config.GetSecrets()["VAZIA"] != nil {
		t.Fatal("uma variavel vazia deveria ser nil")
	}
}

func TestGetSecretsIncludesRequiredSecrets(t *testing.T) {
	t.Setenv("ML_KEY", "abc")
	t.Setenv("PORTA", "8080")

	config, err := backend.NewConfig([]string{"PORTA"}, []string{"ML_KEY"})

	if err != nil {
		t.Fatalf("nao esperava erro: %v", err)
	}

	secrets := config.GetSecrets()

	if secrets["ML_KEY"] == nil || *secrets["ML_KEY"] != "abc" {
		t.Fatalf("a obrigatoria deveria estar no mapa: %v", secrets)
	}

	if secrets["PORTA"] == nil || *secrets["PORTA"] != "8080" {
		t.Fatalf("a opcional deveria estar no mapa: %v", secrets)
	}
}

func TestGetSecretsIgnoresRepeatedNames(t *testing.T) {
	t.Setenv("ML_KEY", "abc")

	config, _ := backend.NewConfig([]string{"ML_KEY", "ML_KEY"}, []string{"ML_KEY"})

	if len(config.GetSecrets()) != 1 {
		t.Fatalf("esperava 1 chave, veio %v", config.GetSecrets())
	}
}

func TestGetSecretsReadsCurrentValue(t *testing.T) {
	t.Setenv("MUDA", "antes")

	config, _ := backend.NewConfig([]string{"MUDA"}, nil)

	t.Setenv("MUDA", "depois")

	if got := *config.GetSecrets()["MUDA"]; got != "depois" {
		t.Fatalf("esperava o valor atual %q, veio %q", "depois", got)
	}
}

func TestGetSecretsIsRepeatable(t *testing.T) {
	t.Setenv("ML_KEY", "abc")

	config, _ := backend.NewConfig([]string{"A", "B"}, []string{"ML_KEY"})

	first := config.GetSecrets()
	second := config.GetSecrets()

	if len(first) != 3 || len(second) != 3 {
		t.Fatalf("esperava 3 chaves nas duas chamadas: %v %v", first, second)
	}

	if *first["ML_KEY"] != *second["ML_KEY"] {
		t.Fatal("duas chamadas seguidas deveriam dar o mesmo valor")
	}
}
