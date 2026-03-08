package web

import (
	"context"
	"fmt"

	orgfacade "github.com/luminor-project/luminor-core-go-playground/internal/organization/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/organization/web/templates"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/i18n"

	"github.com/luminor-project/luminor-core-go-playground/internal/organization/domain"
)

// DashboardLoader encapsulates the data-fetching logic for the organization dashboard.
type DashboardLoader struct {
	orgService orgUseCases
	orgFacade  orgNameProvider
	acctFacade accountOrgUseCases
}

// NewDashboardLoader creates a new DashboardLoader.
func NewDashboardLoader(orgService orgUseCases, orgFacade orgNameProvider, acctFacade accountOrgUseCases) *DashboardLoader {
	return &DashboardLoader{orgService: orgService, orgFacade: orgFacade, acctFacade: acctFacade}
}

// Load fetches all data needed to render the organization dashboard.
// It does not set CSRFToken on the returned data; the caller is responsible for that.
func (l *DashboardLoader) Load(ctx context.Context, userID string) (templates.DashboardData, error) {
	var data templates.DashboardData

	activeOrgID, err := l.acctFacade.GetActiveOrgID(ctx, userID)
	if err != nil {
		return data, fmt.Errorf("get active org: %w", err)
	}

	allOrgs, err := l.orgService.GetAllOrganizationsForUser(ctx, userID)
	if err != nil {
		return data, fmt.Errorf("get organizations: %w", err)
	}

	var orgDTOs []orgfacade.OrganizationDTO
	for _, o := range allOrgs {
		name := o.Name
		if name == "" {
			name = i18n.T(ctx, "organization.fallbackName")
		}
		orgDTOs = append(orgDTOs, orgfacade.OrganizationDTO{
			ID:       o.ID,
			Name:     name,
			IsOwned:  o.OwningUsersID == userID,
			IsActive: o.ID == activeOrgID,
		})
	}
	data.Organizations = orgDTOs

	if activeOrgID == "" {
		return data, nil
	}

	orgName, err := l.orgFacade.GetOrganizationNameByID(ctx, activeOrgID)
	if err != nil {
		return data, fmt.Errorf("get organization name: %w", err)
	}
	if orgName == "" {
		orgName = i18n.T(ctx, "organization.fallbackName")
	}
	data.OrganizationName = orgName

	org, err := l.orgService.GetOrganizationByID(ctx, activeOrgID)
	if err != nil {
		return data, fmt.Errorf("get active organization: %w", err)
	}
	data.CurrentOrganizationRawName = org.Name

	isOwner, err := l.orgService.IsOwner(ctx, userID, activeOrgID)
	if err != nil {
		return data, fmt.Errorf("check ownership: %w", err)
	}
	data.IsOwner = isOwner

	canRename, err := l.orgService.UserHasAccessRight(ctx, userID, activeOrgID, domain.AccessRightEditOrganizationName)
	if err != nil {
		return data, fmt.Errorf("check rename right: %w", err)
	}
	data.CanRename = canRename

	canInvite, err := l.orgService.UserHasAccessRight(ctx, userID, activeOrgID, domain.AccessRightInviteOrganizationMembers)
	if err != nil {
		return data, fmt.Errorf("check invite right: %w", err)
	}
	data.CanInvite = canInvite

	if err := l.loadMembersAndInvitations(ctx, &data, org, userID); err != nil {
		return data, err
	}

	return data, nil
}

func (l *DashboardLoader) loadMembersAndInvitations(
	ctx context.Context, data *templates.DashboardData, org domain.Organization, userID string,
) error {
	memberIDs, err := l.orgService.GetMemberIDs(ctx, org.ID)
	if err != nil {
		return fmt.Errorf("get member IDs: %w", err)
	}

	if len(memberIDs) > 0 {
		acctInfos, err := l.acctFacade.GetAccountInfoByIDs(ctx, memberIDs)
		if err != nil {
			return fmt.Errorf("get account info for members: %w", err)
		}

		orgGroups, err := l.orgService.GetGroups(ctx, org.ID)
		if err != nil {
			return fmt.Errorf("get organization groups: %w", err)
		}
		groupMemberMap := make(map[string][]string)
		for _, g := range orgGroups {
			gMembers, err := l.orgService.GetGroupMemberIDs(ctx, g.ID)
			if err != nil {
				return fmt.Errorf("get group members for group %s: %w", g.ID, err)
			}
			groupMemberMap[g.ID] = gMembers
		}

		var members []orgfacade.MemberDTO
		for _, ai := range acctInfos {
			var memberGroupIDs []string
			for gID, gMembers := range groupMemberMap {
				for _, mid := range gMembers {
					if mid == ai.ID {
						memberGroupIDs = append(memberGroupIDs, gID)
						break
					}
				}
			}

			members = append(members, orgfacade.MemberDTO{
				ID:            ai.ID,
				DisplayName:   ai.DisplayName(),
				Email:         ai.Email,
				IsOwner:       ai.ID == org.OwningUsersID,
				IsCurrentUser: ai.ID == userID,
				GroupIDs:      memberGroupIDs,
			})
		}
		data.Members = members

		var groups []orgfacade.GroupDTO
		for _, g := range orgGroups {
			groups = append(groups, orgfacade.GroupDTO{
				ID:        g.ID,
				Name:      g.Name,
				IsDefault: g.IsDefaultForNewMembers,
			})
		}
		data.Groups = groups
	}

	pendingInvs, err := l.orgService.GetPendingInvitations(ctx, org.ID)
	if err != nil {
		return fmt.Errorf("get pending invitations: %w", err)
	}
	var invitations []orgfacade.InvitationDTO
	for _, inv := range pendingInvs {
		invitations = append(invitations, orgfacade.InvitationDTO{
			ID:        inv.ID,
			Email:     inv.Email,
			CreatedAt: inv.CreatedAt,
		})
	}
	data.PendingInvitations = invitations

	return nil
}
