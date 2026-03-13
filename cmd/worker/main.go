package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	accountdomain "github.com/luminor-project/luminor-core-go-playground/internal/account/domain"
	accountfacade "github.com/luminor-project/luminor-core-go-playground/internal/account/facade"
	accountinfra "github.com/luminor-project/luminor-core-go-playground/internal/account/infra"
	accountsub "github.com/luminor-project/luminor-core-go-playground/internal/account/subscriber"
	orgdomain "github.com/luminor-project/luminor-core-go-playground/internal/organization/domain"
	orgfacade "github.com/luminor-project/luminor-core-go-playground/internal/organization/facade"
	orginfra "github.com/luminor-project/luminor-core-go-playground/internal/organization/infra"
	orgsub "github.com/luminor-project/luminor-core-go-playground/internal/organization/subscriber"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/config"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/database"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/outbox"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	bus := eventbus.New()
	outboxStore := outbox.NewPostgresStore(db)
	clk := clock.New()

	// Build verticals for event dispatch.
	accountRepo := accountinfra.NewPostgresRepository(db)
	accountService := accountdomain.NewAccountService(accountRepo, clk)
	acctFacade := accountfacade.New(accountService, bus, outboxStore, cfg.BaseURL)

	orgRepo := orginfra.NewPostgresRepository(db)
	orgService := orgdomain.NewOrgService(orgRepo, clk)
	oFacade := orgfacade.New(orgService, bus)

	orgsub.RegisterAccountCreatedSubscriber(bus, oFacade)
	accountsub.RegisterOrgChangedSubscriber(bus, acctFacade)

	slog.Info("worker starting", "mode", "outbox_dispatch")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			processPending(ctx, outboxStore, bus)
		case <-quit:
			slog.Info("worker shutting down")
			return
		}
	}
}

func processPending(ctx context.Context, store outbox.Store, bus *eventbus.Bus) {
	events, err := store.GetPending(ctx, 100)
	if err != nil {
		slog.Error("failed to fetch pending outbox events", "error", err)
		return
	}
	for _, ev := range events {
		if err := dispatchOutboxEvent(ctx, bus, ev); err != nil {
			slog.Error("outbox event dispatch failed", "error", err, "event_id", ev.ID, "event_type", ev.EventType)
			if markErr := store.MarkFailed(ctx, ev.ID, err, 15*time.Second); markErr != nil {
				slog.Error("failed to mark outbox event failed", "error", markErr, "event_id", ev.ID)
			}
			continue
		}
		if err := store.MarkProcessed(ctx, ev.ID); err != nil {
			slog.Error("failed to mark outbox event processed", "error", err, "event_id", ev.ID)
		}
	}
}

func dispatchOutboxEvent(ctx context.Context, bus *eventbus.Bus, ev outbox.Event) error {
	switch ev.EventType {
	case outbox.EventTypeAccountCreatedV1:
		var payload accountfacade.AccountCreatedEvent
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			return err
		}
		return eventbus.Publish(ctx, bus, payload)
	default:
		return nil
	}
}
