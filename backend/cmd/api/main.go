package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/blocklist"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/bump"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/capability"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/chat"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citycontext"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citymap"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/config"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/httpserver"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/monetization"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/notification"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/onboarding"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/places"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/postgres"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/push"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/ratelimit"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/realtime"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/recovery"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

func main() {
	cfg, err := config.Load()
	if err != nil { slog.Error("invalid configuration", "error", err); os.Exit(1) }

	pushCfg, err := push.LoadRuntimeConfig()
	if err != nil { slog.Error("invalid push configuration", "error", err); os.Exit(1) }

	startupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := postgres.Open(startupCtx, cfg.DatabaseURL)
	if err != nil { slog.Error("database unavailable", "error", err); os.Exit(1) }
	defer pool.Close()

	passwordParams := password.Params{MemoryKiB: cfg.ArgonMemoryKiB, Iterations: cfg.ArgonIterations, Parallel: cfg.ArgonParallel, SaltBytes: 16, KeyBytes: 32}
	accountService, err := account.NewService(postgres.NewAccountStore(pool), passwordParams, cfg.SessionTTL)
	if err != nil { slog.Error("account service init failed", "error", err); os.Exit(1) }
	if cfg.RecoverySMTPAddress != "" {
		notifier, err := recovery.NewSMTP(recovery.SMTPConfig{
			Address: cfg.RecoverySMTPAddress, Host: cfg.RecoverySMTPHost, Username: cfg.RecoverySMTPUsername,
			Password: cfg.RecoverySMTPPassword, From: cfg.RecoveryFrom, ResetURL: cfg.RecoveryResetURL,
			ImplicitTLS: cfg.RecoveryImplicitTLS, Timeout: 10 * time.Second,
		})
		if err != nil { slog.Error("password recovery init failed", "error", err); os.Exit(1) }
		if err := accountService.ConfigureRecovery(notifier, cfg.PasswordResetTTL); err != nil { slog.Error("password recovery service init failed", "error", err); os.Exit(1) }
		slog.Info("password recovery enabled", "smtp_host", cfg.RecoverySMTPHost)
	}

	var onboardingService *onboarding.Service
	var telegramBot onboarding.ContactPrompter
	if cfg.TelegramBotToken != "" {
		onboardingService, err = onboarding.NewService(postgres.NewOnboardingStore(pool), passwordParams, cfg.RegistrationTTL, cfg.SessionTTL, cfg.TelegramBotUsername)
		if err != nil { slog.Error("onboarding service init failed", "error", err); os.Exit(1) }
		telegramBot, err = onboarding.NewTelegramBot(cfg.TelegramBotToken)
		if err != nil { slog.Error("telegram onboarding init failed", "error", err); os.Exit(1) }
		slog.Info("telegram-only v1.0 registration enabled", "bot_username", cfg.TelegramBotUsername)
	}

	blockService, err := blocklist.NewService(postgres.NewBlockStore(pool, cfg.WaitlistRequestTTL))
	if err != nil { slog.Error("block service init failed", "error", err); os.Exit(1) }

	baseSlotStore, err := postgres.NewSlotStore(pool, cfg.IdempotencyTTL)
	if err != nil { slog.Error("slot store init failed", "error", err); os.Exit(1) }
	slotStore, err := postgres.NewV11SlotStore(baseSlotStore, cfg.WaitlistRequestTTL)
	if err != nil { slog.Error("v1.1 slot store init failed", "error", err); os.Exit(1) }
	slotService, err := slot.NewService(slotStore)
	if err != nil { slog.Error("slot service init failed", "error", err); os.Exit(1) }

	chatService, err := chat.NewService(postgres.NewChatStore(pool))
	if err != nil { slog.Error("chat service init failed", "error", err); os.Exit(1) }
	cityContextStore, err := postgres.NewCityContextStore(pool)
	if err != nil { slog.Error("city context store init failed", "error", err); os.Exit(1) }
	cityContextService, err := citycontext.NewService(cityContextStore, citycontext.DefaultPolicy())
	if err != nil { slog.Error("city context service init failed", "error", err); os.Exit(1) }
	mapStore, err := postgres.NewCityMapStore(pool, cfg.WaitlistRequestTTL)
	if err != nil { slog.Error("map store init failed", "error", err); os.Exit(1) }
	mapService, err := citymap.NewService(mapStore)
	if err != nil { slog.Error("map service init failed", "error", err); os.Exit(1) }
	placeService, err := places.NewService(postgres.NewPlaceStore(pool))
	if err != nil { slog.Error("place service init failed", "error", err); os.Exit(1) }
	realtimeStore, err := postgres.NewRealtimeViewerStore(pool, cfg.WaitlistRequestTTL)
	if err != nil { slog.Error("realtime store init failed", "error", err); os.Exit(1) }
	realtimeService, err := realtime.NewFeedService(realtimeStore)
	if err != nil { slog.Error("realtime service init failed", "error", err); os.Exit(1) }
	cityRealtimeService, err := realtime.NewCityFeedService(realtimeStore)
	if err != nil { slog.Error("city realtime service init failed", "error", err); os.Exit(1) }
	capabilityStore, err := postgres.NewCapabilityStore(pool)
	if err != nil { slog.Error("capability store init failed", "error", err); os.Exit(1) }
	capabilityService, err := capability.NewService(capabilityStore)
	if err != nil { slog.Error("capability service init failed", "error", err); os.Exit(1) }

	monetizationService := monetization.NewService(postgres.NewMonetizationStore(pool))

	var pushService *push.Service
	if pushCfg.RegistrationEnabled() {
		var sender push.Sender
		if pushCfg.DeliveryEnabled() {
			firebaseSender, err := push.NewFirebaseSender(pushCfg.FirebaseProjectID, pushCfg.FirebaseCredentialsFile)
			if err != nil { slog.Error("firebase sender init failed", "error", err); os.Exit(1) }
			sender = firebaseSender
		}
		pushService, err = push.NewService(postgres.NewPushStore(pool), pushCfg.TokenKeyID, pushCfg.TokenKeyBase64, sender)
		if err != nil { slog.Error("push service init failed", "error", err); os.Exit(1) }
		slog.Info("android push device registration enabled", "delivery_enabled", pushCfg.DeliveryEnabled())
	}

	// pushService is a *push.Service that may be a nil pointer above; passing
	// a nil pointer of a concrete type into an interface parameter produces a
	// non-nil interface value wrapping that nil pointer (the classic Go
	// typed-nil trap), which would defeat NotificationProjector's own
	// `pusher == nil` check and panic on first call. Branch explicitly so a
	// genuinely nil interface is passed when push isn't configured.
	var notificationProjector *postgres.NotificationProjector
	if pushService != nil {
		notificationProjector, err = postgres.NewNotificationProjector(pool, capabilityService, pushService, cfg.NotificationTTL, cfg.NotificationFrequencyCapWindow, cfg.NotificationFrequencyCapMax)
	} else {
		notificationProjector, err = postgres.NewNotificationProjector(pool, capabilityService, nil, cfg.NotificationTTL, cfg.NotificationFrequencyCapWindow, cfg.NotificationFrequencyCapMax)
	}
	if err != nil { slog.Error("notification projector init failed", "error", err); os.Exit(1) }

	notificationPreferencesStore, err := postgres.NewNotificationPreferencesStore(pool)
	if err != nil { slog.Error("notification preferences store init failed", "error", err); os.Exit(1) }
	notificationPreferencesService, err := notification.NewPreferencesService(notificationPreferencesStore)
	if err != nil { slog.Error("notification preferences service init failed", "error", err); os.Exit(1) }

	bumpStore, err := postgres.NewBumpStore(pool)
	if err != nil { slog.Error("bump store init failed", "error", err); os.Exit(1) }
	bumpService, err := bump.NewService(bumpStore, cfg.BumpChallengeTTL)
	if err != nil { slog.Error("bump service init failed", "error", err); os.Exit(1) }

	reminderScanner, err := postgres.NewReminderScanner(pool, cfg.EventReminderLeadTime)
	if err != nil { slog.Error("reminder scanner init failed", "error", err); os.Exit(1) }

	authLimiter, err := ratelimit.New(ratelimit.Config{Limit: cfg.AuthRateLimit, Window: cfg.AuthRateWindow, IdleTTL: cfg.AuthRateIdleTTL, MaxEntries: cfg.AuthRateMaxEntries})
	if err != nil { slog.Error("auth rate limiter init failed", "error", err); os.Exit(1) }
	userLimiter, err := ratelimit.New(ratelimit.Config{Limit: cfg.SocialRateLimit, Window: cfg.SocialRateWindow, IdleTTL: cfg.SocialRateIdleTTL, MaxEntries: cfg.SocialRateMaxEntries})
	if err != nil { slog.Error("authenticated user rate limiter init failed", "error", err); os.Exit(1) }

	app := httpserver.New(httpserver.Dependencies{
		Accounts: accountService,
		Blocks: blockService,
		Slots: slotService,
		Chats: chatService,
		CityContext: cityContextService,
		Map: mapService,
		Places: placeService,
		Realtime: realtimeService,
		CityRealtime: cityRealtimeService,
		Capabilities: capabilityService,
		Monetization: monetizationService,
		Push: pushService,
		NotificationPreferences: notificationPreferencesService,
		Bump: bumpService,
		Onboarding: onboardingService,
		Telegram: telegramBot,
		TelegramWebhookSecret: cfg.TelegramWebhookSecret,
		Ready: pool.Ping,
		AuthLimiter: authLimiter,
		UserLimiter: userLimiter,
	})
	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: app.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout: 60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 1)
	go func() {
		slog.Info("linkup api listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) { errCh <- err }
	}()

	// README §6.8's notification projector/worker runs in-process on a
	// simple ticker rather than as a separate binary: it is stateless
	// between ticks (all state lives in Postgres via the durable connector
	// cursor), so a restart of this process just resumes from the last
	// checkpoint, and a separate deployable is not required for that
	// property. It stops with the same shutdown signal as the HTTP server.
	go func() {
		ticker := time.NewTicker(cfg.NotificationPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := notificationProjector.ProcessBatch(ctx, cfg.NotificationBatchSize); err != nil && ctx.Err() == nil {
					slog.Warn("notification projector batch failed", "error", err)
				}
			}
		}
	}()

	// EVENT_REMINDER's own ticker (internal/postgres/reminder_scanner.go):
	// separate from the notification projector's above because it scans
	// slots.start_at against wall-clock time rather than consuming the
	// outbox, but the events it emits then flow through the exact same
	// projector/connector-cursor pipeline once emitted.
	go func() {
		ticker := time.NewTicker(cfg.EventReminderPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := reminderScanner.ScanAndEmit(ctx, cfg.EventReminderBatchSize); err != nil && ctx.Err() == nil {
					slog.Warn("event reminder scan failed", "error", err)
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		slog.Error("http server failed", "error", err); os.Exit(1)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil { slog.Error("graceful shutdown failed", "error", err); os.Exit(1) }
}
