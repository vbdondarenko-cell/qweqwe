package slot

import (
	"context"
	"strings"
)

// MySlots returns current relationships; it is not a public user activity feed.
func (s *Service) ListMine(ctx context.Context, actorID, view string) ([]Slot, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" || (view != "HOSTING" && view != "JOINED" && view != "REQUESTED") {
		return nil, ErrInvalidInput
	}
	return s.store.ListMine(ctx, actorID, view, 100)
}
