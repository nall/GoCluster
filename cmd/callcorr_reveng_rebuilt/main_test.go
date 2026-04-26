package main

import (
	"testing"

	rbnfeed "dxcluster/rbn"
)

func TestParseRBNRecordSeparatesSpotClassFromTXMode(t *testing.T) {
	header := mustParseTestRBNHeader(t)
	record := []string{"K1ABC", "3547.0", "80m", "OZ4ADX", "5", "2026-04-26 03:18:00", "CQ", "CW"}

	row, status := parseRBNRecord(record, header)
	if status != rbnRecordAccepted {
		t.Fatalf("expected accepted row, got status %d", status)
	}
	if row.Mode != "CW" {
		t.Fatalf("expected tx_mode CW to populate row.Mode, got %q", row.Mode)
	}
	if row.SpotClass != rbnfeed.SpotClassCQ {
		t.Fatalf("expected spot class CQ, got %q", row.SpotClass)
	}
}

func TestParseRBNRecordSkipsDXSpotClass(t *testing.T) {
	header := mustParseTestRBNHeader(t)
	record := []string{"K1ABC", "3547.0", "80m", "OZ4ADX", "5", "2026-04-26 03:18:00", "DX", "CW"}

	_, status := parseRBNRecord(record, header)
	if status != rbnRecordSkipped {
		t.Fatalf("expected DX spot class to be skipped, got status %d", status)
	}
}

func TestParseRBNRecordAcceptsBeaconSpotClasses(t *testing.T) {
	header := mustParseTestRBNHeader(t)
	for _, tc := range []struct {
		name string
		raw  string
		want rbnfeed.SpotClass
	}{
		{name: "beacon", raw: "BEACON", want: rbnfeed.SpotClassBeacon},
		{name: "ncdxf", raw: "NCDXF B", want: rbnfeed.SpotClassNCDXFB},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := []string{"K1ABC", "14100.0", "20m", "4U1UN", "5", "2026-04-26 03:18:00", tc.raw, "CW"}

			row, status := parseRBNRecord(record, header)
			if status != rbnRecordAccepted {
				t.Fatalf("expected accepted row, got status %d", status)
			}
			if row.SpotClass != tc.want {
				t.Fatalf("expected spot class %q, got %q", tc.want, row.SpotClass)
			}
			if !row.SpotClass.IsBeacon() {
				t.Fatalf("expected %q to be beacon-tagged", row.SpotClass)
			}
		})
	}
}

func mustParseTestRBNHeader(t *testing.T) rbnHeader {
	t.Helper()
	header, err := parseRBNHeader([]string{"callsign", "freq", "band", "dx", "db", "date", "mode", "tx_mode"})
	if err != nil {
		t.Fatalf("parseRBNHeader: %v", err)
	}
	return header
}
