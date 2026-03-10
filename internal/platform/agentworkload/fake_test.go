package agentworkload_test

import (
	"context"
	"testing"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/agentworkload"
)

func TestFakeAdapter_Lookup(t *testing.T) {
	t.Parallel()
	adapter := agentworkload.NewFakeAdapter()
	ctx := context.Background()

	result, err := adapter.Execute(ctx, agentworkload.WorkloadRequest{
		WorkItemID: "wi-1",
		ActionKind: agentworkload.ActionKindLookup,
		Context:    map[string]string{"subject": "test"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ActionKind != agentworkload.ActionKindLookup {
		t.Errorf("expected action kind %q, got %q", agentworkload.ActionKindLookup, result.ActionKind)
	}
	if result.Output == "" {
		t.Error("expected non-empty output")
	}

	// Verify deterministic
	result2, _ := adapter.Execute(ctx, agentworkload.WorkloadRequest{
		WorkItemID: "wi-2",
		ActionKind: agentworkload.ActionKindLookup,
	})
	if result.Output != result2.Output {
		t.Error("expected deterministic output across calls")
	}
}

func TestFakeAdapter_Draft(t *testing.T) {
	t.Parallel()
	adapter := agentworkload.NewFakeAdapter()
	ctx := context.Background()

	result, err := adapter.Execute(ctx, agentworkload.WorkloadRequest{
		WorkItemID: "wi-1",
		ActionKind: agentworkload.ActionKindDraft,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ActionKind != agentworkload.ActionKindDraft {
		t.Errorf("expected action kind %q, got %q", agentworkload.ActionKindDraft, result.ActionKind)
	}
	if result.Output == "" {
		t.Error("expected non-empty output")
	}

	// Verify deterministic
	result2, _ := adapter.Execute(ctx, agentworkload.WorkloadRequest{
		WorkItemID: "wi-2",
		ActionKind: agentworkload.ActionKindDraft,
	})
	if result.Output != result2.Output {
		t.Error("expected deterministic output across calls")
	}
}

func TestFakeAdapter_UnknownAction(t *testing.T) {
	t.Parallel()
	adapter := agentworkload.NewFakeAdapter()
	ctx := context.Background()

	_, err := adapter.Execute(ctx, agentworkload.WorkloadRequest{
		ActionKind: "unknown",
	})
	if err == nil {
		t.Fatal("expected error for unknown action kind")
	}
}
