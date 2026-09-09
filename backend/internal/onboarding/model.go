package onboarding

import (
	"context"
	"errors"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
)

var (
	ErrInvalidInput = errors.New("invalid onboarding input")
	ErrNotFound     = errors.New("onboarding registration not found")
	ErrExpired      = errors.New("onboarding registration expired")
	ErrNotVerified  = errors.New("telegram phone verification required")
	ErrConflict     = errors.New("onboarding conflict")
)

type Preferences struct {
	Time   []string `json:"time"`
	People []string `json:"people"`
	Goals  []string `json:"goals"`
}

type StartInput struct {
	Email       string
	Username    string
	DisplayName string
	Password    string
	Language    string
	DeviceLabel string
	BirthDate   string
	CityID      string
	CityName    string
	Preferences Preferences
}

type PendingRegistration struct {
	ID           string
	Email        string
	Username     string
	DisplayName  string
	PasswordHash string
	Language     string
	DeviceLabel  string
	BirthDate    time.Time
	CityID       string
	CityName     string
	Preferences  Preferences
	TeenMode     bool
	TokenHash    []byte
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

type StartResult struct {
	VerificationToken string    `json:"verificationToken"`
	TelegramDeepLink  string    `json:"telegramDeepLink"`
	ExpiresAt         time.Time `json:"expiresAt"`
}

type Status struct {
	PhoneVerified bool       `json:"phoneVerified"`
	TeenMode      bool       `json:"teenMode"`
	ExpiresAt     time.Time  `json:"expiresAt"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
}

type Store interface {
	IdentityAvailable(ctx context.Context, email, username string) (bool, error)
	Create(ctx context.Context, pending PendingRegistration) error
	BindTelegram(ctx context.Context, tokenHash []byte, telegramUserID int64, now time.Time) error
	VerifyTelegramContact(ctx context.Context, telegramUserID int64, phoneE164 string, now time.Time) (string, error)
	Status(ctx context.Context, tokenHash []byte, now time.Time) (Status, error)
	Finalize(ctx context.Context, tokenHash []byte, userID string, session account.Session, now time.Time) (account.User, error)
}
