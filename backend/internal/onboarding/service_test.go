package onboarding

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/session"
)

type onboardingTestStore struct {
	identityConflict bool
	pending          PendingRegistration
	boundTelegramID  int64
	verifiedPhone    string
	verifiedLang     string
	status           Status
	finalized        bool
}

func (s *onboardingTestStore) IdentityAvailable(_ context.Context, _, _ string) (bool, error) {
	return !s.identityConflict, nil
}

func (s *onboardingTestStore) Create(_ context.Context, pending PendingRegistration) error {
	s.pending = pending
	s.status = Status{TeenMode: pending.TeenMode, ExpiresAt: pending.ExpiresAt}
	return nil
}

func (s *onboardingTestStore) BindTelegram(_ context.Context, tokenHash []byte, telegramUserID int64, _ time.Time) error {
	if !bytes.Equal(tokenHash, s.pending.TokenHash) {
		return ErrNotFound
	}
	s.boundTelegramID = telegramUserID
	return nil
}

func (s *onboardingTestStore) VerifyTelegramContact(_ context.Context, telegramUserID int64, phoneE164 string, _ time.Time) (string, error) {
	if telegramUserID != s.boundTelegramID {
		return "", ErrNotFound
	}
	s.verifiedPhone = phoneE164
	s.status.PhoneVerified = true
	if s.verifiedLang == "" {
		s.verifiedLang = s.pending.Language
	}
	return s.verifiedLang, nil
}

func (s *onboardingTestStore) Status(_ context.Context, tokenHash []byte, _ time.Time) (Status, error) {
	if !bytes.Equal(tokenHash, s.pending.TokenHash) {
		return Status{}, ErrNotFound
	}
	return s.status, nil
}

func (s *onboardingTestStore) Finalize(_ context.Context, tokenHash []byte, userID string, sess account.Session, now time.Time) (account.User, error) {
	if !bytes.Equal(tokenHash, s.pending.TokenHash) {
		return account.User{}, ErrNotFound
	}
	if !s.status.PhoneVerified {
		return account.User{}, ErrNotVerified
	}
	s.finalized = true
	return account.User{
		ID: userID, Email: s.pending.Email, Username: s.pending.Username,
		DisplayName: s.pending.DisplayName, ProfileVisibility: "PUBLIC",
		Language: s.pending.Language, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func validStartInput() StartInput {
	return StartInput{
		Email: "Alice@Example.com", Username: "Alice_1", DisplayName: " Alice ",
		Password: "correct horse battery staple", Language: "uk", DeviceLabel: "Pixel",
		BirthDate: "2010-09-10", CityID: "kyiv", CityName: "Kyiv",
		Preferences: Preferences{
			Time: []string{"walks"}, People: []string{"friends"}, Goals: []string{"new_friends"},
		},
	}
}

func TestStartDerivesTeenModeAndPersistsOnlyTokenHash(t *testing.T) {
	store := &onboardingTestStore{}
	service, err := NewService(store, password.OWASPMinimum(), 20*time.Minute, time.Hour, "LinkUpBot")
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC) }

	out, err := service.Start(context.Background(), validStartInput())
	if err != nil {
		t.Fatal(err)
	}
	if out.VerificationToken == "" || out.TelegramDeepLink != "https://t.me/LinkUpBot?start="+out.VerificationToken {
		t.Fatalf("unexpected start result: %+v", out)
	}
	if !store.pending.TeenMode {
		t.Fatal("expected Teen Mode for age 15")
	}
	if store.pending.Email != "alice@example.com" || store.pending.Username != "alice_1" || store.pending.DisplayName != "Alice" {
		t.Fatalf("normalization failed: %+v", store.pending)
	}
	if store.pending.PasswordHash == validStartInput().Password || store.pending.PasswordHash == "" {
		t.Fatal("raw password persisted instead of Argon2id hash")
	}
	digest, err := session.Hash(out.VerificationToken)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(store.pending.TokenHash, digest[:]) {
		t.Fatal("stored verification token is not the expected digest")
	}
}

func TestStartRejectsExistingIdentityBeforePendingCreation(t *testing.T) {
	store := &onboardingTestStore{identityConflict: true}
	service, err := NewService(store, password.OWASPMinimum(), 20*time.Minute, time.Hour, "LinkUpBot")
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC) }
	if _, err := service.Start(context.Background(), validStartInput()); err != ErrConflict {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
	if store.pending.ID != "" {
		t.Fatal("conflicting identity must not create pending registration")
	}
}

func TestTeenModeRejectsRomanticPreference(t *testing.T) {
	store := &onboardingTestStore{}
	service, err := NewService(store, password.OWASPMinimum(), 20*time.Minute, time.Hour, "LinkUpBot")
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC) }
	in := validStartInput()
	in.Preferences.Goals = []string{"romantic"}
	if _, err := service.Start(context.Background(), in); err != ErrInvalidInput {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestTelegramContactIsNormalizedAndRequiredForCompletion(t *testing.T) {
	store := &onboardingTestStore{}
	service, err := NewService(store, password.OWASPMinimum(), 20*time.Minute, time.Hour, "LinkUpBot")
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC) }
	out, err := service.Start(context.Background(), validStartInput())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Complete(context.Background(), out.VerificationToken); err != ErrNotVerified {
		t.Fatalf("completion before Telegram verification: %v", err)
	}
	if err := service.BindTelegram(context.Background(), out.VerificationToken, 12345); err != nil {
		t.Fatal(err)
	}
	language, err := service.VerifyTelegramContact(context.Background(), 12345, "+380 (67) 123-45-67")
	if err != nil {
		t.Fatal(err)
	}
	if language != "uk" || store.verifiedPhone != "+380671234567" {
		t.Fatalf("unexpected verification result language=%q phone=%q", language, store.verifiedPhone)
	}
	auth, err := service.Complete(context.Background(), out.VerificationToken)
	if err != nil {
		t.Fatal(err)
	}
	if !store.finalized || auth.Token == "" || auth.User.Email != "alice@example.com" {
		t.Fatalf("unexpected completion: %+v", auth)
	}
}
