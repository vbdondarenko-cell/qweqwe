package notification

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrInvalidPreferences = errors.New("invalid notification preferences")

// StoredPreferences is a user's notification preferences as configured and
// returned by the CRUD API — the storage/wire shape (README §6.8:
// "preference changes take effect server-side for future projections/
// delivery decisions"). It carries the raw configured IANA zone name rather
// than a resolved *time.Location: NotificationProjector reads the same
// notification_preferences table but resolves TimezoneName into a live
// Preferences.Location itself (see postgres.NotificationProjector.
// loadPreferences), which is the shape Decide actually needs. Keeping the
// two separate means this CRUD surface can round-trip exactly what the user
// configured without forcing every caller to carry a *time.Location.
type StoredPreferences struct {
	SocialEnabled         bool   `json:"socialEnabled"`
	EventEnabled          bool   `json:"eventEnabled"`
	RecommendationEnabled bool   `json:"recommendationEnabled"`
	PromoEnabled          bool   `json:"promoEnabled"`
	QuietHoursStartMinute int    `json:"quietHoursStartMinute"`
	QuietHoursEndMinute   int    `json:"quietHoursEndMinute"`
	TimezoneName          string `json:"timezoneName"`
}

// DefaultStoredPreferences mirrors notification_preferences' own column
// defaults and DefaultPreferences() exactly, so a user who has never saved
// preferences sees the same values the projector already assumes for them.
func DefaultStoredPreferences() StoredPreferences {
	d := DefaultPreferences()
	return StoredPreferences{
		SocialEnabled:         d.SocialEnabled,
		EventEnabled:          d.EventEnabled,
		RecommendationEnabled: d.RecommendationEnabled,
		PromoEnabled:          d.PromoEnabled,
		QuietHoursStartMinute: d.QuietHoursStartMinute,
		QuietHoursEndMinute:   d.QuietHoursEndMinute,
		TimezoneName:          "UTC",
	}
}

// Validate enforces exactly what notification_preferences' own CHECK
// constraints enforce (db/migrations/000027), plus confirming the timezone
// name actually resolves — a value that fails to load here would otherwise
// only be caught later, at projection time, where NotificationProjector
// fails closed to UTC rather than blocking delivery (see loadPreferences).
// Catching it here means a user seldom experiences the "silently downgraded
// to UTC" behavior at all: bad input is refused up front instead.
func (p StoredPreferences) Validate() error {
	if p.QuietHoursStartMinute < 0 || p.QuietHoursStartMinute > 1439 {
		return ErrInvalidPreferences
	}
	if p.QuietHoursEndMinute < 0 || p.QuietHoursEndMinute > 1439 {
		return ErrInvalidPreferences
	}
	name := strings.TrimSpace(p.TimezoneName)
	if len(name) < 1 || len(name) > 80 {
		return ErrInvalidPreferences
	}
	if _, err := time.LoadLocation(name); err != nil {
		return ErrInvalidPreferences
	}
	return nil
}

// PreferencesStore is implemented by the postgres package.
type PreferencesStore interface {
	// Get returns DefaultStoredPreferences() when the user has never saved
	// preferences, never an error for that case: "no row yet" is not a
	// failure, it is every user's actual starting state.
	Get(ctx context.Context, userID string) (StoredPreferences, error)
	Update(ctx context.Context, userID string, prefs StoredPreferences) error
}

type PreferencesService struct {
	store PreferencesStore
}

func NewPreferencesService(store PreferencesStore) (*PreferencesService, error) {
	if store == nil {
		return nil, ErrInvalidPreferences
	}
	return &PreferencesService{store: store}, nil
}

func (s *PreferencesService) Get(ctx context.Context, userID string) (StoredPreferences, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return StoredPreferences{}, ErrInvalidPreferences
	}
	return s.store.Get(ctx, userID)
}

// Update validates before writing: an invalid submission never reaches the
// store, and the store is therefore never responsible for enforcing shape
// rules Validate already covers.
func (s *PreferencesService) Update(ctx context.Context, userID string, prefs StoredPreferences) (StoredPreferences, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return StoredPreferences{}, ErrInvalidPreferences
	}
	if err := prefs.Validate(); err != nil {
		return StoredPreferences{}, err
	}
	prefs.TimezoneName = strings.TrimSpace(prefs.TimezoneName)
	if err := s.store.Update(ctx, userID, prefs); err != nil {
		return StoredPreferences{}, err
	}
	return prefs, nil
}
