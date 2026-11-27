package midlleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/database/redis"
)

const (
	// redisTimeout e o tempo maximo de cada contagem no Redis; se passar, usa a memoria.
	redisTimeout = 300 * time.Millisecond

	// redisCooldown e quanto tempo o Redis fica de lado depois de uma falha, para nao atrasar
	// todas as requisicoes enquanto ele esta fora do ar.
	redisCooldown = 5 * time.Second

	defaultPrefix = "ratelimit"
)

// countScript soma 1 na chave e, se ela e nova (ou ficou sem expiracao), define a janela. Tudo de uma vez,
// no Redis, entao varias instancias do backend contam juntas e sem race condition.
// Devolve {contagem, milissegundos que faltam para zerar}.
var countScript = goredis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
	redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
local ttl = redis.call('PTTL', KEYS[1])
if ttl < 0 then
	redis.call('PEXPIRE', KEYS[1], ARGV[1])
	ttl = tonumber(ARGV[1])
end
return {count, ttl}
`)

var ErrInvalidRule = errors.New("invalid rate limit rule: limit and window must be both greater than zero")

// Rule e um limite: no maximo Limit requisicoes a cada Window. Limit 0 desliga a regra.
type Rule struct {
	Limit  int
	Window time.Duration
}

func (r Rule) enabled() bool {
	return r.Limit > 0 && r.Window > 0
}

func (r Rule) valid() bool {
	return r.Limit == 0 && r.Window == 0 || r.enabled()
}

// Config e a configuracao do rate limit.
type Config struct {
	// User e o limite de cada usuario logado (a contagem e por id do usuario).
	User Rule

	// Public e o limite das rotas publicas, sem login (a contagem e por IP).
	// Se for deixado vazio, usa o mesmo limite de User.
	Public Rule

	// Global e o limite de todas as requisicoes juntas, de todo mundo. Vazio desliga.
	Global Rule

	// Identify devolve o id do usuario logado da requisicao, ou "" se for uma rota publica.
	// Se for nil, todas as requisicoes sao tratadas como publicas (contadas por IP).
	// O middleware de login precisa rodar antes deste para o id estar disponivel.
	Identify func(r *http.Request) string

	// TrustProxy faz o IP vir do header X-Forwarded-For. Ligue so se o backend estiver atras
	// de um proxy confiavel; senao qualquer um pode fingir ser outro IP.
	TrustProxy bool

	// Prefix e o comeco das chaves no Redis (padrao "ratelimit").
	Prefix string
}

// Limiter e o middleware de rate limit.
// Conta primeiro no Redis (varias instancias contam juntas). Se o Redis nao existir ou falhar,
// conta na memoria (Counter), so dessa instancia. Ao trocar de um para o outro a contagem recomeca.
type Limiter struct {
	logger  *backend.Logger
	config  Config
	counter *Counter
	redis   *redis.Connection

	// redisDown e redisUntil guardam se o Redis falhou e ate quando ele fica de lado.
	redisDown  atomic.Bool
	redisUntil atomic.Int64
}

// New cria o rate limit. Passe redisManage nil (ou sem conexao) para contar so na memoria.
// Se logger for nil, os logs vao para o terminal.
func New(logger *backend.Logger, redisManage *redis.Manage, config Config) (*Limiter, error) {
	if logger == nil {
		logger = backend.NewLogger()
	}

	for _, rule := range []Rule{config.User, config.Public, config.Global} {
		if !rule.valid() {
			return nil, ErrInvalidRule
		}
	}

	if !config.Public.enabled() {
		config.Public = config.User
	}

	if config.Prefix == "" {
		config.Prefix = defaultPrefix
	}

	limiter := &Limiter{logger: logger, config: config, counter: NewCounter()}

	if redisManage != nil {
		limiter.redis = redisManage.Connection
	}

	if limiter.redis == nil {
		logger.Warning("Redis is not available, rate limit will count in memory")
	}

	return limiter, nil
}

// Counter devolve o contador da memoria (a reserva do Redis).
func (l *Limiter) Counter() *Counter {
	return l.counter
}

// Handler e o middleware: conta a requisicao e responde 429 se passou do limite.
// Primeiro conta no limite global; depois no do usuario (por id) ou no da rota publica (por IP).
func (l *Limiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if l.config.Global.enabled() {
			result := l.hit(r.Context(), l.key("global"), l.config.Global)

			if !result.Allowed {
				l.deny(w, "global", result)

				return
			}
		}

		scope, rule, id := "public", l.config.Public, clientIP(r, l.config.TrustProxy)

		if l.config.Identify != nil {
			if user := l.config.Identify(r); user != "" {
				scope, rule, id = "user", l.config.User, user
			}
		}

		if rule.enabled() {
			result := l.hit(r.Context(), l.key(scope+":"+id), rule)

			setHeaders(w, result)

			if !result.Allowed {
				l.deny(w, scope, result)

				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// HandlerFunc e o Handler para quem usa http.HandlerFunc.
func (l *Limiter) HandlerFunc(next http.HandlerFunc) http.Handler {
	return l.Handler(next)
}

func (l *Limiter) key(name string) string {
	return l.config.Prefix + ":" + name
}

// hit conta no Redis e, se ele nao estiver disponivel ou falhar, na memoria.
func (l *Limiter) hit(ctx context.Context, key string, rule Rule) Result {
	if l.redisReady() {
		result, err := l.hitRedis(ctx, key, rule)

		if err == nil {
			l.redisRecovered()

			return result
		}

		l.redisFailed(err)
	}

	return l.counter.Hit(key, rule.Limit, rule.Window)
}

// redisReady diz se vale tentar o Redis agora: ele existe, ja terminou de conectar e nao esta de castigo.
func (l *Limiter) redisReady() bool {
	if l.redis == nil {
		return false
	}

	select {
	case <-l.redis.Done():
	default:
		return false
	}

	return time.Now().UnixNano() >= l.redisUntil.Load()
}

func (l *Limiter) hitRedis(ctx context.Context, key string, rule Rule) (Result, error) {
	client := l.redis.Client()

	if client == nil {
		return Result{}, errors.New("redis connection failed")
	}

	ctx, cancel := context.WithTimeout(ctx, redisTimeout)

	defer cancel()

	values, err := countScript.Run(ctx, client, []string{key}, max(rule.Window.Milliseconds(), 1)).Slice()

	if err != nil {
		return Result{}, err
	}

	if len(values) != 2 {
		return Result{}, fmt.Errorf("unexpected redis reply: %v", values)
	}

	count, okCount := values[0].(int64)
	ttl, okTTL := values[1].(int64)

	if !okCount || !okTTL {
		return Result{}, fmt.Errorf("unexpected redis reply: %v", values)
	}

	return Result{
		Allowed:   count <= int64(rule.Limit),
		Count:     int(count),
		Limit:     rule.Limit,
		Remaining: max(rule.Limit-int(count), 0),
		ResetIn:   time.Duration(ttl) * time.Millisecond,
	}, nil
}

// redisFailed poe o Redis de castigo e avisa (so uma vez, para nao encher o log).
func (l *Limiter) redisFailed(err error) {
	l.redisUntil.Store(time.Now().Add(redisCooldown).UnixNano())

	if l.redisDown.CompareAndSwap(false, true) {
		l.logger.Warning("Redis failed for rate limit, counting in memory: %v", err)
	}
}

// redisRecovered avisa que o Redis voltou (so uma vez).
func (l *Limiter) redisRecovered() {
	if l.redisDown.CompareAndSwap(true, false) {
		l.logger.Success("Redis is back for rate limit")
	}
}

// deny responde 429. O log mostra so o tipo do limite, nunca o IP nem o usuario.
func (l *Limiter) deny(w http.ResponseWriter, scope string, result Result) {
	l.logger.Warning("Rate limit exceeded (%s)", scope)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", strconv.Itoa(seconds(result.ResetIn)))
	w.WriteHeader(http.StatusTooManyRequests)

	json.NewEncoder(w).Encode(map[string]string{"detail": "rate limit exceeded"})
}

// setHeaders poe os headers X-RateLimit-* da regra por usuario ou IP.
func setHeaders(w http.ResponseWriter, result Result) {
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.Itoa(seconds(result.ResetIn)))
}

// seconds arredonda para cima (e nunca devolve menos que 1).
func seconds(duration time.Duration) int {
	return max(int(math.Ceil(duration.Seconds())), 1)
}

// clientIP acha o IP de quem fez a requisicao.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		forwarded, _, _ := strings.Cut(r.Header.Get("X-Forwarded-For"), ",")

		if ip := net.ParseIP(strings.TrimSpace(forwarded)); ip != nil {
			return ip.String()
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)

	if err != nil {
		return r.RemoteAddr
	}

	return host
}
