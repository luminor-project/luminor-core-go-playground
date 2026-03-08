package facade

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/luminor-project/luminor-core-go-playground/internal/account/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/outbox"
)

type accountService interface {
	Register(ctx context.Context, email, plainPassword string) (domain.AccountCore, error)
	Authenticate(ctx context.Context, email, plainPassword string) (domain.AccountCore, error)
	FindByEmail(ctx context.Context, email string) (domain.AccountCore, error)
	FindByID(ctx context.Context, id string) (domain.AccountCore, error)
	FindByIDs(ctx context.Context, ids []string) ([]domain.AccountCore, error)
	SetActiveOrganization(ctx context.Context, accountID, orgID string) error
	SetPassword(ctx context.Context, accountID, newPlainPassword string) error
}

// Compile-time interface assertion: facadeImpl satisfies all consumer interfaces.
var _ interface {
	Register(ctx context.Context, dto RegistrationDTO) (string, error)
	Authenticate(ctx context.Context, email, password string) (AccountInfoDTO, error)
	MustSetPassword(ctx context.Context, email string) (bool, error)
	GetAccountInfoByID(ctx context.Context, id string) (AccountInfoDTO, error)
	GetAccountInfoByIDs(ctx context.Context, ids []string) ([]AccountInfoDTO, error)
	GetAccountIDByEmail(ctx context.Context, email string) (string, error)
	AccountWithIDExists(ctx context.Context, accountID string) (bool, error)
	GetActiveOrgID(ctx context.Context, accountID string) (string, error)
	GetAccountEmailByID(ctx context.Context, accountID string) (string, error)
	SetActiveOrganization(ctx context.Context, accountID, orgID string) error
	SetPassword(ctx context.Context, accountID, newPassword string) error
} = (*facadeImpl)(nil)

type facadeImpl struct {
	service accountService
	bus     *eventbus.Bus
	outbox  outbox.Store
}

// New creates a new account facade implementation.
func New(service accountService, bus *eventbus.Bus, outboxStore outbox.Store) *facadeImpl {
	return &facadeImpl{
		service: service,
		bus:     bus,
		outbox:  outboxStore,
	}
}

func (f *facadeImpl) Register(ctx context.Context, dto RegistrationDTO) (string, error) {
	account, err := f.service.Register(ctx, dto.Email, dto.PlainPassword)
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyTaken) {
			return "", &domain.ValidationError{Key: "auth.validation.emailTaken"}
		}
		if errors.Is(err, domain.ErrPasswordTooShort) {
			return "", &domain.ValidationError{Key: "auth.validation.passwordTooShort"}
		}
		return "", fmt.Errorf("register: %w", err)
	}

	if err := eventbus.Publish(ctx, f.bus, AccountCreatedEvent{
		AccountID: account.ID,
		Email:     account.Email,
	}); err != nil {
		// Best-effort fallback for long-term reliability.
		// If synchronous dispatch fails, enqueue for worker retry.
		if f.outbox != nil {
			outboxErr := f.outbox.Enqueue(ctx, outbox.EventTypeAccountCreatedV1, AccountCreatedEvent{
				AccountID: account.ID,
				Email:     account.Email,
			})
			if outboxErr != nil {
				return "", fmt.Errorf("publish AccountCreatedEvent: %w (outbox enqueue failed: %v)", err, outboxErr)
			}
			slog.Warn("account created event publish failed; enqueued to outbox", "error", err, "account_id", account.ID)
		} else {
			return "", fmt.Errorf("publish AccountCreatedEvent: %w", err)
		}
	}

	return account.ID, nil
}

func (f *facadeImpl) Authenticate(ctx context.Context, email, password string) (AccountInfoDTO, error) {
	account, err := f.service.Authenticate(ctx, email, password)
	if err != nil {
		return AccountInfoDTO{}, err
	}
	return toAccountInfoDTO(account), nil
}

func (f *facadeImpl) GetAccountIDByEmail(ctx context.Context, email string) (string, error) {
	account, err := f.service.FindByEmail(ctx, email)
	if err != nil {
		return "", err
	}
	return account.ID, nil
}

func (f *facadeImpl) AccountWithIDExists(ctx context.Context, accountID string) (bool, error) {
	_, err := f.service.FindByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, domain.ErrAccountNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (f *facadeImpl) GetActiveOrgID(ctx context.Context, accountID string) (string, error) {
	account, err := f.service.FindByID(ctx, accountID)
	if err != nil {
		return "", err
	}
	return account.CurrentlyActiveOrganizationID, nil
}

func (f *facadeImpl) GetAccountEmailByID(ctx context.Context, accountID string) (string, error) {
	account, err := f.service.FindByID(ctx, accountID)
	if err != nil {
		return "", err
	}
	return account.Email, nil
}

func (f *facadeImpl) MustSetPassword(ctx context.Context, email string) (bool, error) {
	account, err := f.service.FindByEmail(ctx, email)
	if err != nil {
		return false, err
	}
	return account.MustSetPassword, nil
}

func (f *facadeImpl) GetAccountInfoByIDs(ctx context.Context, ids []string) ([]AccountInfoDTO, error) {
	accounts, err := f.service.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make([]AccountInfoDTO, len(accounts))
	for i, a := range accounts {
		result[i] = toAccountInfoDTO(a)
	}
	return result, nil
}

func (f *facadeImpl) GetAccountInfoByID(ctx context.Context, id string) (AccountInfoDTO, error) {
	account, err := f.service.FindByID(ctx, id)
	if err != nil {
		return AccountInfoDTO{}, err
	}
	return toAccountInfoDTO(account), nil
}

func (f *facadeImpl) SetActiveOrganization(ctx context.Context, accountID, orgID string) error {
	return f.service.SetActiveOrganization(ctx, accountID, orgID)
}

func (f *facadeImpl) SetPassword(ctx context.Context, accountID, newPassword string) error {
	return f.service.SetPassword(ctx, accountID, newPassword)
}

func toAccountInfoDTO(a domain.AccountCore) AccountInfoDTO {
	return AccountInfoDTO{
		ID:                            a.ID,
		Email:                         a.Email,
		Roles:                         a.RoleStrings(),
		CreatedAt:                     a.CreatedAt,
		CurrentlyActiveOrganizationID: a.CurrentlyActiveOrganizationID,
	}
}
