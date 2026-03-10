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

			// WorkItem value types (re-exported from domain via facade aliases)
			modulePath + "/internal/workitem/facade.Status":                    {},
			modulePath + "/internal/workitem/facade.StatusNew":                 {},
			modulePath + "/internal/workitem/facade.StatusInProgress":          {},
			modulePath + "/internal/workitem/facade.StatusPendingConfirmation": {},
			modulePath + "/internal/workitem/facade.StatusResolved":            {},
			modulePath + "/internal/workitem/facade.ActionKind":                {},
			modulePath + "/internal/workitem/facade.ActionKindLookup":          {},
			modulePath + "/internal/workitem/facade.ActionKindDraft":           {},
			modulePath + "/internal/workitem/facade.DraftStatus":               {},
			modulePath + "/internal/workitem/facade.DraftStatusNone":           {},
			modulePath + "/internal/workitem/facade.DraftStatusPending":        {},
			modulePath + "/internal/workitem/facade.PartyRole":                 {},
			modulePath + "/internal/workitem/facade.PartyRoleSender":           {},
			modulePath + "/internal/workitem/facade.PartyRoleHandler":          {},
			modulePath + "/internal/workitem/facade.PartyRoleAgent":            {},
			modulePath + "/internal/workitem/domain.Status":                    {},
			modulePath + "/internal/workitem/domain.ActionKind":                {},
			modulePath + "/internal/workitem/domain.DraftStatus":               {},
			modulePath + "/internal/workitem/domain.PartyRole":                 {},

			// Party vertical
			modulePath + "/internal/party/facade.PartyInfoDTO":       {},
			modulePath + "/internal/party/facade.ErrPartyNotFound":   {},
			modulePath + "/internal/party/facade.ActorKind":          {},
			modulePath + "/internal/party/facade.ActorKindHuman":     {},
			modulePath + "/internal/party/facade.ActorKindAssistant": {},

			// Subject vertical
			modulePath + "/internal/subject/facade.SubjectInfoDTO":     {},
			modulePath + "/internal/subject/facade.ErrSubjectNotFound": {},
		},
	}
}
