package domain

// Status represents the lifecycle state of a work item.
type Status string

const (
	StatusNew                 Status = "new"
	StatusInProgress          Status = "in_progress"
	StatusPendingConfirmation Status = "pending_confirmation"
	StatusResolved            Status = "resolved"
)

// ActionKind represents the type of action an AI assistant performs.
type ActionKind string

const (
	ActionKindLookup ActionKind = "lookup"
	ActionKindDraft  ActionKind = "draft"
)

// DraftStatus indicates whether an assistant action produced a pending draft.
type DraftStatus string

const (
	DraftStatusNone    DraftStatus = ""
	DraftStatusPending DraftStatus = "pending"
)

// PartyRole describes a party's relationship to a work item.
type PartyRole string

const (
	PartyRoleSender  PartyRole = "sender"
	PartyRoleHandler PartyRole = "handler"
	PartyRoleAgent   PartyRole = "agent"
)
