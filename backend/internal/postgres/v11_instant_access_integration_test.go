package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

func TestV11InstantJoinCapacityRaceIntegration(t *testing.T) {
	dsn := os.Getenv("LINKUP_TEST_DATABASE_URL")
	if dsn == "" || os.Getenv("LINKUP_TEST_DATABASE_DESTRUCTIVE") != "1" {
		t.Skip("disposable PostgreSQL requires LINKUP_TEST_DATABASE_URL and LINKUP_TEST_DATABASE_DESTRUCTIVE=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := migrate.Apply(ctx, pool, migrationDir(t)); err != nil {
		t.Fatal(err)
	}

	accounts, err := account.NewService(NewAccountStore(pool), password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	host := registerIntegrationUser(t, ctx, accounts, "ih", suffix)
	joiners := []account.AuthResult{
		registerIntegrationUser(t, ctx, accounts, "i1", suffix),
		registerIntegrationUser(t, ctx, accounts, "i2", suffix),
		registerIntegrationUser(t, ctx, accounts, "i3", suffix),
	}

	base, err := NewSlotStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewV11SlotStore(base, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	service, err := slot.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	instant := slot.AccessInstant
	draft, err := service.CreateDraft(ctx, host.User.ID, slot.CreateInput{
		Title: "Instant race", Activity: "coffee", PlaceText: "Center", Capacity: 2, AccessMode: &instant,
	}, "v11-instant-draft-race-0001")
	if err != nil {
		t.Fatal(err)
	}
	published, err := service.PublishDraft(ctx, host.User.ID, draft.ID, draft.Version, "v11-instant-publish-race-0001")
	if err != nil {
		t.Fatal(err)
	}
	if published.AccessMode != slot.AccessInstant || published.State != slot.StateFilling {
		t.Fatalf("instant draft did not publish correctly: %#v", published)
	}

	start := make(chan struct{})
	results := make(chan error, len(joiners))
	var wg sync.WaitGroup
	for i, user := range joiners {
		wg.Add(1)
		go func(index int, actor string) {
			defer wg.Done()
			<-start
			_, err := service.Join(context.Background(), actor, draft.ID, fmt.Sprintf("v11-instant-join-race-%04d", index))
			results <- err
		}(i, user.User.ID)
	}
	close(start)
	wg.Wait()
	close(results)

	success, full := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, slot.ErrCapacityFull):
			full++
		default:
			t.Fatalf("unexpected concurrent join error: %v", err)
		}
	}
	if success != 2 || full != 1 {
		t.Fatalf("instant capacity race mismatch: success=%d full=%d", success, full)
	}

	var acceptedCount, memberships int
	var state string
	if err := pool.QueryRow(ctx, `SELECT accepted_count,state FROM slots WHERE id=$1`, draft.ID).Scan(&acceptedCount, &state); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM slot_memberships WHERE slot_id=$1`, draft.ID).Scan(&memberships); err != nil {
		t.Fatal(err)
	}
	if acceptedCount != 2 || memberships != 2 || state != string(slot.StateFull) {
		t.Fatalf("instant join oversubscription invariant failed: accepted=%d memberships=%d state=%s", acceptedCount, memberships, state)
	}
}
