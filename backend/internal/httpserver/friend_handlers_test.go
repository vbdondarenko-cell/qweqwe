package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/friend"
)

type fakeFriendStore struct {
	requestResult friend.RequestResult
	requestErr    error
	acceptErr     error
	rejectErr     error
	cancelErr     error
	removeErr     error
	friends       []friend.Friend
	incoming      []friend.PendingRequest
	outgoing      []friend.PendingRequest
	listErr       error
}

func (f *fakeFriendStore) Request(_ context.Context, _, _ string, _ time.Time) (friend.RequestResult, error) {
	return f.requestResult, f.requestErr
}
func (f *fakeFriendStore) Accept(_ context.Context, _, _ string, _ time.Time) error {
	return f.acceptErr
}
func (f *fakeFriendStore) Reject(_ context.Context, _, _ string) error { return f.rejectErr }
func (f *fakeFriendStore) Cancel(_ context.Context, _, _ string) error { return f.cancelErr }
func (f *fakeFriendStore) Remove(_ context.Context, _, _ string) error { return f.removeErr }
func (f *fakeFriendStore) ListFriends(_ context.Context, _ string, _ int) ([]friend.Friend, error) {
	return f.friends, f.listErr
}
func (f *fakeFriendStore) ListIncoming(_ context.Context, _ string, _ int) ([]friend.PendingRequest, error) {
	return f.incoming, f.listErr
}
func (f *fakeFriendStore) ListOutgoing(_ context.Context, _ string, _ int) ([]friend.PendingRequest, error) {
	return f.outgoing, f.listErr
}
func (f *fakeFriendStore) AreFriends(_ context.Context, _, _ string) (bool, error) { return false, nil }

func requestWithFriendAuth(method, target string) *http.Request {
	request := httptest.NewRequest(method, target, nil)
	request.SetPathValue("userID", "22222222-2222-2222-2222-222222222222")
	ctx := context.WithValue(request.Context(), authKey, authContext{User: account.User{ID: "11111111-1111-1111-1111-111111111111"}})
	return request.WithContext(ctx)
}

func TestSendFriendRequestReturnsOutcome(t *testing.T) {
	store := &fakeFriendStore{requestResult: friend.RequestResult{Outcome: friend.OutcomeRequested}}
	svc, err := friend.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Friends: svc}}
	recorder := httptest.NewRecorder()
	server.sendFriendRequest(recorder, requestWithFriendAuth(http.MethodPost, "/v1/me/friends/requests/target"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var got friendRequestResponse
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Outcome != string(friend.OutcomeRequested) {
		t.Fatalf("unexpected outcome: %#v", got)
	}
}

func TestSendFriendRequestRequiresAuth(t *testing.T) {
	server := &Server{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/me/friends/requests/target", nil)
	request.SetPathValue("userID", "target")
	server.sendFriendRequest(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSendFriendRequestFailsClosedWhenServiceUnavailable(t *testing.T) {
	server := &Server{}
	recorder := httptest.NewRecorder()
	server.sendFriendRequest(recorder, requestWithFriendAuth(http.MethodPost, "/v1/me/friends/requests/target"))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSendFriendRequestPropagatesAlreadyFriendsAsConflict(t *testing.T) {
	store := &fakeFriendStore{requestErr: friend.ErrAlreadyFriends}
	svc, err := friend.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Friends: svc}}
	recorder := httptest.NewRecorder()
	server.sendFriendRequest(recorder, requestWithFriendAuth(http.MethodPost, "/v1/me/friends/requests/target"))
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSendFriendRequestPropagatesForbiddenForBlockedPair(t *testing.T) {
	store := &fakeFriendStore{requestErr: friend.ErrForbidden}
	svc, err := friend.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Friends: svc}}
	recorder := httptest.NewRecorder()
	server.sendFriendRequest(recorder, requestWithFriendAuth(http.MethodPost, "/v1/me/friends/requests/target"))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAcceptFriendRequestReturnsNoContent(t *testing.T) {
	store := &fakeFriendStore{}
	svc, err := friend.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Friends: svc}}
	recorder := httptest.NewRecorder()
	server.acceptFriendRequest(recorder, requestWithFriendAuth(http.MethodPost, "/v1/me/friends/requests/target/accept"))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAcceptFriendRequestPropagatesNotFound(t *testing.T) {
	store := &fakeFriendStore{acceptErr: friend.ErrNotFound}
	svc, err := friend.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Friends: svc}}
	recorder := httptest.NewRecorder()
	server.acceptFriendRequest(recorder, requestWithFriendAuth(http.MethodPost, "/v1/me/friends/requests/target/accept"))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRejectFriendRequestReturnsNoContent(t *testing.T) {
	store := &fakeFriendStore{}
	svc, err := friend.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Friends: svc}}
	recorder := httptest.NewRecorder()
	server.rejectFriendRequest(recorder, requestWithFriendAuth(http.MethodPost, "/v1/me/friends/requests/target/reject"))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestCancelFriendRequestReturnsNoContent(t *testing.T) {
	store := &fakeFriendStore{}
	svc, err := friend.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Friends: svc}}
	recorder := httptest.NewRecorder()
	server.cancelFriendRequest(recorder, requestWithFriendAuth(http.MethodDelete, "/v1/me/friends/requests/target"))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRemoveFriendReturnsNoContent(t *testing.T) {
	store := &fakeFriendStore{}
	svc, err := friend.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Friends: svc}}
	recorder := httptest.NewRecorder()
	server.removeFriend(recorder, requestWithFriendAuth(http.MethodDelete, "/v1/me/friends/target"))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestListFriendsReturnsItems(t *testing.T) {
	store := &fakeFriendStore{friends: []friend.Friend{{User: friend.UserSummary{ID: "user-2"}, Since: time.Unix(1000, 0)}}}
	svc, err := friend.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Friends: svc}}
	recorder := httptest.NewRecorder()
	server.listFriends(recorder, requestWithFriendAuth(http.MethodGet, "/v1/me/friends"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var got friendListResponse
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].User.ID != "user-2" {
		t.Fatalf("unexpected response: %#v", got)
	}
}

func TestListIncomingFriendRequestsReturnsItems(t *testing.T) {
	store := &fakeFriendStore{incoming: []friend.PendingRequest{{RequestID: "req-1", User: friend.UserSummary{ID: "user-3"}}}}
	svc, err := friend.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Friends: svc}}
	recorder := httptest.NewRecorder()
	server.listIncomingFriendRequests(recorder, requestWithFriendAuth(http.MethodGet, "/v1/me/friends/requests/incoming"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var got friendPendingRequestListResponse
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].RequestID != "req-1" {
		t.Fatalf("unexpected response: %#v", got)
	}
}

func TestListOutgoingFriendRequestsReturnsItems(t *testing.T) {
	store := &fakeFriendStore{outgoing: []friend.PendingRequest{{RequestID: "req-2", User: friend.UserSummary{ID: "user-4"}}}}
	svc, err := friend.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Friends: svc}}
	recorder := httptest.NewRecorder()
	server.listOutgoingFriendRequests(recorder, requestWithFriendAuth(http.MethodGet, "/v1/me/friends/requests/outgoing"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var got friendPendingRequestListResponse
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].RequestID != "req-2" {
		t.Fatalf("unexpected response: %#v", got)
	}
}

func TestWriteFriendErrorMapsInvalidInputToBadRequest(t *testing.T) {
	store := &fakeFriendStore{requestErr: friend.ErrInvalidInput}
	svc, err := friend.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Friends: svc}}
	recorder := httptest.NewRecorder()
	server.sendFriendRequest(recorder, requestWithFriendAuth(http.MethodPost, "/v1/me/friends/requests/target"))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestWriteFriendErrorMapsUnknownErrorToInternalError(t *testing.T) {
	store := &fakeFriendStore{requestErr: errors.New("db down")}
	svc, err := friend.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Friends: svc}}
	recorder := httptest.NewRecorder()
	server.sendFriendRequest(recorder, requestWithFriendAuth(http.MethodPost, "/v1/me/friends/requests/target"))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
