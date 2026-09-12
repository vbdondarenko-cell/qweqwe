package chat

import (
	"context"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
)

const (
	// MaxBillSplitTotalMinor bounds the calculator's input against
	// nonsense/overflow amounts. It is not a real currency ceiling (this
	// package never knows the actor's real currency) — just a defensive
	// sanity bound.
	MaxBillSplitTotalMinor = 100_000_000
)

type Service struct {
	store Store
}

func NewService(store Store) (*Service, error) {
	if store == nil {
		return nil, ErrInvalidInput
	}
	return &Service{store: store}, nil
}

func (s *Service) Send(ctx context.Context, actorID, slotID, idempotencyKey, text string) (Message, error) {
	actorID = strings.TrimSpace(actorID)
	slotID = strings.TrimSpace(slotID)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	text = strings.TrimSpace(text)
	if actorID == "" || slotID == "" || text == "" || !utf8.ValidString(text) || utf8.RuneCountInString(text) > MaxMessageRunes || len(idempotencyKey) < MinIdempotencyKeyLen || len(idempotencyKey) > MaxIdempotencyKeyLen {
		return Message{}, ErrInvalidInput
	}
	messageID, err := identifier.NewUUID()
	if err != nil {
		return Message{}, err
	}
	return s.store.Send(ctx, actorID, slotID, messageID, idempotencyKey, text)
}

func (s *Service) ListRecent(ctx context.Context, actorID, slotID string, limit int) ([]Message, error) {
	actorID = strings.TrimSpace(actorID)
	slotID = strings.TrimSpace(slotID)
	if actorID == "" || slotID == "" || limit < 1 || limit > MaxRecent {
		return nil, ErrInvalidInput
	}
	return s.store.ListRecent(ctx, actorID, slotID, limit)
}

// SplitBill implements README §6.18's Bill Splitter calculator: an even
// split of totalMinor among participantIDs (or, if empty, the Slot's whole
// current roster) using deterministic minor-unit rounding — no floats, no
// payment movement, purely a calculation. actorID must be the Slot's host
// or a currently-accepted member (the same authorization boundary as
// Send/ListRecent), and every requested participant must actually be on
// the Slot's roster right now.
//
// Rounding: totalMinor/n is each participant's base share; the
// totalMinor%n leftover minor units go one each to the alphabetically
// first `remainder` user IDs (participants are always sorted by ID before
// splitting) — an arbitrary but fully deterministic and reproducible tie-
// break, stated here rather than left implicit.
func (s *Service) SplitBill(ctx context.Context, actorID, slotID string, totalMinor int, participantIDs []string) (BillSplit, error) {
	actorID = strings.TrimSpace(actorID)
	slotID = strings.TrimSpace(slotID)
	if actorID == "" || slotID == "" || totalMinor <= 0 || totalMinor > MaxBillSplitTotalMinor {
		return BillSplit{}, ErrInvalidInput
	}

	roster, err := s.store.SplitBillRoster(ctx, actorID, slotID)
	if err != nil {
		return BillSplit{}, err
	}
	inRoster := make(map[string]bool, len(roster))
	for _, a := range roster {
		inRoster[a.ID] = true
	}

	ids := participantIDs
	if len(ids) == 0 {
		ids = make([]string, len(roster))
		for i, a := range roster {
			ids[i] = a.ID
		}
	}
	seen := make(map[string]bool, len(ids))
	participants := make([]string, 0, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" || !inRoster[id] {
			return BillSplit{}, ErrInvalidInput
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		participants = append(participants, id)
	}
	if len(participants) < 2 {
		return BillSplit{}, ErrInvalidInput
	}
	sort.Strings(participants)

	base := totalMinor / len(participants)
	remainder := totalMinor % len(participants)
	shares := make([]ParticipantShare, len(participants))
	for i, id := range participants {
		amount := base
		if i < remainder {
			amount++
		}
		shares[i] = ParticipantShare{UserID: id, AmountMinor: amount}
	}
	return BillSplit{SlotID: slotID, TotalMinor: totalMinor, Shares: shares}, nil
}
