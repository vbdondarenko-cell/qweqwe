package chat

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type memoryStore struct {
	lastActor string
	lastSlot  string
	lastKey   string
	lastText  string
	list      []Message
	err       error
	roster    []Author
	rosterErr error
}

func (m *memoryStore) Send(_ context.Context, actorID, slotID, messageID, key, text string) (Message, error) {
	if m.err != nil {
		return Message{}, m.err
	}
	m.lastActor, m.lastSlot, m.lastKey, m.lastText = actorID, slotID, key, text
	return Message{ID: messageID, SlotID: slotID, Author: &Author{ID: actorID}, Text: text, CreatedAt: time.Unix(1, 0).UTC()}, nil
}

func (m *memoryStore) ListRecent(_ context.Context, actorID, slotID string, _ int) ([]Message, error) {
	m.lastActor, m.lastSlot = actorID, slotID
	if m.err != nil {
		return nil, m.err
	}
	return m.list, nil
}

func (m *memoryStore) SplitBillRoster(_ context.Context, actorID, slotID string) ([]Author, error) {
	m.lastActor, m.lastSlot = actorID, slotID
	if m.rosterErr != nil {
		return nil, m.rosterErr
	}
	return m.roster, nil
}

func TestSendTrimsAndBoundsText(t *testing.T) {
	store := &memoryStore{}
	svc, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	key := "12345678-1234-4234-8234-123456789abc"
	out, err := svc.Send(context.Background(), " user ", " slot ", key, "  hello world  ")
	if err != nil {
		t.Fatal(err)
	}
	if out.Text != "hello world" || store.lastActor != "user" || store.lastSlot != "slot" || store.lastKey != key {
		t.Fatalf("unexpected output: %#v", out)
	}
	if _, err := svc.Send(context.Background(), "u", "s", key, strings.Repeat("x", MaxMessageRunes+1)); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
	if _, err := svc.Send(context.Background(), "u", "s", key, "   "); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected blank rejection, got %v", err)
	}
	if _, err := svc.Send(context.Background(), "u", "s", "short", "hello"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected idempotency key rejection, got %v", err)
	}
}

func TestListRecentEnforcesBound(t *testing.T) {
	svc, _ := NewService(&memoryStore{})
	if _, err := svc.ListRecent(context.Background(), "u", "s", 0); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected limit error, got %v", err)
	}
	if _, err := svc.ListRecent(context.Background(), "u", "s", MaxRecent+1); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected max limit error, got %v", err)
	}
}

func TestSplitBillDefaultsToWholeRosterWithDeterministicRounding(t *testing.T) {
	store := &memoryStore{roster: []Author{{ID: "c"}, {ID: "a"}, {ID: "b"}}}
	svc, _ := NewService(store)
	// 100 minor units among 3 people: base=33, remainder=1 -- the
	// alphabetically-first participant ("a") gets the extra unit.
	out, err := svc.SplitBill(context.Background(), "a", "slot-1", 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.TotalMinor != 100 || len(out.Shares) != 3 {
		t.Fatalf("unexpected split: %#v", out)
	}
	want := map[string]int{"a": 34, "b": 33, "c": 33}
	sum := 0
	for _, share := range out.Shares {
		if share.AmountMinor != want[share.UserID] {
			t.Fatalf("share for %s = %d, want %d", share.UserID, share.AmountMinor, want[share.UserID])
		}
		sum += share.AmountMinor
	}
	if sum != 100 {
		t.Fatalf("shares must sum to the total exactly, got %d", sum)
	}
	// Shares are always returned in the same sorted-by-ID order, regardless
	// of the roster's own order -- a deterministic, reproducible result.
	if out.Shares[0].UserID != "a" || out.Shares[1].UserID != "b" || out.Shares[2].UserID != "c" {
		t.Fatalf("unexpected order: %#v", out.Shares)
	}
}

func TestSplitBillRejectsParticipantNotOnRoster(t *testing.T) {
	store := &memoryStore{roster: []Author{{ID: "a"}, {ID: "b"}}}
	svc, _ := NewService(store)
	if _, err := svc.SplitBill(context.Background(), "a", "slot-1", 100, []string{"a", "stranger"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestSplitBillRejectsFewerThanTwoParticipants(t *testing.T) {
	store := &memoryStore{roster: []Author{{ID: "a"}, {ID: "b"}}}
	svc, _ := NewService(store)
	if _, err := svc.SplitBill(context.Background(), "a", "slot-1", 100, []string{"a"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input for a single participant, got %v", err)
	}
	if _, err := svc.SplitBill(context.Background(), "a", "slot-1", 100, []string{"a", "a"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("a deduplicated single participant must also be rejected, got %v", err)
	}
}

func TestSplitBillRejectsInvalidTotal(t *testing.T) {
	store := &memoryStore{roster: []Author{{ID: "a"}, {ID: "b"}}}
	svc, _ := NewService(store)
	for _, total := range []int{0, -1, MaxBillSplitTotalMinor + 1} {
		if _, err := svc.SplitBill(context.Background(), "a", "slot-1", total, nil); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("total=%d: expected invalid input, got %v", total, err)
		}
	}
}

func TestSplitBillPropagatesAuthorizationErrors(t *testing.T) {
	store := &memoryStore{rosterErr: ErrForbidden}
	svc, _ := NewService(store)
	if _, err := svc.SplitBill(context.Background(), "a", "slot-1", 100, nil); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
