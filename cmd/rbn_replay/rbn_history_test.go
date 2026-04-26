package main

import (
	"encoding/csv"
	"strings"
	"testing"

	rbnfeed "dxcluster/rbn"
)

func TestRBNHistoryCSVSeparatesSpotClassFromTXMode(t *testing.T) {
	parser := newTestRBNHistoryCSV(t, strings.Join([]string{
		"callsign,freq,band,dx,db,date,mode,tx_mode",
		"K1ABC,3547.0,80m,OZ4ADX,5,2026-04-26 03:18:00,CQ,CW",
	}, "\n"))

	row, ok, err := parser.Read()
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if !ok {
		t.Fatalf("expected row to parse")
	}
	if row.Mode != "CW" {
		t.Fatalf("expected tx_mode CW to populate row.Mode, got %q", row.Mode)
	}
	if row.SpotClass != rbnfeed.SpotClassCQ {
		t.Fatalf("expected spot class CQ, got %q", row.SpotClass)
	}
}

func TestRBNHistoryCSVReadsDXSpotClassAsSkippedClass(t *testing.T) {
	parser := newTestRBNHistoryCSV(t, strings.Join([]string{
		"callsign,freq,band,dx,db,date,mode,tx_mode",
		"K1ABC,3547.0,80m,OZ4ADX,5,2026-04-26 03:18:00,DX,CW",
	}, "\n"))

	row, ok, err := parser.Read()
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if !ok {
		t.Fatalf("expected valid row with skipped class to parse")
	}
	if row.SpotClass.Accepted() {
		t.Fatalf("expected DX spot class not to be accepted")
	}
}

func TestNewReplaySpotTagsBeaconSpotClasses(t *testing.T) {
	row := rbnHistoryRow{
		DXCall:    "4U1UN",
		Spotter:   "K1ABC",
		FreqKHz:   14100.0,
		Mode:      "CW",
		SpotClass: rbnfeed.SpotClassNCDXFB,
		ReportDB:  5,
	}

	spotEntry := newReplaySpot(row)
	if !spotEntry.IsBeacon {
		t.Fatalf("expected NCDXF B spot class to set IsBeacon")
	}
	if spotEntry.Mode != "CW" {
		t.Fatalf("expected Spot.Mode to remain RF mode CW, got %q", spotEntry.Mode)
	}
}

func newTestRBNHistoryCSV(t *testing.T, data string) *rbnHistoryCSV {
	t.Helper()
	reader := csv.NewReader(strings.NewReader(data))
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = true
	header, err := reader.Read()
	if err != nil {
		t.Fatalf("read header: %v", err)
	}
	parser := &rbnHistoryCSV{reader: reader, header: header}
	parser.initColumns()
	if err := parser.requiredColumnsPresent(); err != nil {
		t.Fatalf("required columns: %v", err)
	}
	return parser
}
