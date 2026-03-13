package facade

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"

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
	CreatePasswordResetToken(ctx context.Context, accountID string) (domain.PasswordResetToken, string, error)
	ValidateAndConsumeToken(ctx context.Context, rawToken string) (string, error)
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
	RequestPasswordReset(ctx context.Context, dto PasswordResetRequestDTO) error
	CompletePasswordReset(ctx context.Context, dto PasswordResetCompletionDTO) error
} = (*facadeImpl)(nil)

type facadeImpl struct {
	service accountService
	bus     *eventbus.Bus
	outbox  outbox.Store
	baseURL string
}

// New creates a new account facade implementation.
func New(service accountService, bus *eventbus.Bus, outboxStore outbox.Store, baseURL string) *facadeImpl {
	return &facadeImpl{
		service: service,
		bus:     bus,
		outbox:  outboxStore,
		baseURL: baseURL,
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

// RequestPasswordReset initiates the flow (idempotent - no error if email not found).
func (f *facadeImpl) RequestPasswordReset(ctx context.Context, dto PasswordResetRequestDTO) error {
	// 1. Find account by email (silent return if not found - security)
	account, err := f.service.FindByEmail(ctx, dto.Email)
	if errors.Is(err, domain.ErrAccountNotFound) {
		return nil // Don't reveal email doesn't exist
	}
	if err != nil {
		slog.Error("password reset request: find account failed", "error", err)
		return fmt.Errorf("find account: %w", err)
	}

	// 2. Create token via domain (gets entity + raw token)
	token, rawToken, err := f.service.CreatePasswordResetToken(ctx, account.ID)
	if err != nil {
		slog.Error("password reset request: create token failed",
			"error", err,
			"account_id", account.ID)
		return fmt.Errorf("create token: %w", err)
	}

	// 3. Build reset URL with properly encoded token
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", f.baseURL, url.QueryEscape(rawToken))

	// 4. Publish event (email will be sent via subscriber or outbox)
	// Note: rawToken is NOT included in the event - only the pre-built URL contains it
	event := PasswordResetRequestedEvent{
		AccountID: account.ID,
		Email:     account.Email,
		ResetURL:  resetURL,
		TokenID:   token.ID,
	}
	if err := eventbus.Publish(ctx, f.bus, event); err != nil {
		// Fallback to outbox for reliability
		if f.outbox != nil {
			if outboxErr := f.outbox.Enqueue(ctx, outbox.EventTypePasswordResetRequestedV1, event); outboxErr != nil {
				slog.Error("password reset request: both publish and outbox failed",
					"error", err,
					"outbox_error", outboxErr,
					"account_id", account.ID)
			}
		} else {
			slog.Error("password reset request: publish failed and no outbox configured",
				"error", err,
				"account_id", account.ID)
		}
	}
	return nil
}

// CompletePasswordReset validates token and updates password.
// This operation is atomic - the token is consumed (validated + marked used) in one operation.
func (f *facadeImpl) CompletePasswordReset(ctx context.Context, dto PasswordResetCompletionDTO) error {
	// 1. Validate password length
	if len(dto.NewPassword) < 8 {
		return domain.ErrPasswordTooShort
	}

	// 2. Atomically validate and consume token (prevents race conditions)
	accountID, err := f.service.ValidateAndConsumeToken(ctx, dto.RawToken)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidResetToken) {
			slog.Warn("password reset completion: invalid or expired token",
				"error", err)
			return domain.ErrInvalidResetToken
		}
		slog.Error("password reset completion: token validation failed", "error", err)
		return fmt.Errorf("validate token: %w", err)
	}

	if accountID == "" {
		slog.Warn("password reset completion: token not found or already used")
		return domain.ErrInvalidResetToken
	}

	// 3. Update password
	if err := f.service.SetPassword(ctx, accountID, dto.NewPassword); err != nil {
		slog.Error("password reset completion: set password failed",
			"error", err,
			"account_id", accountID)
		return fmt.Errorf("set password: %w", err)
	}

	// 4. Publish completion event
	account, err := f.service.FindByID(ctx, accountID)
	if err != nil {
		slog.Error("password reset completion: find account for event failed",
			"error", err,
			"account_id", accountID)
	} else {
		event := PasswordResetCompletedEvent{
			AccountID: accountID,
			Email:     account.Email,
		}
		if pubErr := eventbus.Publish(ctx, f.bus, event); pubErr != nil {
			slog.Error("password reset completion: publish event failed",
				"error", pubErr,
				"account_id", accountID)
			// Fallback to outbox
			if f.outbox != nil {
				if outboxErr := f.outbox.Enqueue(ctx, outbox.EventTypePasswordResetCompletedV1, event); outboxErr != nil {
					slog.Error("password reset completion: both publish and outbox failed",
						"error", pubErr,
						"outbox_error", outboxErr,
						"account_id", accountID)
				}
			}
		}
	}

	return nil
}
