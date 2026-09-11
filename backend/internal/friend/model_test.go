package friend

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStore struct {
	requestResult RequestResult
	requestErr    error
	acceptErr     error
	rejectErr     error
	cancelErr     error
	removeErr     error
	friends       []Friend
	incoming      []PendingRequest
	outgoing      []PendingRequest
	areFriends    bool
	areFriendsErr error

	lastRequestArgs [2]string
	lastAcceptArgs  [2]string
	lastRejectArgs  [2]string
	lastCancelArgs  [2]string
	lastRemoveArgs  [2]string
}

func (f *fakeStore) Request(_ context.Context, actorID, targetID string, _ time.Time) (RequestResult, error) {
	f.lastRequestArgs = [2]string{actorID, targetID}
	return f.requestResult, f.requestErr
}

func (f *fakeStore) Accept(_ context.Context, actorID, requesterID string, _ time.Time) error {
	f.lastAcceptArgs = [2]string{actorID, requesterID}
	return f.acceptErr
}

func (f *fakeStore) Reject(_ context.Context, actorID, requesterID string) error {
	f.lastRejectArgs = [2]string{actorID, requesterID}
	return f.rejectErr
}

func (f *fakeStore) Cancel(_ context.Context, actorID, targetID string) error {
	f.lastCancelArgs = [2]string{actorID, targetID}
	return f.cancelErr
}

func (f *fakeStore) Remove(_ context.Context, actorID, friendID string) error {
	f.lastRemoveArgs = [2]string{actorID, friendID}
	return f.removeErr
}

func (f *fakeStore) ListFriends(_ context.Context, _ string, _ int) ([]Friend, error) {
	return f.friends, nil
}

func (f *fakeStore) ListIncoming(_ context.Context, _ string, _ int) ([]PendingRequest, error) {
	return f.incoming, nil
}

func (f *fakeStore) ListOutgoing(_ context.Context, _ string, _ int) ([]PendingRequest, error) {
	return f.outgoing, nil
}

func (f *fakeStore) AreFriends(_ context.Context, _, _ string) (bool, error) {
	return f.areFriends, f.areFriendsErr
}

func TestNewServiceRejectsNilStore(t *testing.T) {
	if _, err := NewService(nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestServiceRequestRejectsSelfAndEmpty(t *testing.T) {
	svc, err := NewService(&fakeStore{})
	if err != nil {
		t.Fatal(err)
	}
	cases := [][2]string{{"", "target"}, {"actor", ""}, {"same", "same"}}
	for _, c := range cases {
		if _, err := svc.Request(context.Background(), c[0], c[1]); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("Request(%q,%q): expected ErrInvalidInput, got %v", c[0], c[1], err)
		}
	}
}

func TestServiceRequestTrimsAndForwards(t *testing.T) {
	store := &fakeStore{requestResult: RequestResult{Outcome: OutcomeRequested}}
	svc, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	out, err := svc.Request(context.Background(), "  actor  ", "  target  ")
	if err != nil {
		t.Fatal(err)
	}
	if out.Outcome != OutcomeRequested {
		t.Fatalf("unexpected outcome: %#v", out)
	}
	if store.lastRequestArgs != [2]string{"actor", "target"} {
		t.Fatalf("expected trimmed args forwarded, got %#v", store.lastRequestArgs)
	}
}

func TestServiceRequestPropagatesStoreErrors(t *testing.T) {
	store := &fakeStore{requestErr: ErrAlreadyFriends}
	svc, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Request(context.Background(), "a", "b"); !errors.Is(err, ErrAlreadyFriends) {
		t.Fatalf("expected ErrAlreadyFriends to propagate, got %v", err)
	}
}

func TestServiceAcceptRejectCancelRejectSelfAndEmpty(t *testing.T) {
	svc, err := NewService(&fakeStore{})
	if err != nil {
		t.Fatal(err)
	}
	cases := [][2]string{{"", "b"}, {"a", ""}, {"same", "same"}}
	for _, c := range cases {
		if err := svc.Accept(context.Background(), c[0], c[1]); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("Accept(%q,%q): expected ErrInvalidInput, got %v", c[0], c[1], err)
		}
		if err := svc.Reject(context.Background(), c[0], c[1]); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("Reject(%q,%q): expected ErrInvalidInput, got %v", c[0], c[1], err)
		}
		if err := svc.Cancel(context.Background(), c[0], c[1]); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("Cancel(%q,%q): expected ErrInvalidInput, got %v", c[0], c[1], err)
		}
	}
}

func TestServiceAcceptTrimsAndForwards(t *testing.T) {
	store := &fakeStore{}
	svc, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Accept(context.Background(), "  actor  ", "  requester  "); err != nil {
		t.Fatal(err)
	}
	if store.lastAcceptArgs != [2]string{"actor", "requester"} {
		t.Fatalf("expected trimmed args forwarded, got %#v", store.lastAcceptArgs)
	}
}

func TestServiceAcceptPropagatesNotFound(t *testing.T) {
	store := &fakeStore{acceptErr: ErrNotFound}
	svc, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Accept(context.Background(), "a", "b"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound to propagate, got %v", err)
	}
}

func TestServiceRemoveRejectsSelfAndEmpty(t *testing.T) {
	svc, err := NewService(&fakeStore{})
	if err != nil {
		t.Fatal(err)
	}
	cases := [][2]string{{"", "b"}, {"a", ""}, {"same", "same"}}
	for _, c := range cases {
		if err := svc.Remove(context.Background(), c[0], c[1]); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("Remove(%q,%q): expected ErrInvalidInput, got %v", c[0], c[1], err)
		}
	}
}

func TestServiceListersRejectEmptyActor(t *testing.T) {
	svc, err := NewService(&fakeStore{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ListFriends(context.Background(), ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("ListFriends: expected ErrInvalidInput, got %v", err)
	}
	if _, err := svc.ListIncoming(context.Background(), ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("ListIncoming: expected ErrInvalidInput, got %v", err)
	}
	if _, err := svc.ListOutgoing(context.Background(), ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("ListOutgoing: expected ErrInvalidInput, got %v", err)
	}
}

func TestServiceAreFriendsRejectsEmptyAndSelfShortCircuits(t *testing.T) {
	store := &fakeStore{areFriends: true}
	svc, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AreFriends(context.Background(), "", "b"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for empty a, got %v", err)
	}
	// Self-pair short-circuits to false without ever reaching the store —
	// a user is never their own "friend" edge, and the store's AreFriends
	// implementation is not required to handle a==b sensibly.
	got, err := svc.AreFriends(context.Background(), "same", "same")
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Fatalf("expected AreFriends(x,x)=false without querying the store")
	}
}
