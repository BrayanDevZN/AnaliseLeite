// Package midlleware tem o middleware de rate limit (por usuario, por IP e global).
package midlleware

import (
	"sync"
	"time"
)

const (
	// sweepInterval e de quanto em quanto tempo as contagens vencidas sao apagadas da memoria.
	sweepInterval = time.Minute
)

// Result e o resultado de contar uma requisicao.
type Result struct {
	// Allowed diz se a requisicao esta dentro do limite.
	Allowed bool

	// Count e quantas requisicoes ja foram contadas na janela (contando esta).
	Count int

	// Limit e o limite da janela.
	Limit int

	// Remaining e quantas requisicoes ainda cabem na janela.
	Remaining int

	// ResetIn e quanto falta para a janela acabar e a contagem zerar.
	ResetIn time.Duration
}

type entry struct {
	count   int
	resetAt time.Time
}

// Counter conta requisicoes por chave na memoria RAM, em janelas fixas.
// E um dicionario (chave -> contagem) protegido por um mutex, entao e seguro usar de varias
// goroutines ao mesmo tempo. A contagem de cada chave comeca na primeira requisicao e zera
// quando a janela acaba. Serve de reserva quando o Redis nao esta disponivel.
type Counter struct {
	mu        sync.Mutex
	entries   map[string]*entry
	now       func() time.Time
	nextSweep time.Time
}

// NewCounter cria um contador vazio.
func NewCounter() *Counter {
	return &Counter{
		entries: make(map[string]*entry),
		now:     time.Now,
	}
}

// SetClock troca o relogio (util em testes).
func (c *Counter) SetClock(now func() time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.now = now
}

// Hit conta uma requisicao da chave e compara a contagem com o limite.
// Todas as requisicoes contam, inclusive as que passaram do limite.
func (c *Counter) Hit(key string, limit int, window time.Duration) Result {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()

	c.sweep(now)

	current, found := c.entries[key]

	if !found || !now.Before(current.resetAt) {
		current = &entry{resetAt: now.Add(window)}

		c.entries[key] = current
	}

	current.count++

	return Result{
		Allowed:   current.count <= limit,
		Count:     current.count,
		Limit:     limit,
		Remaining: max(limit-current.count, 0),
		ResetIn:   current.resetAt.Sub(now),
	}
}

// Reset apaga a contagem de uma chave.
func (c *Counter) Reset(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
}

// Len devolve quantas chaves estao sendo contadas agora.
func (c *Counter) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.entries)
}

// sweep apaga as contagens vencidas, no maximo uma vez por sweepInterval, para a memoria nao crescer.
// Quem chama ja segura o mutex.
func (c *Counter) sweep(now time.Time) {
	if now.Before(c.nextSweep) {
		return
	}

	c.nextSweep = now.Add(sweepInterval)

	for key, current := range c.entries {
		if !now.Before(current.resetAt) {
			delete(c.entries, key)
		}
	}
}
