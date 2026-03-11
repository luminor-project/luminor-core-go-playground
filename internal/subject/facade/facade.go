package facade

import "fmt"

// ErrSubjectNotFound is returned when a subject ID is not recognized.
var ErrSubjectNotFound = fmt.Errorf("subject not found")

// SubjectInfoDTO holds subject data for cross-vertical communication.
type SubjectInfoDTO struct {
	ID     string
	Name   string
	Detail string
}

// CreateSubjectDTO holds data for creating a new subject.
type CreateSubjectDTO struct {
	Name               string
	Detail             string
	OwningOrgID        string
	CreatedByAccountID string
}
