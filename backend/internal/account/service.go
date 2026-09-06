package account

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/session"
)

type Service struct {
	store      Store
	password   password.Params
	sessionTTL time.Duration
	now        func() time.Time
}

func NewService(store Store, passwordParams password.Params, sessionTTL time.Duration) (*Service, error) {
	if store == nil || sessionTTL <= 0 { return nil, errors.New("invalid account service dependencies") }
	if err := passwordParams.Validate(); err != nil { return nil, err }
	return &Service{store: store, password: passwordParams, sessionTTL: sessionTTL, now: time.Now}, nil
}

func (s *Service) Register(ctx context.Context, in Registration) (AuthResult, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if in.Language == "" { in.Language = "uk" }
	if !validEmail(in.Email) || !validUsername(in.Username) || len([]rune(in.DisplayName)) < 1 || len([]rune(in.DisplayName)) > 80 || (in.Language != "uk" && in.Language != "en") {
		return AuthResult{}, ErrInvalidInput
	}
	hash, err := password.Hash(in.Password, s.password)
	if err != nil { return AuthResult{}, ErrInvalidInput }
	userID, err := identifier.NewUUID(); if err != nil { return AuthResult{}, err }
	sessionID, err := identifier.NewUUID(); if err != nil { return AuthResult{}, err }
	raw, digest, err := session.Generate(); if err != nil { return AuthResult{}, err }
	now := s.now().UTC()
	user := User{ID: userID, Email: in.Email, Username: in.Username, DisplayName: in.DisplayName, ProfileVisibility: "PUBLIC", Language: in.Language, CreatedAt: now, UpdatedAt: now}
	sess := Session{ID: sessionID, UserID: userID, TokenHash: digest[:], CreatedAt: now, ExpiresAt: now.Add(s.sessionTTL), DeviceLabel: strings.TrimSpace(in.DeviceLabel)}
	if err := s.store.Register(ctx, user, hash, sess); err != nil { return AuthResult{}, err }
	return AuthResult{User: user, Token: raw, ExpiresAt: sess.ExpiresAt}, nil
}

func (s *Service) Login(ctx context.Context, in Login) (AuthResult, error) {
	identifierText := strings.ToLower(strings.TrimSpace(in.Identifier))
	if identifierText == "" || in.Password == "" { return AuthResult{}, ErrUnauthorized }
	found, err := s.store.FindByLogin(ctx, identifierText)
	if err != nil { return AuthResult{}, ErrUnauthorized }
	ok, err := password.Verify(found.PasswordHash, in.Password)
	if err != nil || !ok { return AuthResult{}, ErrUnauthorized }
	sessionID, err := identifier.NewUUID(); if err != nil { return AuthResult{}, err }
	raw, digest, err := session.Generate(); if err != nil { return AuthResult{}, err }
	now := s.now().UTC()
	sess := Session{ID: sessionID, UserID: found.ID, TokenHash: digest[:], CreatedAt: now, ExpiresAt: now.Add(s.sessionTTL), DeviceLabel: strings.TrimSpace(in.DeviceLabel)}
	if err := s.store.CreateSession(ctx, sess); err != nil { return AuthResult{}, err }
	return AuthResult{User: found.User, Token: raw, ExpiresAt: sess.ExpiresAt}, nil
}

func (s *Service) Authenticate(ctx context.Context, raw string) (User, string, error) {
	digest, err := session.Hash(raw)
	if err != nil { return User{}, "", ErrUnauthorized }
	return s.store.Authenticate(ctx, digest[:], s.now().UTC())
}

func (s *Service) Logout(ctx context.Context, raw string) error {
	digest, err := session.Hash(raw)
	if err != nil { return nil }
	if err := s.store.RevokeSession(ctx, digest[:], s.now().UTC()); err != nil && !errors.Is(err, ErrNotFound) { return err }
	return nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, patch ProfilePatch) (User, error) {
	if patch.DisplayName != nil { v := strings.TrimSpace(*patch.DisplayName); if len([]rune(v)) < 1 || len([]rune(v)) > 80 { return User{}, ErrInvalidInput }; patch.DisplayName = &v }
	if patch.AvatarURL != nil { v := strings.TrimSpace(*patch.AvatarURL); if len(v) > 2048 { return User{}, ErrInvalidInput }; patch.AvatarURL = &v }
	if patch.ProfileVisibility != nil && *patch.ProfileVisibility != "PUBLIC" && *patch.ProfileVisibility != "HIDDEN" { return User{}, ErrInvalidInput }
	if patch.Language != nil && *patch.Language != "uk" && *patch.Language != "en" { return User{}, ErrInvalidInput }
	return s.store.UpdateProfile(ctx, userID, patch, s.now().UTC())
}

func validEmail(v string) bool {
	if len(v) < 3 || len(v) > 320 || strings.ContainsAny(v, " \t\r\n") { return false }
	at := strings.LastIndexByte(v, '@')
	return at > 0 && at < len(v)-3 && strings.Contains(v[at+1:], ".")
}

func validUsername(v string) bool {
	if len(v) < 3 || len(v) > 32 { return false }
	for _, r := range v { if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '.') { return false } }
	return true
}
