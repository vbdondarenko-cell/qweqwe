package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/notification"
)

type fakeNotificationInboxStore struct {
	items  []notification.Delivery
	unread int
	marked bool
	err    error
}

func (f *fakeNotificationInboxStore) List(_ context.Context, _ string, limit int, _ int64) ([]notification.Delivery, error) {
	if f.err != nil {
		return nil, f.err
	}
	if limit < len(f.items) {
		return f.items[:limit], nil
	}
	return f.items, nil
}

func (f *fakeNotificationInboxStore) UnreadCount(_ context.Context, _ string) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.unread, nil
}

func (f *fakeNotificationInboxStore) MarkAllRead(_ context.Context, _ string) error {
	if f.err != nil {
		return f.err
	}
	f.marked = true
	return nil
}

func requestWithNotificationInboxAuth(method, target string) *http.Request {
	request := httptest.NewRequest(method, target, nil)
	ctx := context.WithValue(request.Context(), authKey, authContext{User: account.User{ID: "11111111-1111-1111-1111-111111111111"}})
	return request.WithContext(ctx)
}

func TestListNotificationsFailsClosedWhenServiceUnavailable(t *testing.T) {
	server := &Server{}
	recorder := httptest.NewRecorder()
	server.listNotifications(recorder, requestWithNotificationInboxAuth(http.MethodGet, "/v1/me/notifications"))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestListNotificationsReturnsSnapshot(t *testing.T) {
	store := &fakeNotificationInboxStore{
		items: []notification.Delivery{
			{ID: "d1", Type: notification.TypeMessage, Title: "New message", Body: "hey", CreatedAt: 1000, Read: false},
		},
		unread: 3,
	}
	svc, err := notification.NewInboxService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{NotificationInbox: svc}}
	recorder := httptest.NewRecorder()
	server.listNotifications(recorder, requestWithNotificationInboxAuth(http.MethodGet, "/v1/me/notifications"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var snapshot notification.InboxSnapshot
	if err := json.NewDecoder(recorder.Body).Decode(&snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.UnreadCount != 3 || len(snapshot.Items) != 1 || snapshot.Items[0].ID != "d1" {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
}

func TestListNotificationsHonorsLimitAndBeforeQueryParams(t *testing.T) {
	store := &fakeNotificationInboxStore{
		items: []notification.Delivery{
			{ID: "d1", CreatedAt: 100},
			{ID: "d2", CreatedAt: 200},
		},
	}
	svc, err := notification.NewInboxService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{NotificationInbox: svc}}
	recorder := httptest.NewRecorder()
	server.listNotifications(recorder, requestWithNotificationInboxAuth(http.MethodGet, "/v1/me/notifications?limit=1&before=500"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var snapshot notification.InboxSnapshot
	if err := json.NewDecoder(recorder.Body).Decode(&snapshot); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Items) != 1 {
		t.Fatalf("expected limit=1 to be honored, got %d items", len(snapshot.Items))
	}
}

func TestListNotificationsPropagatesStoreError(t *testing.T) {
	store := &fakeNotificationInboxStore{err: errors.New("db down")}
	svc, err := notification.NewInboxService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{NotificationInbox: svc}}
	recorder := httptest.NewRecorder()
	server.listNotifications(recorder, requestWithNotificationInboxAuth(http.MethodGet, "/v1/me/notifications"))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestMarkNotificationsReadFailsClosedWhenServiceUnavailable(t *testing.T) {
	server := &Server{}
	recorder := httptest.NewRecorder()
	server.markNotificationsRead(recorder, requestWithNotificationInboxAuth(http.MethodPost, "/v1/me/notifications/read"))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestMarkNotificationsReadSucceeds(t *testing.T) {
	store := &fakeNotificationInboxStore{}
	svc, err := notification.NewInboxService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{NotificationInbox: svc}}
	recorder := httptest.NewRecorder()
	server.markNotificationsRead(recorder, requestWithNotificationInboxAuth(http.MethodPost, "/v1/me/notifications/read"))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !store.marked {
		t.Fatal("expected MarkAllRead to reach the store")
	}
}

func TestMarkNotificationsReadPropagatesStoreError(t *testing.T) {
	store := &fakeNotificationInboxStore{err: errors.New("db down")}
	svc, err := notification.NewInboxService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{NotificationInbox: svc}}
	recorder := httptest.NewRecorder()
	server.markNotificationsRead(recorder, requestWithNotificationInboxAuth(http.MethodPost, "/v1/me/notifications/read"))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
