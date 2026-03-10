package agentworkload

import "context"

// ActionKind represents the type of action an agent performs.
type ActionKind string

const (
	ActionKindLookup ActionKind = "lookup"
	ActionKindDraft  ActionKind = "draft"
)

// WorkloadRequest describes an agent workload to execute.
type WorkloadRequest struct {
	WorkItemID string
	ActionKind ActionKind
	Context    map[string]string // e.g. subject info for lookup
}

// WorkloadResult is the output of an agent workload execution.
type WorkloadResult struct {
	ActionKind ActionKind
	Output     string
	Metadata   map[string]string
}

// Port is the interface for executing agentic workloads.
type Port interface {
	Execute(ctx context.Context, req WorkloadRequest) (WorkloadResult, error)
}
