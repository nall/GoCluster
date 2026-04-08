package archive

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"
	"time"

	"dxcluster/spot"
)

func encodeRecordV2ForTest(s *spot.Spot) []byte {
	dxCall := strings.TrimSpace(s.DXCallNorm)
	if dxCall == "" {
		dxCall = strings.TrimSpace(s.DXCall)
	}
	deCall := strings.TrimSpace(s.DECallNorm)
	if deCall == "" {
		deCall = strings.TrimSpace(s.DECall)
	}
	mode := strings.TrimSpace(strings.ToUpper(s.Mode))
	lengths := [fieldCount]int{
		len(dxCall),
		len(deCall),
		0,
		len(mode),
		0,
		0,
		0,
		0,
		0,
		0,
		0,
		0,
		0,
	}
	total := recordHeaderSizeV2
	for _, l := range lengths {
		total += l
	}
	raw := make([]byte, total)
	raw[0] = recordVersionV2
	if s.HasReport {
		raw[1] |= flagHasReport
	}
	if s.IsHuman {
		raw[1] |= flagIsHuman
	}
	raw[2] = s.TTL
	binary.BigEndian.PutUint64(raw[4:], math.Float64bits(s.Frequency))
	binary.BigEndian.PutUint32(raw[12:], uint32(int32(s.Report)))
	offset := recordFixedHeaderSizeV2
	for i := 0; i < fieldCount; i++ {
		binary.BigEndian.PutUint16(raw[offset:], uint16(lengths[i]))
		offset += 2
	}
	writeOffset := recordHeaderSizeV2
	copy(raw[writeOffset:], dxCall)
	writeOffset += len(dxCall)
	copy(raw[writeOffset:], deCall)
	writeOffset += len(deCall)
	copy(raw[writeOffset:], mode)
	return raw
}

func TestArchiveRecordStoresStrippedDECall(t *testing.T) {
	s := spot.NewSpot("K1ABC", "W1XYZ-1", 14074.0, "FT8")
	s.DECallStripped = "W1XYZ"
	s.DECallNormStripped = "W1XYZ"

	raw := encodeRecord(s)
	rec, err := decodeRecord(raw)
	if err != nil {
		t.Fatalf("decodeRecord failed: %v", err)
	}
	if rec.deCall != "W1XYZ-1" {
		t.Fatalf("expected raw DE call, got %q", rec.deCall)
	}
	if rec.deCallStripped != "W1XYZ" {
		t.Fatalf("expected stripped DE call, got %q", rec.deCallStripped)
	}

	decoded, err := decodeSpot(time.Now().UTC().UnixNano(), raw)
	if err != nil {
		t.Fatalf("decodeSpot failed: %v", err)
	}
	if decoded.DECall != "W1XYZ-1" {
		t.Fatalf("expected decoded raw DE call, got %q", decoded.DECall)
	}
	if decoded.DECallStripped != "W1XYZ" {
		t.Fatalf("expected decoded stripped DE call, got %q", decoded.DECallStripped)
	}
}

func TestArchiveRecordPreservesDerivedGridFlags(t *testing.T) {
	s := spot.NewSpot("K1ABC", "W1XYZ", 14074.0, "FT8")
	s.DXMetadata.Grid = "FN20"
	s.DXMetadata.GridDerived = true
	s.DEMetadata.Grid = "EM10"
	s.DEMetadata.GridDerived = true

	raw := encodeRecord(s)
	rec, err := decodeRecord(raw)
	if err != nil {
		t.Fatalf("decodeRecord failed: %v", err)
	}
	if !rec.dxGridDerived || !rec.deGridDerived {
		t.Fatalf("expected derived grid flags to be set")
	}
	decoded, err := decodeSpot(time.Now().UTC().UnixNano(), raw)
	if err != nil {
		t.Fatalf("decodeSpot failed: %v", err)
	}
	if !decoded.DXMetadata.GridDerived || !decoded.DEMetadata.GridDerived {
		t.Fatalf("expected decoded spots to retain derived flags")
	}
}

func TestArchiveRecordPreservesObservedFrequency(t *testing.T) {
	s := spot.NewSpot("K1ABC", "W1XYZ", 14074.0, "FT8")
	s.ObservedFrequency = 14076.11

	raw := encodeRecord(s)
	rec, err := decodeRecord(raw)
	if err != nil {
		t.Fatalf("decodeRecord failed: %v", err)
	}
	if rec.freq != 14074.0 {
		t.Fatalf("expected canonical frequency 14074.0, got %.2f", rec.freq)
	}
	if rec.observedFreq != 14076.11 {
		t.Fatalf("expected observed frequency 14076.11, got %.2f", rec.observedFreq)
	}

	decoded, err := decodeSpot(time.Now().UTC().UnixNano(), raw)
	if err != nil {
		t.Fatalf("decodeSpot failed: %v", err)
	}
	if decoded.Frequency != 14074.0 {
		t.Fatalf("expected decoded canonical frequency 14074.0, got %.2f", decoded.Frequency)
	}
	if decoded.ObservedFrequency != 14076.11 {
		t.Fatalf("expected decoded observed frequency 14076.11, got %.2f", decoded.ObservedFrequency)
	}
}

func TestDecodeRecordV2BackfillsObservedFrequency(t *testing.T) {
	s := spot.NewSpot("K1ABC", "W1XYZ", 14076.11, "FT8")

	raw := encodeRecordV2ForTest(s)
	rec, err := decodeRecord(raw)
	if err != nil {
		t.Fatalf("decodeRecord failed: %v", err)
	}
	if rec.freq != 14076.11 {
		t.Fatalf("expected v2 frequency 14076.11, got %.2f", rec.freq)
	}
	if rec.observedFreq != 14076.11 {
		t.Fatalf("expected v2 observed frequency fallback, got %.2f", rec.observedFreq)
	}

	decoded, err := decodeSpot(time.Now().UTC().UnixNano(), raw)
	if err != nil {
		t.Fatalf("decodeSpot failed: %v", err)
	}
	if decoded.ObservedFrequency != 14076.11 {
		t.Fatalf("expected decoded observed frequency fallback, got %.2f", decoded.ObservedFrequency)
	}
}
