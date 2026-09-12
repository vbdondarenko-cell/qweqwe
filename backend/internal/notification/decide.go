package notification

import "time"

// PushOutcome mirrors notification_deliveries.push_outcome's CHECK
// constraint exactly (db/migrations/000027_v11_notifications.sql), so the
// postgres projector can write a Decide() result directly without a second
// translation table that could drift out of sync with the schema.
type PushOutcome string

const (
	OutcomeSent                   PushOutcome = "SENT"
	OutcomeExpired                PushOutcome = "EXPIRED"
	OutcomeSuppressedPreference   PushOutcome = "SUPPRESSED_PREFERENCE"
	OutcomeSuppressedQuietHours   PushOutcome = "SUPPRESSED_QUIET_HOURS"
	OutcomeSuppressedFrequencyCap PushOutcome = "SUPPRESSED_FREQUENCY_CAP"
	OutcomeSuppressedGrouped      PushOutcome = "SUPPRESSED_GROUPED"
	OutcomeSkippedNoDevice        PushOutcome = "SKIPPED_NO_DEVICE"
	OutcomeSendFailed             PushOutcome = "SEND_FAILED"
)

// DecisionInput is everything Decide needs to resolve one candidate
// notification. It carries no database or clock dependency of its own —
// every time-sensitive field is passed in explicitly — so callers (and
// tests) fully control what "now" means for a given decision.
type DecisionInput struct {
	Type Type
	// Now is the instant the decision is made, not necessarily when the
	// underlying domain event occurred.
	Now time.Time
	// ExpiresAt is the notification's TTL cutoff. A zero value means "never
	// expires" (used sparingly; most types should set a real TTL).
	ExpiresAt   time.Time
	Preferences Preferences
	// RecentFrequencyCappedCount is the number of frequency-capped
	// (recommendation/promo) notifications already sent to this user
	// within the caller's own trailing window. Ignored for a Type that
	// isn't FrequencyCapped().
	RecentFrequencyCappedCount int
	// FrequencyCapMax is the maximum allowed within that window. <= 0
	// disables the cap entirely (treated as "not configured", not as
	// "cap everything").
	FrequencyCapMax int
	// LastGroupAnchorAt is when this recipient's most recent NON-grouped
	// (i.e. actually decided, whether sent or suppressed for another
	// reason) notification of this exact (Slot, Type) pair was created —
	// nil if there isn't one. Only consulted when Type.Groupable(). The
	// caller (the postgres projector) is responsible for finding this: it
	// is the anchor of a fixed-size, non-overlapping grouping window, not
	// a rolling one — once one notification in a burst is let through, an
	// arbitrarily-fast burst inside the same GroupWindow always collapses
	// to that single one, and the window resets from whichever
	// notification is the next one Decide lets through.
	LastGroupAnchorAt *time.Time
	// GroupWindow is how long a burst collapses for. <= 0 disables
	// grouping entirely (treated as "not configured").
	GroupWindow time.Duration
}

// Decide resolves a candidate notification against expiry, preference,
// quiet-hours and frequency-cap rules, in that order — matching README
// §6.8's canonical pipeline ("TTL check -> quiet-hours check -> frequency-
// cap check", which runs after dedupe; the caller is responsible for having
// already applied dedupe via source_event_id's UNIQUE constraint before
// ever constructing a DecisionInput). It returns "" when the notification
// should be delivered — the caller then attempts push and records
// OutcomeSent/OutcomeSkippedNoDevice/OutcomeSendFailed based on what
// actually happened, which Decide cannot know in advance.
func Decide(in DecisionInput) PushOutcome {
	if !in.ExpiresAt.IsZero() && !in.Now.Before(in.ExpiresAt) {
		// "Expired notifications are dropped before provider delivery"
		// (README §6.8) — checked first, ahead of preference/quiet-hours/
		// frequency-cap, since none of those gates are meaningful once the
		// notification is already past the point of being deliverable.
		return OutcomeExpired
	}
	if !in.Preferences.categoryEnabled(in.Type.Category()) {
		return OutcomeSuppressedPreference
	}
	if !in.Type.QuietHoursExempt() && in.Preferences.InQuietHours(in.Now) {
		return OutcomeSuppressedQuietHours
	}
	if in.Type.FrequencyCapped() && in.FrequencyCapMax > 0 && in.RecentFrequencyCappedCount >= in.FrequencyCapMax {
		return OutcomeSuppressedFrequencyCap
	}
	if in.Type.Groupable() && in.GroupWindow > 0 && in.LastGroupAnchorAt != nil &&
		in.Now.Before(in.LastGroupAnchorAt.Add(in.GroupWindow)) {
		return OutcomeSuppressedGrouped
	}
	return ""
}
