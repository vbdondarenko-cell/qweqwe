package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/blocklist"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/chat"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// TestV1SocialCorePostgresIntegration exercises the real v1.0 stores and
// services on an explicitly opted-in disposable PostgreSQL database. It is
// intentionally not a fake/temp-table query test: the canonical migrations are
// applied through the production migration runner first.
//
// Safety: LINKUP_TEST_DATABASE_URL alone is insufficient because Apply mutates
// schema. The operator must also set LINKUP_TEST_DATABASE_DESTRUCTIVE=1.
func TestV1SocialCorePostgresIntegration(t *testing.T) {
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
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	if err := migrate.Apply(ctx, pool, migrationDir(t)); err != nil {
		t.Fatalf("apply canonical migrations: %v", err)
	}
	assertMigrationCount(t, ctx, pool, 11)

	accountService, err := account.NewService(NewAccountStore(pool), password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	slotStore, err := NewSlotStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	slotService, err := slot.NewService(slotStore)
	if err != nil {
		t.Fatal(err)
	}
	chatService, err := chat.NewService(NewChatStore(pool))
	if err != nil {
		t.Fatal(err)
	}
	blockService, err := blocklist.NewService(NewBlockStore(pool))
	if err != nil {
		t.Fatal(err)
	}

	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	host := registerIntegrationUser(t, ctx, accountService, "h", suffix)
	memberA := registerIntegrationUser(t, ctx, accountService, "a", suffix)
	memberB := registerIntegrationUser(t, ctx, accountService, "b", suffix)
	memberC := registerIntegrationUser(t, ctx, accountService, "c", suffix)
	userIDs := []string{host.User.ID, memberA.User.ID, memberB.User.ID, memberC.User.ID}
	t.Cleanup(func() { cleanupIntegrationRows(pool, userIDs) })

	keyCounter := 0
	nextKey := func(label string) string {
		keyCounter++
		return fmt.Sprintf("v1-integration-%s-%06d", label, keyCounter)
	}

	t.Run("two user lifecycle chat purge and revocation", func(t *testing.T) {
		start := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
		details := "real PostgreSQL integration"
		created, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
			Title: "Integration Coffee", Activity: "coffee", Details: &details,
			PlaceText: "Integration Zone", StartAt: &start, Capacity: 2,
		}, nextKey("create-core"))
		if err != nil {
			t.Fatal(err)
		}
		if created.State != slot.StateFilling || created.ViewerState != slot.ViewerHost || created.AcceptedCount != 0 {
			t.Fatalf("unexpected created slot: %#v", created)
		}

		pulse, err := slotService.ListPulse(ctx, memberA.User.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !containsSlotWithViewer(pulse, created.ID, slot.ViewerNone) {
			t.Fatalf("new public slot missing from member Pulse: %#v", pulse)
		}

		if _, err := chatService.ListRecent(ctx, memberC.User.ID, created.ID, 100); !errors.Is(err, chat.ErrForbidden) {
			t.Fatalf("stranger chat must be forbidden, got %v", err)
		}

		pendingState, err := slotService.Request(ctx, memberA.User.ID, created.ID, nextKey("request-a"))
		if err != nil {
			t.Fatal(err)
		}
		if pendingState.ViewerState != slot.ViewerPending {
			t.Fatalf("request did not produce PENDING: %#v", pendingState)
		}
		if _, err := chatService.ListRecent(ctx, memberA.User.ID, created.ID, 100); !errors.Is(err, chat.ErrForbidden) {
			t.Fatalf("pending requester chat must be forbidden, got %v", err)
		}

		pending, err := slotService.ListPending(ctx, host.User.ID, created.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(pending) != 1 || pending[0].User.ID != memberA.User.ID {
			t.Fatalf("host pending list mismatch: %#v", pending)
		}

		approved, err := slotService.Approve(ctx, host.User.ID, created.ID, memberA.User.ID, nextKey("approve-a"))
		if err != nil {
			t.Fatal(err)
		}
		if approved.AcceptedCount != 1 || approved.State != slot.StateFilling {
			t.Fatalf("approval mismatch: %#v", approved)
		}
		roster, err := slotService.ListAccepted(ctx, host.User.ID, created.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(roster) != 1 || roster[0].ID != memberA.User.ID {
			t.Fatalf("accepted roster mismatch: %#v", roster)
		}

		hostMessage, err := chatService.Send(ctx, host.User.ID, created.ID, "v1-chat-host-replay-0001", "Host ready")
		if err != nil {
			t.Fatal(err)
		}
		memberMessage, err := chatService.Send(ctx, memberA.User.ID, created.ID, "v1-chat-member-replay-01", "On my way")
		if err != nil {
			t.Fatal(err)
		}
		replay, err := chatService.Send(ctx, memberA.User.ID, created.ID, "v1-chat-member-replay-01", "On my way")
		if err != nil {
			t.Fatal(err)
		}
		if replay.ID != memberMessage.ID {
			t.Fatalf("chat replay created another message: first=%s replay=%s", memberMessage.ID, replay.ID)
		}
		if _, err := chatService.Send(ctx, memberA.User.ID, created.ID, "v1-chat-member-replay-01", "Different text"); !errors.Is(err, chat.ErrIdempotencyConflict) {
			t.Fatalf("same chat key with different payload must conflict, got %v", err)
		}
		messages, err := chatService.ListRecent(ctx, memberA.User.ID, created.ID, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(messages) != 2 || messages[0].ID != hostMessage.ID || messages[1].ID != memberMessage.ID {
			t.Fatalf("chat thread mismatch: %#v", messages)
		}

		active, err := slotService.Start(ctx, host.User.ID, created.ID, nextKey("start-core"))
		if err != nil {
			t.Fatal(err)
		}
		if active.State != slot.StateActive {
			t.Fatalf("expected ACTIVE, got %#v", active)
		}
		if _, err := chatService.Send(ctx, memberA.User.ID, created.ID, "v1-chat-active-member-01", "Active coordination"); err != nil {
			t.Fatalf("accepted member lost chat during ACTIVE: %v", err)
		}

		completed, err := slotService.Complete(ctx, host.User.ID, created.ID, nextKey("complete-core"))
		if err != nil {
			t.Fatal(err)
		}
		if completed.State != slot.StateCompleted {
			t.Fatalf("expected COMPLETED, got %#v", completed)
		}
		assertChatRows(t, ctx, pool, created.ID, 0)
		if _, err := chatService.ListRecent(ctx, memberA.User.ID, created.ID, 100); !errors.Is(err, chat.ErrClosed) {
			t.Fatalf("completed chat must be closed, got %v", err)
		}
	})

	t.Run("leave and block revoke membership chat immediately", func(t *testing.T) {
		leaveSlot, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
			Title: "Leave Revocation", Activity: "walk", PlaceText: "Zone", Capacity: 2,
		}, nextKey("create-leave"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := slotService.Request(ctx, memberA.User.ID, leaveSlot.ID, nextKey("request-leave")); err != nil {
			t.Fatal(err)
		}
		approved, err := slotService.Approve(ctx, host.User.ID, leaveSlot.ID, memberA.User.ID, nextKey("approve-leave"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := chatService.Send(ctx, memberA.User.ID, leaveSlot.ID, "v1-chat-leave-member-01", "Before leave"); err != nil {
			t.Fatal(err)
		}
		left, err := slotService.Leave(ctx, memberA.User.ID, leaveSlot.ID, nextKey("leave-member"))
		if err != nil {
			t.Fatal(err)
		}
		if left.AcceptedCount != 0 || left.ViewerState != slot.ViewerNone {
			t.Fatalf("leave did not revoke membership: %#v (approved=%#v)", left, approved)
		}
		if _, err := chatService.ListRecent(ctx, memberA.User.ID, leaveSlot.ID, 100); !errors.Is(err, chat.ErrForbidden) {
			t.Fatalf("left member retained chat access: %v", err)
		}

		blockSlot, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
			Title: "Block Revocation", Activity: "walk", PlaceText: "Zone", Capacity: 2,
		}, nextKey("create-block"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := slotService.Request(ctx, memberC.User.ID, blockSlot.ID, nextKey("request-block")); err != nil {
			t.Fatal(err)
		}
		if _, err := slotService.Approve(ctx, host.User.ID, blockSlot.ID, memberC.User.ID, nextKey("approve-block")); err != nil {
			t.Fatal(err)
		}
		if _, err := chatService.Send(ctx, memberC.User.ID, blockSlot.ID, "v1-chat-block-member-01", "Before block"); err != nil {
			t.Fatal(err)
		}
		if err := blockService.Block(ctx, host.User.ID, memberC.User.ID); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = blockService.Unblock(ctx, host.User.ID, memberC.User.ID) }()
		if _, err := chatService.ListRecent(ctx, memberC.User.ID, blockSlot.ID, 100); !errors.Is(err, chat.ErrForbidden) {
			t.Fatalf("blocked member retained chat access: %v", err)
		}
		var acceptedCount, membershipCount int
		if err := pool.QueryRow(ctx, `SELECT accepted_count FROM slots WHERE id=$1`, blockSlot.ID).Scan(&acceptedCount); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM slot_memberships WHERE slot_id=$1 AND user_id=$2`, blockSlot.ID, memberC.User.ID).Scan(&membershipCount); err != nil {
			t.Fatal(err)
		}
		if acceptedCount != 0 || membershipCount != 0 {
			t.Fatalf("block cleanup failed: accepted_count=%d membership_count=%d", acceptedCount, membershipCount)
		}
	})

	t.Run("cancel physically purges chat", func(t *testing.T) {
		cancelSlot, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
			Title: "Cancel Purge", Activity: "coffee", PlaceText: "Zone", Capacity: 2,
		}, nextKey("create-cancel"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := slotService.Request(ctx, memberB.User.ID, cancelSlot.ID, nextKey("request-cancel")); err != nil {
			t.Fatal(err)
		}
		approved, err := slotService.Approve(ctx, host.User.ID, cancelSlot.ID, memberB.User.ID, nextKey("approve-cancel"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := chatService.Send(ctx, host.User.ID, cancelSlot.ID, "v1-chat-cancel-host-001", "Will cancel"); err != nil {
			t.Fatal(err)
		}
		assertChatRows(t, ctx, pool, cancelSlot.ID, 1)
		cancelled, err := slotService.Cancel(ctx, host.User.ID, cancelSlot.ID, approved.Version, nextKey("cancel-slot"))
		if err != nil {
			t.Fatal(err)
		}
		if cancelled.State != slot.StateCancelled {
			t.Fatalf("expected CANCELLED, got %#v", cancelled)
		}
		assertChatRows(t, ctx, pool, cancelSlot.ID, 0)
	})

	t.Run("concurrent last seat never exceeds capacity", func(t *testing.T) {
		raceSlot, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
			Title: "Last Seat Race", Activity: "social", PlaceText: "Zone", Capacity: 2,
		}, nextKey("create-race"))
		if err != nil {
			t.Fatal(err)
		}
		for i, user := range []account.AuthResult{memberA, memberB, memberC} {
			if _, err := slotService.Request(ctx, user.User.ID, raceSlot.ID, nextKey(fmt.Sprintf("race-request-%d", i))); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := slotService.Approve(ctx, host.User.ID, raceSlot.ID, memberA.User.ID, nextKey("race-approve-a")); err != nil {
			t.Fatal(err)
		}

		keyB := nextKey("race-approve-b")
		keyC := nextKey("race-approve-c")
		type result struct {
			userID string
			slot   slot.Slot
			err    error
		}
		results := make(chan result, 2)
		var wg sync.WaitGroup
		for _, candidate := range []struct {
			id  string
			key string
		}{{memberB.User.ID, keyB}, {memberC.User.ID, keyC}} {
			candidate := candidate
			wg.Add(1)
			go func() {
				defer wg.Done()
				out, err := slotService.Approve(ctx, host.User.ID, raceSlot.ID, candidate.id, candidate.key)
				results <- result{userID: candidate.id, slot: out, err: err}
			}()
		}
		wg.Wait()
		close(results)

		successes, fullErrors := 0, 0
		for result := range results {
			if result.err == nil {
				successes++
				if result.slot.AcceptedCount != 2 || result.slot.State != slot.StateFull {
					t.Fatalf("race winner returned invalid state for %s: %#v", result.userID, result.slot)
				}
				continue
			}
			if errors.Is(result.err, slot.ErrCapacityFull) {
				fullErrors++
				continue
			}
			t.Fatalf("unexpected concurrent approval error for %s: %v", result.userID, result.err)
		}
		if successes != 1 || fullErrors != 1 {
			t.Fatalf("expected one last-seat winner and one capacity rejection, success=%d full=%d", successes, fullErrors)
		}

		var acceptedCount, membershipCount, pendingCount int
		if err := pool.QueryRow(ctx, `SELECT accepted_count FROM slots WHERE id=$1`, raceSlot.ID).Scan(&acceptedCount); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM slot_memberships WHERE slot_id=$1`, raceSlot.ID).Scan(&membershipCount); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM slot_requests WHERE slot_id=$1`, raceSlot.ID).Scan(&pendingCount); err != nil {
			t.Fatal(err)
		}
		if acceptedCount != 2 || membershipCount != 2 || pendingCount != 1 {
			t.Fatalf("last-seat invariant failed: accepted_count=%d memberships=%d pending=%d", acceptedCount, membershipCount, pendingCount)
		}
	})
}

func migrationDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve integration test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../db/migrations"))
}

func registerIntegrationUser(t *testing.T, ctx context.Context, service *account.Service, prefix, suffix string) account.AuthResult {
	t.Helper()
	username := prefix + suffix
	if len(username) > 30 {
		username = username[:30]
	}
	result, err := service.Register(ctx, account.Registration{
		Email:       username + "@integration.test",
		Username:    username,
		DisplayName: "Integration " + prefix,
		Password:    "CorrectHorseBattery1!",
		Language:    "en",
		DeviceLabel: "postgres-integration",
	})
	if err != nil {
		t.Fatalf("register %s: %v", prefix, err)
	}
	return result
}

func containsSlotWithViewer(items []slot.Slot, id string, viewer slot.ViewerState) bool {
	for _, item := range items {
		if item.ID == id && item.ViewerState == viewer {
			return true
		}
	}
	return false
}

func assertMigrationCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, want int) {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM linkup_schema_migrations WHERE name ~ '^[0-9]{6}_.+\.sql$'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count < want {
		t.Fatalf("expected at least %d canonical migrations, got %d", want, count)
	}
}

func assertChatRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool, slotID string, want int) {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM slot_messages WHERE slot_id=$1`, slotID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("slot %s chat rows=%d want=%d", slotID, count, want)
	}
}

func cleanupIntegrationRows(pool *pgxpool.Pool, userIDs []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, id := range userIDs {
		_, _ = pool.Exec(ctx, `DELETE FROM slots WHERE host_id=$1`, id)
	}
	for _, id := range userIDs {
		_, _ = pool.Exec(ctx, `DELETE FROM app_users WHERE id=$1`, id)
	}
}
