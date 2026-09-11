package bump

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestComputeBand(t *testing.T) {
	cases := []struct {
		count int
		want  Band
	}{
		{-1, BandNew}, // defensive: never actually stored, but ComputeBand must not panic/misclassify
		{0, BandNew},
		{1, BandBuilding},
		{2, BandBuilding},
		{3, BandReliable},
		{9, BandReliable},
		{10, BandTrusted},
		{1000, BandTrusted},
	}
	for _, c := range cases {
		if got := ComputeBand(c.count); got != c.want {
			t.Errorf("ComputeBand(%d) = %s, want %s", c.count, got, c.want)
		}
	}
}

type fakeStore struct {
	challenge       Challenge
	challengeErr    error
	confirmResult   ConfirmResult
	confirmErr      error
	reliability     Reliability
	reliabilityErr  error
	vault           []VaultEntry
	vaultErr        error
	lastConfirmArgs [4]string // actorID, slotID, counterpartID, nonce
	lastVaultLimit  int
}

func (f *fakeStore) IssueChallenge(_ context.Context, _, _ string, _ time.Duration) (Challenge, error) {
	return f.challenge, f.challengeErr
}

func (f *fakeStore) Confirm(_ context.Context, actorID, slotID, counterpartID, nonce string) (ConfirmResult, error) {
	f.lastConfirmArgs = [4]string{actorID, slotID, counterpartID, nonce}
	return f.confirmResult, f.confirmErr
}

func (f *fakeStore) Reliability(_ context.Context, _ string) (Reliability, error) {
	return f.reliability, f.reliabilityErr
}

func (f *fakeStore) Vault(_ context.Context, _ string, limit int) ([]VaultEntry, error) {
	f.lastVaultLimit = limit
	return f.vault, f.vaultErr
}

func TestNewServiceRejectsInvalidDependencies(t *testing.T) {
	if _, err := NewService(nil, time.Minute); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for nil store, got %v", err)
	}
	if _, err := NewService(&fakeStore{}, 0); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for zero TTL, got %v", err)
	}
	if _, err := NewService(&fakeStore{}, -time.Second); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for negative TTL, got %v", err)
	}
}

func TestServiceIssueChallengeRejectsEmptyInput(t *testing.T) {
	svc, err := NewService(&fakeStore{}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.IssueChallenge(context.Background(), "", "slot-1"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if _, err := svc.IssueChallenge(context.Background(), "user-1", ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestServiceConfirmRejectsInvalidInput(t *testing.T) {
	svc, err := NewService(&fakeStore{}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name                            string
		actor, slot, counterpart, nonce string
	}{
		{"empty actor", "", "slot-1", "user-2", "nonce-1"},
		{"empty slot", "user-1", "", "user-2", "nonce-1"},
		{"empty counterpart", "user-1", "slot-1", "", "nonce-1"},
		{"empty nonce", "user-1", "slot-1", "user-2", ""},
		{"self-bump", "user-1", "slot-1", "user-1", "nonce-1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := svc.Confirm(context.Background(), c.actor, c.slot, c.counterpart, c.nonce); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestServiceConfirmTrimsAndForwardsArgs(t *testing.T) {
	store := &fakeStore{confirmResult: ConfirmResult{Verified: true, ConfirmedAt: time.Unix(100, 0)}}
	svc, err := NewService(store, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Confirm(context.Background(), " user-1 ", " slot-1 ", " user-2 ", " nonce-1 ")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Verified {
		t.Fatal("expected Verified=true to pass through from the store")
	}
	want := [4]string{"user-1", "slot-1", "user-2", "nonce-1"}
	if store.lastConfirmArgs != want {
		t.Fatalf("args not trimmed before reaching the store: %#v", store.lastConfirmArgs)
	}
}

func TestServiceReliabilityRejectsEmptyUserID(t *testing.T) {
	svc, err := NewService(&fakeStore{}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Reliability(context.Background(), "  "); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestServiceVaultClampsLimit(t *testing.T) {
	store := &fakeStore{}
	svc, err := NewService(store, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ in, want int }{
		{0, 50}, {-5, 50}, {101, 50}, {1, 1}, {100, 100},
	}
	for _, c := range cases {
		if _, err := svc.Vault(context.Background(), "user-1", c.in); err != nil {
			t.Fatal(err)
		}
		if store.lastVaultLimit != c.want {
			t.Errorf("Vault(limit=%d): forwarded limit=%d, want %d", c.in, store.lastVaultLimit, c.want)
		}
	}
	if _, err := svc.Vault(context.Background(), "", 10); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for empty userID, got %v", err)
	}
}
