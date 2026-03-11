package domain

import (
	"context"
	"fmt"
)

// Repository defines the persistence interface for subjects.
type Repository interface {
	Create(ctx context.Context, subject Subject) error
	FindByID(ctx context.Context, id string) (Subject, error)
	FindByIDs(ctx context.Context, ids []string) ([]Subject, error)
	FindByOrganizationID(ctx context.Context, orgID string) ([]Subject, error)
}

// SubjectService handles subject business logic.
type SubjectService struct {
	repo  Repository
	clock Clock
}

// NewSubjectService creates a new SubjectService.
func NewSubjectService(repo Repository, clock Clock) *SubjectService {
	return &SubjectService{repo: repo, clock: clock}
}

// CreateSubject validates inputs and persists a new subject.
func (s *SubjectService) CreateSubject(ctx context.Context, name, detail, orgID, createdByAccountID string) (Subject, error) {
	subject, err := NewSubject(name, detail, orgID, createdByAccountID, s.clock.Now())
	if err != nil {
		return Subject{}, err
	}

	if err := s.repo.Create(ctx, subject); err != nil {
		return Subject{}, fmt.Errorf("create subject: %w", err)
	}

	return subject, nil
}

// FindByID returns a subject by ID.
func (s *SubjectService) FindByID(ctx context.Context, id string) (Subject, error) {
	return s.repo.FindByID(ctx, id)
}

// FindByIDs returns subjects by IDs.
func (s *SubjectService) FindByIDs(ctx context.Context, ids []string) ([]Subject, error) {
	return s.repo.FindByIDs(ctx, ids)
}

// FindByOrganizationID returns all subjects belonging to an organization.
func (s *SubjectService) FindByOrganizationID(ctx context.Context, orgID string) ([]Subject, error) {
	return s.repo.FindByOrganizationID(ctx, orgID)
}
