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
	ActionKind  ActionKind
	Output      string
	DraftStatus DraftStatus
}

// ConfirmOutboundMessageDTO holds the data for confirming an outbound message.
type ConfirmOutboundMessageDTO struct {
	ConfirmedByPartyID string
	Body               string
}

// AddNoteDTO holds the data for adding a note to a timeline entry.
type AddNoteDTO struct {
	EntryIndex int
	AuthorID   string
	Body       string
}

// EditNoteDTO holds the data for editing a note.
type EditNoteDTO struct {
	NoteID string
	Body   string
}

// DeleteNoteDTO holds the data for soft-deleting a note.
type DeleteNoteDTO struct {
	NoteID string
}
