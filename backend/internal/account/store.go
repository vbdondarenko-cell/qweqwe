package account

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound     = errors.New("account not found")
	ErrConflict     = errors.New("account conflict")
	ErrUnauthorized = errors.New("unauthorized")
	ErrInvalidInput = errors.New("invalid input")
)

type Store interface {
	Register(ctx context.Context, user User, passwordHash string, session Session) error
	FindByLogin(ctx context.Context, identifier string) (UserWithPassword, error)
	CreateSession(ctx context.Context, session Session) error
	Authenticate(ctx context.Context, tokenHash []byte, now time.Time) (User, string, error)
	RevokeSession(ctx context.Context, tokenHash []byte, now time.Time) error
	UpdateProfile(ctx context.Context, userID string, patch ProfilePatch, now time.Time) (User, error)
}
