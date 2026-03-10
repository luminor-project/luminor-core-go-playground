package facade

// IntakeInboundMessageDTO holds the data for creating a work item with an inbound message.
type IntakeInboundMessageDTO struct {
	SenderPartyID  string
	SubjectID      string
	Body           string
	HandlerPartyID string
	AgentPartyID   string
}

// RecordAssistantActionDTO holds the data for recording an AI assistant action.
type RecordAssistantActionDTO struct {
	ActorID     string
	ActionKind  string // "lookup", "draft"
	Output      string
	DraftStatus string // "" or "pending"
}

// ConfirmOutboundMessageDTO holds the data for confirming an outbound message.
type ConfirmOutboundMessageDTO struct {
	ConfirmedByPartyID string
	Body               string
}
