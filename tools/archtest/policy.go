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
		verticals:               []string{"account", "organization", "content", "rag"},
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
		},
	}
}
