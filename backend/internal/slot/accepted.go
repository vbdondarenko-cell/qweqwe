package slot

import (
	"context"
	"strings"
)

// ListAccepted is the host-only current roster, not a public activity history.
func (s *Service) ListAccepted(ctx context.Context, actorID, slotID string) ([]Organizer, error) {
	actorID, slotID = strings.TrimSpace(actorID), strings.TrimSpace(slotID)
	if actorID == "" || slotID == "" { return nil, ErrInvalidInput }
	return s.store.ListAccepted(ctx, actorID, slotID)
}
