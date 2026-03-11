package facade

import (
	"context"
	"errors"

	"github.com/luminor-project/luminor-core-go-playground/internal/subject/domain"
)

type subjectService interface {
	CreateSubject(ctx context.Context, name, detail, orgID, createdByAccountID string) (domain.Subject, error)
	FindByID(ctx context.Context, id string) (domain.Subject, error)
	FindByIDs(ctx context.Context, ids []string) ([]domain.Subject, error)
	FindByOrganizationID(ctx context.Context, orgID string) ([]domain.Subject, error)
}

// Compile-time interface assertion.
var _ interface {
	GetSubjectInfo(ctx context.Context, subjectID string) (SubjectInfoDTO, error)
	CreateSubject(ctx context.Context, dto CreateSubjectDTO) (string, error)
	ListSubjectsByOrg(ctx context.Context, orgID string) ([]SubjectInfoDTO, error)
	GetSubjectsByIDs(ctx context.Context, ids []string) ([]SubjectInfoDTO, error)
} = (*facadeImpl)(nil)

type facadeImpl struct {
	service subjectService
}

// New creates a new subject facade.
func New(service subjectService) *facadeImpl {
	return &facadeImpl{service: service}
}

func (f *facadeImpl) GetSubjectInfo(ctx context.Context, subjectID string) (SubjectInfoDTO, error) {
	s, err := f.service.FindByID(ctx, subjectID)
	if err != nil {
		if errors.Is(err, domain.ErrSubjectNotFound) {
			return SubjectInfoDTO{}, ErrSubjectNotFound
		}
		return SubjectInfoDTO{}, err
	}
	return toDTO(s), nil
}

func (f *facadeImpl) CreateSubject(ctx context.Context, dto CreateSubjectDTO) (string, error) {
	s, err := f.service.CreateSubject(ctx, dto.Name, dto.Detail, dto.OwningOrgID, dto.CreatedByAccountID)
	if err != nil {
		return "", err
	}
	return s.ID, nil
}

func (f *facadeImpl) ListSubjectsByOrg(ctx context.Context, orgID string) ([]SubjectInfoDTO, error) {
	subjects, err := f.service.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return toDTOs(subjects), nil
}

func (f *facadeImpl) GetSubjectsByIDs(ctx context.Context, ids []string) ([]SubjectInfoDTO, error) {
	subjects, err := f.service.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return toDTOs(subjects), nil
}

func toDTO(s domain.Subject) SubjectInfoDTO {
	return SubjectInfoDTO{
		ID:     s.ID,
		Name:   s.Name,
		Detail: s.Detail,
	}
}

func toDTOs(subjects []domain.Subject) []SubjectInfoDTO {
	result := make([]SubjectInfoDTO, len(subjects))
	for i, s := range subjects {
		result[i] = toDTO(s)
	}
	return result
}
