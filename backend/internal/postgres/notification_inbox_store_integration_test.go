package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/notification"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// TestNotificationInboxStoreListsRealProjectorOutput exercises the inbox
// store against real notification_deliveries rows the projector itself
// wrote (the same real request-created/approved flow
// TestNotificationProjectorApprovalAndRequestCreated already drives) --
// not hand-inserted synthetic rows -- so this proves the inbox actually
// reads back what production traffic produces, migration 000040's read_at
// column included.
func TestNotificationInboxStoreListsRealProjectorOutput(t *testing.T) {
	ctx, pool, slotService, capService, host, member := newNotificationFixture(t)
	enableNotificationsForAll(t, ctx, pool)

	created, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Inbox Coffee", Activity: "coffee", PlaceText: "Center", Capacity: 2,
	}, "inbox-create-0001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Request(ctx, member.User.ID, created.ID, "inbox-request-0001"); err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Approve(ctx, host.User.ID, created.ID, member.User.ID, "inbox-approve-0001"); err != nil {
		t.Fatal(err)
	}

	recorder := &recordingPusher{}
	projector, err := NewNotificationProjector(pool, capService, recorder, 14*24*time.Hour, 24*time.Hour, 5, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	drainProjector(t, ctx, projector, 200)

	store, err := NewNotificationInboxStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := notification.NewInboxService(store)
	if err != nil {
		t.Fatal(err)
	}

	// The member received exactly one real "You're in!" delivery (asserted
	// already by the sibling projector test); it must start out unread.
	snapshot, err := svc.List(ctx, member.User.ID, 30, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Items) != 1 {
		t.Fatalf("expected exactly 1 delivery for member, got %d: %#v", len(snapshot.Items), snapshot.Items)
	}
	if snapshot.UnreadCount != 1 {
		t.Fatalf("unreadCount = %d, want 1", snapshot.UnreadCount)
	}
	item := snapshot.Items[0]
	if item.Read {
		t.Fatal("expected a freshly-projected delivery to start unread")
	}
	if item.Title == "" || item.Body == "" || item.DeepLink == "" {
		t.Fatalf("expected real title/body/deepLink, got %#v", item)
	}
	if item.SlotID != created.ID {
		t.Fatalf("slotId = %q, want %q", item.SlotID, created.ID)
	}

	// Marking all read must actually flip read_at, and must not affect a
	// different user's rows.
	if err := svc.MarkAllRead(ctx, member.User.ID); err != nil {
		t.Fatal(err)
	}
	afterRead, err := svc.List(ctx, member.User.ID, 30, 0)
	if err != nil {
		t.Fatal(err)
	}
	if afterRead.UnreadCount != 0 {
		t.Fatalf("unreadCount after MarkAllRead = %d, want 0", afterRead.UnreadCount)
	}
	if !afterRead.Items[0].Read {
		t.Fatal("expected item to be marked read")
	}

	hostSnapshot, err := svc.List(ctx, host.User.ID, 30, 0)
	if err != nil {
		t.Fatal(err)
	}
	if hostSnapshot.UnreadCount == 0 {
		t.Fatal("marking the member's notifications read must not touch the host's")
	}
}

func TestNotificationInboxStoreListPaginatesByBeforeCursor(t *testing.T) {
	ctx, pool, slotService, capService, host, member := newNotificationFixture(t)
	enableNotificationsForAll(t, ctx, pool)

	// Two separate slots, each generating its own "New request" delivery to
	// the host, gives two real, distinctly-timed rows to paginate across.
	first, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Inbox Page One", Activity: "coffee", PlaceText: "Center", Capacity: 2,
	}, "inbox-page-create-0001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Request(ctx, member.User.ID, first.ID, "inbox-page-request-0001"); err != nil {
		t.Fatal(err)
	}
	second, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Inbox Page Two", Activity: "coffee", PlaceText: "Center", Capacity: 2,
	}, "inbox-page-create-0002")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Request(ctx, member.User.ID, second.ID, "inbox-page-request-0002"); err != nil {
		t.Fatal(err)
	}

	recorder := &recordingPusher{}
	projector, err := NewNotificationProjector(pool, capService, recorder, 14*24*time.Hour, 24*time.Hour, 5, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	drainProjector(t, ctx, projector, 200)

	store, err := NewNotificationInboxStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := notification.NewInboxService(store)
	if err != nil {
		t.Fatal(err)
	}

	firstPage, err := svc.List(ctx, host.User.ID, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(firstPage.Items) != 1 || firstPage.NextCursor == 0 {
		t.Fatalf("expected a full first page with a cursor, got %#v", firstPage)
	}

	secondPage, err := svc.List(ctx, host.User.ID, 1, firstPage.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if len(secondPage.Items) != 1 {
		t.Fatalf("expected exactly one older item on the second page, got %d", len(secondPage.Items))
	}
	if secondPage.Items[0].ID == firstPage.Items[0].ID {
		t.Fatal("second page must not repeat the first page's item")
	}
}

func TestNotificationInboxStoreUnreadCountIsPerUser(t *testing.T) {
	_, pool, _, _, host, _ := newNotificationFixture(t)
	store, err := NewNotificationInboxStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	count, err := store.UnreadCount(context.Background(), host.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected 0 unread for a fresh user with no deliveries, got %d", count)
	}
}
