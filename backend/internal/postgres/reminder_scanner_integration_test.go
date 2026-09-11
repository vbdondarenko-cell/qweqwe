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
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

type reminderFixture struct {
	ctx         context.Context
	pool        *pgxpool.Pool
	slotService *slot.Service
	host        account.AuthResult
	memberA     account.AuthResult
	memberB     account.AuthResult
}

func newReminderFixture(t *testing.T) *reminderFixture {
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
	host := registerIntegrationUser(t, ctx, accountService, "rh", suffix)
	memberA := registerIntegrationUser(t, ctx, accountService, "ra", suffix)
	memberB := registerIntegrationUser(t, ctx, accountService, "rb", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{host.User.ID, memberA.User.ID, memberB.User.ID}) })

	baseStore, err := NewSlotStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	slotService, err := slot.NewService(baseStore)
	if err != nil {
		t.Fatal(err)
	}

	return &reminderFixture{ctx: ctx, pool: pool, slotService: slotService, host: host, memberA: memberA, memberB: memberB}
}

func (f *reminderFixture) createApprovedSlot(t *testing.T, startAt time.Time, suffix string) slot.Slot {
	t.Helper()
	created, err := f.slotService.Create(f.ctx, f.host.User.ID, slot.CreateInput{
		Title: "Reminder " + suffix, Activity: "coffee", PlaceText: "Center", StartAt: &startAt, Capacity: 3,
	}, "reminder-create-"+suffix)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []account.AuthResult{f.memberA, f.memberB} {
		if _, err := f.slotService.Request(f.ctx, m.User.ID, created.ID, "reminder-request-"+suffix+"-"+m.User.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := f.slotService.Approve(f.ctx, f.host.User.ID, created.ID, m.User.ID, "reminder-approve-"+suffix+"-"+m.User.ID); err != nil {
			t.Fatal(err)
		}
	}
	return created
}

func countStartingSoonEvents(t *testing.T, ctx context.Context, pool *pgxpool.Pool, slotID string) []string {
	t.Helper()
	rows, err := pool.Query(ctx, `
		SELECT subject_user_id::text FROM domain_outbox_events
		WHERE event_type='slot.starting_soon' AND slot_id=$1::uuid`, slotID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var recipients []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			t.Fatal(err)
		}
		recipients = append(recipients, userID)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return recipients
}

func TestReminderScannerEmitsOncePerRecipientAndIsIdempotent(t *testing.T) {
	f := newReminderFixture(t)
	scanner, err := NewReminderScanner(f.pool, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	due := f.createApprovedSlot(t, time.Now().UTC().Add(20*time.Minute), "due-0001")

	claimed, err := scanner.ScanAndEmit(f.ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if claimed != 1 {
		t.Fatalf("expected exactly 1 Slot claimed, got %d", claimed)
	}

	recipients := countStartingSoonEvents(t, f.ctx, f.pool, due.ID)
	want := map[string]bool{f.host.User.ID: true, f.memberA.User.ID: true, f.memberB.User.ID: true}
	if len(recipients) != 3 {
		t.Fatalf("expected 3 starting_soon events (host + 2 members), got %#v", recipients)
	}
	for _, r := range recipients {
		if !want[r] {
			t.Errorf("unexpected recipient %s", r)
		}
		delete(want, r)
	}
	if len(want) != 0 {
		t.Fatalf("missing recipients: %#v", want)
	}

	// Re-scanning must not claim the same Slot again or duplicate events.
	claimedAgain, err := scanner.ScanAndEmit(f.ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if claimedAgain != 0 {
		t.Fatalf("expected 0 newly claimed Slots on re-scan, got %d", claimedAgain)
	}
	recipientsAfter := countStartingSoonEvents(t, f.ctx, f.pool, due.ID)
	if len(recipientsAfter) != 3 {
		t.Fatalf("re-scan must not duplicate events: got %d, want 3", len(recipientsAfter))
	}
}

func TestReminderScannerIgnoresSlotOutsideLeadTime(t *testing.T) {
	f := newReminderFixture(t)
	scanner, err := NewReminderScanner(f.pool, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	farFuture := f.createApprovedSlot(t, time.Now().UTC().Add(2*time.Hour), "far-0001")

	claimed, err := scanner.ScanAndEmit(f.ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if claimed != 0 {
		t.Fatalf("a Slot outside the lead-time window must not be claimed, got claimed=%d", claimed)
	}
	if recipients := countStartingSoonEvents(t, f.ctx, f.pool, farFuture.ID); len(recipients) != 0 {
		t.Fatalf("no events should exist for a Slot outside the window: %#v", recipients)
	}
}

func TestReminderScannerNeverRemindsAfterStartAtHasPassed(t *testing.T) {
	f := newReminderFixture(t)
	scanner, err := NewReminderScanner(f.pool, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	// A Slot whose start_at is already in the past (e.g. the service was
	// down through the reminder window) must never be reminded late: this
	// is a "starting soon" notice, not a historical record.
	past := f.createApprovedSlot(t, time.Now().UTC().Add(-5*time.Minute), "past-0001")

	claimed, err := scanner.ScanAndEmit(f.ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if claimed != 0 {
		t.Fatalf("a Slot whose start_at already passed must not be claimed, got claimed=%d", claimed)
	}
	if recipients := countStartingSoonEvents(t, f.ctx, f.pool, past.ID); len(recipients) != 0 {
		t.Fatalf("no late reminder should ever be emitted: %#v", recipients)
	}
}

func TestReminderScannerRejectsInvalidConstruction(t *testing.T) {
	if _, err := NewReminderScanner(nil, time.Minute); err == nil {
		t.Fatal("expected error for nil pool")
	}
	f := newReminderFixture(t)
	if _, err := NewReminderScanner(f.pool, 0); err == nil {
		t.Fatal("expected error for zero lead time")
	}
	scanner, err := NewReminderScanner(f.pool, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scanner.ScanAndEmit(f.ctx, 0); err == nil {
		t.Fatal("expected error for zero scan limit")
	}
}
