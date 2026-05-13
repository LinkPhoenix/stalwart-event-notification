package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	telegram "github.com/go-telegram/bot"
	telegrammodels "github.com/go-telegram/bot/models"

	appbot "stalwart-event-notification/internal/bot"
	"stalwart-event-notification/internal/config"
	"stalwart-event-notification/internal/httpserver"
	"stalwart-event-notification/internal/i18n"
	"stalwart-event-notification/internal/logging"
	"stalwart-event-notification/internal/notifications"
	"stalwart-event-notification/internal/storage"
)

func main() {
	_ = config.LoadDotEnv(".env")

	bootstrapLogger := logging.New(os.Getenv("LOG_LEVEL"))
	slog.SetDefault(bootstrapLogger)

	if err := run(); err != nil {
		bootstrapLogger.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(cfg.LogLevel)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	translations, err := i18n.Load(cfg.LocalesDir, cfg.DefaultLocale)
	if err != nil {
		return err
	}

	store, err := storage.OpenPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	if err := store.EnsureSchema(ctx); err != nil {
		return err
	}
	if err := store.SyncIgnoredIPs(ctx, cfg.IgnoredIPsByEvent); err != nil {
		return err
	}
	if cfg.EventsRetentionDays > 0 {
		startPurgeJob(ctx, logger, store, cfg.EventsRetentionDays)
	}

	var handler *appbot.Handler
	telegramBot, err := telegram.New(
		cfg.TelegramBotToken,
		telegram.WithHTTPClient(30*time.Second, &http.Client{Timeout: 40 * time.Second}),
		telegram.WithAllowedUpdates(telegram.AllowedUpdates{"message", "callback_query"}),
		telegram.WithDefaultHandler(func(ctx context.Context, _ *telegram.Bot, update *telegrammodels.Update) {
			if handler == nil {
				return
			}
			if err := appbot.HandleUpdate(ctx, handler, update); err != nil {
				logger.Error("telegram update failed", "error", err)
			}
		}),
		telegram.WithErrorsHandler(func(err error) {
			logger.Error("telegram sdk error", "error", err)
		}),
	)
	if err != nil {
		return err
	}

	messenger := appbot.NewTelegramMessenger(telegramBot)
	if err := messenger.SetCommands(ctx, translations, cfg.DefaultLocale); err != nil {
		logger.Warn("set telegram commands failed", "error", err)
	}

	metrics := notifications.NewMetrics()
	handler = appbot.NewHandler(store, messenger, translations, cfg)
	notifier := notifications.NewService(store, messenger, translations, cfg, metrics, logger)

	server := httpserver.New(cfg, store, notifier, metrics, logger, func(ctx context.Context) bool {
		_, err := telegramBot.GetMe(ctx)
		return err == nil
	})

	httpErr := make(chan error, 1)
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Port)
		logger.Info("http server started", "addr", addr)
		httpErr <- httpserver.ListenAndServe(ctx, addr, server.Handler(), logger)
	}()

	logger.Info("stalwart event notification started", "port", cfg.Port)
	botDone := make(chan struct{})
	go func() {
		telegramBot.Start(ctx)
		close(botDone)
	}()

	select {
	case <-ctx.Done():
		if err := <-httpErr; err != nil {
			return err
		}
		<-botDone
		return nil
	case err := <-httpErr:
		stop()
		<-botDone
		return err
	}
}

func startPurgeJob(ctx context.Context, logger *slog.Logger, store storage.Store, retentionDays int) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			deleted, err := store.PurgeOldEvents(ctx, retentionDays)
			if err != nil {
				logger.Error("events purge failed", "error", err)
			} else {
				logger.Info("events purge completed", "deleted", deleted)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
