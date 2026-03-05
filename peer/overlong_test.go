package peer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRotateOverlongLogRollsBackups(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "peering_overlong.log")
	if err := os.WriteFile(path, []byte("current"), 0o644); err != nil {
		t.Fatalf("write current: %v", err)
	}
	if err := os.WriteFile(path+".1", []byte("older"), 0o644); err != nil {
		t.Fatalf("write backup: %v", err)
	}

	rotateOverlongLog(path, 1, 2)

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected active log to be rotated away, stat err=%v", err)
	}
	got1, err := os.ReadFile(path + ".1")
	if err != nil {
		t.Fatalf("read .1: %v", err)
	}
	if string(got1) != "current" {
		t.Fatalf("expected .1 to contain current data, got %q", string(got1))
	}
	got2, err := os.ReadFile(path + ".2")
	if err != nil {
		t.Fatalf("read .2: %v", err)
	}
	if string(got2) != "older" {
		t.Fatalf("expected .2 to contain previous .1 data, got %q", string(got2))
	}
}
