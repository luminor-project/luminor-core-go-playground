package testharness

import (
	"github.com/luminor-project/luminor-core-go-playground/internal/workitem/facade"
)

// MakeIntakeDTO creates an IntakeInboundMessageDTO with golden-path defaults.
func MakeIntakeDTO() facade.IntakeInboundMessageDTO {
	return facade.IntakeInboundMessageDTO{
		SenderPartyID:  "party-anna-schmidt",
		SubjectID:      "subject-flussufer-12a",
		Body:           "Ich möchte meinen Mietvertrag für die Einheit 12A in den Flussufer Apartments verlängern. Können Sie mir die aktuellen Konditionen mitteilen?",
		HandlerPartyID: "party-sarah",
		AgentPartyID:   "party-ki-assistent",
	}
}

// MakeLookupDTO creates a RecordAssistantActionDTO for a lookup action.
func MakeLookupDTO(output string) facade.RecordAssistantActionDTO {
	return facade.RecordAssistantActionDTO{
		ActorID:     "party-ki-assistent",
		ActionKind:  "lookup",
		Output:      output,
		DraftStatus: "",
	}
}

// MakeDraftDTO creates a RecordAssistantActionDTO for a draft action.
func MakeDraftDTO(output string) facade.RecordAssistantActionDTO {
	return facade.RecordAssistantActionDTO{
		ActorID:     "party-ki-assistent",
		ActionKind:  "draft",
		Output:      output,
		DraftStatus: "pending",
	}
}

// MakeConfirmDTO creates a ConfirmOutboundMessageDTO with golden-path defaults.
func MakeConfirmDTO(body string) facade.ConfirmOutboundMessageDTO {
	return facade.ConfirmOutboundMessageDTO{
		ConfirmedByPartyID: "party-sarah",
		Body:               body,
	}
}
