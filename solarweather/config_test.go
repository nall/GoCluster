package solarweather

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "solarweather.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func writeConfigWithoutKey(t *testing.T, path ...string) string {
	t.Helper()
	base, err := os.ReadFile(filepath.Join("..", "data", "config", "solarweather.yaml"))
	if err != nil {
		t.Fatalf("read shipped solarweather config: %v", err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(base, &doc); err != nil {
		t.Fatalf("parse shipped solarweather config: %v", err)
	}
	removeYAMLKey(t, doc, path...)
	data, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal solarweather config: %v", err)
	}
	return writeTempConfig(t, string(data))
}

func writeShippedConfigCopy(t *testing.T) string {
	t.Helper()
	base, err := os.ReadFile(filepath.Join("..", "data", "config", "solarweather.yaml"))
	if err != nil {
		t.Fatalf("read shipped solarweather config: %v", err)
	}
	return writeTempConfig(t, string(base))
}

func removeYAMLKey(t *testing.T, doc map[string]any, path ...string) {
	t.Helper()
	if len(path) == 0 {
		t.Fatalf("empty key path")
	}
	var current any = doc
	for _, key := range path[:len(path)-1] {
		switch node := current.(type) {
		case map[string]any:
			current = node[key]
		case []any:
			index, err := strconv.Atoi(key)
			if err != nil || index < 0 || index >= len(node) {
				t.Fatalf("invalid test sequence index %q in %s", key, strings.Join(path, "."))
			}
			current = node[index]
		default:
			t.Fatalf("test path %s missing before final key", strings.Join(path, "."))
		}
	}
	last := path[len(path)-1]
	switch node := current.(type) {
	case map[string]any:
		delete(node, last)
	default:
		t.Fatalf("test path %s does not end in a mapping", strings.Join(path, "."))
	}
}

func TestLoadFileRejectsMissingRequiredYAMLSettings(t *testing.T) {
	cases := []struct {
		name string
		path []string
		want string
	}{
		{name: "enabled", path: []string{"enabled"}, want: "enabled"},
		{name: "terminator hold", path: []string{"sun", "near_terminator_hold"}, want: "sun.near_terminator_hold"},
		{name: "kp boundary", path: []string{"high_lat", "use_kp_boundary"}, want: "high_lat.use_kp_boundary"},
		{name: "path key band", path: []string{"path_key_include_band"}, want: "path_key_include_band"},
		{name: "r level name", path: []string{"r_levels", "0", "name"}, want: "r_levels[0].name"},
		{name: "g level name", path: []string{"g_levels", "0", "name"}, want: "g_levels[0].name"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadFile(writeConfigWithoutKey(t, tc.path...))
			if err == nil {
				t.Fatalf("expected missing %s to fail", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error to mention %s, got %v", tc.want, err)
			}
		})
	}
}

func TestLoadFileRejectsNullRequiredYAMLSetting(t *testing.T) {
	path := writeShippedConfigCopy(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	text := strings.Replace(string(data), "path_key_include_band: true", "path_key_include_band:", 1)
	_, err = LoadFile(writeTempConfig(t, text))
	if err == nil {
		t.Fatalf("expected null path_key_include_band to fail")
	}
	if !strings.Contains(err.Error(), "path_key_include_band") {
		t.Fatalf("expected error to mention path_key_include_band, got %v", err)
	}
}

func TestLoadFileRejectsEmptyLevelNames(t *testing.T) {
	path := writeShippedConfigCopy(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	text := strings.Replace(string(data), "name: R3", "name: \"\"", 1)
	_, err = LoadFile(writeTempConfig(t, text))
	if err == nil {
		t.Fatalf("expected empty r level name to fail")
	}
	if !strings.Contains(err.Error(), "r_levels[0] name") {
		t.Fatalf("expected r level name error, got %v", err)
	}
}
