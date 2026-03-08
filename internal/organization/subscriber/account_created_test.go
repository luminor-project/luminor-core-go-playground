package subscriber

import (
	"context"
	"errors"
	"strings"
	"testing"

	accountfacade "github.com/luminor-project/luminor-core-go-playground/internal/account/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
)

func TestRegisterAccountCreatedSubscriber_CallsCreateDefaultOrg(t *testing.T) {
	t.Parallel()

	bus := eventbus.New()
	facade := &fakeOrgFacade{
		createDefaultOrgFunc: func(_ context.Context, accountID string) error {
			if accountID != "acct-1" {
				t.Fatalf("unexpected account id: %s", accountID)
			}
			return nil
		},
	}
	RegisterAccountCreatedSubscriber(bus, facade)

	err := eventbus.Publish(context.Background(), bus, accountfacade.AccountCreatedEvent{
		AccountID: "acct-1",
		Email:     "user@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if facade.createDefaultOrgCalls != 1 {
		t.Fatalf("expected CreateDefaultOrg called once, got %d", facade.createDefaultOrgCalls)
	}
}

func TestRegisterAccountCreatedSubscriber_WrapsFacadeError(t *testing.T) {
	t.Parallel()

	bus := eventbus.New()
	facade := &fakeOrgFacade{
		createDefaultOrgFunc: func(_ context.Context, _ string) error {
			return errors.New("db unavailable")
		},
	}
	RegisterAccountCreatedSubscriber(bus, facade)

	err := eventbus.Publish(context.Background(), bus, accountfacade.AccountCreatedEvent{
		AccountID: "acct-2",
		Email:     "user2@example.com",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create default org for account acct-2") {
		t.Fatalf("expected wrapped context in error, got %q", err.Error())
	}
	if facade.createDefaultOrgCalls != 1 {
		t.Fatalf("expected CreateDefaultOrg called once, got %d", facade.createDefaultOrgCalls)
	}
}

type fakeOrgFacade struct {
	createDefaultOrgFunc  func(ctx context.Context, accountID string) error
	createDefaultOrgCalls int
}

func (f *fakeOrgFacade) CreateDefaultOrg(ctx context.Context, accountID string) error {
	f.createDefaultOrgCalls++
	if f.createDefaultOrgFunc == nil {
		return nil
	}
	return f.createDefaultOrgFunc(ctx, accountID)
}
