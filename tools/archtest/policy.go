package main

const modulePath = "github.com/luminor-project/luminor-core-go-playground"

type policy struct {
	rootDir                 string
	modulePath              string
	verticals               []string
	sharedPackages          []string
	forbiddenCrossSubpkgs   []string
	allowedCrossVerticalPkg []string
	allowedCrossSymbols     map[string]struct{}
}

func defaultPolicy() policy {
	return policy{
		rootDir:                 ".",
		modulePath:              modulePath,
		verticals:               []string{"account", "organization", "content", "rag", "workitem", "app_casehandling", "party", "subject"},
		sharedPackages:          []string{"common", "shared", "platform"},
		forbiddenCrossSubpkgs:   []string{"domain", "infra", "web", "subscriber", "testharness"},
		allowedCrossVerticalPkg: []string{"facade"},
		allowedCrossSymbols: map[string]struct{}{
			modulePath + "/internal/account/facade.AccountInfoDTO":             {},
			modulePath + "/internal/account/facade.RegistrationDTO":            {},
			modulePath + "/internal/account/facade.AccountCreatedEvent":        {},
			modulePath + "/internal/organization/facade.ActiveOrgChangedEvent": {},
			modulePath + "/internal/organization/facade.OrganizationDTO":       {},
			modulePath + "/internal/organization/facade.MemberDTO":             {},
			modulePath + "/internal/organization/facade.GroupDTO":              {},
			modulePath + "/internal/organization/facade.InvitationDTO":         {},
			modulePath + "/internal/rag/facade.IndexDocumentDTO":               {},
			modulePath + "/internal/rag/facade.DocumentDTO":                    {},
			modulePath + "/internal/rag/facade.SearchResultDTO":                {},
			modulePath + "/internal/rag/facade.ChatResponseDTO":                {},
			modulePath + "/internal/rag/facade.DocumentIndexedEvent":           {},

			// WorkItem vertical
			modulePath + "/internal/workitem/facade.IntakeInboundMessageDTO":           {},
			modulePath + "/internal/workitem/facade.RecordAssistantActionDTO":          {},
			modulePath + "/internal/workitem/facade.ConfirmOutboundMessageDTO":         {},
			modulePath + "/internal/workitem/facade.WorkItemCreatedEvent":              {},
			modulePath + "/internal/workitem/facade.PartyLinkedEvent":                  {},
			modulePath + "/internal/workitem/facade.SubjectLinkedEvent":                {},
			modulePath + "/internal/workitem/facade.InboundMessageRecordedEvent":       {},
			modulePath + "/internal/workitem/facade.AssistantActionRecordedEvent":      {},
			modulePath + "/internal/workitem/facade.OutboundMessageRecordedEvent":      {},
			modulePath + "/internal/workitem/facade.WorkItemStatusChangedEvent":        {},
			modulePath + "/internal/workitem/facade.AddNoteDTO":                        {},
			modulePath + "/internal/workitem/facade.EditNoteDTO":                       {},
			modulePath + "/internal/workitem/facade.DeleteNoteDTO":                     {},
			modulePath + "/internal/workitem/facade.NoteAddedToTimelineEntryEvent":     {},
			modulePath + "/internal/workitem/facade.NoteEditedOnTimelineEntryEvent":    {},
			modulePath + "/internal/workitem/facade.NoteDeletedFromTimelineEntryEvent": {},

			// Party vertical
			modulePath + "/internal/party/facade.PartyInfoDTO":     {},
			modulePath + "/internal/party/facade.ErrPartyNotFound": {},

			// Subject vertical
			modulePath + "/internal/subject/facade.SubjectInfoDTO":     {},
			modulePath + "/internal/subject/facade.ErrSubjectNotFound": {},
		},
	}
}
