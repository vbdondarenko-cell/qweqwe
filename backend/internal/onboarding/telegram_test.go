package onboarding

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestExplainVerificationStartShowsContactButton(t *testing.T) {
	var sent map[string]any
	bot := &TelegramBot{
		token: "test-token",
		client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(body, &sent); err != nil {
				t.Fatal(err)
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}, nil
		})},
	}
	if err := bot.ExplainVerificationStart(context.Background(), 123); err != nil {
		t.Fatal(err)
	}
	markup, ok := sent["reply_markup"].(map[string]any)
	if !ok {
		t.Fatalf("missing reply_markup: %#v", sent)
	}
	rows, ok := markup["keyboard"].([]any)
	if !ok || len(rows) == 0 {
		t.Fatalf("missing keyboard: %#v", markup)
	}
	firstRow, ok := rows[0].([]any)
	if !ok || len(firstRow) == 0 {
		t.Fatalf("missing first row: %#v", rows)
	}
	button, ok := firstRow[0].(map[string]any)
	if !ok {
		t.Fatalf("missing contact button: %#v", firstRow)
	}
	if button["request_contact"] != true {
		t.Fatalf("request_contact button required: %#v", button)
	}
}
