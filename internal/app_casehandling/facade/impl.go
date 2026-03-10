package facade

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/agentworkload"
	subjectfacade "github.com/luminor-project/luminor-core-go-playground/internal/subject/facade"
	workitemfacade "github.com/luminor-project/luminor-core-go-playground/internal/workitem/facade"
)

type workitemUseCases interface {
	IntakeInboundMessage(ctx context.Context, dto workitemfacade.IntakeInboundMessageDTO) (string, error)
	RecordAssistantAction(ctx context.Context, workItemID string, dto workitemfacade.RecordAssistantActionDTO) error
	ConfirmOutboundMessage(ctx context.Context, workItemID string, dto workitemfacade.ConfirmOutboundMessageDTO) error
	AddNote(ctx context.Context, workItemID string, dto workitemfacade.AddNoteDTO) (string, error)
	EditNote(ctx context.Context, workItemID string, dto workitemfacade.EditNoteDTO) error
	DeleteNote(ctx context.Context, workItemID string, dto workitemfacade.DeleteNoteDTO) error
}

type subjectLookup interface {
	GetSubjectInfo(ctx context.Context, subjectID string) (subjectfacade.SubjectInfoDTO, error)
}

// Compile-time interface assertion.
var _ interface {
	HandleInboundInquiry(ctx context.Context, dto InquiryDTO) (string, error)
	ConfirmAndSend(ctx context.Context, workItemID, operatorPartyID, body string) error
	AddNote(ctx context.Context, workItemID string, entryIndex int, authorID, body string) (string, error)
	EditNote(ctx context.Context, workItemID, noteID, body string) error
	DeleteNote(ctx context.Context, workItemID, noteID string) error
} = (*facadeImpl)(nil)

type facadeImpl struct {
	workitems workitemUseCases
	agent     agentworkload.Port
	subjects  subjectLookup
}

// New creates a new case handling facade.
func New(workitems workitemUseCases, agent agentworkload.Port, subjects subjectLookup) *facadeImpl {
	return &facadeImpl{
		workitems: workitems,
		agent:     agent,
		subjects:  subjects,
	}
}

// HandleInboundInquiry orchestrates the full intake + AI support flow:
// 1. Intake inbound message (creates work item)
// 2. Execute agent lookup
// 3. Record lookup action
// 4. Execute agent draft
// 5. Record draft action
func (f *facadeImpl) HandleInboundInquiry(ctx context.Context, dto InquiryDTO) (string, error) {
	// 1. Intake inbound message
	workItemID, err := f.workitems.IntakeInboundMessage(ctx, workitemfacade.IntakeInboundMessageDTO{
		SenderPartyID:  dto.SenderPartyID,
		SubjectID:      dto.SubjectID,
		Body:           dto.Body,
		HandlerPartyID: dto.OperatorPartyID,
		AgentPartyID:   dto.AgentPartyID,
	})
	if err != nil {
		return "", fmt.Errorf("intake inbound message: %w", err)
	}

	slog.Info("work item created", "work_item_id", workItemID)

	// Build context for agent lookup
	agentCtx := map[string]string{
		"subject_id": dto.SubjectID,
	}
	subjectInfo, err := f.subjects.GetSubjectInfo(ctx, dto.SubjectID)
	if err == nil {
		agentCtx["subject_name"] = subjectInfo.Name
		agentCtx["subject_detail"] = subjectInfo.Detail
	}

	// 2. Execute agent lookup
	lookupResult, err := f.agent.Execute(ctx, agentworkload.WorkloadRequest{
		WorkItemID: workItemID,
		ActionKind: "lookup",
		Context:    agentCtx,
	})
	if err != nil {
		return workItemID, fmt.Errorf("agent lookup: %w", err)
	}

	// 3. Record lookup action
	if err := f.workitems.RecordAssistantAction(ctx, workItemID, workitemfacade.RecordAssistantActionDTO{
		ActorID:     dto.AgentPartyID,
		ActionKind:  "lookup",
		Output:      lookupResult.Output,
		DraftStatus: "",
	}); err != nil {
		return workItemID, fmt.Errorf("record lookup action: %w", err)
	}

	// 4. Execute agent draft
	draftCtx := map[string]string{
		"lookup_result": lookupResult.Output,
		"inbound_body":  dto.Body,
	}
	draftResult, err := f.agent.Execute(ctx, agentworkload.WorkloadRequest{
		WorkItemID: workItemID,
		ActionKind: "draft",
		Context:    draftCtx,
	})
	if err != nil {
		return workItemID, fmt.Errorf("agent draft: %w", err)
	}

	// 5. Record draft action
	if err := f.workitems.RecordAssistantAction(ctx, workItemID, workitemfacade.RecordAssistantActionDTO{
		ActorID:     dto.AgentPartyID,
		ActionKind:  "draft",
		Output:      draftResult.Output,
		DraftStatus: "pending",
	}); err != nil {
		return workItemID, fmt.Errorf("record draft action: %w", err)
	}

	slog.Info("inbound inquiry handled", "work_item_id", workItemID)
	return workItemID, nil
}

// ConfirmAndSend confirms and sends the draft outbound message.
func (f *facadeImpl) ConfirmAndSend(ctx context.Context, workItemID, operatorPartyID, body string) error {
	return f.workitems.ConfirmOutboundMessage(ctx, workItemID, workitemfacade.ConfirmOutboundMessageDTO{
		ConfirmedByPartyID: operatorPartyID,
		Body:               body,
	})
}

// AddNote adds a note to a timeline entry.
func (f *facadeImpl) AddNote(ctx context.Context, workItemID string, entryIndex int, authorID, body string) (string, error) {
	return f.workitems.AddNote(ctx, workItemID, workitemfacade.AddNoteDTO{
		EntryIndex: entryIndex,
		AuthorID:   authorID,
		Body:       body,
	})
}

// EditNote edits an existing note.
func (f *facadeImpl) EditNote(ctx context.Context, workItemID, noteID, body string) error {
	return f.workitems.EditNote(ctx, workItemID, workitemfacade.EditNoteDTO{
		NoteID: noteID,
		Body:   body,
	})
}

// DeleteNote soft-deletes a note.
func (f *facadeImpl) DeleteNote(ctx context.Context, workItemID, noteID string) error {
	return f.workitems.DeleteNote(ctx, workItemID, workitemfacade.DeleteNoteDTO{
		NoteID: noteID,
	})
}
