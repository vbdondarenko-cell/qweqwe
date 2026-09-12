package postgres

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/capability"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/chat"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/push"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// recordingPusher is a test double for the unexported `pusher` interface:
// it records every call instead of reaching a real FCM/APNs provider.
type recordingPusher struct {
	mu    sync.Mutex
	calls []recordedPush
	err   error
}

type recordedPush struct {
	userID  string
	message push.Message
}

func (r *recordingPusher) NotifyUser(_ context.Context, userID string, message push.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.calls = append(r.calls, recordedPush{userID: userID, message: message})
	return nil
}

func (r *recordingPusher) callsFor(userID string) []recordedPush {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []recordedPush
	for _, c := range r.calls {
		if c.userID == userID {
			out = append(out, c)
		}
	}
	return out
}

// reset discards every recorded call so far — used when a test needs to
// drive unrelated setup events (e.g. slot approval, which pushes its own
// real "You're in!" notification) through the projector before asserting
// on calls a later, specific action produces.
func (r *recordingPusher) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = nil
}

func enableNotificationsForAll(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `UPDATE capability_registry
		SET enabled=true, scope_type='ALL', scope_user_ids='{}'::uuid[], reason='integration test'
		WHERE capability_key='notifications'`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `UPDATE capability_registry
			SET enabled=false, scope_type='ALL', scope_user_ids='{}'::uuid[], reason='integration reset'
			WHERE capability_key='notifications'`)
	})
}

func newNotificationFixture(t *testing.T) (context.Context, *pgxpool.Pool, *slot.Service, *capability.Service, account.AuthResult, account.AuthResult) {
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
	host := registerIntegrationUser(t, ctx, accountService, "nh", suffix)
	member := registerIntegrationUser(t, ctx, accountService, "nm", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{host.User.ID, member.User.ID}) })

	baseStore, err := NewSlotStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	slotService, err := slot.NewService(baseStore)
	if err != nil {
		t.Fatal(err)
	}

	capStore, err := NewCapabilityStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	capService, err := capability.NewService(capStore)
	if err != nil {
		t.Fatal(err)
	}

	return ctx, pool, slotService, capService, host, member
}

func TestNotificationProjectorApprovalAndRequestCreated(t *testing.T) {
	ctx, pool, slotService, capService, host, member := newNotificationFixture(t)
	enableNotificationsForAll(t, ctx, pool)

	created, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Notification Coffee", Activity: "coffee", PlaceText: "Center", Capacity: 2,
	}, "notif-create-0001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Request(ctx, member.User.ID, created.ID, "notif-request-0001"); err != nil {
		t.Fatal(err)
	}

	recorder := &recordingPusher{}
	projector, err := NewNotificationProjector(pool, capService, recorder, 14*24*time.Hour, 24*time.Hour, 5, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	processed := drainProjector(t, ctx, projector, 200)
	if processed == 0 {
		t.Fatal("expected at least the request_created event to be processed")
	}

	// The host should have received exactly one "New request" notification.
	hostCalls := recorder.callsFor(host.User.ID)
	if len(hostCalls) != 1 || hostCalls[0].message.Title != "New request" {
		t.Fatalf("expected host to receive one New request push, got %#v", hostCalls)
	}
	assertNotificationRow(t, ctx, pool, host.User.ID, "EVENT", "SENT", func(deepLink string) bool {
		return deepLink == "app://slot/"+created.ID
	})

	if _, err := slotService.Approve(ctx, host.User.ID, created.ID, member.User.ID, "notif-approve-0001"); err != nil {
		t.Fatal(err)
	}
	drainProjector(t, ctx, projector, 200)

	memberCalls := recorder.callsFor(member.User.ID)
	if len(memberCalls) != 1 || memberCalls[0].message.Title != "You're in!" {
		t.Fatalf("expected member to receive one You're in! push, got %#v", memberCalls)
	}
	assertNotificationRow(t, ctx, pool, member.User.ID, "EVENT", "SENT", func(deepLink string) bool {
		return deepLink == "app://slot/"+created.ID
	})

	// Re-running must not duplicate anything: no new events past the cursor.
	// A single bounded call suffices here — the drain above already caught
	// the cursor up to the current end of the outbox.
	processedAgain, err := projector.ProcessBatch(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	if processedAgain != 0 {
		t.Fatalf("expected 0 newly processed events on the second run, got %d", processedAgain)
	}
	if len(recorder.callsFor(host.User.ID)) != 1 || len(recorder.callsFor(member.User.ID)) != 1 {
		t.Fatal("re-running ProcessBatch must not duplicate push calls")
	}
}

func TestNotificationProjectorSuppressedByPreference(t *testing.T) {
	ctx, pool, slotService, capService, host, member := newNotificationFixture(t)
	enableNotificationsForAll(t, ctx, pool)

	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_preferences (user_id, event_enabled)
		VALUES ($1, false)`, host.User.ID); err != nil {
		t.Fatal(err)
	}

	created, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Preference Off", Activity: "coffee", PlaceText: "Center", Capacity: 2,
	}, "notif-pref-create-0001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Request(ctx, member.User.ID, created.ID, "notif-pref-request-0001"); err != nil {
		t.Fatal(err)
	}

	recorder := &recordingPusher{}
	projector, err := NewNotificationProjector(pool, capService, recorder, 14*24*time.Hour, 24*time.Hour, 5, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	drainProjector(t, ctx, projector, 200)

	if calls := recorder.callsFor(host.User.ID); len(calls) != 0 {
		t.Fatalf("host has EVENT notifications disabled, expected no push, got %#v", calls)
	}
	assertNotificationRow(t, ctx, pool, host.User.ID, "EVENT", "SUPPRESSED_PREFERENCE", nil)
}

func TestNotificationProjectorSuppressedByQuietHours(t *testing.T) {
	ctx, pool, slotService, capService, host, member := newNotificationFixture(t)
	enableNotificationsForAll(t, ctx, pool)

	created, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Quiet Hours", Activity: "coffee", PlaceText: "Center", Capacity: 2,
	}, "notif-quiet-create-0001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Request(ctx, member.User.ID, created.ID, "notif-quiet-request-0001"); err != nil {
		t.Fatal(err)
	}

	recorder := &recordingPusher{}
	projector, err := NewNotificationProjector(pool, capService, recorder, 14*24*time.Hour, 24*time.Hour, 5, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	// Default quiet hours are 23:00-08:00 UTC; force "now" well inside them.
	projector.now = func() time.Time { return time.Date(2026, 6, 1, 23, 30, 0, 0, time.UTC) }

	drainProjector(t, ctx, projector, 200)
	if calls := recorder.callsFor(host.User.ID); len(calls) != 0 {
		t.Fatalf("expected no push during quiet hours, got %#v", calls)
	}
	assertNotificationRow(t, ctx, pool, host.User.ID, "EVENT", "SUPPRESSED_QUIET_HOURS", nil)
}

func TestNotificationProjectorCapabilityDisabledCreatesNoRow(t *testing.T) {
	ctx, pool, slotService, capService, host, member := newNotificationFixture(t)
	// Deliberately do NOT enable the notifications capability.

	created, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Capability Off", Activity: "coffee", PlaceText: "Center", Capacity: 2,
	}, "notif-cap-create-0001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Request(ctx, member.User.ID, created.ID, "notif-cap-request-0001"); err != nil {
		t.Fatal(err)
	}

	recorder := &recordingPusher{}
	projector, err := NewNotificationProjector(pool, capService, recorder, 14*24*time.Hour, 24*time.Hour, 5, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	processed := drainProjector(t, ctx, projector, 200)
	if processed == 0 {
		t.Fatal("the event must still be consumed (cursor advances) even though it is skipped")
	}
	if len(recorder.callsFor(host.User.ID)) != 0 {
		t.Fatal("capability-disabled recipient must never receive a push")
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notification_deliveries WHERE user_id=$1`, host.User.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("capability-disabled recipient must have no notification_deliveries row at all, got %d", count)
	}
}

func TestNotificationProjectorNoPusherConfigured(t *testing.T) {
	ctx, pool, slotService, capService, host, member := newNotificationFixture(t)
	enableNotificationsForAll(t, ctx, pool)

	created, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "No Pusher", Activity: "coffee", PlaceText: "Center", Capacity: 2,
	}, "notif-nopusher-create-0001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Request(ctx, member.User.ID, created.ID, "notif-nopusher-request-0001"); err != nil {
		t.Fatal(err)
	}

	projector, err := NewNotificationProjector(pool, capService, nil, 14*24*time.Hour, 24*time.Hour, 5, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	drainProjector(t, ctx, projector, 200)
	assertNotificationRow(t, ctx, pool, host.User.ID, "EVENT", "SKIPPED_NO_DEVICE", nil)
}

func TestNotificationProjectorRecentFrequencyCappedCount(t *testing.T) {
	ctx, pool, _, capService, host, _ := newNotificationFixture(t)
	projector, err := NewNotificationProjector(pool, capService, nil, 14*24*time.Hour, 24*time.Hour, 5, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()

	// notification_deliveries.source_event_id is a UNIQUE FK into
	// domain_outbox_events, so each synthetic row here needs its own
	// distinct real outbox event to reference. The event_id must be freshly
	// generated per run, not a fixed literal: domain_outbox_events.
	// subject_user_id is ON DELETE SET NULL (migration 000014), not CASCADE,
	// so a synthetic row outlives the test's own host user being cleaned up
	// — a fixed literal would collide with the previous run's leftover row
	// the moment this test runs twice against the same disposable database
	// (the normal case for this package's test suite, which reuses one live
	// database across many `go test` invocations rather than a fresh one
	// per test).
	newSyntheticID := func() string {
		t.Helper()
		id, err := identifier.NewUUID()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	seedEvent := func(id string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO domain_outbox_events (event_id,event_type,aggregate_type,aggregate_id,subject_user_id,payload)
			VALUES ($1::uuid,'test.synthetic','test',$1::uuid,$2::uuid,'{}'::jsonb)`, id, host.User.ID); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM domain_outbox_events WHERE event_id=$1::uuid`, id)
		})
	}
	insertWithEvent := func(notifType, outcome string, createdAt time.Time) {
		t.Helper()
		eventID := newSyntheticID()
		seedEvent(eventID)
		if _, err := pool.Exec(ctx, `
			INSERT INTO notification_deliveries (id,user_id,notification_type,source_event_id,deep_link,title,body,created_at,expires_at,push_outcome)
			VALUES ($1::uuid,$2,$3,$1::uuid,'app://home','t','b',$4,$5,$6)`,
			eventID, host.User.ID, notifType, createdAt, createdAt.Add(24*time.Hour), outcome); err != nil {
			t.Fatal(err)
		}
	}

	insertWithEvent("PROMO", "SENT", now.Add(-time.Hour))
	insertWithEvent("EVENT_RECOMMENDATION", "SENT", now.Add(-2*time.Hour))
	insertWithEvent("PROMO", "SUPPRESSED_QUIET_HOURS", now.Add(-time.Hour)) // not SENT: excluded
	insertWithEvent("PROMO", "SENT", now.Add(-48*time.Hour))                // outside window: excluded
	insertWithEvent("EVENT", "SENT", now.Add(-time.Hour))                   // not a capped type: excluded

	count, err := projector.recentFrequencyCappedCount(ctx, host.User.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected 2 recent frequency-capped SENT notifications, got %d", count)
	}
}

// TestNotificationProjectorEventReminder exercises the one recognized
// outbox event type this projector never receives from a DB trigger:
// slot.starting_soon (emitted only by internal/postgres.ReminderScanner —
// see reminder_scanner_integration_test.go for scanner-side coverage of
// eligibility/idempotency/lead-time). This test seeds that event directly
// via the same linkup_enqueue_outbox function the scanner itself calls, so
// it verifies buildStartingSoonCandidate's own logic in isolation from the
// scanner's eligibility scan.
func TestNotificationProjectorEventReminder(t *testing.T) {
	ctx, pool, slotService, capService, host, _ := newNotificationFixture(t)
	enableNotificationsForAll(t, ctx, pool)

	start := time.Now().UTC().Add(20 * time.Minute)
	created, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Reminder Coffee", Activity: "coffee", PlaceText: "Center", StartAt: &start, Capacity: 2,
	}, "notif-reminder-create-0001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		SELECT linkup_enqueue_outbox('slot.starting_soon','slot',$1::uuid,$2::uuid,$1::uuid,'{}'::jsonb)`,
		created.ID, host.User.ID); err != nil {
		t.Fatal(err)
	}

	recorder := &recordingPusher{}
	projector, err := NewNotificationProjector(pool, capService, recorder, 14*24*time.Hour, 24*time.Hour, 5, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	drainProjector(t, ctx, projector, 200)

	hostCalls := recorder.callsFor(host.User.ID)
	if len(hostCalls) != 1 || hostCalls[0].message.Title != "Starting soon" {
		t.Fatalf("expected host to receive one Starting soon push, got %#v", hostCalls)
	}
	assertNotificationRow(t, ctx, pool, host.User.ID, "EVENT_REMINDER", "SENT", func(deepLink string) bool {
		return deepLink == "app://slot/"+created.ID
	})
}

// drainProjector repeatedly calls ProcessBatch until it reports nothing left
// to process, returning the total events processed across every call.
//
// The `internal/postgres` package's integration tests all share one live
// disposable database across the whole `go test` invocation (not a fresh
// database per test), and the "notifications" connector's cursor
// (connector_cursors) is a single durable row shared by every test that
// exercises NotificationProjector, not scoped per test. A single bounded
// ProcessBatch(ctx, N) call — this file's original pattern — silently
// assumes the cursor is already within N events of "now", which holds only
// by accident depending on how much unrelated outbox volume every other
// integration test in the package happened to generate first. Draining to
// completion instead matches what production's own poll ticker actually
// guarantees (cmd/api/main.go calls ProcessBatch repeatedly, forever, until
// caught up) and removes the dependency on total prior outbox volume.
func drainProjector(t *testing.T, ctx context.Context, projector *NotificationProjector, batchSize int) int {
	t.Helper()
	total := 0
	for {
		n, err := projector.ProcessBatch(ctx, batchSize)
		if err != nil {
			t.Fatal(err)
		}
		total += n
		if n < batchSize {
			// A short batch means the connector reached the current end of
			// the outbox; there is nothing left to drain right now.
			return total
		}
	}
}

func assertNotificationRow(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID, wantType, wantOutcome string, deepLinkCheck func(string) bool) {
	t.Helper()
	var gotType, gotOutcome, deepLink string
	err := pool.QueryRow(ctx, `
		SELECT notification_type,push_outcome,deep_link
		FROM notification_deliveries
		WHERE user_id=$1
		ORDER BY created_at DESC LIMIT 1`, userID).Scan(&gotType, &gotOutcome, &deepLink)
	if err != nil {
		t.Fatal(err)
	}
	if gotType != wantType || gotOutcome != wantOutcome {
		t.Fatalf("user %s: type=%s outcome=%s, want type=%s outcome=%s", userID, gotType, gotOutcome, wantType, wantOutcome)
	}
	if deepLinkCheck != nil && !deepLinkCheck(deepLink) {
		t.Fatalf("user %s: unexpected deep link %q", userID, deepLink)
	}
}

// TestNotificationProjectorChatMessageFanOutAndGrouping is this session's
// own added coverage for two real gaps found while auditing buildCandidate
// (now buildCandidates) before assuming chat notifications already
// existed: (1) no candidate builder ever produced notification.TypeMessage
// at all, so a chat message never generated a push notification; (2)
// README §6.8's grouping/collapse rule ("multiple MESSAGE notifications
// from the same sender/thread collapse into one grouped surface") had no
// implementation. This test proves both against a real chat send, real
// Slot membership, and a real block relationship — not a synthetic event.
func TestNotificationProjectorChatMessageFanOutAndGrouping(t *testing.T) {
	ctx, pool, slotService, capService, host, memberA := newNotificationFixture(t)
	enableNotificationsForAll(t, ctx, pool)

	accountService, err := account.NewService(NewAccountStore(pool), password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	memberB := registerIntegrationUser(t, ctx, accountService, "nb", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{memberB.User.ID}) })

	created, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Chat Notify Coffee", Activity: "coffee", PlaceText: "Center", Capacity: 4,
	}, "notif-chat-create-0001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Request(ctx, memberA.User.ID, created.ID, "notif-chat-request-a-0001"); err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Approve(ctx, host.User.ID, created.ID, memberA.User.ID, "notif-chat-approve-a-0001"); err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Request(ctx, memberB.User.ID, created.ID, "notif-chat-request-b-0001"); err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Approve(ctx, host.User.ID, created.ID, memberB.User.ID, "notif-chat-approve-b-0001"); err != nil {
		t.Fatal(err)
	}

	chatService, err := chat.NewService(NewChatStore(pool))
	if err != nil {
		t.Fatal(err)
	}
	recorder := &recordingPusher{}
	projector, err := NewNotificationProjector(pool, capService, recorder, 14*24*time.Hour, 24*time.Hour, 5, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	t0 := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC) // clearly outside default quiet hours

	// Drain and discard the setup events first, strictly before t0:
	// Request/Approve above already produced their own real EVENT-type
	// notifications (a "New request" push to the host, a "You're in!"
	// push to each approved member) — correct behavior, but not what this
	// test is about. A strictly earlier fake clock (not just "reset the
	// recorder") matters too: assertNotificationRow below picks the most
	// recently created row per user with no secondary tie-breaker, so a
	// setup event sharing message 1's exact created_at could otherwise be
	// picked non-deterministically ahead of it.
	projector.now = func() time.Time { return t0.Add(-time.Hour) }
	drainProjector(t, ctx, projector, 200)
	recorder.reset()
	projector.now = func() time.Time { return t0 }

	// (1) Host sends the first message: fans out to BOTH other
	// participants (memberA, memberB), never to the sender.
	if _, err := chatService.Send(ctx, host.User.ID, created.ID, "notif-chat-msg-one-000001", "hey team"); err != nil {
		t.Fatal(err)
	}
	drainProjector(t, ctx, projector, 200)
	assertNotificationRow(t, ctx, pool, memberA.User.ID, "MESSAGE", "SENT", nil)
	assertNotificationRow(t, ctx, pool, memberB.User.ID, "MESSAGE", "SENT", nil)
	if len(recorder.callsFor(memberA.User.ID)) != 1 || len(recorder.callsFor(memberB.User.ID)) != 1 {
		t.Fatalf("expected exactly one push each to memberA/memberB, got %d/%d", len(recorder.callsFor(memberA.User.ID)), len(recorder.callsFor(memberB.User.ID)))
	}
	if len(recorder.callsFor(host.User.ID)) != 0 {
		t.Fatalf("the sender must never be notified about their own message, got %d pushes", len(recorder.callsFor(host.User.ID)))
	}

	// (2) A block from memberB toward the host: from this point on,
	// memberB must never be notified about the host's messages again,
	// even though memberA still is.
	if _, err := pool.Exec(ctx, `INSERT INTO user_blocks (blocker_id, blocked_id) VALUES ($1,$2)`, memberB.User.ID, host.User.ID); err != nil {
		t.Fatal(err)
	}

	// (3) A second message shortly after: within the grouping window, so
	// memberA's notification collapses (SUPPRESSED_GROUPED, no second
	// push) rather than spamming a second one.
	projector.now = func() time.Time { return t0.Add(time.Minute) }
	if _, err := chatService.Send(ctx, host.User.ID, created.ID, "notif-chat-msg-two-000002", "you there?"); err != nil {
		t.Fatal(err)
	}
	drainProjector(t, ctx, projector, 200)
	assertNotificationRow(t, ctx, pool, memberA.User.ID, "MESSAGE", "SUPPRESSED_GROUPED", nil)
	if len(recorder.callsFor(memberA.User.ID)) != 1 {
		t.Fatalf("expected still exactly one push to memberA (second message grouped), got %d", len(recorder.callsFor(memberA.User.ID)))
	}
	var memberBRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notification_deliveries WHERE user_id=$1 AND notification_type='MESSAGE'`, memberB.User.ID).Scan(&memberBRows); err != nil {
		t.Fatal(err)
	}
	if memberBRows != 1 {
		t.Fatalf("expected memberB to have no new MESSAGE row after blocking the host (still just the first), got %d rows", memberBRows)
	}

	// (4) A third message after the grouping window has elapsed: memberA
	// is notified again, a real second push, not another collapse.
	projector.now = func() time.Time { return t0.Add(6 * time.Minute) }
	if _, err := chatService.Send(ctx, host.User.ID, created.ID, "notif-chat-msg-three-000003", "ping"); err != nil {
		t.Fatal(err)
	}
	drainProjector(t, ctx, projector, 200)
	assertNotificationRow(t, ctx, pool, memberA.User.ID, "MESSAGE", "SENT", nil)
	if len(recorder.callsFor(memberA.User.ID)) != 2 {
		t.Fatalf("expected a real second push to memberA once the grouping window elapsed, got %d", len(recorder.callsFor(memberA.User.ID)))
	}
}
