package facade_test

import (
	"context"
	"errors"
	"testing"

	"github.com/luminor-project/luminor-core-go-playground/internal/subject/facade"
)

func TestDemoSubjectFacade_KnownSubject(t *testing.T) {
	t.Parallel()
	f := facade.NewDemoSubjectFacade()
	ctx := context.Background()

	info, err := f.GetSubjectInfo(ctx, "subject-flussufer-12a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Name != "Flussufer Apartments" {
		t.Errorf("expected name %q, got %q", "Flussufer Apartments", info.Name)
	}
	if info.Detail != "Unit 12A" {
		t.Errorf("expected detail %q, got %q", "Unit 12A", info.Detail)
	}
}

func TestDemoSubjectFacade_UnknownSubject(t *testing.T) {
	t.Parallel()
	f := facade.NewDemoSubjectFacade()
	ctx := context.Background()

	_, err := f.GetSubjectInfo(ctx, "subject-unknown")
	if !errors.Is(err, facade.ErrSubjectNotFound) {
		t.Fatalf("expected ErrSubjectNotFound, got: %v", err)
	}
}
