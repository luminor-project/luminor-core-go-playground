package facade

import "context"

var demoSubjects = map[string]SubjectInfoDTO{
	"subject-flussufer-12a": {
		ID:     "subject-flussufer-12a",
		Name:   "Flussufer Apartments",
		Detail: "Unit 12A",
	},
}

type demoSubjectFacade struct{}

// NewDemoSubjectFacade creates a facade with hardcoded demo subject data.
func NewDemoSubjectFacade() *demoSubjectFacade {
	return &demoSubjectFacade{}
}

func (f *demoSubjectFacade) GetSubjectInfo(_ context.Context, subjectID string) (SubjectInfoDTO, error) {
	s, ok := demoSubjects[subjectID]
	if !ok {
		return SubjectInfoDTO{}, ErrSubjectNotFound
	}
	return s, nil
}
