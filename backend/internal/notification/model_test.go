package notification

import (
	"testing"
	"time"
)

func mustDate(t *testing.T, year int, month time.Month, day, hour, minute int, loc *time.Location) time.Time {
	t.Helper()
	return time.Date(year, month, day, hour, minute, 0, 0, loc)
}

func TestCategoryMapping(t *testing.T) {
	cases := []struct {
		typ  Type
		want Category
	}{
		{TypeMessage, CategorySocial},
		{TypeFriendRequest, CategorySocial},
		{TypeFriendAccepted, CategorySocial},
		{TypeEvent, CategoryEvent},
		{TypeEventReminder, CategoryEvent},
		{TypeEventRecommendation, CategoryRecommendation},
		{TypePromo, CategoryPromo},
		{TypeAdvertisement, CategoryPromo},
		{TypeSecurity, CategoryCritical},
		{TypeAccount, CategoryCritical},
		{TypeSystem, CategoryCritical},
		{Type("UNKNOWN_FUTURE_TYPE"), CategoryCritical},
	}
	for _, c := range cases {
		if got := c.typ.Category(); got != c.want {
			t.Errorf("%s.Category() = %s, want %s", c.typ, got, c.want)
		}
	}
}

func TestFrequencyCappedOnlyRecommendationAndPromo(t *testing.T) {
	capped := map[Type]bool{
		TypeMessage: false, TypeEvent: false, TypeEventReminder: false,
		TypeFriendRequest: false, TypeFriendAccepted: false,
		TypeSecurity: false, TypeAccount: false, TypeSystem: false,
		TypeEventRecommendation: true, TypePromo: true, TypeAdvertisement: true,
	}
	for typ, want := range capped {
		if got := typ.FrequencyCapped(); got != want {
			t.Errorf("%s.FrequencyCapped() = %v, want %v", typ, got, want)
		}
	}
}

func TestGroupableOnlyMessageAndEvent(t *testing.T) {
	groupable := map[Type]bool{
		TypeMessage: true, TypeEvent: true,
		TypeEventReminder: false, TypeFriendRequest: false, TypeFriendAccepted: false,
		TypeSecurity: false, TypeAccount: false, TypeSystem: false,
		TypeEventRecommendation: false, TypePromo: false, TypeAdvertisement: false,
	}
	for typ, want := range groupable {
		if got := typ.Groupable(); got != want {
			t.Errorf("%s.Groupable() = %v, want %v", typ, got, want)
		}
	}
}

func TestQuietHoursExemptOnlyCritical(t *testing.T) {
	for _, typ := range []Type{TypeSecurity, TypeAccount, TypeSystem} {
		if !typ.QuietHoursExempt() {
			t.Errorf("%s should be quiet-hours exempt", typ)
		}
	}
	for _, typ := range []Type{TypeMessage, TypeEvent, TypeEventReminder, TypeEventRecommendation, TypeFriendRequest, TypeFriendAccepted, TypePromo, TypeAdvertisement} {
		if typ.QuietHoursExempt() {
			t.Errorf("%s should not be quiet-hours exempt", typ)
		}
	}
}

func TestDeepLinkMatchesReadmeRoutingTable(t *testing.T) {
	cases := []struct {
		typ    Type
		slotID string
		want   string
	}{
		{TypeMessage, "slot-1", "app://chat/slot-1"},
		{TypeEvent, "slot-1", "app://slot/slot-1"},
		{TypeEventReminder, "slot-1", "app://slot/slot-1"},
		{TypeEventRecommendation, "slot-1", "app://pulse?slotId=slot-1"},
		{TypeFriendRequest, "", "app://me/requests"},
		{TypeFriendAccepted, "", "app://me/connections"},
		{TypeSecurity, "", "app://me/security"},
		{TypeAccount, "", "app://me/account"},
		{TypePromo, "", "app://linkup-plus"},
		{TypeAdvertisement, "", "app://linkup-plus"},
	}
	for _, c := range cases {
		if got := DeepLink(c.typ, c.slotID); got != c.want {
			t.Errorf("DeepLink(%s,%q) = %q, want %q", c.typ, c.slotID, got, c.want)
		}
	}
}

func TestInQuietHoursDefaultWraparoundWindow(t *testing.T) {
	prefs := DefaultPreferences() // 23:00-08:00 UTC
	cases := []struct {
		hour, minute int
		want         bool
	}{
		{22, 59, false},
		{23, 0, true},
		{23, 59, true},
		{0, 0, true},
		{3, 30, true},
		{7, 59, true},
		{8, 0, false},
		{12, 0, false},
	}
	for _, c := range cases {
		now := mustDate(t, 2026, 3, 10, c.hour, c.minute, time.UTC)
		if got := prefs.InQuietHours(now); got != c.want {
			t.Errorf("InQuietHours at %02d:%02d = %v, want %v", c.hour, c.minute, got, c.want)
		}
	}
}

func TestInQuietHoursZeroWidthWindowMeansDisabled(t *testing.T) {
	prefs := DefaultPreferences()
	prefs.QuietHoursStartMinute = 500
	prefs.QuietHoursEndMinute = 500
	now := mustDate(t, 2026, 3, 10, 8, 20, time.UTC) // 500 minutes past midnight
	if prefs.InQuietHours(now) {
		t.Fatal("zero-width quiet-hours window must be treated as disabled")
	}
}

func TestInQuietHoursNonWraparoundWindow(t *testing.T) {
	prefs := DefaultPreferences()
	prefs.QuietHoursStartMinute = 13 * 60 // 13:00
	prefs.QuietHoursEndMinute = 15 * 60   // 15:00
	if prefs.InQuietHours(mustDate(t, 2026, 3, 10, 12, 59, time.UTC)) {
		t.Fatal("12:59 must be outside 13:00-15:00")
	}
	if !prefs.InQuietHours(mustDate(t, 2026, 3, 10, 14, 0, time.UTC)) {
		t.Fatal("14:00 must be inside 13:00-15:00")
	}
	if prefs.InQuietHours(mustDate(t, 2026, 3, 10, 15, 0, time.UTC)) {
		t.Fatal("15:00 must be outside 13:00-15:00 (end is exclusive)")
	}
}

func TestInQuietHoursIsDSTSafe(t *testing.T) {
	kyiv, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		t.Skipf("Europe/Kyiv tzdata unavailable: %v", err)
	}
	prefs := DefaultPreferences()
	prefs.Location = kyiv

	// Ukraine's spring-forward transition (clocks jump 03:00 -> 04:00 local)
	// historically falls on the last Sunday in March. Evaluate the same
	// wall-clock local hour (23:30) on either side of the transition; both
	// must read as inside quiet hours because InQuietHours works in local
	// wall-clock minutes, not a fixed UTC offset that a DST shift would
	// silently invalidate.
	before := time.Date(2026, 3, 28, 23, 30, 0, 0, kyiv)
	after := time.Date(2026, 3, 29, 23, 30, 0, 0, kyiv)
	if !prefs.InQuietHours(before) {
		t.Fatal("23:30 local before DST transition must be in quiet hours")
	}
	if !prefs.InQuietHours(after) {
		t.Fatal("23:30 local after DST transition must be in quiet hours")
	}
	// The same instant in UTC differs by an hour across the transition,
	// which is exactly the failure mode a fixed-UTC-offset implementation
	// would get wrong; asserting both is the point of this test.
	if before.UTC().Hour() == after.UTC().Hour() {
		t.Fatal("test fixture did not actually cross a DST transition")
	}
}
