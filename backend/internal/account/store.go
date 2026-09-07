package account

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound             = errors.New("account not found")
	ErrConflict             = errors.New("account conflict")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrInvalidInput         = errors.New("invalid input")
	ErrRecoveryUnavailable  = errors.New("password recovery unavailable")
)

type Store interface {
	Register(ctx context.Context, user User, passwordHash string, session Session) error
	FindByLogin(ctx context.Context, identifier string) (UserWithPassword, error)
	CreateSession(ctx context.Context, session Session, expectedPasswordHash string) error
	Authenticate(ctx context.Context, tokenHash []byte, now time.Time) (User, string, error)
	RevokeSession(ctx context.Context, tokenHash []byte, now time.Time) error
	UpdateProfile(ctx context.Context, userID string, patch ProfilePatch, now time.Time) (User, error)
	CreatePasswordReset(ctx context.Context, reset PasswordReset, now time.Time) error
	ResetPassword(ctx context.Context, tokenHash []byte, newPasswordHash string, now time.Time) error
}

type RecoveryNotifier interface {
	SendPasswordReset(ctx context.Context, recipientEmail, rawToken string, expiresAt time.Time) error
}
