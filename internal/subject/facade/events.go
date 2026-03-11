package facade

import "time"

type SubjectRegisteredEvent struct {
	SubjectID          string
	SubjectKind        SubjectKind
	Name               string
	Detail             string
	OrgID              string
	CreatedByAccountID string
	RegisteredAt       time.Time
}
