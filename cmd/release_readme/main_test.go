package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRendersTelnetPortFromCheckedInConfig(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "README.md.template")
	outputPath := filepath.Join(dir, "README.md")

	template := "connect with:\n\ntelnet localhost {{ .TelnetPort }}\n"
	if err := os.WriteFile(templatePath, []byte(template), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	if err := run(templatePath, filepath.Join("..", "..", "data", "config"), outputPath); err != nil {
		t.Fatalf("run() error: %v", err)
	}

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.Contains(string(rendered), "telnet localhost 8300") {
		t.Fatalf("rendered README did not use checked-in telnet.port:\n%s", rendered)
	}
}

func TestRunRejectsUnresolvedTemplateMarker(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "README.md.template")
	outputPath := filepath.Join(dir, "README.md")

	template := "telnet localhost {{ .TelnetPort }}\n{{ \"{{UNRESOLVED}}\" }}\n"
	if err := os.WriteFile(templatePath, []byte(template), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	err := run(templatePath, filepath.Join("..", "..", "data", "config"), outputPath)
	if err == nil || !strings.Contains(err.Error(), "unresolved template marker") {
		t.Fatalf("expected unresolved marker error, got %v", err)
	}
}
