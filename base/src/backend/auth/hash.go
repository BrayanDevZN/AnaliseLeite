package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
)

const (
	// DefaultCost e o custo do bcrypt (cada +1 dobra o tempo de calculo).
	DefaultCost = 12

	// limite para nao aceitar um hash forjado que force um calculo enorme
	maxCost = 15

	// o bcrypt so usa os 72 primeiros bytes da senha
	maxPasswordLength = 72
)

var (
	ErrEmptyPassword   = errors.New("password must not be empty")
	ErrPasswordTooLong = fmt.Errorf("password must have at most %d bytes", maxPasswordLength)
	ErrInvalidHash     = errors.New("invalid password hash format")
)

// Hasher gera e confere hashes de senha com bcrypt.
type Hasher struct {
	logger *backend.Logger
	cost   int
}

// NewHasher cria um Hasher com o custo padrao.
// Se logger for nil, os logs vao para o terminal.
func NewHasher(logger *backend.Logger) *Hasher {
	return NewHasherWithCost(logger, DefaultCost)
}

// NewHasherWithCost cria um Hasher com um custo escolhido (util em testes).
// Um custo fora do intervalo aceito (de bcrypt.MinCost ate 15) usa o padrao.
func NewHasherWithCost(logger *backend.Logger, cost int) *Hasher {
	if logger == nil {
		logger = backend.NewLogger()
	}

	if cost < bcrypt.MinCost || cost > maxCost {
		cost = DefaultCost
	}

	return &Hasher{logger: logger, cost: cost}
}

// Hash gera o hash bcrypt de uma senha, com um salt aleatorio.
// A senha e o hash nunca vao para os logs.
func (h *Hasher) Hash(password string) (string, error) {
	h.logger.Info("Generating password hash...")

	if password == "" {
		h.logger.Error("Password must not be empty")

		return "", ErrEmptyPassword
	}

	if len(password) > maxPasswordLength {
		h.logger.Error("Password is too long")

		return "", ErrPasswordTooLong
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)

	if err != nil {
		h.logger.Error("Could not generate hash: %v", err)

		return "", fmt.Errorf("generate hash: %w", err)
	}

	h.logger.Success("Password hash generated with success")

	return string(hashed), nil
}

// Verify confere se a senha corresponde ao hash.
// Senha errada devolve (false, nil); hash em formato invalido devolve ErrInvalidHash.
func (h *Hasher) Verify(password string, encoded string) (bool, error) {
	h.logger.Info("Verifying password...")

	cost, err := bcrypt.Cost([]byte(encoded))

	if err != nil || cost > maxCost {
		h.logger.Error("Invalid password hash format")

		return false, ErrInvalidHash
	}

	// o bcrypt ignora o que passa de 72 bytes; como o Hash nunca aceita uma senha assim,
	// uma senha maior que isso nao pode corresponder a nenhum hash gerado aqui
	if len(password) > maxPasswordLength {
		h.logger.Warning("Password does not match")

		return false, nil
	}

	err = bcrypt.CompareHashAndPassword([]byte(encoded), []byte(password))

	switch {
	case err == nil:
		h.logger.Success("Password verified with success")

		return true, nil

	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		h.logger.Warning("Password does not match")

		return false, nil

	default:
		h.logger.Error("Invalid password hash format")

		return false, ErrInvalidHash
	}
}
