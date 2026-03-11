package main

const modulePath = "github.com/luminor-project/luminor-core-go-playground"

type policy struct {
	rootDir                 string
	modulePath              string
	verticals               []string
	sharedPackages          []string
	allowedCrossVerticalPkg []string
	allowedCrossSymbols     map[string]struct{}
	allowedVerticalSubpkgs  []string
	facadeOnlyVerticals     []string // verticals that export interfaces (no concrete impl)
}

func defaultPolicy() policy {
	return policy{
		rootDir:                 ".",
		modulePath:              modulePath,
		verticals:               []string{"account", "organization", "content", "rag", "workitem", "app_casehandling", "app_propertymanagement", "app_inquiry", "party", "subject", "rental"},
		sharedPackages:          []string{"common", "shared", "platform"},
		allowedCrossVerticalPkg: []string{"facade"},
		allowedVerticalSubpkgs:  []string{"domain", "facade", "infra", "web", "subscriber", "testharness"},
		facadeOnlyVerticals:     []string{},
		// allowedCrossSymbols is populated at runtime by discoverFacadeValueSymbols.
		// No manual allowlist needed — the facade package IS the allowlist.
		allowedCrossSymbols: nil,
	}
}
