package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoints(t *testing.T) {
	t.Parallel()

	srv := New(Dependencies{})
	for _, path := range []string{"/livez", "/healthz"} {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rr := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("%s status = %d, want %d", path, rr.Code, http.StatusOK)
			}
			if got := rr.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
				t.Fatalf("%s content-type = %q", path, got)
			}
		}
	}
}
