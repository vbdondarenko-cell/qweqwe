package notification

import (
	"context"
	"errors"
	"testing"
)

type fakePreferencesStore struct {
	saved map[string]StoredPreferences
	err   error
}

func newFakePreferencesStore() *fakePreferencesStore {
	return &fakePreferencesStore{saved: make(map[string]StoredPreferences)}
}

func (f *fakePreferencesStore) Get(_ context.Context, userID string) (StoredPreferences, error) {
	if f.err != nil {
		return StoredPreferences{}, f.err
	}
	if p, ok := f.saved[userID]; ok {
		return p, nil
	}
	return DefaultStoredPreferences(), nil
}

func (f *fakePreferencesStore) Update(_ context.Context, userID string, prefs StoredPreferences) error {
	if f.err != nil {
		return f.err
	}
	f.saved[userID] = prefs
	return nil
}

func TestPreferencesServiceGetReturnsDefaultsWhenUnset(t *testing.T) {
	svc, err := NewPreferencesService(newFakePreferencesStore())
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if got != DefaultStoredPreferences() {
		t.Fatalf("expected defaults, got %#v", got)
	}
}

func TestPreferencesServiceUpdateThenGetRoundTrips(t *testing.T) {
	svc, err := NewPreferencesService(newFakePreferencesStore())
	if err != nil {
		t.Fatal(err)
	}
	in := StoredPreferences{
		SocialEnabled: false, EventEnabled: true, RecommendationEnabled: false, PromoEnabled: false,
		QuietHoursStartMinute: 22 * 60, QuietHoursEndMinute: 7 * 60, TimezoneName: "Europe/Kyiv",
	}
	saved, err := svc.Update(context.Background(), "user-1", in)
	if err != nil {
		t.Fatal(err)
	}
	if saved != in {
		t.Fatalf("Update did not return what was saved: %#v", saved)
	}
	got, err := svc.Get(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if got != in {
		t.Fatalf("Get after Update mismatch: %#v", got)
	}
}

func TestPreferencesServiceUpdateRejectsInvalidInput(t *testing.T) {
	svc, err := NewPreferencesService(newFakePreferencesStore())
	if err != nil {
		t.Fatal(err)
	}
	cases := []StoredPreferences{
		{QuietHoursStartMinute: -1, QuietHoursEndMinute: 0, TimezoneName: "UTC"},
		{QuietHoursStartMinute: 1440, QuietHoursEndMinute: 0, TimezoneName: "UTC"},
		{QuietHoursStartMinute: 0, QuietHoursEndMinute: 1440, TimezoneName: "UTC"},
		{QuietHoursStartMinute: 0, QuietHoursEndMinute: 0, TimezoneName: ""},
		{QuietHoursStartMinute: 0, QuietHoursEndMinute: 0, TimezoneName: "Not/A/Real/Zone"},
	}
	for _, c := range cases {
		if _, err := svc.Update(context.Background(), "user-1", c); !errors.Is(err, ErrInvalidPreferences) {
			t.Errorf("case %#v: expected ErrInvalidPreferences, got %v", c, err)
		}
	}
}

func TestPreferencesServiceRejectsEmptyUserID(t *testing.T) {
	svc, err := NewPreferencesService(newFakePreferencesStore())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(context.Background(), "  "); !errors.Is(err, ErrInvalidPreferences) {
		t.Fatalf("expected ErrInvalidPreferences, got %v", err)
	}
	if _, err := svc.Update(context.Background(), "", DefaultStoredPreferences()); !errors.Is(err, ErrInvalidPreferences) {
		t.Fatalf("expected ErrInvalidPreferences, got %v", err)
	}
}

func TestNewPreferencesServiceRejectsNilStore(t *testing.T) {
	if _, err := NewPreferencesService(nil); !errors.Is(err, ErrInvalidPreferences) {
		t.Fatalf("expected ErrInvalidPreferences, got %v", err)
	}
}
