package facade

import "fmt"

// ErrSubjectNotFound is returned when a subject ID is not recognized.
var ErrSubjectNotFound = fmt.Errorf("subject not found")

// SubjectKind is the business classification of a subject.
type SubjectKind string

const (
	SubjectKindDwelling SubjectKind = "dwelling"
)

// SubjectInfoDTO holds subject data for cross-vertical communication.
type SubjectInfoDTO struct {
	ID          string
	SubjectKind SubjectKind
	Name        string
	Detail      string
}

// CreateSubjectDTO holds data for creating a new subject.
type CreateSubjectDTO struct {
	SubjectKind        SubjectKind
	Name               string
	Detail             string
	OwningOrgID        string
	CreatedByAccountID string
}
