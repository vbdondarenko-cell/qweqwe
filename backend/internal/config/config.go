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
	HTTPAddr             string
	DatabaseURL          string
	SessionTTL           time.Duration
	PasswordResetTTL     time.Duration
	RegistrationTTL      time.Duration
	IdempotencyTTL       time.Duration
	WaitlistRequestTTL   time.Duration
	MigrationDir         string
	ArgonMemoryKiB       uint32
	ArgonIterations      uint32
	ArgonParallel        uint8
	AuthRateLimit        int
	AuthRateWindow       time.Duration
	AuthRateIdleTTL      time.Duration
	AuthRateMaxEntries   int
	SocialRateLimit      int
	SocialRateWindow     time.Duration
	SocialRateIdleTTL    time.Duration
	SocialRateMaxEntries int
	// MonetizationRate* is a separate, tighter budget for entitlement-
	// sensitive endpoints (referral binding, rewarded-view submission,
	// purchase verification), independent of SocialRate's general
	// per-authenticated-request limit — docs/LINKUP_PLUS_MONETIZATION.md
	// §8.2.
	MonetizationRateLimit          int
	MonetizationRateWindow         time.Duration
	MonetizationRateIdleTTL        time.Duration
	MonetizationRateMaxEntries     int
	RecoverySMTPAddress            string
	RecoverySMTPHost               string
	RecoverySMTPUsername           string
	RecoverySMTPPassword           string
	RecoveryFrom                   string
	RecoveryResetURL               string
	RecoveryImplicitTLS            bool
	TelegramBotToken               string
	TelegramBotUsername            string
	TelegramWebhookSecret          string
	NotificationTTL                time.Duration
	NotificationFrequencyCapMax    int
	NotificationFrequencyCapWindow time.Duration
	NotificationGroupWindow        time.Duration
	NotificationPollInterval       time.Duration
	NotificationBatchSize          int
	BumpChallengeTTL               time.Duration
	EventReminderLeadTime          time.Duration
	EventReminderPollInterval      time.Duration
	EventReminderBatchSize         int
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:                       envOr("LINKUP_HTTP_ADDR", ":8080"),
		DatabaseURL:                    os.Getenv("DATABASE_URL"),
		SessionTTL:                     30 * 24 * time.Hour,
		PasswordResetTTL:               30 * time.Minute,
		RegistrationTTL:                20 * time.Minute,
		IdempotencyTTL:                 24 * time.Hour,
		WaitlistRequestTTL:             48 * time.Hour,
		MigrationDir:                   envOr("LINKUP_MIGRATIONS_DIR", "../db/migrations"),
		ArgonMemoryKiB:                 19 * 1024,
		ArgonIterations:                2,
		ArgonParallel:                  1,
		AuthRateLimit:                  10,
		AuthRateWindow:                 time.Minute,
		AuthRateIdleTTL:                10 * time.Minute,
		AuthRateMaxEntries:             20_000,
		SocialRateLimit:                120,
		SocialRateWindow:               time.Minute,
		SocialRateIdleTTL:              10 * time.Minute,
		SocialRateMaxEntries:           20_000,
		MonetizationRateLimit:          20,
		MonetizationRateWindow:         time.Hour,
		MonetizationRateIdleTTL:        2 * time.Hour,
		MonetizationRateMaxEntries:     20_000,
		RecoverySMTPAddress:            strings.TrimSpace(os.Getenv("LINKUP_RECOVERY_SMTP_ADDR")),
		RecoverySMTPHost:               strings.TrimSpace(os.Getenv("LINKUP_RECOVERY_SMTP_HOST")),
		RecoverySMTPUsername:           strings.TrimSpace(os.Getenv("LINKUP_RECOVERY_SMTP_USERNAME")),
		RecoverySMTPPassword:           os.Getenv("LINKUP_RECOVERY_SMTP_PASSWORD"),
		RecoveryFrom:                   strings.TrimSpace(os.Getenv("LINKUP_RECOVERY_FROM")),
		RecoveryResetURL:               strings.TrimSpace(os.Getenv("LINKUP_RECOVERY_RESET_URL")),
		TelegramBotToken:               strings.TrimSpace(os.Getenv("LINKUP_TELEGRAM_BOT_TOKEN")),
		TelegramBotUsername:            strings.TrimPrefix(strings.TrimSpace(os.Getenv("LINKUP_TELEGRAM_BOT_USERNAME")), "@"),
		TelegramWebhookSecret:          strings.TrimSpace(os.Getenv("LINKUP_TELEGRAM_WEBHOOK_SECRET")),
		NotificationTTL:                14 * 24 * time.Hour,
		NotificationFrequencyCapMax:    5,
		NotificationFrequencyCapWindow: 24 * time.Hour,
		// README §6.8 grouping/collapse: a burst of same-(user,Slot,Type)
		// notifications collapses to one within this window. 5 minutes is
		// long enough to absorb a realistic rapid-fire chat burst without
		// meaningfully delaying a genuinely new conversation's first
		// notification, and short enough that a real EVENT state change an
		// hour later is never mistaken for the same burst.
		NotificationGroupWindow:   5 * time.Minute,
		NotificationPollInterval:  5 * time.Second,
		NotificationBatchSize:     200,
		BumpChallengeTTL:          10 * time.Minute,
		EventReminderLeadTime:     30 * time.Minute,
		EventReminderPollInterval: time.Minute,
		EventReminderBatchSize:    100,
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
	if cfg.WaitlistRequestTTL, err = durationEnv("LINKUP_WAITLIST_REQUEST_TTL", cfg.WaitlistRequestTTL); err != nil {
		return Config{}, err
	}
	if cfg.NotificationTTL, err = durationEnv("LINKUP_NOTIFICATION_TTL", cfg.NotificationTTL); err != nil {
		return Config{}, err
	}
	if cfg.NotificationFrequencyCapMax, err = positiveIntEnv("LINKUP_NOTIFICATION_FREQUENCY_CAP_MAX", cfg.NotificationFrequencyCapMax); err != nil {
		return Config{}, err
	}
	if cfg.NotificationFrequencyCapWindow, err = durationEnv("LINKUP_NOTIFICATION_FREQUENCY_CAP_WINDOW", cfg.NotificationFrequencyCapWindow); err != nil {
		return Config{}, err
	}
	if cfg.NotificationGroupWindow, err = durationEnv("LINKUP_NOTIFICATION_GROUP_WINDOW", cfg.NotificationGroupWindow); err != nil {
		return Config{}, err
	}
	if cfg.NotificationPollInterval, err = durationEnv("LINKUP_NOTIFICATION_POLL_INTERVAL", cfg.NotificationPollInterval); err != nil {
		return Config{}, err
	}
	if cfg.NotificationBatchSize, err = positiveIntEnv("LINKUP_NOTIFICATION_BATCH_SIZE", cfg.NotificationBatchSize); err != nil {
		return Config{}, err
	}
	if cfg.BumpChallengeTTL, err = durationEnv("LINKUP_BUMP_CHALLENGE_TTL", cfg.BumpChallengeTTL); err != nil {
		return Config{}, err
	}
	if cfg.EventReminderLeadTime, err = durationEnv("LINKUP_EVENT_REMINDER_LEAD_TIME", cfg.EventReminderLeadTime); err != nil {
		return Config{}, err
	}
	if cfg.EventReminderPollInterval, err = durationEnv("LINKUP_EVENT_REMINDER_POLL_INTERVAL", cfg.EventReminderPollInterval); err != nil {
		return Config{}, err
	}
	if cfg.EventReminderBatchSize, err = positiveIntEnv("LINKUP_EVENT_REMINDER_BATCH_SIZE", cfg.EventReminderBatchSize); err != nil {
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
		MemoryKiB:  cfg.ArgonMemoryKiB,
		Iterations: cfg.ArgonIterations,
		Parallel:   cfg.ArgonParallel,
		SaltBytes:  16,
		KeyBytes:   32,
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
	if cfg.MonetizationRateLimit, err = positiveIntEnv("LINKUP_MONETIZATION_RATE_LIMIT", cfg.MonetizationRateLimit); err != nil {
		return Config{}, err
	}
	if cfg.MonetizationRateWindow, err = durationEnv("LINKUP_MONETIZATION_RATE_WINDOW", cfg.MonetizationRateWindow); err != nil {
		return Config{}, err
	}
	if cfg.MonetizationRateIdleTTL, err = durationEnv("LINKUP_MONETIZATION_RATE_IDLE_TTL", cfg.MonetizationRateIdleTTL); err != nil {
		return Config{}, err
	}
	if cfg.MonetizationRateMaxEntries, err = positiveIntEnv("LINKUP_MONETIZATION_RATE_MAX_ENTRIES", cfg.MonetizationRateMaxEntries); err != nil {
		return Config{}, err
	}
	if cfg.MonetizationRateIdleTTL < cfg.MonetizationRateWindow {
		return Config{}, errors.New("LINKUP_MONETIZATION_RATE_IDLE_TTL must be >= LINKUP_MONETIZATION_RATE_WINDOW")
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
	if cfg.TelegramBotToken != "" {
		configured++
	}
	if cfg.TelegramBotUsername != "" {
		configured++
	}
	if cfg.TelegramWebhookSecret != "" {
		configured++
	}
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
