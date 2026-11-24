package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
)

// formato esperado de cada linha: [NIVEL] => 2026-01-02 15:04:05.000000+00:00 | mensagem
var lineFormat = regexp.MustCompile(`^\[(INFO|SUCCESS|WARNING|ERROR)\] => \d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{6}\+00:00 \| .*$`)

func newTestLogger() (*Logger, *bytes.Buffer) {
	var buffer bytes.Buffer

	return NewLoggerWriter(&buffer), &buffer
}

func TestEachLevelWritesItsTag(t *testing.T) {
	logger, buffer := newTestLogger()

	logger.Info("a")
	logger.Success("b")
	logger.Warning("c")
	logger.Error("d")

	lines := strings.Split(strings.TrimSuffix(buffer.String(), "\n"), "\n")

	if len(lines) != 4 {
		t.Fatalf("esperava 4 linhas, veio %d: %q", len(lines), lines)
	}

	for i, want := range []string{"[INFO] => ", "[SUCCESS] => ", "[WARNING] => ", "[ERROR] => "} {
		if !strings.HasPrefix(lines[i], want) {
			t.Errorf("linha %d: esperava comecar com %q, veio %q", i, want, lines[i])
		}
	}
}

func TestLineFormat(t *testing.T) {
	logger, buffer := newTestLogger()

	logger.Info("mensagem de teste")

	line := strings.TrimSuffix(buffer.String(), "\n")

	if !lineFormat.MatchString(line) {
		t.Fatalf("linha fora do formato esperado: %q", line)
	}

	if !strings.HasSuffix(line, "| mensagem de teste") {
		t.Fatalf("a mensagem deveria ser o final da linha: %q", line)
	}
}

func TestMessageFormatting(t *testing.T) {
	logger, buffer := newTestLogger()

	logger.Info("modelo %s carregado em %d ms", "anl_v3", 42)

	if !strings.Contains(buffer.String(), "modelo anl_v3 carregado em 42 ms") {
		t.Fatalf("mensagem formatada errada: %q", buffer.String())
	}
}

func TestMessageWithoutArgsKeepsPercent(t *testing.T) {
	logger, buffer := newTestLogger()

	logger.Info("100%% de sucesso")

	if !strings.Contains(buffer.String(), "100% de sucesso") {
		t.Fatalf("esperava o percentual na saida: %q", buffer.String())
	}
}

func TestEveryLineEndsWithNewline(t *testing.T) {
	logger, buffer := newTestLogger()

	logger.Info("a")
	logger.Error("b")

	if strings.Count(buffer.String(), "\n") != 2 || !strings.HasSuffix(buffer.String(), "\n") {
		t.Fatalf("cada log deveria ocupar uma linha: %q", buffer.String())
	}
}

func TestNoColorWhenOutputIsNotTerminal(t *testing.T) {
	logger, buffer := newTestLogger()

	logger.Error("sem cor")

	if strings.Contains(buffer.String(), "\033[") {
		t.Fatalf("um buffer nao deveria receber cores: %q", buffer.String())
	}
}

func TestNoColorWhenOutputIsARegularFile(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "saida.log"))

	if err != nil {
		t.Fatal(err)
	}

	defer file.Close()

	if NewLoggerWriter(file).color {
		t.Fatal("um arquivo comum nao deveria receber cores")
	}
}

func TestColorIsAppliedWhenEnabled(t *testing.T) {
	var buffer bytes.Buffer

	logger := &Logger{out: &buffer, color: true}

	logger.Info("a")
	logger.Success("b")
	logger.Warning("c")
	logger.Error("d")

	for level, color := range levelColors {
		want := color + string(level) + colorReset

		if !strings.Contains(buffer.String(), want) {
			t.Errorf("faltou a tag colorida de %s: %q", level, buffer.String())
		}
	}
}

func TestEachLevelHasItsOwnColor(t *testing.T) {
	seen := map[string]Level{}

	for level, color := range levelColors {
		if other, repeated := seen[color]; repeated {
			t.Errorf("%s e %s usam a mesma cor", level, other)
		}

		seen[color] = level
	}

	if len(levelColors) != 4 {
		t.Fatalf("esperava uma cor para cada um dos 4 niveis, veio %d", len(levelColors))
	}
}

func TestLoggerDefaultsToStdout(t *testing.T) {
	logger := NewLogger()

	if logger.out != os.Stdout {
		t.Fatal("NewLogger deveria escrever em stdout")
	}
}

func TestLoggerNeverWritesToFiles(t *testing.T) {
	dir := t.TempDir()

	t.Chdir(dir)

	logger, _ := newTestLogger()

	logger.Info("nada em disco")
	logger.Error("nada em disco")

	entries, err := os.ReadDir(dir)

	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 0 {
		t.Fatalf("o Logger nao deveria criar arquivos, criou %d", len(entries))
	}
}

func TestConcurrentLogsDoNotMixLines(t *testing.T) {
	logger, buffer := newTestLogger()

	const goroutines = 100

	var wait sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wait.Add(1)

		go func() {
			defer wait.Done()
			logger.Info("mensagem concorrente")
		}()
	}

	wait.Wait()

	lines := strings.Split(strings.TrimSuffix(buffer.String(), "\n"), "\n")

	if len(lines) != goroutines {
		t.Fatalf("esperava %d linhas, veio %d", goroutines, len(lines))
	}

	for _, line := range lines {
		if !lineFormat.MatchString(line) {
			t.Fatalf("linha misturada ou fora do formato: %q", line)
		}
	}
}
