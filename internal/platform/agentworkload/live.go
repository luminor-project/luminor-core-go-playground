package agentworkload

import (
	"context"
	"errors"
)

// LiveAdapter is a placeholder for the real agent workload implementation.
type LiveAdapter struct{}

// NewLiveAdapter creates a new live agent workload adapter.
func NewLiveAdapter() *LiveAdapter {
	return &LiveAdapter{}
}

// Execute is not yet implemented.
func (a *LiveAdapter) Execute(_ context.Context, _ WorkloadRequest) (WorkloadResult, error) {
	return WorkloadResult{}, errors.New("live agent workload adapter not yet implemented")
}
