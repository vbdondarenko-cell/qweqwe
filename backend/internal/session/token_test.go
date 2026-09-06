package session

import (
	"bytes"
	"testing"
)

func TestGenerateAndHash(t *testing.T) {
	t.Parallel()

	rawA, digestA, err := Generate()
	if err != nil {
		t.Fatalf("Generate A: %v", err)
	}
	rawB, digestB, err := Generate()
	if err != nil {
		t.Fatalf("Generate B: %v", err)
	}

	if rawA == rawB {
		t.Fatal("generated bearer tokens must be unique")
	}
	if bytes.Equal(digestA[:], digestB[:]) {
		t.Fatal("generated token digests must be unique")
	}

	hashed, err := Hash(rawA)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if !bytes.Equal(hashed[:], digestA[:]) {
		t.Fatal("Hash(raw) does not match Generate digest")
	}
}

func TestHashRejectsMalformedTokens(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"", "abc", "not+base64", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"} {
		if _, err := Hash(raw); err == nil {
			t.Fatalf("Hash(%q) unexpectedly succeeded", raw)
		}
	}
}
