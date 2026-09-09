package httpserver

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTelegramStartParsing(t *testing.T) {
	token := "abcdefghijklmnopqrstuvwxyz0123456789_-ABC"
	got, ok := telegramStartToken("/start " + token)
	if !ok || got != token {
		t.Fatalf("telegramStartToken() = %q, %v", got, ok)
	}
	got, ok = telegramStartToken("/start@LinkUpNetworkbot " + token)
	if !ok || got != token {
		t.Fatalf("telegramStartToken(bot) = %q, %v", got, ok)
	}
	if _, ok := telegramStartToken("/start"); ok {
		t.Fatal("plain /start must not be treated as token start")
	}
}

func TestTelegramPlainStart(t *testing.T) {
	for _, text := range []string{"/start", " /start ", "/start@LinkUpNetworkbot"} {
		if !telegramPlainStart(text) {
			t.Fatalf("telegramPlainStart(%q) = false", text)
		}
	}
	for _, text := range []string{"/start token", "/help", "start"} {
		if telegramPlainStart(text) {
			t.Fatalf("telegramPlainStart(%q) = true", text)
		}
	}
}

func TestDecodeTelegramUpdateAllowsTelegramExtraFields(t *testing.T) {
	body := `{"update_id":1,"message":{"message_id":2,"date":1788968000,"from":{"id":123,"is_bot":false,"first_name":"A","username":"u"},"chat":{"id":123,"type":"private","first_name":"A","username":"u"},"text":"/start","entities":[{"offset":0,"length":6,"type":"bot_command"}]}}`
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	rec := httptest.NewRecorder()
	var update telegramUpdate
	if err := decodeTelegramUpdate(rec, req, &update); err != nil {
		t.Fatalf("decodeTelegramUpdate() error = %v", err)
	}
	if update.UpdateID != 1 || update.Message == nil || update.Message.Text != "/start" {
		t.Fatalf("unexpected update: %#v", update)
	}
}

func TestDecodeTelegramUpdateRejectsSecondJSONValue(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"update_id":1} {"update_id":2}`))
	rec := httptest.NewRecorder()
	var update telegramUpdate
	if err := decodeTelegramUpdate(rec, req, &update); err == nil {
		t.Fatal("expected trailing JSON value to be rejected")
	}
}
