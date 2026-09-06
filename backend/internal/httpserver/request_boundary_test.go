package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/session"
)

func TestDecodeJSONConsumesWholeBoundedBody(t *testing.T) {
	for _, tc := range []struct { name, body string; valid bool }{
		{"single", `{"name":"Alice"}`, true},
		{"whitespace", "{\"name\":\"Alice\"}\n\t ", true},
		{"second object", `{"name":"Alice"}{"name":"Bob"}`, false},
		{"trailing null", `{"name":"Alice"}null`, false},
		{"trailing garbage", `{"name":"Alice"}oops`, false},
		{"unknown field", `{"admin":true}`, false},
		{"empty", "", false},
		{"oversized suffix", `{"name":"Alice"}` + strings.Repeat(" ", 64<<10), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body struct { Name string `json:"name"` }
			err := decodeJSON(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body)), &body)
			if (err == nil) != tc.valid { t.Fatalf("valid=%v, error=%v", tc.valid, err) }
		})
	}
}

type unavailableAuthStore struct { authTestStore }

func (*unavailableAuthStore) Authenticate(context.Context, []byte, time.Time) (account.User, string, error) {
	return account.User{}, "", errors.New("private database connection detail")
}
func (*unavailableAuthStore) FindByLogin(context.Context, string) (account.UserWithPassword, error) {
	return account.UserWithPassword{}, errors.New("private database connection detail")
}

func TestAuthOutageDoesNotRevokeClientSession(t *testing.T) {
	accounts, err := account.NewService(&unavailableAuthStore{}, password.OWASPMinimum(), time.Hour)
	if err != nil { t.Fatal(err) }
	srv := New(Dependencies{Accounts: accounts})
	token, _, err := session.Generate()
	if err != nil { t.Fatal(err) }
	for _, tc := range []struct { method, path, body string }{
		{http.MethodGet, "/v1/me", ""},
		{http.MethodPost, "/v1/auth/login", `{"identifier":"alice","password":"some password"}`},
	} {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer " + token)
			response := httptest.NewRecorder()
			srv.Handler().ServeHTTP(response, req)
			if response.Code != http.StatusServiceUnavailable { t.Fatalf("status=%d body=%s", response.Code, response.Body.String()) }
			if strings.Contains(response.Body.String(), "private database") { t.Fatal("storage error leaked") }
		})
	}
}

func TestRegisterRejectsTrailingCommandBeforeWriting(t *testing.T) {
	store := &authTestStore{}
	accounts, err := account.NewService(store, password.OWASPMinimum(), time.Hour)
	if err != nil { t.Fatal(err) }
	body := `{"email":"alice@example.com","username":"alice","displayName":"Alice","password":"some password"}{}`
	response := httptest.NewRecorder()
	New(Dependencies{Accounts: accounts}).Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(body)))
	if response.Code != http.StatusBadRequest || store.user.ID != "" { t.Fatalf("status=%d user=%q", response.Code, store.user.ID) }
}
