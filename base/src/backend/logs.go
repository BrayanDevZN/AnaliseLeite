package backend

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level indica a gravidade de uma mensagem de log.
type Level string

const (
	LevelInfo    Level = "INFO"
	LevelSuccess Level = "SUCCESS"
	LevelWarning Level = "WARNING"
	LevelError   Level = "ERROR"
)

// cores ANSI de cada nivel, usadas so quando a saida e o terminal
var levelColors = map[Level]string{
	LevelInfo:    "\033[36m",
	LevelSuccess: "\033[32m",
	LevelWarning: "\033[33m",
	LevelError:   "\033[31m",
}

const colorReset = "\033[0m"

// Logger mostra os logs na saida escolhida. Nao grava em arquivo.
// Pode ser usado por varias goroutines ao mesmo tempo.
type Logger struct {
	mu    sync.Mutex
	out   io.Writer
	color bool
}

// NewLogger cria um Logger que escreve no terminal (stdout).
func NewLogger() *Logger {
	return NewLoggerWriter(os.Stdout)
}

// NewLoggerWriter cria um Logger que escreve em out.
// As cores so sao usadas quando out e o terminal e a variavel NO_COLOR nao esta definida.
func NewLoggerWriter(out io.Writer) *Logger {
	_, noColor := os.LookupEnv("NO_COLOR")

	return &Logger{out: out, color: !noColor && isTerminal(out)}
}

// isTerminal diz se out e o terminal (e nao um arquivo, pipe ou buffer).
func isTerminal(out io.Writer) bool {
	file, ok := out.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()

	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// log monta e escreve uma linha: [NIVEL] => data/hora | mensagem
func (l *Logger) log(level Level, message string) {
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000-07:00")

	tag := string(level)
	if l.color {
		tag = levelColors[level] + tag + colorReset
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	fmt.Fprintf(l.out, "[%s] => %s | %s\n", tag, now, message)
}

// Info registra uma mensagem informativa.
func (l *Logger) Info(format string, args ...any) {
	l.log(LevelInfo, fmt.Sprintf(format, args...))
}

// Success registra que uma etapa terminou com sucesso.
func (l *Logger) Success(format string, args ...any) {
	l.log(LevelSuccess, fmt.Sprintf(format, args...))
}

// Warning registra um aviso.
func (l *Logger) Warning(format string, args ...any) {
	l.log(LevelWarning, fmt.Sprintf(format, args...))
}

// Error registra um erro.
func (l *Logger) Error(format string, args ...any) {
	l.log(LevelError, fmt.Sprintf(format, args...))
}
