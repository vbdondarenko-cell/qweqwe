package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
)

type Config struct {
	HTTPAddr               string
	DatabaseURL            string
	SessionTTL             time.Duration
	PasswordResetTTL       time.Duration
	RegistrationTTL        time.Duration
	IdempotencyTTL         time.Duration
	MigrationDir           string
	ArgonMemoryKiB         uint32
	ArgonIterations        uint32
	ArgonParallel          uint8
	AuthRateLimit          int
	AuthRateWindow         time.Duration
	AuthRateIdleTTL        time.Duration
	AuthRateMaxEntries     int
	SocialRateLimit        int
	SocialRateWindow       time.Duration
	SocialRateIdleTTL      time.Duration
	SocialRateMaxEntries   int
	RecoverySMTPAddress    string
	RecoverySMTPHost       string
	RecoverySMTPUsername   string
	RecoverySMTPPassword   string
	RecoveryFrom           string
	RecoveryResetURL       string
	RecoveryImplicitTLS    bool
	TelegramBotToken       string
	TelegramBotUsername    string
	TelegramWebhookSecret  string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:              envOr("LINKUP_HTTP_ADDR", ":8080"),
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		SessionTTL:            30 * 24 * time.Hour,
		PasswordResetTTL:      30 * time.Minute,
		RegistrationTTL:       20 * time.Minute,
		IdempotencyTTL:        24 * time.Hour,
		MigrationDir:          envOr("LINKUP_MIGRATIONS_DIR", "../db/migrations"),
		ArgonMemoryKiB:        19 * 1024,
		ArgonIterations:       2,
		ArgonParallel:         1,
		AuthRateLimit:         10,
		AuthRateWindow:        time.Minute,
		AuthRateIdleTTL:       10 * time.Minute,
		AuthRateMaxEntries:    20_000,
		SocialRateLimit:       120,
		SocialRateWindow:      time.Minute,
		SocialRateIdleTTL:     10 * time.Minute,
		SocialRateMaxEntries:  20_000,
		RecoverySMTPAddress:   strings.TrimSpace(os.Getenv("LINKUP_RECOVERY_SMTP_ADDR")),
		RecoverySMTPHost:      strings.TrimSpace(os.Getenv("LINKUP_RECOVERY_SMTP_HOST")),
		RecoverySMTPUsername:  strings.TrimSpace(os.Getenv("LINKUP_RECOVERY_SMTP_USERNAME")),
		RecoverySMTPPassword:  os.Getenv("LINKUP_RECOVERY_SMTP_PASSWORD"),
		RecoveryFrom:          strings.TrimSpace(os.Getenv("LINKUP_RECOVERY_FROM")),
		RecoveryResetURL:      strings.TrimSpace(os.Getenv("LINKUP_RECOVERY_RESET_URL")),
		TelegramBotToken:      strings.TrimSpace(os.Getenv("LINKUP_TELEGRAM_BOT_TOKEN")),
		TelegramBotUsername:   strings.TrimPrefix(strings.TrimSpace(os.Getenv("LINKUP_TELEGRAM_BOT_USERNAME")), "@"),
		TelegramWebhookSecret: strings.TrimSpace(os.Getenv("LINKUP_TELEGRAM_WEBHOOK_SECRET")),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	var err error
	if cfg.SessionTTL, err = durationEnv("LINKUP_SESSION_TTL", cfg.SessionTTL); err != nil {
		return Config{}, err
	}
	if cfg.PasswordResetTTL, err = durationEnv("LINKUP_PASSWORD_RESET_TTL", cfg.PasswordResetTTL); err != nil {
		return Config{}, err
	}
	if cfg.RegistrationTTL, err = durationEnv("LINKUP_REGISTRATION_TTL", cfg.RegistrationTTL); err != nil {
		return Config{}, err
	}
	if cfg.IdempotencyTTL, err = durationEnv("LINKUP_IDEMPOTENCY_TTL", cfg.IdempotencyTTL); err != nil {
		return Config{}, err
	}
	if cfg.ArgonMemoryKiB, err = uint32Env("LINKUP_ARGON_MEMORY_KIB", cfg.ArgonMemoryKiB); err != nil {
		return Config{}, err
	}
	if cfg.ArgonIterations, err = uint32Env("LINKUP_ARGON_ITERATIONS", cfg.ArgonIterations); err != nil {
		return Config{}, err
	}
	parallel, err := uint32Env("LINKUP_ARGON_PARALLELISM", uint32(cfg.ArgonParallel))
	if err != nil || parallel > 255 {
		if err == nil {
			err = errors.New("LINKUP_ARGON_PARALLELISM must be <=255")
		}
		return Config{}, err
	}
	cfg.ArgonParallel = uint8(parallel)
	if err := (password.Params{
		MemoryKiB: cfg.ArgonMemoryKiB,
		Iterations: cfg.ArgonIterations,
		Parallel: cfg.ArgonParallel,
		SaltBytes: 16,
		KeyBytes: 32,
	}).Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid Argon2id configuration: %w", err)
	}
	if cfg.AuthRateLimit, err = positiveIntEnv("LINKUP_AUTH_RATE_LIMIT", cfg.AuthRateLimit); err != nil {
		return Config{}, err
	}
	if cfg.AuthRateWindow, err = durationEnv("LINKUP_AUTH_RATE_WINDOW", cfg.AuthRateWindow); err != nil {
		return Config{}, err
	}
	if cfg.AuthRateIdleTTL, err = durationEnv("LINKUP_AUTH_RATE_IDLE_TTL", cfg.AuthRateIdleTTL); err != nil {
		return Config{}, err
	}
	if cfg.AuthRateMaxEntries, err = positiveIntEnv("LINKUP_AUTH_RATE_MAX_ENTRIES", cfg.AuthRateMaxEntries); err != nil {
		return Config{}, err
	}
	if cfg.AuthRateIdleTTL < cfg.AuthRateWindow {
		return Config{}, errors.New("LINKUP_AUTH_RATE_IDLE_TTL must be >= LINKUP_AUTH_RATE_WINDOW")
	}
	if cfg.SocialRateLimit, err = positiveIntEnv("LINKUP_SOCIAL_RATE_LIMIT", cfg.SocialRateLimit); err != nil {
		return Config{}, err
	}
	if cfg.SocialRateWindow, err = durationEnv("LINKUP_SOCIAL_RATE_WINDOW", cfg.SocialRateWindow); err != nil {
		return Config{}, err
	}
	if cfg.SocialRateIdleTTL, err = durationEnv("LINKUP_SOCIAL_RATE_IDLE_TTL", cfg.SocialRateIdleTTL); err != nil {
		return Config{}, err
	}
	if cfg.SocialRateMaxEntries, err = positiveIntEnv("LINKUP_SOCIAL_RATE_MAX_ENTRIES", cfg.SocialRateMaxEntries); err != nil {
		return Config{}, err
	}
	if cfg.SocialRateIdleTTL < cfg.SocialRateWindow {
		return Config{}, errors.New("LINKUP_SOCIAL_RATE_IDLE_TTL must be >= LINKUP_SOCIAL_RATE_WINDOW")
	}
	if cfg.RecoveryImplicitTLS, err = boolEnv("LINKUP_RECOVERY_SMTP_IMPLICIT_TLS", false); err != nil {
		return Config{}, err
	}
	if recoveryTransportFieldsPresent(cfg) && cfg.RecoverySMTPAddress == "" {
		return Config{}, errors.New("LINKUP_RECOVERY_SMTP_ADDR is required when recovery SMTP is configured")
	}
	if err := validateTelegramConfig(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func recoveryTransportFieldsPresent(cfg Config) bool {
	return cfg.RecoverySMTPAddress != "" || cfg.RecoverySMTPHost != "" || cfg.RecoverySMTPUsername != "" || cfg.RecoverySMTPPassword != "" || cfg.RecoveryFrom != ""
}

func validateTelegramConfig(cfg Config) error {
	configured := 0
	if cfg.TelegramBotToken != "" { configured++ }
	if cfg.TelegramBotUsername != "" { configured++ }
	if cfg.TelegramWebhookSecret != "" { configured++ }
	if configured == 0 {
		return nil
	}
	if configured != 3 {
		return errors.New("LINKUP_TELEGRAM_BOT_TOKEN, LINKUP_TELEGRAM_BOT_USERNAME and LINKUP_TELEGRAM_WEBHOOK_SECRET must be configured together")
	}
	if len(cfg.TelegramWebhookSecret) < 16 || len(cfg.TelegramWebhookSecret) > 256 {
		return errors.New("LINKUP_TELEGRAM_WEBHOOK_SECRET must be 16..256 characters")
	}
	for _, r := range cfg.TelegramWebhookSecret {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return errors.New("LINKUP_TELEGRAM_WEBHOOK_SECRET contains unsupported characters")
		}
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	if seconds, err := strconv.ParseInt(v, 10, 64); err == nil {
		const maxDurationSeconds = int64(^uint64(0)>>1) / int64(time.Second)
		if seconds <= 0 || seconds > maxDurationSeconds {
			return 0, fmt.Errorf("%s integer seconds are outside time.Duration range", key)
		}
		return time.Duration(seconds) * time.Second, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration or integer seconds", key)
	}
	return d, nil
}

func uint32Env(key string, fallback uint32) (uint32, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.ParseUint(v, 10, 32)
	if err != nil || n == 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return uint32(n), nil
}

func positiveIntEnv(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return n, nil
}

func boolEnv(key string, fallback bool) (bool, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean", key)
	}
	return b, nil
}
