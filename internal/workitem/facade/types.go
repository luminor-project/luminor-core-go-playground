package facade

import "github.com/luminor-project/luminor-core-go-playground/internal/workitem/domain"

// Re-export domain value types as the public API.
// Canonical definitions live in domain/types.go; the facade exposes them
// for cross-vertical consumers so that no one imports domain directly.

type Status = domain.Status

const (
	StatusNew                 = domain.StatusNew
	StatusInProgress          = domain.StatusInProgress
	StatusPendingConfirmation = domain.StatusPendingConfirmation
	StatusResolved            = domain.StatusResolved
)

type ActionKind = domain.ActionKind

const (
	ActionKindLookup = domain.ActionKindLookup
	ActionKindDraft  = domain.ActionKindDraft
)

type DraftStatus = domain.DraftStatus

const (
	DraftStatusNone    = domain.DraftStatusNone
	DraftStatusPending = domain.DraftStatusPending
)

type PartyRole = domain.PartyRole

const (
	PartyRoleSender  = domain.PartyRoleSender
	PartyRoleHandler = domain.PartyRoleHandler
	PartyRoleAgent   = domain.PartyRoleAgent
)
