package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
)

const (
	// formato do hash guardado: pbkdf2-sha256$iteracoes$salt$hash
	hashScheme = "pbkdf2-sha256"

	// DefaultIterations e o numero de iteracoes do PBKDF2-HMAC-SHA256 recomendado pela OWASP.
	DefaultIterations = 600_000

	// limite para nao aceitar um hash que force um calculo enorme
	maxIterations = 10_000_000

	saltSize = 16
	keySize  = 32
)

var (
	ErrEmptyPassword = errors.New("password must not be empty")
	ErrInvalidHash   = errors.New("invalid password hash format")
)

// Hasher gera e confere hashes de senha.
type Hasher struct {
	logger     *backend.Logger
	iterations int
}

// NewHasher cria um Hasher com o numero de iteracoes padrao.
// Se logger for nil, os logs vao para o terminal.
func NewHasher(logger *backend.Logger) *Hasher {
	return NewHasherWithIterations(logger, DefaultIterations)
}

// NewHasherWithIterations cria um Hasher com um numero de iteracoes escolhido (util em testes).
func NewHasherWithIterations(logger *backend.Logger, iterations int) *Hasher {
	if logger == nil {
		logger = backend.NewLogger()
	}

	if iterations < 1 {
		iterations = DefaultIterations
	}

	return &Hasher{logger: logger, iterations: iterations}
}

// Hash gera o hash de uma senha, com um salt aleatorio.
// A senha e o hash nunca vao para os logs.
func (h *Hasher) Hash(password string) (string, error) {
	h.logger.Info("Generating password hash...")

	if password == "" {
		h.logger.Error("Password must not be empty")

		return "", ErrEmptyPassword
	}

	salt := make([]byte, saltSize)

	if _, err := rand.Read(salt); err != nil {
		h.logger.Error("Could not generate salt: %v", err)

		return "", fmt.Errorf("generate salt: %w", err)
	}

	key, err := pbkdf2.Key(sha256.New, password, salt, h.iterations, keySize)

	if err != nil {
		h.logger.Error("Could not generate hash: %v", err)

		return "", fmt.Errorf("generate hash: %w", err)
	}

	encoded := strings.Join([]string{
		hashScheme,
		strconv.Itoa(h.iterations),
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	}, "$")

	h.logger.Success("Password hash generated with success")

	return encoded, nil
}

// Verify confere se a senha corresponde ao hash.
// Senha errada devolve (false, nil); hash em formato invalido devolve ErrInvalidHash.
func (h *Hasher) Verify(password string, encoded string) (bool, error) {
	h.logger.Info("Verifying password...")

	iterations, salt, key, err := parseHash(encoded)

	if err != nil {
		h.logger.Error("Invalid password hash format")

		return false, err
	}

	derived, err := pbkdf2.Key(sha256.New, password, salt, iterations, keySize)

	if err != nil {
		h.logger.Error("Could not verify password: %v", err)

		return false, fmt.Errorf("verify password: %w", err)
	}

	if subtle.ConstantTimeCompare(derived, key) != 1 {
		h.logger.Warning("Password does not match")

		return false, nil
	}

	h.logger.Success("Password verified with success")

	return true, nil
}

// parseHash separa o hash guardado em iteracoes, salt e chave.
func parseHash(encoded string) (int, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")

	if len(parts) != 4 || parts[0] != hashScheme {
		return 0, nil, nil, ErrInvalidHash
	}

	iterations, err := strconv.Atoi(parts[1])

	if err != nil || iterations < 1 || iterations > maxIterations {
		return 0, nil, nil, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[2])

	if err != nil || len(salt) != saltSize {
		return 0, nil, nil, ErrInvalidHash
	}

	key, err := base64.RawStdEncoding.DecodeString(parts[3])

	if err != nil || len(key) != keySize {
		return 0, nil, nil, ErrInvalidHash
	}

	return iterations, salt, key, nil
}
