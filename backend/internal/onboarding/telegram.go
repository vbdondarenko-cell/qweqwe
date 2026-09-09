package onboarding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ContactPrompter interface {
	RequestContact(ctx context.Context, chatID int64, language string) error
	ConfirmContact(ctx context.Context, chatID int64, language string) error
}

type TelegramBot struct {
	token  string
	client *http.Client
}

func NewTelegramBot(token string) (*TelegramBot, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("telegram bot token is required")
	}
	return &TelegramBot{token: token, client: &http.Client{Timeout: 8 * time.Second}}, nil
}

func (b *TelegramBot) RequestContact(ctx context.Context, chatID int64, language string) error {
	text := "Підтвердьте номер телефону для LinkUp. Натисніть кнопку нижче."
	button := "Поділитися номером"
	if language == "en" {
		text = "Verify your phone number for LinkUp. Tap the button below."
		button = "Share phone number"
	}
	payload := map[string]any{
		"chat_id": chatID,
		"text": text,
		"reply_markup": map[string]any{
			"keyboard": [][]map[string]any{{{"text": button, "request_contact": true}}},
			"resize_keyboard": true,
			"one_time_keyboard": true,
			"input_field_placeholder": button,
		},
	}
	return b.call(ctx, "sendMessage", payload)
}

func (b *TelegramBot) ConfirmContact(ctx context.Context, chatID int64, language string) error {
	text := "Номер підтверджено. Поверніться в LinkUp, щоб завершити реєстрацію."
	if language == "en" {
		text = "Phone verified. Return to LinkUp to finish registration."
	}
	payload := map[string]any{
		"chat_id": chatID,
		"text": text,
		"reply_markup": map[string]any{"remove_keyboard": true},
	}
	return b.call(ctx, "sendMessage", payload)
}

func (b *TelegramBot) call(ctx context.Context, method string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/%s", b.token, method)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 32<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram api returned status %d", resp.StatusCode)
	}
	return nil
}
