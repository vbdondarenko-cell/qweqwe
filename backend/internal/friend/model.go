// Package friend implements README's Friends/Links domain: the prerequisite
// for §4.3's "Friends/Links" visibility mode, the "My LINKs dashboard"
// (§5.4), §6.8's FRIEND_REQUEST/FRIEND_ACCEPTED notification types, and the
// foundation §6.13's "Links Graph" later builds on.
//
// A friend request is a directed, single-use proposal (README gives it no
// separate spec beyond the notification types and deep links, so this
// package follows the same shape every other pairwise-relationship domain
// in this codebase already uses — blocklist for a symmetric relationship
// with no request step, BUMP for a mutual-confirmation request step). Once
// both sides agree, the relationship becomes a symmetric Friendship, mirror
// of how bump_confirmations/user_blocks are stored: an unordered pair
// canonicalized to a stable (lo, hi) ordering so either side reading it
// gets the same row.
package friend

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidInput   = errors.New("invalid friend input")
	ErrNotFound       = errors.New("friend request not found")
	ErrForbidden      = errors.New("friend action forbidden")
	ErrAlreadyFriends = errors.New("users are already friends")
)

type UserSummary struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"displayName"`
	AvatarURL   *string `json:"avatarUrl,omitempty"`
}

// PendingRequest is one side's view of a directed, unanswered request: User
// is always the OTHER party (the requester, for an incoming request; the
// target, for an outgoing one) — never the viewer themselves.
type PendingRequest struct {
	RequestID string      `json:"requestId"`
	User      UserSummary `json:"user"`
	CreatedAt time.Time   `json:"createdAt"`
}

type Friend struct {
	User  UserSummary `json:"user"`
	Since time.Time   `json:"since"`
}

// Outcome reports what a Request call actually did. A request from A to B
// while B already has a pending request to A is not a second, parallel
// request — it is answered as a mutual match immediately (OutcomeAccepted),
// the same "you both already wanted this" resolution most friend-request
// products use rather than forcing a redundant second accept step.
type Outcome string

const (
	OutcomeRequested      Outcome = "REQUESTED"
	OutcomeAccepted       Outcome = "ACCEPTED"
	OutcomeAlreadyPending Outcome = "ALREADY_PENDING"
)

type RequestResult struct {
	Outcome Outcome `json:"outcome"`
}

// Store is implemented by the postgres package. Every mutation here is
// designed to be naturally idempotent through its own constraints/state
// checks (ON CONFLICT / conditional UPDATE) rather than the generic
// mutation_idempotency mechanism the Slot domain uses — mirroring how
// internal/blocklist's Block/Unblock already work in this codebase, since
// a repeat Request/Accept/Reject/Cancel/Remove call has an obviously
// correct, safe no-op or already-current-state answer, unlike a Slot
// mutation with side effects (capacity, chat messages) that must not
// double-apply.
type Store interface {
	// Request fails with ErrForbidden if either side has blocked the
	// other, ErrAlreadyFriends if they are already friends, ErrInvalidInput
	// if actorID==targetID or targetID does not exist.
	Request(ctx context.Context, actorID, targetID string, now time.Time) (RequestResult, error)
	// Accept fails with ErrNotFound if no PENDING request from requesterID
	// to actorID exists.
	Accept(ctx context.Context, actorID, requesterID string, now time.Time) error
	// Reject fails with ErrNotFound if no PENDING request from requesterID
	// to actorID exists.
	Reject(ctx context.Context, actorID, requesterID string) error
	// Cancel fails with ErrNotFound if no PENDING request from actorID to
	// targetID exists.
	Cancel(ctx context.Context, actorID, targetID string) error
	// Remove (unfriend) is idempotent: removing a non-existent friendship
	// is a harmless no-op, not an error, matching Unblock's existing
	// contract in this codebase.
	Remove(ctx context.Context, actorID, friendID string) error
	ListFriends(ctx context.Context, actorID string, limit int) ([]Friend, error)
	ListIncoming(ctx context.Context, actorID string, limit int) ([]PendingRequest, error)
	ListOutgoing(ctx context.Context, actorID string, limit int) ([]PendingRequest, error)
	// AreFriends is the query surface a future visibility mode (README
	// §4.3 "Friends/Links") consumes; not wired into any discovery query
	// by this block, but part of the domain contract now so that
	// integration does not require a Store interface change later.
	AreFriends(ctx context.Context, a, b string) (bool, error)
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) (*Service, error) {
	if store == nil {
		return nil, ErrInvalidInput
	}
	return &Service{store: store, now: time.Now}, nil
}

func (s *Service) Request(ctx context.Context, actorID, targetID string) (RequestResult, error) {
	actorID = strings.TrimSpace(actorID)
	targetID = strings.TrimSpace(targetID)
	if actorID == "" || targetID == "" || actorID == targetID {
		return RequestResult{}, ErrInvalidInput
	}
	return s.store.Request(ctx, actorID, targetID, s.now().UTC())
}

func (s *Service) Accept(ctx context.Context, actorID, requesterID string) error {
	actorID = strings.TrimSpace(actorID)
	requesterID = strings.TrimSpace(requesterID)
	if actorID == "" || requesterID == "" || actorID == requesterID {
		return ErrInvalidInput
	}
	return s.store.Accept(ctx, actorID, requesterID, s.now().UTC())
}

func (s *Service) Reject(ctx context.Context, actorID, requesterID string) error {
	actorID = strings.TrimSpace(actorID)
	requesterID = strings.TrimSpace(requesterID)
	if actorID == "" || requesterID == "" || actorID == requesterID {
		return ErrInvalidInput
	}
	return s.store.Reject(ctx, actorID, requesterID)
}

func (s *Service) Cancel(ctx context.Context, actorID, targetID string) error {
	actorID = strings.TrimSpace(actorID)
	targetID = strings.TrimSpace(targetID)
	if actorID == "" || targetID == "" || actorID == targetID {
		return ErrInvalidInput
	}
	return s.store.Cancel(ctx, actorID, targetID)
}

func (s *Service) Remove(ctx context.Context, actorID, friendID string) error {
	actorID = strings.TrimSpace(actorID)
	friendID = strings.TrimSpace(friendID)
	if actorID == "" || friendID == "" || actorID == friendID {
		return ErrInvalidInput
	}
	return s.store.Remove(ctx, actorID, friendID)
}

func (s *Service) ListFriends(ctx context.Context, actorID string) ([]Friend, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return nil, ErrInvalidInput
	}
	return s.store.ListFriends(ctx, actorID, 200)
}

func (s *Service) ListIncoming(ctx context.Context, actorID string) ([]PendingRequest, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return nil, ErrInvalidInput
	}
	return s.store.ListIncoming(ctx, actorID, 200)
}

func (s *Service) ListOutgoing(ctx context.Context, actorID string) ([]PendingRequest, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return nil, ErrInvalidInput
	}
	return s.store.ListOutgoing(ctx, actorID, 200)
}

func (s *Service) AreFriends(ctx context.Context, a, b string) (bool, error) {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false, ErrInvalidInput
	}
	if a == b {
		return false, nil
	}
	return s.store.AreFriends(ctx, a, b)
}
