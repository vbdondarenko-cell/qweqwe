package guardian

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStore struct {
	createErr    error
	revokeErr    error
	accessStatus Status
	accessErr    error
	lastMode     Mode
	lastTTLEnds  time.Time
	lastNow      time.Time
	lastRevoked  string
	lastAccessed []byte
}

func (f *fakeStore) CreateLink(_ context.Context, _, _, _ string, mode Mode, _ []byte, now, expiresAt time.Time) error {
	f.lastMode = mode
	f.lastNow = now
	f.lastTTLEnds = expiresAt
	return f.createErr
}

func (f *fakeStore) Revoke(_ context.Context, linkID, _ string, _ time.Time) error {
	f.lastRevoked = linkID
	return f.revokeErr
}

func (f *fakeStore) Access(_ context.Context, tokenHash []byte, _ time.Time) (Status, error) {
	f.lastAccessed = tokenHash
	if f.accessErr != nil {
		return Status{}, f.accessErr
	}
	return f.accessStatus, nil
}

func TestCreateLinkRejectsUnsupportedModes(t *testing.T) {
	svc, err := NewService(&fakeStore{})
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []Mode{ModeETAApproximate, ModeSafetyRadar, ModeLivePrecise, Mode("BOGUS")} {
		if _, err := svc.CreateLink(context.Background(), "user-1", "slot-1", mode, time.Hour); !errors.Is(err, ErrModeNotSupported) {
			t.Fatalf("mode=%q: expected ErrModeNotSupported, got %v", mode, err)
		}
	}
}

func TestCreateLinkEnforcesTTLBounds(t *testing.T) {
	svc, err := NewService(&fakeStore{})
	if err != nil {
		t.Fatal(err)
	}
	for _, ttl := range []time.Duration{0, MinTTL - time.Second, MaxTTL + time.Second} {
		if _, err := svc.CreateLink(context.Background(), "user-1", "slot-1", ModeStatusOnly, ttl); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("ttl=%v: expected ErrInvalidInput, got %v", ttl, err)
		}
	}
}

func TestCreateLinkReturnsRawTokenOnceAndStoresOnlyItsHash(t *testing.T) {
	store := &fakeStore{}
	svc, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	out, err := svc.CreateLink(context.Background(), "user-1", "slot-1", ModeStatusOnly, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if out.Token == "" {
		t.Fatal("expected a raw token")
	}
	if out.Mode != ModeStatusOnly || out.SlotID != "slot-1" || out.CreatedBy != "user-1" {
		t.Fatalf("unexpected link: %#v", out)
	}
	if !out.ExpiresAt.Equal(store.lastTTLEnds) {
		t.Fatalf("expiresAt mismatch: %v vs %v", out.ExpiresAt, store.lastTTLEnds)
	}
}

func TestCreateLinkPropagatesStoreErrors(t *testing.T) {
	svc, err := NewService(&fakeStore{createErr: ErrForbidden})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateLink(context.Background(), "user-1", "slot-1", ModeStatusOnly, time.Hour); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestRevokeRequiresActorAndLinkID(t *testing.T) {
	svc, err := NewService(&fakeStore{})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Revoke(context.Background(), "", "link-1"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if err := svc.Revoke(context.Background(), "user-1", ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestRevokePropagatesForbidden(t *testing.T) {
	svc, err := NewService(&fakeStore{revokeErr: ErrForbidden})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Revoke(context.Background(), "user-1", "link-1"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestAccessRejectsBlankOrMalformedTokenWithoutTouchingStore(t *testing.T) {
	store := &fakeStore{}
	svc, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"", "   ", "not-a-valid-token"} {
		if _, err := svc.Access(context.Background(), raw); !errors.Is(err, ErrLinkUnavailable) {
			t.Fatalf("token=%q: expected ErrLinkUnavailable, got %v", raw, err)
		}
	}
	if store.lastAccessed != nil {
		t.Fatal("a malformed token must never reach the store")
	}
}

func TestAccessReturnsStatusForValidToken(t *testing.T) {
	store := &fakeStore{accessStatus: Status{SharerDisplayName: "Alice", SlotStatus: SlotStatusActive}}
	svc, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	created, err := svc.CreateLink(context.Background(), "user-1", "slot-1", ModeStatusOnly, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Access(context.Background(), created.Token)
	if err != nil {
		t.Fatal(err)
	}
	if got.SharerDisplayName != "Alice" || got.SlotStatus != SlotStatusActive {
		t.Fatalf("unexpected status: %#v", got)
	}
}

func TestAccessPropagatesUnavailable(t *testing.T) {
	store := &fakeStore{accessErr: ErrLinkUnavailable}
	svc, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	created, err := svc.CreateLink(context.Background(), "user-1", "slot-1", ModeStatusOnly, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Access(context.Background(), created.Token); !errors.Is(err, ErrLinkUnavailable) {
		t.Fatalf("expected ErrLinkUnavailable, got %v", err)
	}
}
