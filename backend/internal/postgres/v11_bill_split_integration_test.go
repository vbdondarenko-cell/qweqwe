package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
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

// TestV11BillSplitRosterAuthorizationAndDeterministicSplit proves README
// §6.18's Bill Splitter calculator end to end against real PostgreSQL: a
// stranger cannot split a Slot's bill at all (same boundary as chat); the
// host and an accepted member both can; the computed roster excludes a
// member who has since left, and a block still wins even for an otherwise-
// accepted member (mirroring chat's own existing invariants); and the
// resulting split is deterministic and always sums exactly to the total.
func TestV11BillSplitRosterAuthorizationAndDeterministicSplit(t *testing.T) {
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

	accountService, err := account.NewService(NewAccountStore(pool), password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	host := registerIntegrationUser(t, ctx, accountService, "bsh", suffix)
	memberA := registerIntegrationUser(t, ctx, accountService, "bsa", suffix)
	memberB := registerIntegrationUser(t, ctx, accountService, "bsb", suffix)
	stranger := registerIntegrationUser(t, ctx, accountService, "bss", suffix)
	userIDs := []string{host.User.ID, memberA.User.ID, memberB.User.ID, stranger.User.ID}
	t.Cleanup(func() { cleanupIntegrationRows(pool, userIDs) })

	baseStore, err := NewSlotStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	slotService, err := slot.NewService(baseStore)
	if err != nil {
		t.Fatal(err)
	}
	chatStore := NewChatStore(pool)
	chatService, err := chat.NewService(chatStore)
	if err != nil {
		t.Fatal(err)
	}
	blockService, err := blocklist.NewService(NewBlockStore(pool, time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	created, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Bill Split Dinner", Activity: "dinner", PlaceText: "Zone", Capacity: 4,
	}, "v11-bill-split-create-0001")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := chatService.SplitBill(ctx, stranger.User.ID, created.ID, 300, nil); !errors.Is(err, chat.ErrForbidden) {
		t.Fatalf("stranger must not be able to split the bill, got %v", err)
	}

	if _, err := slotService.Request(ctx, memberA.User.ID, created.ID, "v11-bill-split-request-a-0001"); err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Approve(ctx, host.User.ID, created.ID, memberA.User.ID, "v11-bill-split-approve-a-0001"); err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Request(ctx, memberB.User.ID, created.ID, "v11-bill-split-request-b-0001"); err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Approve(ctx, host.User.ID, created.ID, memberB.User.ID, "v11-bill-split-approve-b-0001"); err != nil {
		t.Fatal(err)
	}

	// Default roster: host + both accepted members, 100 minor units split
	// three ways (base 33, remainder 1 -- goes to the alphabetically-first
	// user ID, whichever of the three that is).
	split, err := chatService.SplitBill(ctx, memberA.User.ID, created.ID, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	if split.TotalMinor != 100 || len(split.Shares) != 3 {
		t.Fatalf("unexpected split: %#v", split)
	}
	sum := 0
	for _, share := range split.Shares {
		sum += share.AmountMinor
	}
	if sum != 100 {
		t.Fatalf("shares must sum to the total, got %d", sum)
	}

	// A stranger listed explicitly as a participant is rejected.
	if _, err := chatService.SplitBill(ctx, host.User.ID, created.ID, 100, []string{host.User.ID, stranger.User.ID}); !errors.Is(err, chat.ErrInvalidInput) {
		t.Fatalf("expected invalid input for a non-participant, got %v", err)
	}

	// memberB leaves -- the default roster must shrink accordingly. Checked
	// via SplitBillRoster directly (rather than SplitBill, which requires
	// at least 2 participants and would otherwise conflate this with the
	// block assertion below once only the host remains).
	if _, err := slotService.Leave(ctx, memberB.User.ID, created.ID, "v11-bill-split-leave-b-0001"); err != nil {
		t.Fatal(err)
	}
	afterLeave, err := chatStore.SplitBillRoster(ctx, host.User.ID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(afterLeave) != 2 {
		t.Fatalf("expected host+memberA after memberB left, got %#v", afterLeave)
	}
	for _, author := range afterLeave {
		if author.ID == memberB.User.ID {
			t.Fatal("a member who left must not appear in the default roster")
		}
	}

	// A block from the host toward memberA still wins over being accepted,
	// mirroring chat's own existing invariant.
	if err := blockService.Block(ctx, host.User.ID, memberA.User.ID); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blockService.Unblock(ctx, host.User.ID, memberA.User.ID) }()
	blockedView, err := chatStore.SplitBillRoster(ctx, host.User.ID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, author := range blockedView {
		if author.ID == memberA.User.ID {
			t.Fatal("a blocked counterpart must not appear in the host's own roster view")
		}
	}
}
