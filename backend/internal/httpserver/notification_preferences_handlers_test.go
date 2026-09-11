package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/notification"
)

type fakeNotificationPreferencesStore struct {
	saved map[string]notification.StoredPreferences
	err   error
}

func (f *fakeNotificationPreferencesStore) Get(_ context.Context, userID string) (notification.StoredPreferences, error) {
	if f.err != nil {
		return notification.StoredPreferences{}, f.err
	}
	if p, ok := f.saved[userID]; ok {
		return p, nil
	}
	return notification.DefaultStoredPreferences(), nil
}

func (f *fakeNotificationPreferencesStore) Update(_ context.Context, userID string, prefs notification.StoredPreferences) error {
	if f.err != nil {
		return f.err
	}
	if f.saved == nil {
		f.saved = make(map[string]notification.StoredPreferences)
	}
	f.saved[userID] = prefs
	return nil
}

func requestWithNotificationPreferencesAuth(method, body string) *http.Request {
	var reader *bytes.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	} else {
		reader = bytes.NewReader(nil)
	}
	request := httptest.NewRequest(method, "/v1/me/notifications/preferences", reader)
	ctx := context.WithValue(request.Context(), authKey, authContext{User: account.User{ID: "11111111-1111-1111-1111-111111111111"}})
	return request.WithContext(ctx)
}

func TestGetNotificationPreferencesReturnsDefaultsWhenUnset(t *testing.T) {
	store := &fakeNotificationPreferencesStore{}
	svc, err := notification.NewPreferencesService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{NotificationPreferences: svc}}
	recorder := httptest.NewRecorder()
	server.getNotificationPreferences(recorder, requestWithNotificationPreferencesAuth(http.MethodGet, ""))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var got notification.StoredPreferences
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got != notification.DefaultStoredPreferences() {
		t.Fatalf("expected defaults, got %#v", got)
	}
}

func TestGetNotificationPreferencesFailsClosedWhenServiceUnavailable(t *testing.T) {
	server := &Server{}
	recorder := httptest.NewRecorder()
	server.getNotificationPreferences(recorder, requestWithNotificationPreferencesAuth(http.MethodGet, ""))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestUpdateNotificationPreferencesRoundTrips(t *testing.T) {
	store := &fakeNotificationPreferencesStore{}
	svc, err := notification.NewPreferencesService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{NotificationPreferences: svc}}
	body := `{"socialEnabled":false,"eventEnabled":true,"recommendationEnabled":false,"promoEnabled":true,"quietHoursStartMinute":1320,"quietHoursEndMinute":420,"timezoneName":"Europe/Kyiv"}`
	recorder := httptest.NewRecorder()
	server.updateNotificationPreferences(recorder, requestWithNotificationPreferencesAuth(http.MethodPut, body))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var got notification.StoredPreferences
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	want := notification.StoredPreferences{
		SocialEnabled: false, EventEnabled: true, RecommendationEnabled: false, PromoEnabled: true,
		QuietHoursStartMinute: 1320, QuietHoursEndMinute: 420, TimezoneName: "Europe/Kyiv",
	}
	if got != want {
		t.Fatalf("got %#v, want %#v", got, want)
	}

	getRecorder := httptest.NewRecorder()
	server.getNotificationPreferences(getRecorder, requestWithNotificationPreferencesAuth(http.MethodGet, ""))
	var roundTripped notification.StoredPreferences
	if err := json.NewDecoder(getRecorder.Body).Decode(&roundTripped); err != nil {
		t.Fatal(err)
	}
	if roundTripped != want {
		t.Fatalf("GET after PUT mismatch: got %#v, want %#v", roundTripped, want)
	}
}

func TestUpdateNotificationPreferencesRejectsInvalidBody(t *testing.T) {
	store := &fakeNotificationPreferencesStore{}
	svc, err := notification.NewPreferencesService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{NotificationPreferences: svc}}
	body := `{"quietHoursStartMinute":-1,"timezoneName":"UTC"}`
	recorder := httptest.NewRecorder()
	server.updateNotificationPreferences(recorder, requestWithNotificationPreferencesAuth(http.MethodPut, body))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var problemBody problem
	if err := json.NewDecoder(recorder.Body).Decode(&problemBody); err != nil {
		t.Fatal(err)
	}
	if problemBody.Code != "invalid_preferences" {
		t.Fatalf("code=%q", problemBody.Code)
	}
}

func TestUpdateNotificationPreferencesRejectsInvalidJSON(t *testing.T) {
	store := &fakeNotificationPreferencesStore{}
	svc, err := notification.NewPreferencesService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{NotificationPreferences: svc}}
	recorder := httptest.NewRecorder()
	server.updateNotificationPreferences(recorder, requestWithNotificationPreferencesAuth(http.MethodPut, "{not json"))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestGetNotificationPreferencesPropagatesStoreError(t *testing.T) {
	store := &fakeNotificationPreferencesStore{err: errors.New("db down")}
	svc, err := notification.NewPreferencesService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{NotificationPreferences: svc}}
	recorder := httptest.NewRecorder()
	server.getNotificationPreferences(recorder, requestWithNotificationPreferencesAuth(http.MethodGet, ""))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
