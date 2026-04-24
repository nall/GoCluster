package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"dxcluster/config"
)

type templateData struct {
	TelnetPort int
}

func main() {
	templatePath := flag.String("template", "", "path to release README template")
	configDir := flag.String("config-dir", "", "path to staged config directory")
	outputPath := flag.String("out", "", "path for rendered README")
	flag.Parse()

	if err := run(*templatePath, *configDir, *outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "release_readme: %v\n", err)
		os.Exit(1)
	}
}

func run(templatePath, configDir, outputPath string) error {
	if strings.TrimSpace(templatePath) == "" {
		return fmt.Errorf("-template is required")
	}
	if strings.TrimSpace(configDir) == "" {
		return fmt.Errorf("-config-dir is required")
	}
	if strings.TrimSpace(outputPath) == "" {
		return fmt.Errorf("-out is required")
	}

	cfg, err := config.Load(configDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.Telnet.Port <= 0 {
		return fmt.Errorf("invalid telnet.port %d", cfg.Telnet.Port)
	}

	tmpl, err := template.New(filepath.Base(templatePath)).Option("missingkey=error").ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, templateData{TelnetPort: cfg.Telnet.Port}); err != nil {
		return fmt.Errorf("render template: %w", err)
	}
	if strings.Contains(rendered.String(), "{{") || strings.Contains(rendered.String(), "}}") {
		return fmt.Errorf("rendered README contains an unresolved template marker")
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(outputPath, rendered.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
