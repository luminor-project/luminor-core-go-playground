package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTypeAwareChecker_FailsOnForeignConcreteFunction(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/archtest\n\ngo 1.26\n")
	writeFile(t, root, "internal/organization/facade/client.go", `package facade
func NewConcrete() string { return "x" }
`)
	writeFile(t, root, "internal/account/web/handler.go", `package web
import orgfacade "example.com/archtest/internal/organization/facade"
func Use() string { return orgfacade.NewConcrete() }
`)

	p := policy{
		rootDir:                 root,
		modulePath:              "example.com/archtest",
		verticals:               []string{"account", "organization", "content"},
		sharedPackages:          []string{"common", "shared", "platform"},
		forbiddenCrossSubpkgs:   []string{"domain", "infra", "web", "subscriber", "testharness"},
		allowedCrossVerticalPkg: []string{"facade"},
		allowedCrossSymbols:     map[string]struct{}{},
	}
	violations, err := checkTypeBoundaries(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected violation for foreign concrete function usage")
	}
}

func TestTypeAwareChecker_AllowsForeignInterfaceAndMethodCalls(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/archtest\n\ngo 1.26\n")
	writeFile(t, root, "internal/organization/facade/client.go", `package facade
type Client interface { Ping() error }
`)
	writeFile(t, root, "internal/account/web/handler.go", `package web
import orgfacade "example.com/archtest/internal/organization/facade"
type localClient interface { Ping() error }
func Use(c orgfacade.Client) error { return c.Ping() }
`)

	p := policy{
		rootDir:                 root,
		modulePath:              "example.com/archtest",
		verticals:               []string{"account", "organization", "content"},
		sharedPackages:          []string{"common", "shared", "platform"},
		forbiddenCrossSubpkgs:   []string{"domain", "infra", "web", "subscriber", "testharness"},
		allowedCrossVerticalPkg: []string{"facade"},
		allowedCrossSymbols:     map[string]struct{}{},
	}
	violations, err := checkTypeBoundaries(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got: %v", violations)
	}
}

func TestTypeAwareChecker_AllowsAllowlistedValueTypes(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/archtest\n\ngo 1.26\n")
	writeFile(t, root, "internal/organization/facade/events.go", `package facade
type ActiveOrgChangedEvent struct { OrganizationID string }
`)
	writeFile(t, root, "internal/account/subscriber/sub.go", `package subscriber
import orgfacade "example.com/archtest/internal/organization/facade"
func Handle(e orgfacade.ActiveOrgChangedEvent) string { return e.OrganizationID }
`)

	p := policy{
		rootDir:                 root,
		modulePath:              "example.com/archtest",
		verticals:               []string{"account", "organization", "content"},
		sharedPackages:          []string{"common", "shared", "platform"},
		forbiddenCrossSubpkgs:   []string{"domain", "infra", "web", "subscriber", "testharness"},
		allowedCrossVerticalPkg: []string{"facade"},
		allowedCrossSymbols: map[string]struct{}{
			"example.com/archtest/internal/organization/facade.ActiveOrgChangedEvent": {},
		},
	}
	violations, err := checkTypeBoundaries(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got: %v", violations)
	}
}

func TestCurrentRepoPolicyHasNoViolations(t *testing.T) {
	p := defaultPolicy()
	p.rootDir = filepath.Join("..", "..")
	importViolations, err := checkImportBoundaries(p)
	if err != nil {
		t.Fatalf("unexpected import-check error: %v", err)
	}
	typeViolations, err := checkTypeBoundaries(p)
	if err != nil {
		t.Fatalf("unexpected type-check error: %v", err)
	}
	all := append(importViolations, typeViolations...)
	if len(all) != 0 {
		t.Fatalf("expected no violations in current repo, got: %s", strings.Join(all, "\n"))
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", abs, err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
}
