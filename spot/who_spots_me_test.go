package spot

import (
	"reflect"
	"testing"
	"time"
)

func TestWhoSpotsMeStoreGroupsByContinentAndExpires(t *testing.T) {
	store := NewWhoSpotsMeStoreWithOptions(WhoSpotsMeOptions{
		Window:               3 * time.Second,
		Shards:               1,
		MaxEntries:           16,
		MaxCountriesPerEntry: 8,
		CleanupInterval:      time.Hour,
	})
	base := time.Unix(100, 0).UTC()

	store.Record("w1aw", "20M", 291, "NA", base)
	store.Record("W1AW", "20m", 291, "NA", base.Add(time.Second))
	store.Record("W1AW", "20m", 230, "EU", base.Add(2*time.Second))

	got := store.CountryCountsByContinent("W1AW", "20m", base.Add(2*time.Second))
	want := map[string][]WhoSpotsMeCountryCount{
		"EU": {{ADIF: 230, Count: 1}},
		"NA": {{ADIF: 291, Count: 2}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("counts before expiry mismatch: got %#v want %#v", got, want)
	}

	got = store.CountryCountsByContinent("W1AW", "20m", base.Add(3*time.Second))
	want = map[string][]WhoSpotsMeCountryCount{
		"EU": {{ADIF: 230, Count: 1}},
		"NA": {{ADIF: 291, Count: 1}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("counts at expiry boundary mismatch: got %#v want %#v", got, want)
	}

	got = store.CountryCountsByContinent("W1AW", "20m", base.Add(5*time.Second))
	if got != nil {
		t.Fatalf("expected all counts to expire, got %#v", got)
	}
}

func TestWhoSpotsMeStoreSkipsInvalidCountryMetadata(t *testing.T) {
	store := NewWhoSpotsMeStoreWithOptions(WhoSpotsMeOptions{
		Window:               10 * time.Second,
		Shards:               1,
		MaxEntries:           8,
		MaxCountriesPerEntry: 4,
		CleanupInterval:      time.Hour,
	})
	now := time.Unix(200, 0).UTC()

	store.Record("W1AW", "20m", 0, "NA", now)
	store.Record("W1AW", "20m", 291, "", now)
	store.Record("W1AW", "20m", 291, "xx", now)

	if got := store.CountryCountsByContinent("W1AW", "20m", now); got != nil {
		t.Fatalf("expected invalid country metadata to be skipped, got %#v", got)
	}
}

func TestWhoSpotsMeStoreEvictsOldestKeyAtCapacity(t *testing.T) {
	store := NewWhoSpotsMeStoreWithOptions(WhoSpotsMeOptions{
		Window:               10 * time.Second,
		Shards:               1,
		MaxEntries:           1,
		MaxCountriesPerEntry: 4,
		CleanupInterval:      time.Hour,
	})
	base := time.Unix(300, 0).UTC()

	store.Record("W1AW", "20m", 291, "NA", base)
	store.Record("K1JT", "20m", 230, "EU", base.Add(time.Second))

	if got := store.CountryCountsByContinent("W1AW", "20m", base.Add(time.Second)); got != nil {
		t.Fatalf("expected oldest key eviction, got %#v", got)
	}
	if got := store.CountryCountsByContinent("K1JT", "20m", base.Add(time.Second)); !reflect.DeepEqual(got, map[string][]WhoSpotsMeCountryCount{
		"EU": {{ADIF: 230, Count: 1}},
	}) {
		t.Fatalf("expected surviving key counts, got %#v", got)
	}
}

func TestWhoSpotsMeStoreSkipsNewCountriesPastPerKeyCap(t *testing.T) {
	store := NewWhoSpotsMeStoreWithOptions(WhoSpotsMeOptions{
		Window:               10 * time.Second,
		Shards:               1,
		MaxEntries:           8,
		MaxCountriesPerEntry: 1,
		CleanupInterval:      time.Hour,
	})
	now := time.Unix(400, 0).UTC()

	store.Record("W1AW", "20m", 291, "NA", now)
	store.Record("W1AW", "20m", 230, "EU", now.Add(time.Second))

	got := store.CountryCountsByContinent("W1AW", "20m", now.Add(time.Second))
	want := map[string][]WhoSpotsMeCountryCount{
		"NA": {{ADIF: 291, Count: 1}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected capped country set, got %#v want %#v", got, want)
	}
}
