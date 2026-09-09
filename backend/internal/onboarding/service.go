package onboarding

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/session"
)

type Service struct {
	store       Store
	password    password.Params
	verifyTTL   time.Duration
	sessionTTL  time.Duration
	botUsername string
	now         func() time.Time
}

func NewService(store Store, passwordParams password.Params, verifyTTL, sessionTTL time.Duration, botUsername string) (*Service, error) {
	botUsername = strings.TrimPrefix(strings.TrimSpace(botUsername), "@")
	if store == nil || verifyTTL <= 0 || sessionTTL <= 0 || botUsername == "" {
		return nil, errors.New("invalid onboarding dependencies")
	}
	if err := passwordParams.Validate(); err != nil {
		return nil, err
	}
	return &Service{store: store, password: passwordParams, verifyTTL: verifyTTL, sessionTTL: sessionTTL, botUsername: botUsername, now: time.Now}, nil
}

func (s *Service) Start(ctx context.Context, in StartInput) (StartResult, error) {
	now := s.now().UTC()
	birthDate, teenMode, err := validateAndNormalize(&in, now)
	if err != nil {
		return StartResult{}, err
	}
	passwordHash, err := password.Hash(in.Password, s.password)
	if err != nil {
		return StartResult{}, ErrInvalidInput
	}
	id, err := identifier.NewUUID()
	if err != nil {
		return StartResult{}, err
	}
	raw, digest, err := session.Generate()
	if err != nil {
		return StartResult{}, err
	}
	pending := PendingRegistration{
		ID: id, Email: in.Email, Username: in.Username, DisplayName: in.DisplayName,
		PasswordHash: passwordHash, Language: in.Language, DeviceLabel: strings.TrimSpace(in.DeviceLabel),
		BirthDate: birthDate, CityID: in.CityID, CityName: in.CityName, Preferences: in.Preferences,
		TeenMode: teenMode, TokenHash: digest[:], CreatedAt: now, ExpiresAt: now.Add(s.verifyTTL),
	}
	if err := s.store.Create(ctx, pending); err != nil {
		return StartResult{}, err
	}
	return StartResult{
		VerificationToken: raw,
		TelegramDeepLink: fmt.Sprintf("https://t.me/%s?start=%s", s.botUsername, raw),
		ExpiresAt: pending.ExpiresAt,
	}, nil
}

func (s *Service) BindTelegram(ctx context.Context, verificationToken string, telegramUserID int64) error {
	if telegramUserID <= 0 {
		return ErrInvalidInput
	}
	digest, err := session.Hash(strings.TrimSpace(verificationToken))
	if err != nil {
		return ErrNotFound
	}
	return s.store.BindTelegram(ctx, digest[:], telegramUserID, s.now().UTC())
}

func (s *Service) VerifyTelegramContact(ctx context.Context, telegramUserID int64, phone string) (string, error) {
	if telegramUserID <= 0 {
		return "", ErrInvalidInput
	}
	phoneE164, ok := normalizePhone(phone)
	if !ok {
		return "", ErrInvalidInput
	}
	return s.store.VerifyTelegramContact(ctx, telegramUserID, phoneE164, s.now().UTC())
}

func (s *Service) Status(ctx context.Context, verificationToken string) (Status, error) {
	digest, err := session.Hash(strings.TrimSpace(verificationToken))
	if err != nil {
		return Status{}, ErrNotFound
	}
	return s.store.Status(ctx, digest[:], s.now().UTC())
}

func (s *Service) Complete(ctx context.Context, verificationToken string) (account.AuthResult, error) {
	digest, err := session.Hash(strings.TrimSpace(verificationToken))
	if err != nil {
		return account.AuthResult{}, ErrNotFound
	}
	userID, err := identifier.NewUUID()
	if err != nil {
		return account.AuthResult{}, err
	}
	sessionID, err := identifier.NewUUID()
	if err != nil {
		return account.AuthResult{}, err
	}
	rawSession, sessionDigest, err := session.Generate()
	if err != nil {
		return account.AuthResult{}, err
	}
	now := s.now().UTC()
	sess := account.Session{ID: sessionID, UserID: userID, TokenHash: sessionDigest[:], CreatedAt: now, ExpiresAt: now.Add(s.sessionTTL)}
	user, err := s.store.Finalize(ctx, digest[:], userID, sess, now)
	if err != nil {
		return account.AuthResult{}, err
	}
	return account.AuthResult{User: user, Token: rawSession, ExpiresAt: sess.ExpiresAt}, nil
}

func validateAndNormalize(in *StartInput, now time.Time) (time.Time, bool, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.Language = strings.TrimSpace(in.Language)
	in.CityID = strings.TrimSpace(in.CityID)
	in.CityName = strings.TrimSpace(in.CityName)
	if in.Language == "" {
		in.Language = "uk"
	}
	if !validEmail(in.Email) || !validUsername(in.Username) || len([]rune(in.DisplayName)) < 1 || len([]rune(in.DisplayName)) > 80 {
		return time.Time{}, false, ErrInvalidInput
	}
	if (in.Language != "uk" && in.Language != "en") || len(in.CityID) > 128 || len([]rune(in.CityName)) < 1 || len([]rune(in.CityName)) > 160 {
		return time.Time{}, false, ErrInvalidInput
	}
	birthDate, err := time.Parse("2006-01-02", strings.TrimSpace(in.BirthDate))
	if err != nil {
		return time.Time{}, false, ErrInvalidInput
	}
	age := ageAt(birthDate, now)
	if age < 14 || age > 120 {
		return time.Time{}, false, ErrInvalidInput
	}
	teenMode := age < 18
	if !validPreferences(in.Preferences, teenMode) {
		return time.Time{}, false, ErrInvalidInput
	}
	return birthDate, teenMode, nil
}

func ageAt(birthDate, now time.Time) int {
	age := now.Year() - birthDate.Year()
	anniversary := time.Date(now.Year(), birthDate.Month(), birthDate.Day(), 0, 0, 0, 0, time.UTC)
	if now.Before(anniversary) {
		age--
	}
	return age
}

func validPreferences(p Preferences, teenMode bool) bool {
	timeAllowed := map[string]bool{"walks": true, "cafes": true, "sports": true, "games": true, "arts_culture": true, "nightlife": true, "travel": true, "networking": true, "unsure": true}
	peopleAllowed := map[string]bool{"new_people": true, "friends": true, "romantic": true, "groups": true, "anyone": true, "unsure": true}
	goalsAllowed := map[string]bool{"new_friends": true, "romantic": true, "activities_events": true, "networking": true, "browsing": true, "unsure": true}
	if !validSelection(p.Time, timeAllowed) || !validSelection(p.People, peopleAllowed) || !validSelection(p.Goals, goalsAllowed) {
		return false
	}
	if teenMode && (contains(p.People, "romantic") || contains(p.Goals, "romantic")) {
		return false
	}
	return true
}

func validSelection(values []string, allowed map[string]bool) bool {
	if len(values) < 1 || len(values) > len(allowed) {
		return false
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !allowed[value] {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func normalizePhone(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	var digits strings.Builder
	for i, r := range raw {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case r == '+' && i == 0:
		case r == ' ' || r == '-' || r == '(' || r == ')':
		default:
			return "", false
		}
	}
	value := digits.String()
	if len(value) < 8 || len(value) > 15 || value[0] == '0' {
		return "", false
	}
	return "+" + value, true
}

func validEmail(v string) bool {
	if len(v) < 3 || len(v) > 320 || strings.ContainsAny(v, " \t\r\n") {
		return false
	}
	at := strings.LastIndexByte(v, '@')
	return at > 0 && at < len(v)-3 && strings.Contains(v[at+1:], ".")
}

func validUsername(v string) bool {
	if len(v) < 3 || len(v) > 32 {
		return false
	}
	for _, r := range v {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '.') {
			return false
		}
	}
	return true
}
