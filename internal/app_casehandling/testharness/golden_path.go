package testharness

import (
	"context"
	"fmt"
	"log/slog"

	casefacade "github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/facade"
	workitemfacade "github.com/luminor-project/luminor-core-go-playground/internal/workitem/facade"
)

type caseOrchestrator interface {
	HandleInboundInquiry(ctx context.Context, dto casefacade.InquiryDTO) (string, error)
}

type workitemCommands interface {
	ConfirmOutboundMessage(ctx context.Context, workItemID string, dto workitemfacade.ConfirmOutboundMessageDTO) error
}

// SeedGoldenPath runs the FALL-2024-1842 scenario:
// Anna Schmidt requests a lease renewal for Flussufer 12A.
// The AI assistant performs a lookup and drafts a response.
// Sarah (operator) confirms and sends the response.
func SeedGoldenPath(ctx context.Context, cases caseOrchestrator, workitems workitemCommands) error {
	slog.Info("seeding golden path: FALL-2024-1842")

	workItemID, err := cases.HandleInboundInquiry(ctx, casefacade.InquiryDTO{
		SenderPartyID:   "party-anna-schmidt",
		OperatorPartyID: "party-sarah",
		AgentPartyID:    "party-ki-assistent",
		SubjectID:       "subject-flussufer-12a",
		Body:            "Ich möchte meinen Mietvertrag für die Einheit 12A in den Flussufer Apartments verlängern. Können Sie mir die aktuellen Konditionen mitteilen?",
	})
	if err != nil {
		return fmt.Errorf("handle inbound inquiry: %w", err)
	}

	slog.Info("golden path: work item created", "work_item_id", workItemID)

	// Operator confirms the AI-drafted response
	if err := workitems.ConfirmOutboundMessage(ctx, workItemID, workitemfacade.ConfirmOutboundMessageDTO{
		ConfirmedByPartyID: "party-sarah",
		Body:               "Sehr geehrte Frau Schmidt,\n\nvielen Dank für Ihre Anfrage zur Mietvertragsverlängerung für die Einheit 12A in den Flussufer Apartments.\n\nNach Prüfung Ihres Vertrags können wir Ihnen eine Verlängerung zu den aktualisierten Konditionen anbieten. Die angepasste Miete beträgt 1.496 EUR/Monat (Marktanpassung +3,2%).\n\nBitte bestätigen Sie, ob Sie mit den neuen Konditionen einverstanden sind.\n\nMit freundlichen Grüßen,\nIhr Verwaltungsteam",
	}); err != nil {
		return fmt.Errorf("confirm and send: %w", err)
	}

	slog.Info("golden path: case resolved", "work_item_id", workItemID)
	return nil
}

// SeedPendingCase creates a second case that is left in "pending_confirmation" state
// (no confirm step). Useful for demonstrating the confirm UI.
func SeedPendingCase(ctx context.Context, cases caseOrchestrator) (string, error) {
	slog.Info("seeding pending case")

	workItemID, err := cases.HandleInboundInquiry(ctx, casefacade.InquiryDTO{
		SenderPartyID:   "party-anna-schmidt",
		OperatorPartyID: "party-sarah",
		AgentPartyID:    "party-ki-assistent",
		SubjectID:       "subject-flussufer-12a",
		Body:            "Gibt es die Möglichkeit, einen Stellplatz in der Tiefgarage zusätzlich zu meinem Mietvertrag zu buchen?",
	})
	if err != nil {
		return "", fmt.Errorf("handle inbound inquiry (pending): %w", err)
	}

	slog.Info("pending case seeded", "work_item_id", workItemID)
	return workItemID, nil
}
