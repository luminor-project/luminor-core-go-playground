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

	"github.com/jackc/pgx/v5/pgxpool"

	// Account vertical
	accountdomain "github.com/luminor-project/luminor-core-go-playground/internal/account/domain"
	accountfacade "github.com/luminor-project/luminor-core-go-playground/internal/account/facade"
	accountinfra "github.com/luminor-project/luminor-core-go-playground/internal/account/infra"
	accountsub "github.com/luminor-project/luminor-core-go-playground/internal/account/subscriber"
	accountweb "github.com/luminor-project/luminor-core-go-playground/internal/account/web"

	// Organization vertical
	orgdomain "github.com/luminor-project/luminor-core-go-playground/internal/organization/domain"
	orgfacade "github.com/luminor-project/luminor-core-go-playground/internal/organization/facade"
	orginfra "github.com/luminor-project/luminor-core-go-playground/internal/organization/infra"
	orgsub "github.com/luminor-project/luminor-core-go-playground/internal/organization/subscriber"
	orgweb "github.com/luminor-project/luminor-core-go-playground/internal/organization/web"

	// Content vertical
	contentweb "github.com/luminor-project/luminor-core-go-playground/internal/content/web"

	// RAG vertical
	ragdomain "github.com/luminor-project/luminor-core-go-playground/internal/rag/domain"
	ragfacade "github.com/luminor-project/luminor-core-go-playground/internal/rag/facade"
	raginfra "github.com/luminor-project/luminor-core-go-playground/internal/rag/infra"
	ragweb "github.com/luminor-project/luminor-core-go-playground/internal/rag/web"

	// Party vertical
	partyfacade "github.com/luminor-project/luminor-core-go-playground/internal/party/facade"
	partyinfra "github.com/luminor-project/luminor-core-go-playground/internal/party/infra"
	partysub "github.com/luminor-project/luminor-core-go-playground/internal/party/subscriber"

	// Subject vertical
	subjectfacade "github.com/luminor-project/luminor-core-go-playground/internal/subject/facade"
	subjectinfra "github.com/luminor-project/luminor-core-go-playground/internal/subject/infra"
	subjectsub "github.com/luminor-project/luminor-core-go-playground/internal/subject/subscriber"

	// Rental vertical
	rentalfacade "github.com/luminor-project/luminor-core-go-playground/internal/rental/facade"
	rentalinfra "github.com/luminor-project/luminor-core-go-playground/internal/rental/infra"
	rentalsub "github.com/luminor-project/luminor-core-go-playground/internal/rental/subscriber"

	// WorkItem vertical
	workitemfacade "github.com/luminor-project/luminor-core-go-playground/internal/workitem/facade"

	// App Casehandling vertical
	casehandlingfacade "github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/facade"
	caseinfra "github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/infra"
	casesub "github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/subscriber"
	caseweb "github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/web"

	// App Property Management vertical
	pmfacade "github.com/luminor-project/luminor-core-go-playground/internal/app_propertymanagement/facade"
	pmweb "github.com/luminor-project/luminor-core-go-playground/internal/app_propertymanagement/web"

	// App Inquiry vertical
	inquiryfacade "github.com/luminor-project/luminor-core-go-playground/internal/app_inquiry/facade"
	inquiryweb "github.com/luminor-project/luminor-core-go-playground/internal/app_inquiry/web"

	// Platform
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/agentworkload"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/auth"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/config"
	appCSRF "github.com/luminor-project/luminor-core-go-playground/internal/platform/csrf"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/database"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventstore"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/flash"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/httplog"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/i18n"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/ollama"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/outbox"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/session"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := mustLoadConfig()
	db := mustConnect(cfg.DatabaseURL)
	defer db.Close()
	ragDB := mustConnect(cfg.RAGDatabaseURL)
	defer ragDB.Close()

	bus := eventbus.New()
	sessionStore := session.NewStore(cfg.SessionKey)
	translator := mustLoadTranslator()
	clk := clock.New()

	// ── Build Verticals ────────────────────────────────────────────────
	acctFacade := accountfacade.New(accountdomain.NewAccountService(accountinfra.NewPostgresRepository(db), clk), bus, outbox.NewPostgresStore(db))
	orgService := orgdomain.NewOrgService(orginfra.NewPostgresRepository(db), clk)
	oFacade := orgfacade.New(orgService, bus)

	ollamaClient := ollama.NewClient(cfg.LocalInferenceURL)
	ragService := ragdomain.NewRAGService(raginfra.NewPostgresRepository(ragDB), ollamaClient, &ollamaChatAdapter{client: ollamaClient}, cfg.EmbedModel, cfg.ChatModel, clk)
	rFacade := ragfacade.New(ragService, bus)

	partyRepo := partyinfra.NewPostgresRepository(db)
	partyFac := partyfacade.New(eventstore.NewPostgresStore(db), bus, clk, partyRepo)
	subjectRepo := subjectinfra.NewPostgresRepository(db)
	subjectFac := subjectfacade.New(eventstore.NewPostgresStore(db), bus, clk, subjectRepo)
	rentalRepo := rentalinfra.NewPostgresRepository(db)
	rentalFac := rentalfacade.New(eventstore.NewPostgresStore(db), bus, clk, rentalRepo, rentalRepo)

	// ── Wire Event Subscribers ─────────────────────────────────────────
	orgsub.RegisterAccountCreatedSubscriber(bus, oFacade)
	accountsub.RegisterOrgChangedSubscriber(bus, acctFacade)
	partysub.RegisterProjectionSubscribers(bus, partyRepo)
	partysub.RegisterAccountJoinedOrgSubscriber(bus, partyFac, acctFacade)
	subjectsub.RegisterProjectionSubscribers(bus, subjectRepo)
	rentalsub.RegisterProjectionSubscribers(bus, rentalRepo)

	// ── Build HTTP Routes ──────────────────────────────────────────────
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	contentweb.RegisterRoutes(mux, !cfg.IsProduction())
	accountweb.RegisterRoutes(mux, acctFacade, sessionStore, &sessionEnricherAdapter{partyFac: partyFac, orgFacade: oFacade})
	orgweb.RegisterRoutes(mux, orgService, oFacade, acctFacade, sessionStore)
	ragweb.RegisterRoutes(mux, rFacade)

	// Casehandling: event store, workitem, projections, routes, facade
	wiFacade := workitemfacade.New(eventstore.NewPostgresStore(db), bus, clk)
	dashboardStore := caseinfra.NewDashboardStore(db)
	casesub.RegisterProjectionSubscribers(bus, dashboardStore, partyFac, subjectFac)
	caseweb.RegisterRoutes(mux, dashboardStore, wiFacade, dashboardStore)
	caseFac := casehandlingfacade.New(wiFacade, agentworkload.NewFakeAdapter(), subjectFac)

	// Property management + inquiry
	pmweb.RegisterRoutes(mux, pmfacade.New(partyFac, subjectFac, rentalFac, acctFacade, oFacade), partyFac, subjectFac, rentalFac, acctFacade)
	inquiryweb.RegisterRoutes(mux, inquiryfacade.New(rentalFac, caseFac, partyFac), rentalFac, acctFacade)

	// ── Compose Middleware Stack ────────────────────────────────────────
	var handler http.Handler = mux
	handler = appCSRF.Middleware(cfg.CSRFKey, cfg.IsProduction(), cfg.BaseURL)(handler)
	handler = flash.Middleware(sessionStore)(handler)
	handler = auth.LoadUser(sessionStore)(handler)
	handler = i18n.Middleware(translator)(handler)
	handler = httplog.Middleware(handler)

	addr := ":" + cfg.Port
	slog.Info("server starting", "addr", addr, "env", cfg.AppEnv)
	listenAndServe(withHealthChecks(handler, db), addr)
}

func mustLoadConfig() config.Config {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	return cfg
}

func mustConnect(url string) *pgxpool.Pool {
	db, err := database.Connect(url)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	return db
}

func mustLoadTranslator() *i18n.Translator {
	t, err := i18n.LoadEmbeddedTranslator()
	if err != nil {
		slog.Error("failed to load i18n catalogs", "error", err)
		os.Exit(1)
	}
	return t
}

func withHealthChecks(handler http.Handler, db *pgxpool.Pool) *http.ServeMux {
	topMux := http.NewServeMux()
	topMux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	topMux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
	topMux.Handle("/", handler)
	return topMux
}

func listenAndServe(handler http.Handler, addr string) {
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	slog.Info("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped gracefully")
}

// sessionEnricherAdapter bridges party and org facades to the account web handler's sessionEnricher interface.
type sessionEnricherAdapter struct {
	partyFac interface {
		GetPartyInfo(ctx context.Context, partyID string) (partyfacade.PartyInfoDTO, error)
	}
	orgFacade interface {
		GetOrganizationNameByID(ctx context.Context, orgID string) (string, error)
	}
}

func (a *sessionEnricherAdapter) GetPartyNameAndKind(ctx context.Context, partyID string) (string, string, error) {
	info, err := a.partyFac.GetPartyInfo(ctx, partyID)
	if err != nil {
		return "", "", err
	}
	return info.Name, string(info.PartyKind), nil
}

func (a *sessionEnricherAdapter) GetOrgName(ctx context.Context, orgID string) (string, error) {
	return a.orgFacade.GetOrganizationNameByID(ctx, orgID)
}

// ollamaChatAdapter bridges ollama.Client to ragdomain.Generator by converting message types.
type ollamaChatAdapter struct {
	client *ollama.Client
}

func (a *ollamaChatAdapter) Chat(ctx context.Context, model string, messages []ragdomain.Message) (string, error) {
	ollamaMessages := make([]ollama.Message, len(messages))
	for i, m := range messages {
		ollamaMessages[i] = ollama.Message{Role: m.Role, Content: m.Content}
	}
	return a.client.Chat(ctx, model, ollamaMessages)
}
