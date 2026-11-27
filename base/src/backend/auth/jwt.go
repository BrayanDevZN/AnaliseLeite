package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
)

const (
	// SecretName e o nome da variavel de ambiente com a chave que assina os tokens.
	SecretName = "JWT_SECRET"

	// uma chave de HS256 com menos que isso e facil de adivinhar
	minSecretLength = 32

	jwtAlgorithm = "HS256"
	jwtType      = "JWT"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
	ErrEmptySubject = errors.New("subject must not be empty")
	ErrInvalidTTL   = errors.New("ttl must be greater than zero")
	ErrWeakSecret   = fmt.Errorf("%s must have at least %d characters", SecretName, minSecretLength)
)

// Claims sao os dados que um token carrega.
type Claims struct {
	Subject   string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

type header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type payload struct {
	Subject   string `json:"sub"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

// JWT cria e le tokens JWT assinados com HS256.
type JWT struct {
	logger *backend.Logger
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewJWT le a chave de JWT_SECRET pelo Config, como secret obrigatoria.
// Se a variavel nao existir, devolve um *backend.MissingSecretsError.
// ttl e por quanto tempo cada token vale. Se logger for nil, os logs vao para o terminal.
func NewJWT(logger *backend.Logger, ttl time.Duration) (*JWT, error) {
	if logger == nil {
		logger = backend.NewLogger()
	}

	logger.Info("Loading JWT secret...")

	config, err := backend.NewConfig(nil, []string{SecretName})

	if err != nil {
		logger.Error("%v", err)

		return nil, err
	}

	secret := *config.GetSecrets()[SecretName]

	if len(secret) < minSecretLength {
		logger.Error("%v", ErrWeakSecret)

		return nil, ErrWeakSecret
	}

	if ttl <= 0 {
		logger.Error("%v", ErrInvalidTTL)

		return nil, ErrInvalidTTL
	}

	logger.Success("JWT secret loaded with success")

	return &JWT{
		logger: logger,
		secret: []byte(secret),
		ttl:    ttl,
		now:    func() time.Time { return time.Now().UTC() },
	}, nil
}

// TTL devolve o tempo de vida dos tokens criados por este JWT.
func (j *JWT) TTL() time.Duration {
	return j.ttl
}

// SetClock troca o relogio usado para emitir e validar tokens (util em testes).
func (j *JWT) SetClock(now func() time.Time) {
	j.now = now
}

// Create gera um token para o subject (por exemplo, o id do usuario).
// O token e a chave nunca vao para os logs.
func (j *JWT) Create(subject string) (string, error) {
	j.logger.Info("Generating token...")

	if subject == "" {
		j.logger.Error("%v", ErrEmptySubject)

		return "", ErrEmptySubject
	}

	issuedAt := j.now()

	headerPart, err := encodePart(header{Alg: jwtAlgorithm, Typ: jwtType})

	if err != nil {
		j.logger.Error("Could not generate token: %v", err)

		return "", fmt.Errorf("encode header: %w", err)
	}

	payloadPart, err := encodePart(payload{
		Subject:   subject,
		IssuedAt:  issuedAt.Unix(),
		ExpiresAt: issuedAt.Add(j.ttl).Unix(),
	})

	if err != nil {
		j.logger.Error("Could not generate token: %v", err)

		return "", fmt.Errorf("encode payload: %w", err)
	}

	signingInput := headerPart + "." + payloadPart
	token := signingInput + "." + j.sign(signingInput)

	j.logger.Success("Token generated with success")

	return token, nil
}

// Read confere a assinatura e a validade de um token e devolve os dados dele.
// Devolve ErrInvalidToken para qualquer token adulterado ou mal formado e ErrExpiredToken para um token vencido.
func (j *JWT) Read(token string) (*Claims, error) {
	j.logger.Info("Reading token...")

	parts := strings.Split(token, ".")

	if len(parts) != 3 {
		return nil, j.reject(ErrInvalidToken)
	}

	// a assinatura e conferida antes de ler qualquer dado do token
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])

	if err != nil || !hmac.Equal(signature, j.signBytes(parts[0]+"."+parts[1])) {
		return nil, j.reject(ErrInvalidToken)
	}

	var head header

	if err := decodePart(parts[0], &head); err != nil || head.Alg != jwtAlgorithm || head.Typ != jwtType {
		return nil, j.reject(ErrInvalidToken)
	}

	var data payload

	if err := decodePart(parts[1], &data); err != nil || data.Subject == "" {
		return nil, j.reject(ErrInvalidToken)
	}

	now := j.now()

	if now.Unix() >= data.ExpiresAt {
		return nil, j.reject(ErrExpiredToken)
	}

	j.logger.Success("Token read with success")

	return &Claims{
		Subject:   data.Subject,
		IssuedAt:  time.Unix(data.IssuedAt, 0).UTC(),
		ExpiresAt: time.Unix(data.ExpiresAt, 0).UTC(),
	}, nil
}

// reject registra no log que o token foi recusado e devolve o erro.
func (j *JWT) reject(err error) error {
	j.logger.Warning("Token rejected: %v", err)

	return err
}

func (j *JWT) signBytes(input string) []byte {
	mac := hmac.New(sha256.New, j.secret)
	mac.Write([]byte(input))

	return mac.Sum(nil)
}

func (j *JWT) sign(input string) string {
	return base64.RawURLEncoding.EncodeToString(j.signBytes(input))
}

func encodePart(value any) (string, error) {
	data, err := json.Marshal(value)

	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(data), nil
}

func decodePart(part string, value any) error {
	data, err := base64.RawURLEncoding.DecodeString(part)

	if err != nil {
		return err
	}

	return json.Unmarshal(data, value)
}
