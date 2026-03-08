package subscriber

import (
	"context"
	"errors"
	"strings"
	"testing"

	orgfacade "github.com/luminor-project/luminor-core-go-playground/internal/organization/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
)

func TestRegisterOrgChangedSubscriber_SetsActiveOrganization(t *testing.T) {
	t.Parallel()

	bus := eventbus.New()
	facade := &fakeAccountFacade{
		setActiveOrganizationFunc: func(_ context.Context, accountID, orgID string) error {
			if accountID != "user-1" || orgID != "org-1" {
				t.Fatalf("unexpected values account=%s org=%s", accountID, orgID)
			}
			return nil
		},
	}
	RegisterOrgChangedSubscriber(bus, facade)

	err := eventbus.Publish(context.Background(), bus, orgfacade.ActiveOrgChangedEvent{
		OrganizationID: "org-1",
		AffectedUserID: "user-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if facade.setActiveOrganizationCalls != 1 {
		t.Fatalf("expected SetActiveOrganization called once, got %d", facade.setActiveOrganizationCalls)
	}
}

func TestRegisterOrgChangedSubscriber_WrapsFacadeError(t *testing.T) {
	t.Parallel()

	bus := eventbus.New()
	facade := &fakeAccountFacade{
		setActiveOrganizationFunc: func(_ context.Context, _, _ string) error {
			return errors.New("write failed")
		},
	}
	RegisterOrgChangedSubscriber(bus, facade)

	err := eventbus.Publish(context.Background(), bus, orgfacade.ActiveOrgChangedEvent{
		OrganizationID: "org-2",
		AffectedUserID: "user-2",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "set active organization for account user-2") {
		t.Fatalf("expected wrapped context in error, got %q", err.Error())
	}
	if facade.setActiveOrganizationCalls != 1 {
		t.Fatalf("expected SetActiveOrganization called once, got %d", facade.setActiveOrganizationCalls)
	}
}

type fakeAccountFacade struct {
	setActiveOrganizationFunc  func(ctx context.Context, accountID, orgID string) error
	setActiveOrganizationCalls int
}

func (f *fakeAccountFacade) SetActiveOrganization(ctx context.Context, accountID, orgID string) error {
	f.setActiveOrganizationCalls++
	if f.setActiveOrganizationFunc == nil {
		return nil
	}
	return f.setActiveOrganizationFunc(ctx, accountID, orgID)
}
