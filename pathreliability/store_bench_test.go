package pathreliability

import (
	"testing"
	"time"
)

func BenchmarkStoreUpdate(b *testing.B) {
	cfg := DefaultConfig()
	store := NewStore(cfg, []string{"20m"})
	now := time.Now().UTC()
	power := dbToPower(-5)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Update(1, 2, 3, 4, "20m", power, 1.0, now)
	}
}

func BenchmarkStoreRefreshStatsSnapshot(b *testing.B) {
	cfg := DefaultConfig()
	cfg.StaleAfterSeconds = 3600
	store := NewStore(cfg, []string{"20m"})
	now := time.Now().UTC()
	power := dbToPower(-5)

	for i := 0; i < 20000; i++ {
		receiver := CellID((i % 30000) + 1)
		sender := CellID(((i * 7) % 30000) + 1)
		store.Update(receiver, sender, InvalidCell, InvalidCell, "20m", power, 1.0, now)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.RefreshStatsSnapshot(now)
	}
}
