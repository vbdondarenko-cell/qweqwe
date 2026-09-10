package slot

import (
	"context"
	"strings"
	"time"
)

// V11AccessStore adds v1.1 access transitions without widening the stable v1.0 Store contract.
type V11AccessStore interface {
	Join(ctx context.Context, actorID, slotID, idempotencyKey string, requestHash []byte, now time.Time) (Slot, error)
}

func (s *Service) Join(ctx context.Context, actorID, slotID, idempotencyKey string) (Slot, error) {
	actorID = strings.TrimSpace(actorID)
	slotID = strings.TrimSpace(slotID)
	key, ok := validIdempotencyKey(idempotencyKey)
	store, supported := s.store.(V11AccessStore)
	if actorID == "" || slotID == "" || !ok || !supported {
		return Slot{}, ErrInvalidInput
	}
	fingerprint, err := hashRequest(struct{ SlotID string }{slotID})
	if err != nil {
		return Slot{}, err
	}
	return store.Join(ctx, actorID, slotID, key, fingerprint, s.now().UTC())
}
