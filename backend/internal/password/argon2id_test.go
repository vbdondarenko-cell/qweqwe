package password

import (
	"errors"
	"strings"
	"testing"
)

func TestHashVerify(t *testing.T) {
	encoded, err := Hash("correct horse battery staple", OWASPMinimum())
	if err != nil {
		t.Fatal(err)
	}
	ok, err := Verify(encoded, "correct horse battery staple")
	if err != nil || !ok {
		t.Fatalf("verify=%v err=%v", ok, err)
	}
	ok, err = Verify(encoded, "wrong password")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("wrong password accepted")
	}
}

func TestRejectsWeakParameterFloor(t *testing.T) {
	p := OWASPMinimum()
	p.MemoryKiB--
	if _, err := Hash("correct horse battery staple", p); err == nil {
		t.Fatal("expected parameter floor error")
	}
}

func TestRejectsParameterSafetyCeilingBeforeArgonWork(t *testing.T) {
	p := OWASPMinimum()
	p.MemoryKiB = maxMemoryKiB + 1
	if _, err := Hash("correct horse battery staple", p); err == nil {
		t.Fatal("expected parameter ceiling error")
	}

	encoded, err := Hash("correct horse battery staple", OWASPMinimum())
	if err != nil {
		t.Fatal(err)
	}
	hostile := strings.Replace(encoded, "m=19456", "m=262145", 1)
	if ok, err := Verify(hostile, "correct horse battery staple"); !errors.Is(err, ErrInvalidHash) || ok {
		t.Fatalf("hostile encoded params must fail before Argon2 work: ok=%v err=%v", ok, err)
	}
}

func TestVerifyRejectsOversizedPasswordBeforeArgonWork(t *testing.T) {
	encoded, err := Hash("correct horse battery staple", OWASPMinimum())
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := Verify(encoded, strings.Repeat("x", 1025)); !errors.Is(err, ErrInvalidPassword) || ok {
		t.Fatalf("oversized password must fail: ok=%v err=%v", ok, err)
	}
}
