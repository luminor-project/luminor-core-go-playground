package agentworkload

import "context"

// WorkloadRequest describes an agent workload to execute.
type WorkloadRequest struct {
	WorkItemID string
	ActionKind string            // "lookup", "draft"
	Context    map[string]string // e.g. subject info for lookup
}

// WorkloadResult is the output of an agent workload execution.
type WorkloadResult struct {
	ActionKind string
	Output     string
	Metadata   map[string]string
}

// Port is the interface for executing agentic workloads.
type Port interface {
	Execute(ctx context.Context, req WorkloadRequest) (WorkloadResult, error)
}
