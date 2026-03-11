package main

import (
	"context"
	"log/slog"
	"os"

	casefacade "github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/facade"
	caseinfra "github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/infra"
	casesub "github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/subscriber"
	partydomain "github.com/luminor-project/luminor-core-go-playground/internal/party/domain"
	partyfacade "github.com/luminor-project/luminor-core-go-playground/internal/party/facade"
	partyinfra "github.com/luminor-project/luminor-core-go-playground/internal/party/infra"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/agentworkload"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/config"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/database"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventstore"
	rentaldomain "github.com/luminor-project/luminor-core-go-playground/internal/rental/domain"
	rentalfacade "github.com/luminor-project/luminor-core-go-playground/internal/rental/facade"
	rentalinfra "github.com/luminor-project/luminor-core-go-playground/internal/rental/infra"
	subjectdomain "github.com/luminor-project/luminor-core-go-playground/internal/subject/domain"
	subjectfacade "github.com/luminor-project/luminor-core-go-playground/internal/subject/facade"
	subjectinfra "github.com/luminor-project/luminor-core-go-playground/internal/subject/infra"
	workitemfacade "github.com/luminor-project/luminor-core-go-playground/internal/workitem/facade"
)

// demoIDs holds IDs of entities created during seeding.
type demoIDs struct {
	tenantPartyID string
	pmPartyID     string
	agentPartyID  string
	subjectID     string
}

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
	bus := eventbus.New()
	clk := clock.New()

	agentPort := mustBuildAgentPort(cfg.AgentWorkloadMode)

	// Build facades
	partyFac := partyfacade.New(partydomain.NewPartyService(partyinfra.NewPostgresRepository(db), clk))
	subjectFac := subjectfacade.New(subjectdomain.NewSubjectService(subjectinfra.NewPostgresRepository(db), clk))
	rentalFac := rentalfacade.New(rentaldomain.NewRentalService(rentalinfra.NewPostgresRepository(db), clk))
	wiFacade := workitemfacade.New(eventstore.NewPostgresStore(db), bus, clk)
	cFacade := casefacade.New(wiFacade, agentPort, subjectFac)

	casesub.RegisterProjectionSubscribers(bus, caseinfra.NewDashboardStore(db), partyFac, subjectFac)

	// Seed domain entities
	ids := seedDomainEntities(ctx, partyFac, subjectFac, rentalFac)

	// Seed golden path case (resolved)
	seedGoldenPath(ctx, cFacade, wiFacade, ids)

	// Seed pending case (for confirm UI demo)
	seedPendingCase(ctx, cFacade, ids)

	slog.Info("seed completed successfully")
}

func mustBuildAgentPort(mode string) agentworkload.Port {
	switch mode {
	case "fake":
		return agentworkload.NewFakeAdapter()
	case "live":
		return agentworkload.NewLiveAdapter()
	default:
		slog.Error("invalid AGENT_WORKLOAD_MODE", "mode", mode)
		os.Exit(1)
		return nil
	}
}

func seedDomainEntities(ctx context.Context, partyFac partyCreator, subjectFac subjectCreator, rentalFac rentalCreator) demoIDs {
	demoOrgID := "00000000-0000-0000-0000-000000000001"
	demoAccountID := "00000000-0000-0000-0000-000000000002"

	tenantPartyID := mustCreate("tenant party", func() (string, error) {
		return partyFac.CreateParty(ctx, partyfacade.CreatePartyDTO{
			Name: "Anna Schmidt", ActorKind: partyfacade.ActorKindHuman, PartyKind: partyfacade.PartyKindTenant,
			OwningOrgID: demoOrgID, CreatedByAccountID: demoAccountID,
		})
	})
	pmPartyID := mustCreate("PM party", func() (string, error) {
		return partyFac.CreateParty(ctx, partyfacade.CreatePartyDTO{
			Name: "Sarah", ActorKind: partyfacade.ActorKindHuman, PartyKind: partyfacade.PartyKindPropertyManager,
			OwningOrgID: demoOrgID, CreatedByAccountID: demoAccountID,
		})
	})
	agentPartyID := mustCreate("agent party", func() (string, error) {
		return partyFac.CreateParty(ctx, partyfacade.CreatePartyDTO{
			Name: "KI-Assistent", ActorKind: partyfacade.ActorKindAssistant, PartyKind: partyfacade.PartyKindAssistant,
			OwningOrgID: demoOrgID, CreatedByAccountID: demoAccountID,
		})
	})
	subjectID := mustCreate("subject", func() (string, error) {
		return subjectFac.CreateSubject(ctx, subjectfacade.CreateSubjectDTO{
			Name: "Flussufer Apartments, Unit 12A", Detail: "Wohnung im 3. OG, 65m², 2 Zimmer, Balkon mit Flussblick",
			OwningOrgID: demoOrgID, CreatedByAccountID: demoAccountID,
		})
	})
	mustCreate("rental", func() (string, error) {
		return rentalFac.CreateRental(ctx, rentalfacade.CreateRentalDTO{
			SubjectID: subjectID, TenantPartyID: tenantPartyID, OrgID: demoOrgID, CreatedByAccountID: demoAccountID,
		})
	})

	return demoIDs{tenantPartyID: tenantPartyID, pmPartyID: pmPartyID, agentPartyID: agentPartyID, subjectID: subjectID}
}

type partyCreator interface {
	CreateParty(ctx context.Context, dto partyfacade.CreatePartyDTO) (string, error)
}

type subjectCreator interface {
	CreateSubject(ctx context.Context, dto subjectfacade.CreateSubjectDTO) (string, error)
}

type rentalCreator interface {
	CreateRental(ctx context.Context, dto rentalfacade.CreateRentalDTO) (string, error)
}

func mustCreate(name string, fn func() (string, error)) string {
	id, err := fn()
	if err != nil {
		slog.Error("seed: create "+name+" failed", "error", err)
		os.Exit(1)
	}
	slog.Info("seed: created "+name, "id", id)
	return id
}

type caseOrchestrator interface {
	HandleInboundInquiry(ctx context.Context, dto casefacade.InquiryDTO) (string, error)
}

type workitemCommands interface {
	ConfirmOutboundMessage(ctx context.Context, workItemID string, dto workitemfacade.ConfirmOutboundMessageDTO) error
}

func seedGoldenPath(ctx context.Context, cases caseOrchestrator, workitems workitemCommands, ids demoIDs) {
	slog.Info("seeding golden path: FALL-2024-1842")

	workItemID, err := cases.HandleInboundInquiry(ctx, casefacade.InquiryDTO{
		SenderPartyID: ids.tenantPartyID, OperatorPartyID: ids.pmPartyID,
		AgentPartyID: ids.agentPartyID, SubjectID: ids.subjectID,
		Body: "Ich möchte meinen Mietvertrag für die Einheit 12A in den Flussufer Apartments verlängern. Können Sie mir die aktuellen Konditionen mitteilen?",
	})
	if err != nil {
		slog.Error("golden path seed failed", "error", err)
		os.Exit(1)
	}
	slog.Info("golden path: work item created", "work_item_id", workItemID)

	if err := workitems.ConfirmOutboundMessage(ctx, workItemID, workitemfacade.ConfirmOutboundMessageDTO{
		ConfirmedByPartyID: ids.pmPartyID,
		Body:               "Sehr geehrte Frau Schmidt,\n\nvielen Dank für Ihre Anfrage zur Mietvertragsverlängerung für die Einheit 12A in den Flussufer Apartments.\n\nNach Prüfung Ihres Vertrags können wir Ihnen eine Verlängerung zu den aktualisierten Konditionen anbieten. Die angepasste Miete beträgt 1.496 EUR/Monat (Marktanpassung +3,2%).\n\nBitte bestätigen Sie, ob Sie mit den neuen Konditionen einverstanden sind.\n\nMit freundlichen Grüßen,\nIhr Verwaltungsteam",
	}); err != nil {
		slog.Error("golden path: confirm and send failed", "error", err)
		os.Exit(1)
	}
	slog.Info("golden path: case resolved", "work_item_id", workItemID)
}

func seedPendingCase(ctx context.Context, cases caseOrchestrator, ids demoIDs) {
	slog.Info("seeding pending case")

	pendingID, err := cases.HandleInboundInquiry(ctx, casefacade.InquiryDTO{
		SenderPartyID: ids.tenantPartyID, OperatorPartyID: ids.pmPartyID,
		AgentPartyID: ids.agentPartyID, SubjectID: ids.subjectID,
		Body: "Gibt es die Möglichkeit, einen Stellplatz in der Tiefgarage zusätzlich zu meinem Mietvertrag zu buchen?",
	})
	if err != nil {
		slog.Error("pending case seed failed", "error", err)
		os.Exit(1)
	}
	slog.Info("pending case seeded", "work_item_id", pendingID)
}
