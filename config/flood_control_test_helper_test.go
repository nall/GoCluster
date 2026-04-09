package config

import (
	"os"
	"path/filepath"
	"testing"
)

const requiredFloodControlConfig = `flood_control:
  enabled: true
  log_interval_seconds: 30
  partition_mode: exact_source_type
  rails:
    decall:
      enabled: true
      action: observe
      window_seconds: 60
      max_entries_per_partition: 10000
      thresholds_per_source_type:
        MANUAL: 8
        UPSTREAM: 8
        PEER: 8
        RBN: 60
        FT8: 60
        FT4: 60
        PSKREPORTER: 60
    source_node:
      enabled: true
      action: observe
      window_seconds: 60
      max_entries_per_partition: 10000
      thresholds_per_source_type:
        MANUAL: 0
        UPSTREAM: 40
        PEER: 40
        RBN: 0
        FT8: 0
        FT4: 0
        PSKREPORTER: 0
    spotter_ip:
      enabled: true
      action: observe
      window_seconds: 60
      max_entries_per_partition: 10000
      thresholds_per_source_type:
        MANUAL: 0
        UPSTREAM: 0
        PEER: 20
        RBN: 0
        FT8: 0
        FT4: 0
        PSKREPORTER: 0
    dxcall:
      enabled: true
      action: observe
      window_seconds: 60
      max_entries_per_partition: 10000
      active_mode: conservative
      thresholds_by_mode:
        conservative:
          MANUAL: 25
          UPSTREAM: 25
          PEER: 25
          RBN: 80
          FT8: 80
          FT4: 80
          PSKREPORTER: 80
        moderate:
          MANUAL: 15
          UPSTREAM: 15
          PEER: 15
          RBN: 40
          FT8: 40
          FT4: 40
          PSKREPORTER: 40
        aggressive:
          MANUAL: 10
          UPSTREAM: 10
          PEER: 10
          RBN: 25
          FT8: 25
          FT4: 25
          PSKREPORTER: 25
`

func writeRequiredFloodControlFile(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "floodcontrol.yaml"), []byte(requiredFloodControlConfig), 0o644); err != nil {
		t.Fatalf("write floodcontrol.yaml: %v", err)
	}
}
