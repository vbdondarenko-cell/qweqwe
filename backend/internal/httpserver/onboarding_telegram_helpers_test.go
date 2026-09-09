package httpserver

import "testing"

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
