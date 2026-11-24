package backend

import (
	"os"
	"strings"
)

// MissingSecretsError e o erro devolvido quando faltam secrets obrigatorias.
type MissingSecretsError struct {
	Names []string
}

func (e *MissingSecretsError) Error() string {
	return "missing required secrets: " + strings.Join(e.Names, ", ")
}

// Config le secrets das variaveis de ambiente.
type Config struct {
	secrets         []string
	requiredSecrets []string
}

// NewConfig recebe os nomes das variaveis que devem ser lidas (secrets) e,
// opcionalmente, os nomes das que sao obrigatorias (requiredSecrets, pode ser nil).
// Se alguma obrigatoria nao existir, devolve um *MissingSecretsError com o nome de todas as que faltam.
func NewConfig(secrets []string, requiredSecrets []string) (*Config, error) {
	config := &Config{
		secrets:         unique(secrets),
		requiredSecrets: unique(requiredSecrets),
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// validate confere se todas as secrets obrigatorias existem.
func (c *Config) validate() error {
	var missing []string

	for _, name := range c.requiredSecrets {
		if lookup(name) == nil {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return &MissingSecretsError{Names: missing}
	}

	return nil
}

// GetSecrets le as variaveis e devolve um mapa com o nome de cada uma e o valor dela.
// Quando a variavel nao existe, o valor e nil (o equivalente a None).
// As obrigatorias tambem entram no mapa.
func (c *Config) GetSecrets() map[string]*string {
	values := make(map[string]*string)

	for _, name := range unique(append(append([]string{}, c.secrets...), c.requiredSecrets...)) {
		values[name] = lookup(name)
	}

	return values
}

// lookup devolve o valor da variavel de ambiente, ou nil se ela nao existir ou estiver vazia.
func lookup(name string) *string {
	value, ok := os.LookupEnv(name)

	if !ok || value == "" {
		return nil
	}

	return &value
}

// unique tira os nomes repetidos, mantendo a ordem em que apareceram.
func unique(names []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(names))

	for _, name := range names {
		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}

	return result
}
