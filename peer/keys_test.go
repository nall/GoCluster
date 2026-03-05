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
