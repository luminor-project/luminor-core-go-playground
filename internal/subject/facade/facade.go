package facade

import (
	"context"
	"fmt"
)

// ErrSubjectNotFound is returned when a subject ID is not recognized.
var ErrSubjectNotFound = fmt.Errorf("subject not found")

// SubjectInfoDTO holds subject data for cross-vertical communication.
type SubjectInfoDTO struct {
	ID     string
	Name   string
	Detail string
}

// SubjectFacade provides subject lookup operations.
type SubjectFacade interface {
	GetSubjectInfo(ctx context.Context, subjectID string) (SubjectInfoDTO, error)
}
