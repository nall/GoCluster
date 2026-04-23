package spot

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkWhoSpotsMeRecord(b *testing.B) {
	store := NewWhoSpotsMeStoreWithOptions(WhoSpotsMeOptions{
		Window:               10 * time.Minute,
		Shards:               64,
		MaxEntries:           32768,
		MaxCountriesPerEntry: 256,
		CleanupInterval:      time.Hour,
	})
	base := time.Unix(1_700_000_000, 0).UTC()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		call := fmt.Sprintf("K1%04d", i%2048)
		band := "20m"
		adif := 200 + (i % 32)
		continent := "EU"
		if i%3 == 0 {
			continent = "NA"
		}
		seenAt := base.Add(time.Duration(i) * time.Second)
		store.Record(call, band, adif, continent, seenAt)
	}
}

func BenchmarkWhoSpotsMeQuery(b *testing.B) {
	store := NewWhoSpotsMeStoreWithOptions(WhoSpotsMeOptions{
		Window:               10 * time.Minute,
		Shards:               64,
		MaxEntries:           32768,
		MaxCountriesPerEntry: 256,
		CleanupInterval:      time.Hour,
	})
	base := time.Unix(1_700_000_000, 0).UTC()
	for i := 0; i < 10_000; i++ {
		call := fmt.Sprintf("K1%04d", i%1024)
		band := "20m"
		adif := 200 + (i % 24)
		continent := "EU"
		if i%4 == 0 {
			continent = "NA"
		}
		store.Record(call, band, adif, continent, base.Add(time.Duration(i)*time.Second))
	}

	targetCall := "K10001"
	queryAt := base.Add(10_000 * time.Second)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.CountryCountsByContinent(targetCall, "20m", queryAt)
	}
}
