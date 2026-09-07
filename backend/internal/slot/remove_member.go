package slot

import (
    "context"
    "strings"
)

func (s *Service) RemoveMember(ctx context.Context, actorID, slotID, memberID string, expectedVersion int64, idempotencyKey string) (Slot, error) {
    actorID, slotID, key, err := normalizeMutation(actorID, slotID, idempotencyKey)
    if err != nil { return Slot{}, err }
    memberID = strings.TrimSpace(memberID)
    if memberID == "" || memberID == actorID || expectedVersion <= 0 { return Slot{}, ErrInvalidInput }
    fingerprint, err := hashRequest(struct {
        SlotID string
        MemberID string
        ExpectedVersion int64
    }{slotID, memberID, expectedVersion})
    if err != nil { return Slot{}, err }
    return s.store.RemoveMember(ctx, actorID, slotID, memberID, expectedVersion, key, fingerprint, s.now().UTC())
}
