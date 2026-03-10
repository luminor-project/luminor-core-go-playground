package main

import (
	"context"
	"log/slog"
	"os"

	casefacade "github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/facade"
	caseinfra "github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/infra"
	casesub "github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/subscriber"
	"github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/testharness"
	partyfacade "github.com/luminor-project/luminor-core-go-playground/internal/party/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/agentworkload"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/config"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/database"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventstore"
	subjectfacade "github.com/luminor-project/luminor-core-go-playground/internal/subject/facade"
	workitemfacade "github.com/luminor-project/luminor-core-go-playground/internal/workitem/facade"
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

	ctx := context.Background()

	// Wire dependencies
	bus := eventbus.New()
	evStore := eventstore.NewPostgresStore(db)

	var agentPort agentworkload.Port
	switch cfg.AgentWorkloadMode {
	case "fake":
		agentPort = agentworkload.NewFakeAdapter()
	case "live":
		agentPort = agentworkload.NewLiveAdapter()
	default:
		slog.Error("invalid AGENT_WORKLOAD_MODE", "mode", cfg.AgentWorkloadMode)
		os.Exit(1)
	}

	partyFac := partyfacade.NewDemoPartyFacade()
	subjectFac := subjectfacade.NewDemoSubjectFacade()
	wiFacade := workitemfacade.New(evStore, bus)
	dashboardStore := caseinfra.NewDashboardStore(db)
	cFacade := casefacade.New(wiFacade, agentPort, subjectFac)

	// Wire projection subscribers
	casesub.RegisterProjectionSubscribers(bus, dashboardStore, partyFac, subjectFac)

	// Seed golden path (resolved case)
	if err := testharness.SeedGoldenPath(ctx, cFacade); err != nil {
		slog.Error("golden path seed failed", "error", err)
		os.Exit(1)
	}

	// Seed pending case (for confirm UI demo)
	if _, err := testharness.SeedPendingCase(ctx, cFacade); err != nil {
		slog.Error("pending case seed failed", "error", err)
		os.Exit(1)
	}

	slog.Info("seed completed successfully")
}
