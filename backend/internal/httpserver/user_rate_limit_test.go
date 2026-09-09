package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/ratelimit"
)

func TestAuthenticatedUserRateLimitReturns429AndRetryAfter(t *testing.T) {
	store := &authTestStore{}
	accounts, err := account.NewService(store, password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	limiter, err := ratelimit.New(ratelimit.Config{Limit: 1, Window: time.Minute, IdleTTL: time.Minute, MaxEntries: 100})
	if err != nil {
		t.Fatal(err)
	}
	server := New(Dependencies{Accounts: accounts, UserLimiter: limiter})

	token := registerHTTPUser(t, server)

	first := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	firstRequest.Header.Set("Authorization", "Bearer "+token)
	server.Handler().ServeHTTP(first, firstRequest)
	if first.Code != http.StatusOK {
		t.Fatalf("first authenticated request status=%d body=%s", first.Code, first.Body.String())
	}

	second := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	secondRequest.Header.Set("Authorization", "Bearer "+token)
	server.Handler().ServeHTTP(second, secondRequest)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second authenticated request status=%d body=%s", second.Code, second.Body.String())
	}
	if second.Header().Get("Retry-After") == "" {
		t.Fatal("authenticated rate limit response is missing Retry-After")
	}
}
