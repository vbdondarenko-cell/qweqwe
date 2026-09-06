package password

import "testing"

func TestHashVerify(t *testing.T) {
	encoded, err := Hash("correct horse battery staple", OWASPMinimum())
	if err != nil { t.Fatal(err) }
	ok, err := Verify(encoded, "correct horse battery staple")
	if err != nil || !ok { t.Fatalf("verify=%v err=%v", ok, err) }
	ok, err = Verify(encoded, "wrong password")
	if err != nil { t.Fatal(err) }
	if ok { t.Fatal("wrong password accepted") }
}

func TestRejectsWeakParameterFloor(t *testing.T) {
	p := OWASPMinimum(); p.MemoryKiB--
	if _, err := Hash("correct horse battery staple", p); err == nil { t.Fatal("expected parameter floor error") }
}
