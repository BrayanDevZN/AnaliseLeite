package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestNewConfigWithRequiredSecretsPresent(t *testing.T) {
	t.Setenv("ML_KEY", "abc")

	config, err := NewConfig(nil, []string{"ML_KEY"})

	if err != nil {
		t.Fatalf("nao esperava erro, veio: %v", err)
	}

	if config == nil {
		t.Fatal("esperava uma Config, veio nil")
	}
}

func TestNewConfigWithoutRequiredSecretsNeverFails(t *testing.T) {
	config, err := NewConfig([]string{"NAO_EXISTE"}, nil)

	if err != nil {
		t.Fatalf("nao esperava erro sem secrets obrigatorias, veio: %v", err)
	}

	if config == nil {
		t.Fatal("esperava uma Config, veio nil")
	}
}

func TestNewConfigMissingRequiredSecret(t *testing.T) {
	config, err := NewConfig(nil, []string{"SECRET_QUE_NAO_EXISTE"})

	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}

	if config != nil {
		t.Fatal("com erro, a Config deveria ser nil")
	}

	var missing *MissingSecretsError

	if !errors.As(err, &missing) {
		t.Fatalf("esperava *MissingSecretsError, veio %T", err)
	}
}

func TestNewConfigReportsEveryMissingSecret(t *testing.T) {
	t.Setenv("PRESENTE", "ok")

	_, err := NewConfig(nil, []string{"FALTA_1", "PRESENTE", "FALTA_2"})

	var missing *MissingSecretsError

	if !errors.As(err, &missing) {
		t.Fatalf("esperava *MissingSecretsError, veio %v", err)
	}

	want := []string{"FALTA_1", "FALTA_2"}

	if !reflect.DeepEqual(missing.Names, want) {
		t.Fatalf("nomes que faltam: esperava %v, veio %v", want, missing.Names)
	}
}

func TestMissingSecretsErrorMessage(t *testing.T) {
	err := &MissingSecretsError{Names: []string{"A", "B"}}

	want := "missing required secrets: A, B"

	if err.Error() != want {
		t.Fatalf("mensagem: esperava %q, veio %q", want, err.Error())
	}
}

func TestEmptyRequiredSecretCountsAsMissing(t *testing.T) {
	t.Setenv("VAZIA", "")

	_, err := NewConfig(nil, []string{"VAZIA"})

	var missing *MissingSecretsError

	if !errors.As(err, &missing) {
		t.Fatalf("uma secret vazia deveria contar como ausente, veio %v", err)
	}
}

func TestGetSecretsReturnsNameAndValue(t *testing.T) {
	t.Setenv("PORTA", "8080")
	t.Setenv("HOST", "localhost")

	config, _ := NewConfig([]string{"PORTA", "HOST"}, nil)
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
	config, _ := NewConfig([]string{"VARIAVEL_QUE_NAO_EXISTE"}, nil)
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

	config, _ := NewConfig([]string{"VAZIA"}, nil)

	if config.GetSecrets()["VAZIA"] != nil {
		t.Fatal("uma variavel vazia deveria ser nil")
	}
}

func TestGetSecretsIncludesRequiredSecrets(t *testing.T) {
	t.Setenv("ML_KEY", "abc")
	t.Setenv("PORTA", "8080")

	config, err := NewConfig([]string{"PORTA"}, []string{"ML_KEY"})

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

	config, _ := NewConfig([]string{"ML_KEY", "ML_KEY"}, []string{"ML_KEY"})

	if len(config.GetSecrets()) != 1 {
		t.Fatalf("esperava 1 chave, veio %v", config.GetSecrets())
	}
}

func TestGetSecretsReadsCurrentValue(t *testing.T) {
	t.Setenv("MUDA", "antes")

	config, _ := NewConfig([]string{"MUDA"}, nil)

	t.Setenv("MUDA", "depois")

	if got := *config.GetSecrets()["MUDA"]; got != "depois" {
		t.Fatalf("esperava o valor atual %q, veio %q", "depois", got)
	}
}

func TestGetSecretsDoesNotChangeTheConfig(t *testing.T) {
	secrets := []string{"A", "B"}
	required := []string{"C"}

	config, _ := NewConfig(secrets, nil)
	config.requiredSecrets = required

	config.GetSecrets()
	config.GetSecrets()

	if !reflect.DeepEqual(config.secrets, []string{"A", "B"}) || !reflect.DeepEqual(config.requiredSecrets, []string{"C"}) {
		t.Fatalf("GetSecrets alterou a Config: %v %v", config.secrets, config.requiredSecrets)
	}
}

func TestUniqueKeepsOrder(t *testing.T) {
	got := unique([]string{"b", "a", "b", "c", "a"})
	want := []string{"b", "a", "c"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("esperava %v, veio %v", want, got)
	}
}

func TestMissingSecretsErrorListsNamesInMessage(t *testing.T) {
	_, err := NewConfig(nil, []string{"UMA_QUE_FALTA", "OUTRA_QUE_FALTA"})

	if err == nil || !strings.Contains(err.Error(), "UMA_QUE_FALTA, OUTRA_QUE_FALTA") {
		t.Fatalf("a mensagem deveria listar as secrets que faltam, veio %v", err)
	}
}
