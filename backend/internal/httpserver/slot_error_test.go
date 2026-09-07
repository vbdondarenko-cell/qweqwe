package httpserver

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

func TestRequestQueueFullMapsToConflict(t *testing.T) {
    req := httptest.NewRequest(http.MethodPost, "/v1/slots/slot/request", nil)
    req.Header.Set("X-Request-ID", "request-id")
    rec := httptest.NewRecorder()

    (&Server{}).writeSlotError(rec, req, slot.ErrRequestLimit)
    if rec.Code != http.StatusConflict {
        t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
    }
    var problem struct {
        Code string `json:"code"`
    }
    if err := json.NewDecoder(rec.Body).Decode(&problem); err != nil {
        t.Fatal(err)
    }
    if problem.Code != "request_queue_full" {
        t.Fatalf("code=%q", problem.Code)
    }
}
