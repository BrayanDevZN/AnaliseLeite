package unit

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend"
)

// formato esperado de cada linha: [NIVEL] => 2026-01-02 15:04:05.000000+00:00 | mensagem
var lineFormat = regexp.MustCompile(`^\[(INFO|SUCCESS|WARNING|ERROR)\] => \d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{6}\+00:00 \| .*$`)

func newTestLogger() (*backend.Logger, *bytes.Buffer) {
	var buffer bytes.Buffer

	return backend.NewLoggerWriter(&buffer), &buffer
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

func TestLevelConstantsMatchTheTags(t *testing.T) {
	want := map[backend.Level]string{
		backend.LevelInfo:    "INFO",
		backend.LevelSuccess: "SUCCESS",
		backend.LevelWarning: "WARNING",
		backend.LevelError:   "ERROR",
	}

	for level, tag := range want {
		if string(level) != tag {
			t.Errorf("o nivel %q deveria valer %q", level, tag)
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

func TestMessageKeepsPercentSign(t *testing.T) {
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
	path := filepath.Join(t.TempDir(), "saida.log")

	file, err := os.Create(path)

	if err != nil {
		t.Fatal(err)
	}

	logger := backend.NewLoggerWriter(file)
	logger.Error("em arquivo")
	file.Close()

	content, err := os.ReadFile(path)

	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(content), "\033[") {
		t.Fatalf("um arquivo comum nao deveria receber cores: %q", content)
	}

	if !strings.Contains(string(content), "em arquivo") {
		t.Fatalf("a mensagem deveria ter sido escrita: %q", content)
	}
}

func TestNoColorWhenNoColorIsSet(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	logger, buffer := newTestLogger()

	logger.Warning("sem cor")

	if strings.Contains(buffer.String(), "\033[") {
		t.Fatalf("NO_COLOR deveria desligar as cores: %q", buffer.String())
	}
}

func TestNewLoggerWritesToStdout(t *testing.T) {
	reader, writer, err := os.Pipe()

	if err != nil {
		t.Fatal(err)
	}

	original := os.Stdout
	os.Stdout = writer

	defer func() { os.Stdout = original }()

	backend.NewLogger().Info("no terminal")

	writer.Close()

	output, err := io.ReadAll(reader)

	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(output), "no terminal") {
		t.Fatalf("NewLogger deveria escrever em stdout, veio %q", output)
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
