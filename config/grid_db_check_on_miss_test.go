package config

import "testing"

func TestGridDBCheckOnMissDefaultsTrue(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.GridDBCheckOnMiss == nil {
		t.Fatalf("expected GridDBCheckOnMiss to be set")
	}
	if !*cfg.GridDBCheckOnMiss {
		t.Fatalf("expected GridDBCheckOnMiss=true by default")
	}
}

func TestGridDBCheckOnMissAllowsFalse(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	writeTestConfigOverlay(t, dir, "data.yaml", "grid_db_check_on_miss: false\n")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.GridDBCheckOnMiss == nil {
		t.Fatalf("expected GridDBCheckOnMiss to be set")
	}
	if *cfg.GridDBCheckOnMiss {
		t.Fatalf("expected GridDBCheckOnMiss=false when configured")
	}
}
