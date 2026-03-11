package subscriber

import (
	"context"
	"fmt"
	"log/slog"

	accountfacade "github.com/luminor-project/luminor-core-go-playground/internal/account/facade"
	orgfacade "github.com/luminor-project/luminor-core-go-playground/internal/organization/facade"
	partyfacade "github.com/luminor-project/luminor-core-go-playground/internal/party/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
)

type partyCreator interface {
	CreateParty(ctx context.Context, dto partyfacade.CreatePartyDTO) (string, error)
}

type accountPartyLinker interface {
	LinkPartyToAccount(ctx context.Context, accountID, partyID, orgID string) error
	GetPartyMembershipsForAccount(ctx context.Context, accountID, orgID string) ([]accountfacade.PartyMembershipDTO, error)
	SetActiveParty(ctx context.Context, accountID, partyID string) error
	GetAccountInfoByID(ctx context.Context, id string) (accountfacade.AccountInfoDTO, error)
}

// RegisterAccountJoinedOrgSubscriber subscribes to AccountJoinedOrgEvent
// and auto-creates a property_manager party for the account, unless one already exists.
func RegisterAccountJoinedOrgSubscriber(bus *eventbus.Bus, partyFac partyCreator, acctFac accountPartyLinker) {
	eventbus.Subscribe(bus, func(ctx context.Context, e orgfacade.AccountJoinedOrgEvent) error {
		slog.Info("handling AccountJoinedOrgEvent — checking party membership",
			"account_id", e.AccountID, "org_id", e.OrgID,
		)

		// Check if the account already has a party in this org.
		memberships, err := acctFac.GetPartyMembershipsForAccount(ctx, e.AccountID, e.OrgID)
		if err != nil {
			return fmt.Errorf("check party memberships for account %s: %w", e.AccountID, err)
		}
		if len(memberships) > 0 {
			slog.Info("account already has party in org; skipping auto-create",
				"account_id", e.AccountID, "org_id", e.OrgID,
			)
			return nil
		}

		// Resolve the display name for the party.
		name := e.Email
		if name == "" {
			info, err := acctFac.GetAccountInfoByID(ctx, e.AccountID)
			if err != nil {
				return fmt.Errorf("get account info for %s: %w", e.AccountID, err)
			}
			name = info.Email
		}

		// Create a property_manager party for this account.
		partyID, err := partyFac.CreateParty(ctx, partyfacade.CreatePartyDTO{
			Name:               name,
			ActorKind:          partyfacade.ActorKindHuman,
			PartyKind:          partyfacade.PartyKindPropertyManager,
			OwningOrgID:        e.OrgID,
			CreatedByAccountID: e.AccountID,
		})
		if err != nil {
			return fmt.Errorf("create PM party for account %s: %w", e.AccountID, err)
		}

		// Link the party to the account.
		if err := acctFac.LinkPartyToAccount(ctx, e.AccountID, partyID, e.OrgID); err != nil {
			return fmt.Errorf("link party %s to account %s: %w", partyID, e.AccountID, err)
		}

		// Set it as the active party.
		if err := acctFac.SetActiveParty(ctx, e.AccountID, partyID); err != nil {
			return fmt.Errorf("set active party %s for account %s: %w", partyID, e.AccountID, err)
		}

		slog.Info("auto-created PM party for account",
			"account_id", e.AccountID, "party_id", partyID, "org_id", e.OrgID,
		)
		return nil
	})
}
