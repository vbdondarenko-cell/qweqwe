package httpserver

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/onboarding"
)

type onboardingStartRequest struct {
	Email       string                 `json:"email"`
	Username    string                 `json:"username"`
	DisplayName string                 `json:"displayName"`
	Password    string                 `json:"password"`
	Language    string                 `json:"language"`
	DeviceLabel string                 `json:"deviceLabel"`
	BirthDate   string                 `json:"birthDate"`
	CityID      string                 `json:"cityId"`
	CityName    string                 `json:"cityName"`
	Preferences onboarding.Preferences `json:"preferences"`
}

type onboardingTokenRequest struct {
	VerificationToken string `json:"verificationToken"`
}

type telegramUpdate struct {
	UpdateID int64            `json:"update_id"`
	Message  *telegramMessage `json:"message"`
}

type telegramMessage struct {
	MessageID int64            `json:"message_id"`
	From      *telegramUser    `json:"from"`
	Chat      telegramChat     `json:"chat"`
	Text      string           `json:"text"`
	Contact   *telegramContact `json:"contact"`
}

type telegramUser struct {
	ID int64 `json:"id"`
}

type telegramChat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type telegramContact struct {
	PhoneNumber string `json:"phone_number"`
	UserID      int64  `json:"user_id"`
}

func (s *Server) startOnboarding(w http.ResponseWriter, r *http.Request) {
	if s.deps.Onboarding == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "registration onboarding is unavailable")
		return
	}
	var in onboardingStartRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	out, err := s.deps.Onboarding.Start(r.Context(), onboarding.StartInput{
		Email: in.Email, Username: in.Username, DisplayName: in.DisplayName, Password: in.Password,
		Language: in.Language, DeviceLabel: in.DeviceLabel, BirthDate: in.BirthDate,
		CityID: in.CityID, CityName: in.CityName, Preferences: in.Preferences,
	})
	if err != nil {
		s.writeOnboardingError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) onboardingStatus(w http.ResponseWriter, r *http.Request) {
	if s.deps.Onboarding == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "registration onboarding is unavailable")
		return
	}
	var in onboardingTokenRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	out, err := s.deps.Onboarding.Status(r.Context(), in.VerificationToken)
	if err != nil {
		s.writeOnboardingError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) completeOnboarding(w http.ResponseWriter, r *http.Request) {
	if s.deps.Onboarding == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "registration onboarding is unavailable")
		return
	}
	var in onboardingTokenRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	out, err := s.deps.Onboarding.Complete(r.Context(), in.VerificationToken)
	if err != nil {
		s.writeOnboardingError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) telegramOnboardingWebhook(w http.ResponseWriter, r *http.Request) {
	if s.deps.Onboarding == nil || s.deps.Telegram == nil || s.deps.TelegramWebhookSecret == "" {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "telegram onboarding is unavailable")
		return
	}
	provided := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
	if !constantTimeEqual(provided, s.deps.TelegramWebhookSecret) {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "invalid webhook authentication")
		return
	}
	var update telegramUpdate
	if err := decodeJSON(w, r, &update); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid Telegram update")
		return
	}
	message := update.Message
	if message == nil || message.From == nil || message.Chat.ID == 0 || message.Chat.Type != "private" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if token, ok := telegramStartToken(message.Text); ok {
		if err := s.deps.Onboarding.BindTelegram(r.Context(), token, message.From.ID); err != nil {
			// Expired/unknown start links are acknowledged without leaking token validity
			// to a retried webhook or creating an infinite Telegram retry loop.
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err := s.deps.Telegram.RequestContact(r.Context(), message.Chat.ID, "uk"); err != nil {
			writeProblem(w, r, http.StatusBadGateway, "telegram_delivery_failed", "failed to request Telegram contact")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if telegramPlainStart(message.Text) {
		if err := s.deps.Telegram.ExplainVerificationStart(r.Context(), message.Chat.ID); err != nil {
			writeProblem(w, r, http.StatusBadGateway, "telegram_delivery_failed", "failed to explain Telegram verification")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if message.Contact != nil {
		// request_contact must represent the sender's own Telegram account. A
		// manually shared third-party contact never verifies phone ownership.
		if message.Contact.UserID <= 0 || message.Contact.UserID != message.From.ID {
			if err := s.deps.Telegram.ExplainUnboundContact(r.Context(), message.Chat.ID); err != nil {
				writeProblem(w, r, http.StatusBadGateway, "telegram_delivery_failed", "failed to explain Telegram contact verification")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		language, err := s.deps.Onboarding.VerifyTelegramContact(r.Context(), message.From.ID, message.Contact.PhoneNumber)
		if err != nil {
			if err := s.deps.Telegram.ExplainUnboundContact(r.Context(), message.Chat.ID); err != nil {
				writeProblem(w, r, http.StatusBadGateway, "telegram_delivery_failed", "failed to explain Telegram contact verification")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err := s.deps.Telegram.ConfirmContact(r.Context(), message.Chat.ID, language); err != nil {
			writeProblem(w, r, http.StatusBadGateway, "telegram_delivery_failed", "failed to confirm Telegram contact")
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func telegramStartToken(text string) (string, bool) {
	parts := strings.Fields(strings.TrimSpace(text))
	if len(parts) != 2 {
		return "", false
	}
	command := strings.SplitN(parts[0], "@", 2)[0]
	if command != "/start" || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func telegramPlainStart(text string) bool {
	parts := strings.Fields(strings.TrimSpace(text))
	if len(parts) != 1 {
		return false
	}
	return strings.SplitN(parts[0], "@", 2)[0] == "/start"
}

func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) || len(a) == 0 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (s *Server) writeOnboardingError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, onboarding.ErrInvalidInput):
		writeProblem(w, r, http.StatusBadRequest, "invalid_onboarding", "invalid registration data")
	case errors.Is(err, onboarding.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "registration_not_found", "registration session not found")
	case errors.Is(err, onboarding.ErrExpired):
		writeProblem(w, r, http.StatusGone, "registration_expired", "registration session expired")
	case errors.Is(err, onboarding.ErrNotVerified):
		writeProblem(w, r, http.StatusConflict, "phone_not_verified", "Telegram phone verification is required")
	case errors.Is(err, onboarding.ErrConflict):
		writeProblem(w, r, http.StatusConflict, "registration_conflict", "registration data is already in use")
	default:
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
	}
}
