// Package notification implements the pure decision logic of README §6.8's
// canonical notification pipeline:
//
//	domain transaction
//	-> transactional outbox            (db/migrations/000014_v11_realtime_outbox.sql)
//	-> notification projector/worker   (internal/postgres.NotificationProjector)
//	-> dedupe(idempotency key)         (notification_deliveries.source_event_id UNIQUE)
//	-> TTL check                       (Decide)
//	-> quiet-hours check                (Decide / Preferences.InQuietHours)
//	-> frequency-cap check               (Decide)
//	-> FCM/APNs adapter                 (internal/push.Service.NotifyUser)
//
// This package owns everything between "dedupe" and the adapter call: it has
// no database or network dependency, so every rule README §6.8 states as
// "must be verified" can be exercised with plain unit tests against
// deterministic inputs, independent of Postgres or a push provider being
// reachable.
package notification

import "time"

// Type is one of README §6.8's canonical notification types.
type Type string

const (
	TypeMessage             Type = "MESSAGE"
	TypeEvent               Type = "EVENT"
	TypeEventRecommendation Type = "EVENT_RECOMMENDATION"
	TypeEventReminder       Type = "EVENT_REMINDER"
	TypeFriendRequest       Type = "FRIEND_REQUEST"
	TypeFriendAccepted      Type = "FRIEND_ACCEPTED"
	TypeSystem              Type = "SYSTEM"
	TypeSecurity            Type = "SECURITY"
	TypeAccount             Type = "ACCOUNT"
	TypePromo               Type = "PROMO"
	TypeAdvertisement       Type = "ADVERTISEMENT"
)

// Category is the toggle group README §6.8 describes: "category toggles
// exist for social/event/recommendation/promo classes where policy
// permits disabling". Security/account/system notifications are
// deliberately not user-toggleable and get their own category so a
// preference-driven default can never silence them.
type Category string

const (
	CategorySocial         Category = "social"
	CategoryEvent          Category = "event"
	CategoryRecommendation Category = "recommendation"
	CategoryPromo          Category = "promo"
	CategoryCritical       Category = "critical"
)

// Category maps a Type to its toggle group. An unrecognized Type
// fail-closes to CategoryCritical: unknown notification types are never
// silently disable-able, matching this package's fail-closed default
// throughout (mirrors the capability registry's own fail-closed-on-unknown
// rule in internal/capability).
func (t Type) Category() Category {
	switch t {
	case TypeMessage, TypeFriendRequest, TypeFriendAccepted:
		return CategorySocial
	case TypeEvent, TypeEventReminder:
		return CategoryEvent
	case TypeEventRecommendation:
		return CategoryRecommendation
	case TypePromo, TypeAdvertisement:
		return CategoryPromo
	case TypeSecurity, TypeAccount, TypeSystem:
		return CategoryCritical
	default:
		return CategoryCritical
	}
}

// FrequencyCapped reports whether Type is subject to the recommendation/
// promo frequency cap. README §6.8: "recommendation and promo classes have
// explicit frequency caps" and "MESSAGE and SECURITY are not frequency-
// capped by marketing/recommendation limits" — generalized here to every
// category except the two the cap exists for, since nothing in README
// describes a cap on EVENT/social/critical traffic either.
func (t Type) FrequencyCapped() bool {
	switch t.Category() {
	case CategoryRecommendation, CategoryPromo:
		return true
	default:
		return false
	}
}

// QuietHoursExempt reports whether Type bypasses quiet hours entirely.
// Only the critical category (SECURITY/ACCOUNT/SYSTEM) is exempt; README's
// default quiet-hours description is scoped to "non-critical notifications".
func (t Type) QuietHoursExempt() bool {
	return t.Category() == CategoryCritical
}

// DeepLink returns the canonical deep link for a notification, matching
// README §6.8's routing table exactly. slotID is ignored (and may be
// empty) for the link shapes that don't reference one.
func DeepLink(t Type, slotID string) string {
	switch t {
	case TypeMessage:
		return "app://chat/" + slotID
	case TypeEvent, TypeEventReminder:
		return "app://slot/" + slotID
	case TypeEventRecommendation:
		return "app://pulse?slotId=" + slotID
	case TypeFriendRequest:
		return "app://me/requests"
	case TypeFriendAccepted:
		return "app://me/connections"
	case TypeSecurity:
		return "app://me/security"
	case TypeAccount:
		return "app://me/account"
	case TypePromo, TypeAdvertisement:
		return "app://linkup-plus"
	default:
		return "app://home"
	}
}

// Preferences is a user's durable notification configuration. The zero
// value is deliberately NOT safe to use directly for delivery decisions —
// use DefaultPreferences() — because a zero-value Preferences would read as
// "every category disabled", the opposite of this feature's fail-open-for-
// non-critical-categories default (only SECURITY/ACCOUNT/SYSTEM fail
// closed-to-always-on, everything else defaults to enabled per README
// §6.8's "category toggles ... where policy permits disabling").
type Preferences struct {
	SocialEnabled         bool
	EventEnabled          bool
	RecommendationEnabled bool
	PromoEnabled          bool
	QuietHoursStartMinute int // 0..1439, minute-of-day quiet hours begin
	QuietHoursEndMinute   int // 0..1439, minute-of-day quiet hours end
	Location              *time.Location
}

// DefaultPreferences matches the notification_preferences table's own
// column defaults and README §6.8's stated default quiet hours
// (23:00-08:00 local time).
func DefaultPreferences() Preferences {
	return Preferences{
		SocialEnabled:         true,
		EventEnabled:          true,
		RecommendationEnabled: true,
		PromoEnabled:          true,
		QuietHoursStartMinute: 23 * 60,
		QuietHoursEndMinute:   8 * 60,
		Location:              time.UTC,
	}
}

func (p Preferences) categoryEnabled(c Category) bool {
	switch c {
	case CategorySocial:
		return p.SocialEnabled
	case CategoryEvent:
		return p.EventEnabled
	case CategoryRecommendation:
		return p.RecommendationEnabled
	case CategoryPromo:
		return p.PromoEnabled
	default:
		// CategoryCritical (and anything unrecognized): never toggle-able.
		return true
	}
}

// InQuietHours reports whether `now` falls inside the configured quiet-hours
// window, evaluated in the user's own IANA location so the window is
// DST-safe (README §6.8: "quiet hours use the user's current configured
// timezone and must be DST-safe") — converting `now` into p.Location and
// comparing minute-of-day handles a DST transition correctly because the
// wall-clock minute-of-day is what quiet hours are defined in terms of, not
// an elapsed-duration offset from UTC. A zero-width window (start == end)
// is treated as "quiet hours disabled" rather than "always quiet" or
// "never quiet", since either of those silent interpretations of a
// degenerate configuration would be surprising.
func (p Preferences) InQuietHours(now time.Time) bool {
	loc := p.Location
	if loc == nil {
		loc = time.UTC
	}
	local := now.In(loc)
	minute := local.Hour()*60 + local.Minute()
	start, end := p.QuietHoursStartMinute, p.QuietHoursEndMinute
	if start == end {
		return false
	}
	if start < end {
		return minute >= start && minute < end
	}
	// Window wraps past midnight, e.g. the 23:00-08:00 default.
	return minute >= start || minute < end
}
