package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportBoundary_BlocksAnyNonFacadeSubpackage(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/archtest\n\ngo 1.26\n")
	// A novel subpackage name that isn't "domain" or "infra" — still blocked.
	writeFile(t, root, "internal/organization/utils/helper.go", `package utils
func Help() string { return "x" }
`)
	writeFile(t, root, "internal/account/web/handler.go", `package web
import "example.com/archtest/internal/organization/utils"
var _ = utils.Help
`)

	p := policy{
		rootDir:                 root,
		modulePath:              "example.com/archtest",
		verticals:               []string{"account", "organization"},
		sharedPackages:          []string{"common", "shared", "platform"},
		allowedCrossVerticalPkg: []string{"facade"},
	}
	violations, err := checkImportBoundaries(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected violation for cross-vertical import of non-facade subpackage 'utils'")
	}
}

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
		rootDir:        root,
		modulePath:     "example.com/archtest",
		verticals:      []string{"account", "organization", "content"},
		sharedPackages: []string{"common", "shared", "platform"},

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
		rootDir:        root,
		modulePath:     "example.com/archtest",
		verticals:      []string{"account", "organization", "content"},
		sharedPackages: []string{"common", "shared", "platform"},

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

func TestTypeAwareChecker_AutoDiscoversFacadeValueTypes(t *testing.T) {
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
		rootDir:        root,
		modulePath:     "example.com/archtest",
		verticals:      []string{"account", "organization", "content"},
		sharedPackages: []string{"common", "shared", "platform"},

		allowedCrossVerticalPkg: []string{"facade"},
	}
	// Auto-discover instead of manual allowlist
	symbols, err := discoverFacadeValueSymbols(p)
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}
	p.allowedCrossSymbols = symbols

	if _, ok := symbols["example.com/archtest/internal/organization/facade.ActiveOrgChangedEvent"]; !ok {
		t.Fatal("expected auto-discovered symbol ActiveOrgChangedEvent")
	}

	violations, err := checkTypeBoundaries(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got: %v", violations)
	}
}

func TestAutoDiscovery_ExcludesFunctions(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/archtest\n\ngo 1.26\n")
	writeFile(t, root, "internal/organization/facade/impl.go", `package facade
type MyDTO struct { Name string }
func New() *MyDTO { return &MyDTO{} }
`)

	p := policy{
		rootDir:        root,
		modulePath:     "example.com/archtest",
		verticals:      []string{"account", "organization"},
		sharedPackages: []string{"common", "shared", "platform"},

		allowedCrossVerticalPkg: []string{"facade"},
	}
	symbols, err := discoverFacadeValueSymbols(p)
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}

	if _, ok := symbols["example.com/archtest/internal/organization/facade.MyDTO"]; !ok {
		t.Fatal("expected MyDTO to be discovered")
	}
	if _, ok := symbols["example.com/archtest/internal/organization/facade.New"]; ok {
		t.Fatal("expected New (function) to be excluded from discovery")
	}
}

func TestCurrentRepoPolicyHasNoViolations(t *testing.T) {
	p := defaultPolicy()
	p.rootDir = filepath.Join("..", "..")

	// Auto-discover facade value symbols
	symbols, err := discoverFacadeValueSymbols(p)
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}
	p.allowedCrossSymbols = symbols

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

func TestEventSourcingRequired_PassesForCompliantVertical(t *testing.T) {
	root := t.TempDir()
	// Facade imports eventstore.
	writeFile(t, root, "internal/party/facade/impl.go", `package facade
import "example.com/archtest/internal/platform/eventstore"
var _ eventstore.Store
`)
	// Subscriber exists.
	writeFile(t, root, "internal/party/subscriber/projection.go", `package subscriber
`)
	// Domain exports DeserializeEvent.
	writeFile(t, root, "internal/party/domain/serialization.go", `package domain
func DeserializeEvent(eventType string, raw []byte) (any, error) { return nil, nil }
`)

	p := policy{
		rootDir:               root,
		modulePath:            "example.com/archtest",
		eventSourcedVerticals: []string{"party"},
	}
	violations, err := checkEventSourcingRequired(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got: %v", violations)
	}
}

func TestEventSourcingRequired_DetectsMissingEventStore(t *testing.T) {
	root := t.TempDir()
	// Facade does NOT import eventstore (CRUD-style).
	writeFile(t, root, "internal/party/facade/impl.go", `package facade
import "database/sql"
var _ sql.DB
`)
	writeFile(t, root, "internal/party/subscriber/projection.go", `package subscriber
`)
	writeFile(t, root, "internal/party/domain/serialization.go", `package domain
func DeserializeEvent(eventType string, raw []byte) (any, error) { return nil, nil }
`)

	p := policy{
		rootDir:               root,
		modulePath:            "example.com/archtest",
		eventSourcedVerticals: []string{"party"},
	}
	violations, err := checkEventSourcingRequired(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}
	if !strings.Contains(violations[0], "does not import platform/eventstore") {
		t.Fatalf("unexpected violation message: %s", violations[0])
	}
}

func TestEventSourcingRequired_DetectsMissingFacade(t *testing.T) {
	root := t.TempDir()
	// No facade/ directory at all.
	writeFile(t, root, "internal/party/domain/party.go", `package domain
type Party struct{}
`)

	p := policy{
		rootDir:               root,
		modulePath:            "example.com/archtest",
		eventSourcedVerticals: []string{"party"},
	}
	violations, err := checkEventSourcingRequired(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}
	if !strings.Contains(violations[0], "no facade/ package") {
		t.Fatalf("unexpected violation message: %s", violations[0])
	}
}

func TestEventSourcingRequired_DetectsMissingDeserializeEvent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/party/facade/impl.go", `package facade
import "example.com/archtest/internal/platform/eventstore"
var _ eventstore.Store
`)
	writeFile(t, root, "internal/party/subscriber/projection.go", `package subscriber
`)
	// Domain exists but has no DeserializeEvent.
	writeFile(t, root, "internal/party/domain/party.go", `package domain
type Party struct{}
`)

	p := policy{
		rootDir:               root,
		modulePath:            "example.com/archtest",
		eventSourcedVerticals: []string{"party"},
	}
	violations, err := checkEventSourcingRequired(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}
	if !strings.Contains(violations[0], "does not export DeserializeEvent") {
		t.Fatalf("unexpected violation message: %s", violations[0])
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
