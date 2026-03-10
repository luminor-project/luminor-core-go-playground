package agentworkload

import (
	"context"
	"fmt"
)

const (
	fakeLookupOutput = "Mietvertrag Einheit 12A, Flussufer Apartments. Aktuelle Miete: 1.450 EUR/Monat. Vertragslaufzeit bis 30.04.2025. Verlängerungsklausel vorhanden. Marktanpassung: +3,2%."
	fakeDraftOutput  = "Sehr geehrte Frau Schmidt,\n\nvielen Dank für Ihre Anfrage zur Mietvertragsverlängerung für die Einheit 12A in den Flussufer Apartments.\n\nNach Prüfung Ihres Vertrags können wir Ihnen eine Verlängerung zu den aktualisierten Konditionen anbieten. Die angepasste Miete beträgt 1.496 EUR/Monat (Marktanpassung +3,2%).\n\nBitte bestätigen Sie, ob Sie mit den neuen Konditionen einverstanden sind.\n\nMit freundlichen Grüßen,\nIhr Verwaltungsteam"
)

// FakeAdapter returns deterministic hardcoded results for demo/testing.
type FakeAdapter struct{}

// NewFakeAdapter creates a new fake agent workload adapter.
func NewFakeAdapter() *FakeAdapter {
	return &FakeAdapter{}
}

// Execute returns hardcoded results based on the action kind.
func (a *FakeAdapter) Execute(_ context.Context, req WorkloadRequest) (WorkloadResult, error) {
	switch req.ActionKind {
	case "lookup":
		return WorkloadResult{
			ActionKind: "lookup",
			Output:     fakeLookupOutput,
			Metadata:   map[string]string{"source": "contract-db"},
		}, nil
	case "draft":
		return WorkloadResult{
			ActionKind: "draft",
			Output:     fakeDraftOutput,
			Metadata:   map[string]string{"model": "fake"},
		}, nil
	default:
		return WorkloadResult{}, fmt.Errorf("unknown action kind: %s", req.ActionKind)
	}
}
