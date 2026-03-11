package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	// Account vertical
	accountdomain "github.com/luminor-project/luminor-core-go-playground/internal/account/domain"
	accountfacade "github.com/luminor-project/luminor-core-go-playground/internal/account/facade"
	accountinfra "github.com/luminor-project/luminor-core-go-playground/internal/account/infra"
	accountsub "github.com/luminor-project/luminor-core-go-playground/internal/account/subscriber"

	// Organization vertical
	orgdomain "github.com/luminor-project/luminor-core-go-playground/internal/organization/domain"
	orgfacade "github.com/luminor-project/luminor-core-go-playground/internal/organization/facade"
	orginfra "github.com/luminor-project/luminor-core-go-playground/internal/organization/infra"
	orgsub "github.com/luminor-project/luminor-core-go-playground/internal/organization/subscriber"

	// Party vertical
	partydomain "github.com/luminor-project/luminor-core-go-playground/internal/party/domain"
	partyfacade "github.com/luminor-project/luminor-core-go-playground/internal/party/facade"
	partyinfra "github.com/luminor-project/luminor-core-go-playground/internal/party/infra"
	partysub "github.com/luminor-project/luminor-core-go-playground/internal/party/subscriber"

	// Subject vertical
	subjectdomain "github.com/luminor-project/luminor-core-go-playground/internal/subject/domain"
	subjectfacade "github.com/luminor-project/luminor-core-go-playground/internal/subject/facade"
	subjectinfra "github.com/luminor-project/luminor-core-go-playground/internal/subject/infra"

	// Rental vertical
	rentaldomain "github.com/luminor-project/luminor-core-go-playground/internal/rental/domain"
	rentalfacade "github.com/luminor-project/luminor-core-go-playground/internal/rental/facade"
	rentalinfra "github.com/luminor-project/luminor-core-go-playground/internal/rental/infra"

	// App verticals
	casefacade "github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/facade"
	caseinfra "github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/infra"
	casesub "github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/subscriber"
	inquiryfacade "github.com/luminor-project/luminor-core-go-playground/internal/app_inquiry/facade"
	pmfacade "github.com/luminor-project/luminor-core-go-playground/internal/app_propertymanagement/facade"

	// WorkItem vertical
	workitemfacade "github.com/luminor-project/luminor-core-go-playground/internal/workitem/facade"

	// Platform
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/agentworkload"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/config"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/database"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventstore"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/outbox"
)

// seedContext carries IDs resolved during seeding.
type seedContext struct {
	accountID string
	orgID     string
	pmPartyID string
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <email> <password>\n", os.Args[0])
		os.Exit(1)
	}
	email := os.Args[1]
	password := os.Args[2]

	cfg := mustLoadConfig()
	db := mustConnect(cfg.DatabaseURL)
	defer db.Close()

	ctx := context.Background()
	bus := eventbus.New()
	clk := clock.New()

	// ── Build all facades (mirrors cmd/server/main.go wiring) ─────────
	orgService := orgdomain.NewOrgService(orginfra.NewPostgresRepository(db), clk)
	oFacade := orgfacade.New(orgService, bus)
	acctFacade := accountfacade.New(
		accountdomain.NewAccountService(accountinfra.NewPostgresRepository(db), clk),
		bus, outbox.NewPostgresStore(db),
	)
	partyFac := partyfacade.New(partydomain.NewPartyService(partyinfra.NewPostgresRepository(db), clk))
	subjectFac := subjectfacade.New(subjectdomain.NewSubjectService(subjectinfra.NewPostgresRepository(db), clk))
	rentalFac := rentalfacade.New(rentaldomain.NewRentalService(rentalinfra.NewPostgresRepository(db), clk))
	wiFacade := workitemfacade.New(eventstore.NewPostgresStore(db), bus, clk)
	caseFac := casefacade.New(wiFacade, agentworkload.NewFakeAdapter(), subjectFac)
	inqFacade := inquiryfacade.New(rentalFac, caseFac, partyFac)
	pmFac := pmfacade.New(partyFac, subjectFac, rentalFac, acctFacade, oFacade)

	// ── Register all event subscribers ────────────────────────────────
	orgsub.RegisterAccountCreatedSubscriber(bus, oFacade)
	accountsub.RegisterOrgChangedSubscriber(bus, acctFacade)
	partysub.RegisterAccountJoinedOrgSubscriber(bus, partyFac, acctFacade)
	casesub.RegisterProjectionSubscribers(bus, caseinfra.NewDashboardStore(db), partyFac, subjectFac)

	// ── Seed ──────────────────────────────────────────────────────────
	sc := seedAccountAndOrg(ctx, acctFacade, orgService, partyFac, email, password)
	seedDemoData(ctx, pmFac, inqFacade, wiFacade, sc)

	slog.Info("seed completed successfully", "email", email, "org", "Flussufer Verwaltung GmbH")
}

// seedAccountAndOrg registers the account (triggering the full event chain),
// renames the org, and creates an AI assistant party.
func seedAccountAndOrg(
	ctx context.Context,
	acctFacade accountRegistrar,
	orgService orgRenamer,
	partyFac partyCreator,
	email, password string,
) seedContext {
	slog.Info("step 1: registering account", "email", email)
	accountID := must("register account", func() (string, error) {
		return acctFacade.Register(ctx, accountfacade.RegistrationDTO{
			Email:         email,
			PlainPassword: password,
		})
	})

	info, err := acctFacade.GetAccountInfoByID(ctx, accountID)
	if err != nil {
		slog.Error("failed to get account info after registration", "error", err)
		os.Exit(1)
	}
	orgID := info.CurrentlyActiveOrganizationID
	pmPartyID := info.CurrentlyActivePartyID
	slog.Info("account registered",
		"account_id", accountID, "org_id", orgID, "pm_party_id", pmPartyID,
	)

	slog.Info("step 2: renaming organization")
	if err := orgService.RenameOrganization(ctx, orgID, "Flussufer Verwaltung GmbH"); err != nil {
		slog.Error("failed to rename organization", "error", err)
		os.Exit(1)
	}

	slog.Info("step 3: creating AI assistant party")
	must("agent party", func() (string, error) {
		return partyFac.CreateParty(ctx, partyfacade.CreatePartyDTO{
			Name: "KI-Assistent", ActorKind: partyfacade.ActorKindAssistant,
			PartyKind: partyfacade.PartyKindAssistant, OwningOrgID: orgID,
			CreatedByAccountID: accountID,
		})
	})

	return seedContext{accountID: accountID, orgID: orgID, pmPartyID: pmPartyID}
}

// seedDemoData creates properties, tenants, rentals, and inquiry cases.
func seedDemoData(
	ctx context.Context,
	pmFac propertyManager,
	inqFacade inquirySubmitter,
	wiFacade workitemConfirmer,
	sc seedContext,
) {
	slog.Info("step 4: creating properties")
	type prop struct{ name, detail string }
	properties := []prop{
		{"Flussufer Apartments, Unit 12A", "Wohnung im 3. OG, 65m², 2 Zimmer, Balkon mit Flussblick"},
		{"Flussufer Apartments, Unit 7B", "Erdgeschoss, 45m², 1 Zimmer, Gartenanteil"},
		{"Parkblick Residenz, Unit 3C", "Penthouse, 120m², 4 Zimmer, Dachterrasse"},
	}
	propertyIDs := make([]string, len(properties))
	for i, p := range properties {
		propertyIDs[i] = must("property "+p.name, func() (string, error) {
			return pmFac.CreateProperty(ctx, pmfacade.CreatePropertyDTO{
				Name: p.name, Detail: p.detail, OrgID: sc.orgID, CreatedByAccountID: sc.accountID,
			})
		})
	}

	slog.Info("step 5: creating tenants")
	tenantNames := []string{"Anna Schmidt", "Max Weber", "Lisa Müller"}
	tenantIDs := make([]string, len(tenantNames))
	for i, name := range tenantNames {
		tenantIDs[i] = must("tenant "+name, func() (string, error) {
			return pmFac.CreateTenant(ctx, pmfacade.CreateTenantDTO{
				Name: name, OrgID: sc.orgID, CreatedByAccountID: sc.accountID,
			})
		})
	}

	slog.Info("step 6: assigning tenants to properties")
	for i := range tenantIDs {
		must("rental "+tenantNames[i], func() (string, error) {
			return pmFac.AssignTenantToProperty(ctx, pmfacade.AssignTenantDTO{
				SubjectID: propertyIDs[i], TenantPartyID: tenantIDs[i],
				OrgID: sc.orgID, CreatedByAccountID: sc.accountID,
			})
		})
	}

	slog.Info("step 7: submitting inquiry cases")
	seedInquiries(ctx, inqFacade, wiFacade, tenantIDs, tenantNames, sc.orgID, sc.pmPartyID)
}

func seedInquiries(
	ctx context.Context,
	inqFacade inquirySubmitter,
	wiFacade workitemConfirmer,
	tenantIDs, tenantNames []string,
	orgID, pmPartyID string,
) {
	type spec struct {
		tenantIdx int
		body      string
		confirm   bool
	}
	inquiries := []spec{
		{0, "Ich möchte meinen Mietvertrag für die Einheit 12A in den Flussufer Apartments verlängern. " +
			"Können Sie mir die aktuellen Konditionen mitteilen?", true},
		{0, "Gibt es die Möglichkeit, einen Stellplatz in der Tiefgarage zusätzlich " +
			"zu meinem Mietvertrag zu buchen?", false},
		{1, "Seit gestern Abend funktioniert die Heizung in meiner Wohnung nicht mehr. " +
			"Die Raumtemperatur ist bereits auf 16°C gesunken. " +
			"Können Sie bitte schnellstmöglich einen Techniker schicken?", false},
		{2, "Die Abdichtung an der Dachterrasse zeigt Risse und bei starkem Regen " +
			"tritt Feuchtigkeit ein. Könnten Sie bitte eine Inspektion und ggf. " +
			"Reparatur veranlassen?", false},
	}

	for _, inq := range inquiries {
		tenantName := tenantNames[inq.tenantIdx]
		workItemID := must("inquiry from "+tenantName, func() (string, error) {
			return inqFacade.SubmitInquiry(ctx, inquiryfacade.SubmitInquiryDTO{
				TenantPartyID: tenantIDs[inq.tenantIdx],
				OrgID:         orgID,
				Body:          inq.body,
			})
		})
		if inq.confirm {
			confirmCase(ctx, wiFacade, workItemID, pmPartyID)
		}
	}
}

func confirmCase(ctx context.Context, wiFacade workitemConfirmer, workItemID, pmPartyID string) {
	body := "Sehr geehrte Frau Schmidt,\n\nvielen Dank für Ihre Anfrage zur " +
		"Mietvertragsverlängerung für die Einheit 12A in den Flussufer Apartments.\n\n" +
		"Nach Prüfung Ihres Vertrags können wir Ihnen eine Verlängerung zu den aktualisierten " +
		"Konditionen anbieten. Die angepasste Miete beträgt 1.496 EUR/Monat " +
		"(Marktanpassung +3,2%).\n\nBitte bestätigen Sie, ob Sie mit den neuen " +
		"Konditionen einverstanden sind.\n\nMit freundlichen Grüßen,\nIhr Verwaltungsteam"
	if err := wiFacade.ConfirmOutboundMessage(ctx, workItemID, workitemfacade.ConfirmOutboundMessageDTO{
		ConfirmedByPartyID: pmPartyID,
		Body:               body,
	}); err != nil {
		slog.Error("confirm case failed", "error", err)
		os.Exit(1)
	}
	slog.Info("case confirmed", "work_item_id", workItemID)
}

// ── Interfaces (consumed here, defined here) ──────────────────────────────

type accountRegistrar interface {
	Register(ctx context.Context, dto accountfacade.RegistrationDTO) (string, error)
	GetAccountInfoByID(ctx context.Context, id string) (accountfacade.AccountInfoDTO, error)
}

type orgRenamer interface {
	RenameOrganization(ctx context.Context, orgID, newName string) error
}

type partyCreator interface {
	CreateParty(ctx context.Context, dto partyfacade.CreatePartyDTO) (string, error)
}

type propertyManager interface {
	CreateProperty(ctx context.Context, dto pmfacade.CreatePropertyDTO) (string, error)
	CreateTenant(ctx context.Context, dto pmfacade.CreateTenantDTO) (string, error)
	AssignTenantToProperty(ctx context.Context, dto pmfacade.AssignTenantDTO) (string, error)
}

type inquirySubmitter interface {
	SubmitInquiry(ctx context.Context, dto inquiryfacade.SubmitInquiryDTO) (string, error)
}

type workitemConfirmer interface {
	ConfirmOutboundMessage(ctx context.Context, workItemID string, dto workitemfacade.ConfirmOutboundMessageDTO) error
}

// ── Helpers ───────────────────────────────────────────────────────────────

func must(name string, fn func() (string, error)) string {
	id, err := fn()
	if err != nil {
		slog.Error("seed: "+name+" failed", "error", err)
		os.Exit(1)
	}
	slog.Info("seed: created "+name, "id", id)
	return id
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
