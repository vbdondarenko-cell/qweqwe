package notification

import (
	"testing"
	"time"
)

func TestDecideDelivers(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC) // clearly outside default quiet hours
	got := Decide(DecisionInput{
		Type:        TypeEvent,
		Now:         now,
		ExpiresAt:   now.Add(time.Hour),
		Preferences: DefaultPreferences(),
	})
	if got != "" {
		t.Fatalf("expected deliver (empty outcome), got %q", got)
	}
}

func TestDecideExpiredTakesPriorityOverEverythingElse(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	got := Decide(DecisionInput{
		Type:        TypeEvent,
		Now:         now,
		ExpiresAt:   now.Add(-time.Second), // already past
		Preferences: DefaultPreferences(),
	})
	if got != OutcomeExpired {
		t.Fatalf("expected EXPIRED, got %q", got)
	}
}

func TestDecideZeroExpiryNeverExpires(t *testing.T) {
	got := Decide(DecisionInput{
		Type:        TypeSecurity,
		Now:         time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiresAt:   time.Time{}, // zero value: never expires
		Preferences: DefaultPreferences(),
	})
	if got != "" {
		t.Fatalf("zero ExpiresAt must never expire, got %q", got)
	}
}

func TestDecideSuppressedPreference(t *testing.T) {
	prefs := DefaultPreferences()
	prefs.EventEnabled = false
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	got := Decide(DecisionInput{Type: TypeEvent, Now: now, ExpiresAt: now.Add(time.Hour), Preferences: prefs})
	if got != OutcomeSuppressedPreference {
		t.Fatalf("expected SUPPRESSED_PREFERENCE, got %q", got)
	}
}

func TestDecideCriticalCategoryIgnoresPreferenceToggle(t *testing.T) {
	// There is no SecurityEnabled/AccountEnabled field at all — Preferences
	// simply has no way to disable the critical category. This test exists
	// so that if such a field were ever added, the missing wiring in
	// categoryEnabled would be caught rather than silently allowing
	// SECURITY to become toggleable (README §6.8: "SECURITY and critical
	// account-integrity notifications cannot be silently converted into
	// marketing controls").
	prefs := DefaultPreferences()
	prefs.SocialEnabled, prefs.EventEnabled, prefs.RecommendationEnabled, prefs.PromoEnabled = false, false, false, false
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	for _, typ := range []Type{TypeSecurity, TypeAccount, TypeSystem} {
		got := Decide(DecisionInput{Type: typ, Now: now, ExpiresAt: now.Add(time.Hour), Preferences: prefs})
		if got != "" {
			t.Fatalf("%s must deliver despite every toggle disabled, got %q", typ, got)
		}
	}
}

func TestDecideQuietHoursSuppressesNonCritical(t *testing.T) {
	now := time.Date(2026, 3, 10, 23, 30, 0, 0, time.UTC) // inside default 23:00-08:00
	got := Decide(DecisionInput{Type: TypeEvent, Now: now, ExpiresAt: now.Add(time.Hour), Preferences: DefaultPreferences()})
	if got != OutcomeSuppressedQuietHours {
		t.Fatalf("expected SUPPRESSED_QUIET_HOURS, got %q", got)
	}
}

func TestDecideQuietHoursNeverSuppressesCritical(t *testing.T) {
	now := time.Date(2026, 3, 10, 23, 30, 0, 0, time.UTC)
	for _, typ := range []Type{TypeSecurity, TypeAccount, TypeSystem} {
		got := Decide(DecisionInput{Type: typ, Now: now, ExpiresAt: now.Add(time.Hour), Preferences: DefaultPreferences()})
		if got != "" {
			t.Fatalf("%s must not be suppressed by quiet hours, got %q", typ, got)
		}
	}
}

func TestDecideFrequencyCap(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	base := DecisionInput{Type: TypeEventRecommendation, Now: now, ExpiresAt: now.Add(time.Hour), Preferences: DefaultPreferences(), FrequencyCapMax: 3}

	under := base
	under.RecentFrequencyCappedCount = 2
	if got := Decide(under); got != "" {
		t.Fatalf("under cap should deliver, got %q", got)
	}

	atCap := base
	atCap.RecentFrequencyCappedCount = 3
	if got := Decide(atCap); got != OutcomeSuppressedFrequencyCap {
		t.Fatalf("at cap should suppress, got %q", got)
	}
}

func TestDecideFrequencyCapDoesNotApplyToMessageOrSecurity(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	for _, typ := range []Type{TypeMessage, TypeSecurity} {
		got := Decide(DecisionInput{
			Type: typ, Now: now, ExpiresAt: now.Add(time.Hour), Preferences: DefaultPreferences(),
			RecentFrequencyCappedCount: 999, FrequencyCapMax: 1,
		})
		if got != "" {
			t.Fatalf("%s must never be frequency-capped, got %q", typ, got)
		}
	}
}

func TestDecideFrequencyCapMaxZeroOrNegativeDisablesCap(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	for _, max := range []int{0, -1} {
		got := Decide(DecisionInput{
			Type: TypePromo, Now: now, ExpiresAt: now.Add(time.Hour), Preferences: DefaultPreferences(),
			RecentFrequencyCappedCount: 1_000_000, FrequencyCapMax: max,
		})
		if got != "" {
			t.Fatalf("FrequencyCapMax=%d must disable the cap, got %q", max, got)
		}
	}
}

func TestDecidePrecedenceExpiryBeforePreferenceBeforeQuietHoursBeforeCap(t *testing.T) {
	// A candidate that would fail every gate must report the first one,
	// consistently, regardless of which others also apply.
	prefs := DefaultPreferences()
	prefs.PromoEnabled = false
	now := time.Date(2026, 3, 10, 23, 30, 0, 0, time.UTC) // also inside quiet hours
	in := DecisionInput{
		Type: TypePromo, Now: now, ExpiresAt: now.Add(-time.Second), // also expired
		Preferences: prefs, RecentFrequencyCappedCount: 100, FrequencyCapMax: 1, // also over cap
	}
	if got := Decide(in); got != OutcomeExpired {
		t.Fatalf("expiry must take precedence over every other gate, got %q", got)
	}
	in.ExpiresAt = now.Add(time.Hour) // no longer expired
	if got := Decide(in); got != OutcomeSuppressedPreference {
		t.Fatalf("preference must take precedence over quiet-hours/cap once not expired, got %q", got)
	}
}
