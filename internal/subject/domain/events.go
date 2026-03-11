package domain

import "time"

const (
	EventSubjectRegistered = "subject.SubjectRegistered.v1"
)

type DomainEvent struct {
	EventType string
	Payload   any
}

type SubjectRegistered struct {
	SubjectID          string      `json:"subject_id"`
	SubjectKind        SubjectKind `json:"subject_kind"`
	Name               string      `json:"name"`
	Detail             string      `json:"detail"`
	OrgID              string      `json:"org_id"`
	CreatedByAccountID string      `json:"created_by_account_id"`
	RegisteredAt       time.Time   `json:"registered_at"`
}
