package peer

import "testing"

func TestPC92KeyIgnoresHopValue(t *testing.T) {
	a, err := ParseFrame("PC92^NODE1^123^A^^9CALL:ver:build:1.2.3.4^H95^")
	if err != nil {
		t.Fatalf("ParseFrame(a): %v", err)
	}
	b, err := ParseFrame("PC92^NODE1^123^A^^9CALL:ver:build:1.2.3.4^H94^")
	if err != nil {
		t.Fatalf("ParseFrame(b): %v", err)
	}
	keyA := pc92Key(a)
	keyB := pc92Key(b)
	if keyA != keyB {
		t.Fatalf("expected equal keys, got %q vs %q", keyA, keyB)
	}
}

func TestPC92KeyDiffersByRecordType(t *testing.T) {
	add, err := ParseFrame("PC92^NODE1^123^A^^9CALL:ver:build:1.2.3.4^H95^")
	if err != nil {
		t.Fatalf("ParseFrame(add): %v", err)
	}
	del, err := ParseFrame("PC92^NODE1^123^D^^9CALL:ver:build:1.2.3.4^H95^")
	if err != nil {
		t.Fatalf("ParseFrame(del): %v", err)
	}
	keyAdd := pc92Key(add)
	keyDel := pc92Key(del)
	if keyAdd == keyDel {
		t.Fatalf("expected different keys for record types, got %q", keyAdd)
	}
}

func TestWWVKeyIgnoresHopValue(t *testing.T) {
	a, err := ParseFrame("PC23^19-Apr-2026^1200Z^120^5^1^No storms^W1AW^NODE^H95^")
	if err != nil {
		t.Fatalf("ParseFrame(a): %v", err)
	}
	b, err := ParseFrame("PC23^19-Apr-2026^1200Z^120^5^1^No storms^W1AW^NODE^H94^")
	if err != nil {
		t.Fatalf("ParseFrame(b): %v", err)
	}
	if keyA, keyB := wwvKey(a), wwvKey(b); keyA != keyB {
		t.Fatalf("expected equal WWV keys, got %q vs %q", keyA, keyB)
	}
}

func TestPC93KeyIgnoresHopValue(t *testing.T) {
	a, err := ParseFrame("PC93^IZ7AUH-6^79200^*^IZ7AUH-6^*^hello^H97^")
	if err != nil {
		t.Fatalf("ParseFrame(a): %v", err)
	}
	b, err := ParseFrame("PC93^IZ7AUH-6^79200^*^IZ7AUH-6^*^hello^H96^")
	if err != nil {
		t.Fatalf("ParseFrame(b): %v", err)
	}
	if keyA, keyB := pc93Key(a), pc93Key(b); keyA != keyB {
		t.Fatalf("expected equal PC93 keys, got %q vs %q", keyA, keyB)
	}
}
