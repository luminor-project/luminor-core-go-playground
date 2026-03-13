package facade

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

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
	SetActiveParty(ctx context.Context, accountID, partyID string) error
	LinkPartyToAccount(ctx context.Context, accountID, partyID, orgID string) error
	GetPartyMembershipsForAccount(ctx context.Context, accountID, orgID string) ([]domain.PartyMembership, error)
	GetAccountIDsForParty(ctx context.Context, partyID string) ([]string, error)
	CreatePendingPartyLink(ctx context.Context, invitationID, partyID, orgID string) (domain.PendingPartyLink, error)
	ResolvePendingPartyLink(ctx context.Context, invitationID, accountID string) error
	RequestMagicLink(ctx context.Context, accountID string) (domain.MagicLinkToken, string, error)
	VerifyMagicLink(ctx context.Context, rawToken string) (string, error)
}

// Compile-time interface assertion: facadeImpl satisfies all consumer interfaces.
var _ interface {
	Register(ctx context.Context, dto RegistrationDTO) (string, error)
	Authenticate(ctx context.Context, email, password string) (AccountInfoDTO, error)
	MustSetPassword(ctx context.Context, email string) (bool, error)
	GetAccountInfoByID(ctx context.Context, id string) (AccountInfoDTO, error)
	GetAccountInfoByIDs(ctx context.Context, ids []string) ([]AccountInfoDTO, error)
	GetActiveOrgID(ctx context.Context, accountID string) (string, error)
	GetAccountEmailByID(ctx context.Context, accountID string) (string, error)
	SetActiveOrganization(ctx context.Context, accountID, orgID string) error
	SetPassword(ctx context.Context, accountID, newPassword string) error
	SetActiveParty(ctx context.Context, accountID, partyID string) error
	LinkPartyToAccount(ctx context.Context, accountID, partyID, orgID string) error
	GetPartyMembershipsForAccount(ctx context.Context, accountID, orgID string) ([]PartyMembershipDTO, error)
	GetAccountIDsForParty(ctx context.Context, partyID string) ([]string, error)
	CreatePendingPartyLink(ctx context.Context, invitationID, partyID, orgID string) (string, error)
	ResolvePendingPartyLink(ctx context.Context, invitationID, accountID string) error
	RequestMagicLink(ctx context.Context, dto MagicLinkRequestDTO) error
	VerifyMagicLink(ctx context.Context, rawToken string) (MagicLinkResultDTO, error)
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

func (f *facadeImpl) SetActiveParty(ctx context.Context, accountID, partyID string) error {
	return f.service.SetActiveParty(ctx, accountID, partyID)
}

func (f *facadeImpl) LinkPartyToAccount(ctx context.Context, accountID, partyID, orgID string) error {
	err := f.service.LinkPartyToAccount(ctx, accountID, partyID, orgID)
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyLinked) {
			return ErrAlreadyLinked
		}
		return err
	}
	return nil
}

func (f *facadeImpl) GetPartyMembershipsForAccount(ctx context.Context, accountID, orgID string) ([]PartyMembershipDTO, error) {
	memberships, err := f.service.GetPartyMembershipsForAccount(ctx, accountID, orgID)
	if err != nil {
		return nil, err
	}
	result := make([]PartyMembershipDTO, len(memberships))
	for i, m := range memberships {
		result[i] = PartyMembershipDTO{
			AccountID: m.AccountID,
			PartyID:   m.PartyID,
			OrgID:     m.OrgID,
			CreatedAt: m.CreatedAt,
		}
	}
	return result, nil
}

func (f *facadeImpl) GetAccountIDsForParty(ctx context.Context, partyID string) ([]string, error) {
	return f.service.GetAccountIDsForParty(ctx, partyID)
}

func (f *facadeImpl) CreatePendingPartyLink(ctx context.Context, invitationID, partyID, orgID string) (string, error) {
	link, err := f.service.CreatePendingPartyLink(ctx, invitationID, partyID, orgID)
	if err != nil {
		return "", err
	}
	return link.ID, nil
}

func (f *facadeImpl) ResolvePendingPartyLink(ctx context.Context, invitationID, accountID string) error {
	err := f.service.ResolvePendingPartyLink(ctx, invitationID, accountID)
	if err != nil {
		if errors.Is(err, domain.ErrPendingLinkNotFound) {
			return ErrPendingLinkNotFound
		}
		return err
	}
	return nil
}

func (f *facadeImpl) RequestMagicLink(ctx context.Context, dto MagicLinkRequestDTO) error {
	// Find account - if not found, still return success to prevent email enumeration
	account, err := f.service.FindByEmail(ctx, dto.Email)
	if err != nil {
		if errors.Is(err, domain.ErrAccountNotFound) {
			slog.Info("magic link requested for non-existent account", "email", dto.Email)
			return nil
		}
		return fmt.Errorf("find account: %w", err)
	}

	// Request magic link (includes rate limiting)
	token, rawToken, err := f.service.RequestMagicLink(ctx, account.ID)
	if err != nil {
		if errors.Is(err, domain.ErrMagicLinkRateLimitExceeded) {
			slog.Warn("magic link rate limit exceeded", "account_id", account.ID)
			// Still return nil to prevent enumeration, but email won't be sent
			return nil
		}
		return fmt.Errorf("request magic link: %w", err)
	}

	// Publish event for email sending
	event := MagicLinkRequestedEvent{
		AccountID: account.ID,
		Email:     account.Email,
		RawToken:  rawToken,
		ExpiresAt: token.ExpiresAt,
	}

	if err := eventbus.Publish(ctx, f.bus, event); err != nil {
		// Fallback to outbox for reliability
		if f.outbox != nil {
			outboxErr := f.outbox.Enqueue(ctx, outbox.EventTypeMagicLinkRequestedV1, event)
			if outboxErr != nil {
				return fmt.Errorf("publish MagicLinkRequestedEvent: %w (outbox enqueue failed: %v)", err, outboxErr)
			}
			slog.Warn("magic link requested event publish failed; enqueued to outbox", "error", err, "account_id", account.ID)
		} else {
			return fmt.Errorf("publish MagicLinkRequestedEvent: %w", err)
		}
	}

	return nil
}

func (f *facadeImpl) VerifyMagicLink(ctx context.Context, rawToken string) (MagicLinkResultDTO, error) {
	accountID, err := f.service.VerifyMagicLink(ctx, rawToken)
	if err != nil {
		if errors.Is(err, domain.ErrMagicLinkTokenExpired) {
			return MagicLinkResultDTO{}, err
		}
		return MagicLinkResultDTO{}, fmt.Errorf("verify magic link: %w", err)
	}

	// Get account info
	account, err := f.service.FindByID(ctx, accountID)
	if err != nil {
		return MagicLinkResultDTO{}, fmt.Errorf("find account: %w", err)
	}

	// Publish event for audit logging
	_ = eventbus.Publish(ctx, f.bus, MagicLinkUsedEvent{
		AccountID: account.ID,
		Email:     account.Email,
		UsedAt:    time.Now(),
	})

	return MagicLinkResultDTO{
		AccountID: account.ID,
		Email:     account.Email,
		Roles:     account.RoleStrings(),
	}, nil
}

func toAccountInfoDTO(a domain.AccountCore) AccountInfoDTO {
	return AccountInfoDTO{
		ID:                            a.ID,
		Email:                         a.Email,
		Roles:                         a.RoleStrings(),
		CreatedAt:                     a.CreatedAt,
		CurrentlyActiveOrganizationID: a.CurrentlyActiveOrganizationID,
		CurrentlyActivePartyID:        a.CurrentlyActivePartyID,
	}
}
