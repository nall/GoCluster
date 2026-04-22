package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func testConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	srcDir := filepath.Join("..", "data", "config")
	for base, spec := range configFileRegistry {
		if !spec.required {
			continue
		}
		src := filepath.Join(srcDir, base)
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read test config fixture %s: %v", src, err)
		}
		if err := os.WriteFile(filepath.Join(dir, base), data, 0o644); err != nil {
			t.Fatalf("write test config fixture %s: %v", base, err)
		}
	}
	return dir
}

func writeTestConfigOverlay(t *testing.T, dir, base, body string) {
	t.Helper()
	base = strings.ToLower(base)
	if _, ok := configFileRegistry[base]; !ok {
		t.Fatalf("test attempted to write unregistered config file %s", base)
	}
	path := filepath.Join(dir, base)
	merged := make(map[string]any)
	if existing, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(existing))) > 0 {
		if err := yaml.Unmarshal(existing, &merged); err != nil {
			t.Fatalf("parse existing test config %s: %v", base, err)
		}
	}
	var override map[string]any
	if err := yaml.Unmarshal([]byte(body), &override); err != nil {
		t.Fatalf("parse override test config %s: %v", base, err)
	}
	merged = mergeYAMLMaps(merged, override)
	data, err := yaml.Marshal(merged)
	if err != nil {
		t.Fatalf("marshal override test config %s: %v", base, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write override test config %s: %v", base, err)
	}
}

func replaceTestConfigFile(t *testing.T, dir, base, body string) {
	t.Helper()
	base = strings.ToLower(base)
	if _, ok := configFileRegistry[base]; !ok {
		t.Fatalf("test attempted to replace unregistered config file %s", base)
	}
	if err := os.WriteFile(filepath.Join(dir, base), []byte(body), 0o644); err != nil {
		t.Fatalf("replace test config %s: %v", base, err)
	}
}

func removeTestConfigKey(t *testing.T, dir, base string, path ...string) {
	t.Helper()
	if len(path) == 0 {
		t.Fatalf("removeTestConfigKey called without key path")
	}
	base = strings.ToLower(base)
	file := filepath.Join(dir, base)
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read test config %s: %v", base, err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse test config %s: %v", base, err)
	}
	current := doc
	for _, key := range path[:len(path)-1] {
		next, ok := current[key].(map[string]any)
		if !ok {
			t.Fatalf("test config path %s missing before final key", strings.Join(path, "."))
		}
		current = next
	}
	delete(current, path[len(path)-1])
	out, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal test config %s: %v", base, err)
	}
	if err := os.WriteFile(file, out, 0o644); err != nil {
		t.Fatalf("write test config %s: %v", base, err)
	}
}
