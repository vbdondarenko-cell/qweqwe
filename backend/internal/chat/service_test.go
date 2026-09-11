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
