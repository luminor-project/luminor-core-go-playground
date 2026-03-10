package main

const modulePath = "github.com/luminor-project/luminor-core-go-playground"

type policy struct {
	rootDir                 string
	modulePath              string
	verticals               []string
	sharedPackages          []string
	allowedCrossVerticalPkg []string
	allowedCrossSymbols     map[string]struct{}
}

func defaultPolicy() policy {
	return policy{
		rootDir:                 ".",
		modulePath:              modulePath,
		verticals:               []string{"account", "organization", "content", "rag", "workitem", "app_casehandling", "party", "subject"},
		sharedPackages:          []string{"common", "shared", "platform"},
		allowedCrossVerticalPkg: []string{"facade"},
		// allowedCrossSymbols is populated at runtime by discoverFacadeValueSymbols.
		// No manual allowlist needed — the facade package IS the allowlist.
		allowedCrossSymbols: nil,
	}
}
