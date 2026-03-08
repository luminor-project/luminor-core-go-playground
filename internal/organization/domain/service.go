package domain

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrOrganizationNotFound             = errors.New("organization not found")
	ErrGroupNotFound                    = errors.New("group not found")
	ErrInvitationNotFound               = errors.New("invitation not found")
	ErrAlreadyMember                    = errors.New("user is already a member")
	ErrAlreadyInvited                   = errors.New("email already invited")
	ErrForbidden                        = errors.New("forbidden")
	ErrInvitationEmailMismatch          = errors.New("invitation email does not match authenticated account")
	ErrCrossOrganizationGroupAssignment = errors.New("member and group must belong to the same organization")
)

// Repository defines the persistence interface for the organization vertical.
type Repository interface {
	// Organizations
	CreateOrganization(ctx context.Context, org Organization) error
	FindOrganizationByID(ctx context.Context, id string) (Organization, error)
	UpdateOrganization(ctx context.Context, org Organization) error
	GetAllOrganizationsForUser(ctx context.Context, userID string) ([]Organization, error)
	UserIsOwnerOfOrganization(ctx context.Context, userID, orgID string) (bool, error)

	// Members
	AddMember(ctx context.Context, member OrganizationMember) error
	IsMember(ctx context.Context, accountCoreID, orgID string) (bool, error)
	GetMemberIDs(ctx context.Context, orgID string) ([]string, error)
	GetOwnerID(ctx context.Context, orgID string) (string, error)

	// Groups
	CreateGroup(ctx context.Context, group Group) error
	FindGroupByID(ctx context.Context, id string) (Group, error)
	GetGroups(ctx context.Context, orgID string) ([]Group, error)
	GetGroupsOfUser(ctx context.Context, userID, orgID string) ([]Group, error)
	AddUserToGroup(ctx context.Context, gm GroupMember) error
	RemoveUserFromGroup(ctx context.Context, accountCoreID, groupID string) error
	IsGroupMember(ctx context.Context, accountCoreID, groupID string) (bool, error)
	GetGroupMemberIDs(ctx context.Context, groupID string) ([]string, error)
	GetDefaultGroup(ctx context.Context, orgID string) (Group, error)

	// Invitations
	CreateInvitation(ctx context.Context, inv Invitation) error
	FindInvitationByID(ctx context.Context, id string) (Invitation, error)
	GetPendingInvitations(ctx context.Context, orgID string) ([]Invitation, error)
	InvitationExistsForEmail(ctx context.Context, orgID, email string) (bool, error)
	DeleteInvitation(ctx context.Context, id string) error
	ExecuteInTx(ctx context.Context, fn func(repo Repository) error) error
}

// OrgService handles core organization business logic.
type OrgService struct {
	repo Repository
}

// NewOrgService creates a new OrgService.
func NewOrgService(repo Repository) *OrgService {
	return &OrgService{repo: repo}
}

// CreateOrganization creates a new organization with default groups and adds the owner as admin.
func (s *OrgService) CreateOrganization(ctx context.Context, ownerID, name string) (Organization, error) {
	org := NewOrganization(ownerID, name)
	err := s.repo.ExecuteInTx(ctx, func(repo Repository) error {
		if err := repo.CreateOrganization(ctx, org); err != nil {
			return fmt.Errorf("create organization: %w", err)
		}

		// Create default groups
		adminsGroup := NewGroup(org.ID, "Administrators", AllAccessRights(), false)
		if err := repo.CreateGroup(ctx, adminsGroup); err != nil {
			return fmt.Errorf("create administrators group: %w", err)
		}

		teamGroup := NewGroup(org.ID, "Team Members", []AccessRight{}, true)
		if err := repo.CreateGroup(ctx, teamGroup); err != nil {
			return fmt.Errorf("create team members group: %w", err)
		}

		// Add owner as member and to administrators group
		if err := repo.AddMember(ctx, OrganizationMember{AccountCoreID: ownerID, OrganizationID: org.ID}); err != nil {
			return fmt.Errorf("add owner as member: %w", err)
		}

		if err := repo.AddUserToGroup(ctx, GroupMember{AccountCoreID: ownerID, GroupID: adminsGroup.ID}); err != nil {
			return fmt.Errorf("add owner to admins group: %w", err)
		}
		return nil
	})
	if err != nil {
		return Organization{}, err
	}
	return org, nil
}

// RenameOrganization renames an organization.
func (s *OrgService) RenameOrganization(ctx context.Context, orgID, newName string) error {
	org, err := s.repo.FindOrganizationByID(ctx, orgID)
	if err != nil {
		return err
	}
	org.Name = truncate(newName, 256)
	return s.repo.UpdateOrganization(ctx, org)
}

// RenameOrganizationAsActor renames an organization if the actor has permission.
func (s *OrgService) RenameOrganizationAsActor(ctx context.Context, actorUserID, orgID, newName string) error {
	hasRight, err := s.UserHasAccessRight(ctx, actorUserID, orgID, AccessRightEditOrganizationName)
	if err != nil {
		return fmt.Errorf("check edit organization name right: %w", err)
	}
	if !hasRight {
		return ErrForbidden
	}
	return s.RenameOrganization(ctx, orgID, newName)
}

// CreateInvitation creates an invitation for an email to join an organization.
func (s *OrgService) CreateInvitation(ctx context.Context, orgID, email string) (Invitation, error) {
	exists, err := s.repo.InvitationExistsForEmail(ctx, orgID, email)
	if err != nil {
		return Invitation{}, err
	}
	if exists {
		return Invitation{}, ErrAlreadyInvited
	}

	inv := NewInvitation(orgID, email)
	if err := s.repo.CreateInvitation(ctx, inv); err != nil {
		return Invitation{}, fmt.Errorf("create invitation: %w", err)
	}

	return inv, nil
}

// CreateInvitationAsActor creates an invitation if the actor has permission.
func (s *OrgService) CreateInvitationAsActor(ctx context.Context, actorUserID, orgID, email string) (Invitation, error) {
	hasRight, err := s.UserHasAccessRight(ctx, actorUserID, orgID, AccessRightInviteOrganizationMembers)
	if err != nil {
		return Invitation{}, fmt.Errorf("check invite members right: %w", err)
	}
	if !hasRight {
		return Invitation{}, ErrForbidden
	}
	return s.CreateInvitation(ctx, orgID, email)
}

// AcceptInvitation processes an invitation acceptance.
func (s *OrgService) AcceptInvitation(ctx context.Context, invitationID, accountCoreID, accountEmail string) (string, error) {
	var orgID string
	err := s.repo.ExecuteInTx(ctx, func(repo Repository) error {
		inv, err := repo.FindInvitationByID(ctx, invitationID)
		if err != nil {
			return err
		}

		if inv.Email != accountEmail {
			return ErrInvitationEmailMismatch
		}

		orgID = inv.OrganizationID

		// Check if already a member
		isMember, err := repo.IsMember(ctx, accountCoreID, inv.OrganizationID)
		if err != nil {
			return err
		}
		if isMember {
			if err := repo.DeleteInvitation(ctx, invitationID); err != nil {
				return fmt.Errorf("delete invitation for existing member: %w", err)
			}
			return nil
		}

		// Add as member
		if err := repo.AddMember(ctx, OrganizationMember{
			AccountCoreID:  accountCoreID,
			OrganizationID: inv.OrganizationID,
		}); err != nil {
			return fmt.Errorf("add member: %w", err)
		}

		// Add to default group
		defaultGroup, err := repo.GetDefaultGroup(ctx, inv.OrganizationID)
		if err != nil {
			return fmt.Errorf("get default group: %w", err)
		}
		if err := repo.AddUserToGroup(ctx, GroupMember{
			AccountCoreID: accountCoreID,
			GroupID:       defaultGroup.ID,
		}); err != nil {
			return fmt.Errorf("add member to default group: %w", err)
		}

		if err := repo.DeleteInvitation(ctx, invitationID); err != nil {
			return fmt.Errorf("delete invitation: %w", err)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return orgID, nil
}

// CanAccessOrganization returns whether a user is owner or member of an organization.
func (s *OrgService) CanAccessOrganization(ctx context.Context, userID, orgID string) (bool, error) {
	return s.repo.IsMember(ctx, userID, orgID)
}

// AddUserToGroupAsActor adds a user to a group if the actor has permission.
func (s *OrgService) AddUserToGroupAsActor(ctx context.Context, actorUserID, accountCoreID, groupID string) error {
	group, err := s.repo.FindGroupByID(ctx, groupID)
	if err != nil {
		return err
	}
	hasRight, err := s.UserHasAccessRight(ctx, actorUserID, group.OrganizationID, AccessRightMoveOrganizationMembersGroups)
	if err != nil {
		return fmt.Errorf("check move members right: %w", err)
	}
	if !hasRight {
		return ErrForbidden
	}
	isMember, err := s.repo.IsMember(ctx, accountCoreID, group.OrganizationID)
	if err != nil {
		return fmt.Errorf("check member in organization: %w", err)
	}
	if !isMember {
		return ErrCrossOrganizationGroupAssignment
	}
	return s.repo.AddUserToGroup(ctx, GroupMember{AccountCoreID: accountCoreID, GroupID: groupID})
}

// RemoveUserFromGroupAsActor removes a user from a group if the actor has permission.
func (s *OrgService) RemoveUserFromGroupAsActor(ctx context.Context, actorUserID, accountCoreID, groupID string) error {
	group, err := s.repo.FindGroupByID(ctx, groupID)
	if err != nil {
		return err
	}
	hasRight, err := s.UserHasAccessRight(ctx, actorUserID, group.OrganizationID, AccessRightMoveOrganizationMembersGroups)
	if err != nil {
		return fmt.Errorf("check move members right: %w", err)
	}
	if !hasRight {
		return ErrForbidden
	}
	return s.repo.RemoveUserFromGroup(ctx, accountCoreID, groupID)
}

// UserHasAccessRight checks if a user has a specific access right in an organization.
func (s *OrgService) UserHasAccessRight(ctx context.Context, userID, orgID string, right AccessRight) (bool, error) {
	// Organization owner has all rights
	isOwner, err := s.repo.UserIsOwnerOfOrganization(ctx, userID, orgID)
	if err != nil {
		return false, err
	}
	if isOwner {
		return true, nil
	}

	// Check group membership
	groups, err := s.repo.GetGroupsOfUser(ctx, userID, orgID)
	if err != nil {
		return false, err
	}

	for _, g := range groups {
		if g.HasAccessRight(right) {
			return true, nil
		}
	}

	return false, nil
}

// GetOrganizationByID returns an organization by ID.
func (s *OrgService) GetOrganizationByID(ctx context.Context, id string) (Organization, error) {
	return s.repo.FindOrganizationByID(ctx, id)
}

// GetAllOrganizationsForUser returns all organizations a user owns or is a member of.
func (s *OrgService) GetAllOrganizationsForUser(ctx context.Context, userID string) ([]Organization, error) {
	return s.repo.GetAllOrganizationsForUser(ctx, userID)
}

// GetGroups returns all groups in an organization.
func (s *OrgService) GetGroups(ctx context.Context, orgID string) ([]Group, error) {
	return s.repo.GetGroups(ctx, orgID)
}

// GetGroupsOfUser returns groups a user belongs to in an organization.
func (s *OrgService) GetGroupsOfUser(ctx context.Context, userID, orgID string) ([]Group, error) {
	return s.repo.GetGroupsOfUser(ctx, userID, orgID)
}

// GetMemberIDs returns all member IDs for an organization.
func (s *OrgService) GetMemberIDs(ctx context.Context, orgID string) ([]string, error) {
	ownerID, err := s.repo.GetOwnerID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	joinedIDs, err := s.repo.GetMemberIDs(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Combine owner + joined members, deduplicate
	idSet := make(map[string]bool)
	idSet[ownerID] = true
	for _, id := range joinedIDs {
		idSet[id] = true
	}

	var result []string
	for id := range idSet {
		result = append(result, id)
	}
	return result, nil
}

// GetGroupMemberIDs returns member IDs for a specific group.
func (s *OrgService) GetGroupMemberIDs(ctx context.Context, groupID string) ([]string, error) {
	return s.repo.GetGroupMemberIDs(ctx, groupID)
}

// GetPendingInvitations returns pending invitations for an organization.
func (s *OrgService) GetPendingInvitations(ctx context.Context, orgID string) ([]Invitation, error) {
	return s.repo.GetPendingInvitations(ctx, orgID)
}

// AddUserToGroup adds a user to a group.
func (s *OrgService) AddUserToGroup(ctx context.Context, accountCoreID, groupID string) error {
	return s.repo.AddUserToGroup(ctx, GroupMember{AccountCoreID: accountCoreID, GroupID: groupID})
}

// RemoveUserFromGroup removes a user from a group.
func (s *OrgService) RemoveUserFromGroup(ctx context.Context, accountCoreID, groupID string) error {
	return s.repo.RemoveUserFromGroup(ctx, accountCoreID, groupID)
}

// FindInvitationByID returns an invitation by ID.
func (s *OrgService) FindInvitationByID(ctx context.Context, id string) (Invitation, error) {
	return s.repo.FindInvitationByID(ctx, id)
}

// IsOwner checks if a user owns an organization.
func (s *OrgService) IsOwner(ctx context.Context, userID, orgID string) (bool, error) {
	return s.repo.UserIsOwnerOfOrganization(ctx, userID, orgID)
}

// GetOrganizationName returns the organization name (may be empty).
func (s *OrgService) GetOrganizationName(ctx context.Context, orgID string) (string, error) {
	org, err := s.repo.FindOrganizationByID(ctx, orgID)
	if err != nil {
		return "", err
	}
	return org.Name, nil
}
