package slot

import (
	"context"
	"testing"
	"time"
)

type accessStoreStub struct {
	Store
	joinedActor string
	joinedSlot  string
}

func (s *accessStoreStub) Join(_ context.Context, actorID, slotID, _ string, _ []byte, _ time.Time) (Slot, error) {
	s.joinedActor = actorID
	s.joinedSlot = slotID
	return Slot{ID: slotID, AccessMode: AccessInstant, ViewerState: ViewerAccepted, AcceptedCount: 1}, nil
}

func TestJoinDelegatesToV11AccessStore(t *testing.T) {
	store := &accessStoreStub{}
	service, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	out, err := service.Join(context.Background(), "member", "slot", "00000000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatal(err)
	}
	if store.joinedActor != "member" || store.joinedSlot != "slot" || out.ViewerState != ViewerAccepted {
		t.Fatalf("unexpected join delegation: out=%#v store=%#v", out, store)
	}
}
