package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/notification"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
)

func newNotificationPreferencesFixture(t *testing.T) (context.Context, *pgxpool.Pool, string) {
	t.Helper()
	dsn := os.Getenv("LINKUP_TEST_DATABASE_URL")
	if dsn == "" || os.Getenv("LINKUP_TEST_DATABASE_DESTRUCTIVE") != "1" {
		t.Skip("disposable PostgreSQL requires LINKUP_TEST_DATABASE_URL and LINKUP_TEST_DATABASE_DESTRUCTIVE=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := migrate.Apply(ctx, pool, migrationDir(t)); err != nil {
		t.Fatal(err)
	}

	accountService, err := account.NewService(NewAccountStore(pool), password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	user := registerIntegrationUser(t, ctx, accountService, "np", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{user.User.ID}) })

	return ctx, pool, user.User.ID
}

func TestNotificationPreferencesStoreGetReturnsDefaultsWhenUnset(t *testing.T) {
	ctx, pool, userID := newNotificationPreferencesFixture(t)
	store, err := NewNotificationPreferencesStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if got != notification.DefaultStoredPreferences() {
		t.Fatalf("expected defaults, got %#v", got)
	}
}

func TestNotificationPreferencesStoreUpdateThenGetRoundTrips(t *testing.T) {
	ctx, pool, userID := newNotificationPreferencesFixture(t)
	store, err := NewNotificationPreferencesStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	in := notification.StoredPreferences{
		SocialEnabled: false, EventEnabled: true, RecommendationEnabled: false, PromoEnabled: true,
		QuietHoursStartMinute: 23 * 60, QuietHoursEndMinute: 6 * 60, TimezoneName: "Europe/Kyiv",
	}
	if err := store.Update(ctx, userID, in); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if got != in {
		t.Fatalf("Get after Update mismatch: got %#v, want %#v", got, in)
	}
}

func TestNotificationPreferencesStoreUpdateIsIdempotentUpsert(t *testing.T) {
	ctx, pool, userID := newNotificationPreferencesFixture(t)
	store, err := NewNotificationPreferencesStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	first := notification.StoredPreferences{
		SocialEnabled: true, EventEnabled: true, RecommendationEnabled: true, PromoEnabled: true,
		QuietHoursStartMinute: 22 * 60, QuietHoursEndMinute: 7 * 60, TimezoneName: "UTC",
	}
	if err := store.Update(ctx, userID, first); err != nil {
		t.Fatal(err)
	}
	second := notification.StoredPreferences{
		SocialEnabled: false, EventEnabled: false, RecommendationEnabled: false, PromoEnabled: false,
		QuietHoursStartMinute: 0, QuietHoursEndMinute: 0, TimezoneName: "America/New_York",
	}
	if err := store.Update(ctx, userID, second); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if got != second {
		t.Fatalf("second Update should overwrite the first: got %#v, want %#v", got, second)
	}

	var rowCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notification_preferences WHERE user_id=$1`, userID).Scan(&rowCount); err != nil {
		t.Fatal(err)
	}
	if rowCount != 1 {
		t.Fatalf("expected exactly one row after two updates (upsert), got %d", rowCount)
	}
}
