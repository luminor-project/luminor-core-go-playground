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

	// Platform
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/auth"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/config"
	appCSRF "github.com/luminor-project/luminor-core-go-playground/internal/platform/csrf"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/database"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/flash"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/httplog"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/i18n"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/ollama"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/outbox"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/session"
)

func main() {
	// Setup structured logging
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Connect to databases
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	ragDB, err := database.Connect(cfg.RAGDatabaseURL)
	if err != nil {
		slog.Error("failed to connect to RAG database", "error", err)
		os.Exit(1)
	}
	defer ragDB.Close()

	// Create event bus
	bus := eventbus.New()
	outboxStore := outbox.NewPostgresStore(db)

	// Create session store
	sessionStore := session.NewStore(cfg.SessionKey)

	translator, err := i18n.LoadEmbeddedTranslator()
	if err != nil {
		slog.Error("failed to load i18n catalogs", "error", err)
		os.Exit(1)
	}

	// ── Build Account Vertical ──────────────────────────────────────────
	accountRepo := accountinfra.NewPostgresRepository(db)
	accountService := accountdomain.NewAccountService(accountRepo)
	acctFacade := accountfacade.New(accountService, bus, outboxStore)

	// ── Build Organization Vertical ─────────────────────────────────────
	orgRepo := orginfra.NewPostgresRepository(db)
	orgService := orgdomain.NewOrgService(orgRepo)
	oFacade := orgfacade.New(orgService, bus)

	// ── Build RAG Vertical ─────────────────────────────────────────────
	ollamaClient := ollama.NewClient(cfg.LocalInferenceURL)
	ollamaAdapter := &ollamaChatAdapter{client: ollamaClient}
	ragRepo := raginfra.NewPostgresRepository(ragDB)
	ragService := ragdomain.NewRAGService(ragRepo, ollamaClient, ollamaAdapter, cfg.EmbedModel, cfg.ChatModel)
	rFacade := ragfacade.New(ragService, bus)

	// ── Wire Event Subscribers ──────────────────────────────────────────
	// Account created → create default organization
	orgsub.RegisterAccountCreatedSubscriber(bus, oFacade)
	// Active org changed → update account's active org
	accountsub.RegisterOrgChangedSubscriber(bus, acctFacade)

	// ── Build HTTP Routes ───────────────────────────────────────────────
	mux := http.NewServeMux()

	// Static file serving
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Register vertical routes
	contentweb.RegisterRoutes(mux, !cfg.IsProduction())
	accountweb.RegisterRoutes(mux, acctFacade, sessionStore)
	orgweb.RegisterRoutes(mux, orgService, oFacade, acctFacade, sessionStore)
	ragweb.RegisterRoutes(mux, rFacade)

	// ── Compose Middleware Stack ────────────────────────────────────────
	var handler http.Handler = mux

	// CSRF protection
	csrfMiddleware := appCSRF.Middleware(cfg.CSRFKey, cfg.IsProduction(), cfg.BaseURL)
	handler = csrfMiddleware(handler)

	// Flash messages
	handler = flash.Middleware(sessionStore)(handler)

	// Load authenticated user into context
	handler = auth.LoadUser(sessionStore)(handler)

	// Locale-aware routing and translation context.
	handler = i18n.Middleware(translator)(handler)

	// Request logging (outermost)
	handler = httplog.Middleware(handler)

	// ── Health Check Endpoints ──────────────────────────────────────────
	// Registered outside middleware to avoid CSRF/session/i18n overhead on probes.
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

	// ── Start Server ────────────────────────────────────────────────────
	addr := ":" + cfg.Port
	slog.Info("server starting", "addr", addr, "env", cfg.AppEnv)
	listenAndServe(topMux, addr)
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
