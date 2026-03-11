package domain_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/organization/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
)

var testClock = clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))

// mockRepository implements domain.Repository for testing.
type mockRepository struct {
	orgs         map[string]domain.Organization
	members      map[string][]string // orgID -> []accountCoreID
	groups       map[string]domain.Group
	groupMembers map[string][]string // groupID -> []accountCoreID
	invitations  map[string]domain.Invitation
}

func newMockRepo() *mockRepository {
	return &mockRepository{
		orgs:         make(map[string]domain.Organization),
		members:      make(map[string][]string),
		groups:       make(map[string]domain.Group),
		groupMembers: make(map[string][]string),
		invitations:  make(map[string]domain.Invitation),
	}
}

func (r *mockRepository) CreateOrganization(_ context.Context, org domain.Organization) error {
	r.orgs[org.ID] = org
	return nil
}

func (r *mockRepository) FindOrganizationByID(_ context.Context, id string) (domain.Organization, error) {
	org, ok := r.orgs[id]
	if !ok {
		return domain.Organization{}, domain.ErrOrganizationNotFound
	}
	return org, nil
}

func (r *mockRepository) UpdateOrganization(_ context.Context, org domain.Organization) error {
	r.orgs[org.ID] = org
	return nil
}

func (r *mockRepository) GetAllOrganizationsForUser(_ context.Context, userID string) ([]domain.Organization, error) {
	var result []domain.Organization
	for _, org := range r.orgs {
		if org.OwningUsersID == userID {
			result = append(result, org)
			continue
		}
		for _, mid := range r.members[org.ID] {
			if mid == userID {
				result = append(result, org)
				break
			}
		}
	}
	return result, nil
}

func (r *mockRepository) UserIsOwnerOfOrganization(_ context.Context, userID, orgID string) (bool, error) {
	org, ok := r.orgs[orgID]
	if !ok {
		return false, nil
	}
	return org.OwningUsersID == userID, nil
}

func (r *mockRepository) AddMember(_ context.Context, member domain.OrganizationMember) error {
	r.members[member.OrganizationID] = append(r.members[member.OrganizationID], member.AccountCoreID)
	return nil
}

func (r *mockRepository) IsMember(_ context.Context, accountCoreID, orgID string) (bool, error) {
	org, ok := r.orgs[orgID]
	if ok && org.OwningUsersID == accountCoreID {
		return true, nil
	}
	for _, mid := range r.members[orgID] {
		if mid == accountCoreID {
			return true, nil
		}
	}
	return false, nil
}

func (r *mockRepository) GetMemberIDs(_ context.Context, orgID string) ([]string, error) {
	return r.members[orgID], nil
}

func (r *mockRepository) GetOwnerID(_ context.Context, orgID string) (string, error) {
	org, ok := r.orgs[orgID]
	if !ok {
		return "", domain.ErrOrganizationNotFound
	}
	return org.OwningUsersID, nil
}

func (r *mockRepository) CreateGroup(_ context.Context, group domain.Group) error {
	r.groups[group.ID] = group
	return nil
}

func (r *mockRepository) FindGroupByID(_ context.Context, id string) (domain.Group, error) {
	g, ok := r.groups[id]
	if !ok {
		return domain.Group{}, domain.ErrGroupNotFound
	}
	return g, nil
}

func (r *mockRepository) GetGroups(_ context.Context, orgID string) ([]domain.Group, error) {
	var result []domain.Group
	for _, g := range r.groups {
		if g.OrganizationID == orgID {
			result = append(result, g)
		}
	}
	return result, nil
}

func (r *mockRepository) GetGroupsOfUser(_ context.Context, userID, orgID string) ([]domain.Group, error) {
	var result []domain.Group
	for gID, members := range r.groupMembers {
		for _, mid := range members {
			if mid == userID {
				g, ok := r.groups[gID]
				if ok && g.OrganizationID == orgID {
					result = append(result, g)
				}
				break
			}
		}
	}
	return result, nil
}

func (r *mockRepository) AddUserToGroup(_ context.Context, gm domain.GroupMember) error {
	r.groupMembers[gm.GroupID] = append(r.groupMembers[gm.GroupID], gm.AccountCoreID)
	return nil
}

func (r *mockRepository) RemoveUserFromGroup(_ context.Context, accountCoreID, groupID string) error {
	members := r.groupMembers[groupID]
	var updated []string
	for _, m := range members {
		if m != accountCoreID {
			updated = append(updated, m)
		}
	}
	r.groupMembers[groupID] = updated
	return nil
}

func (r *mockRepository) IsGroupMember(_ context.Context, accountCoreID, groupID string) (bool, error) {
	for _, mid := range r.groupMembers[groupID] {
		if mid == accountCoreID {
			return true, nil
		}
	}
	return false, nil
}

func (r *mockRepository) GetGroupMemberIDs(_ context.Context, groupID string) ([]string, error) {
	return r.groupMembers[groupID], nil
}

func (r *mockRepository) GetDefaultGroup(_ context.Context, orgID string) (domain.Group, error) {
	for _, g := range r.groups {
		if g.OrganizationID == orgID && g.IsDefaultForNewMembers {
			return g, nil
		}
	}
	return domain.Group{}, fmt.Errorf("no default group")
}

func (r *mockRepository) CreateInvitation(_ context.Context, inv domain.Invitation) error {
	r.invitations[inv.ID] = inv
	return nil
}

func (r *mockRepository) FindInvitationByID(_ context.Context, id string) (domain.Invitation, error) {
	inv, ok := r.invitations[id]
	if !ok {
		return domain.Invitation{}, domain.ErrInvitationNotFound
	}
	return inv, nil
}

func (r *mockRepository) GetPendingInvitations(_ context.Context, orgID string) ([]domain.Invitation, error) {
	var result []domain.Invitation
	for _, inv := range r.invitations {
		if inv.OrganizationID == orgID {
			result = append(result, inv)
		}
	}
	return result, nil
}

func (r *mockRepository) InvitationExistsForEmail(_ context.Context, orgID, email string) (bool, error) {
	for _, inv := range r.invitations {
		if inv.OrganizationID == orgID && inv.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func (r *mockRepository) DeleteInvitation(_ context.Context, id string) error {
	delete(r.invitations, id)
	return nil
}

func (r *mockRepository) ExecuteInTx(_ context.Context, fn func(repo domain.Repository) error) error {
	return fn(r)
}

func TestCreateOrganization(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewOrgService(repo, testClock)

	org, err := svc.CreateOrganization(context.Background(), "user-1", "Test Org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if org.Name != "Test Org" {
		t.Errorf("expected name 'Test Org', got %q", org.Name)
	}
	if org.OwningUsersID != "user-1" {
		t.Errorf("expected owner 'user-1', got %q", org.OwningUsersID)
	}

	// Should create 2 default groups
	groups, _ := svc.GetGroups(context.Background(), org.ID)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	// Owner should be a member
	isMember, _ := repo.IsMember(context.Background(), "user-1", org.ID)
	if !isMember {
		t.Error("owner should be a member")
	}
}

func TestUserHasAccessRight_Owner(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewOrgService(repo, testClock)

	org, _ := svc.CreateOrganization(context.Background(), "user-1", "Test Org")

	// Owner should have all access rights
	hasRight, err := svc.UserHasAccessRight(context.Background(), "user-1", org.ID, domain.AccessRightEditOrganizationName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasRight {
		t.Error("owner should have EDIT_ORGANIZATION_NAME right")
	}
}

func TestCreateInvitation(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewOrgService(repo, testClock)

	org, _ := svc.CreateOrganization(context.Background(), "user-1", "Test Org")

	inv, err := svc.CreateInvitation(context.Background(), org.ID, "invited@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if inv.Email != "invited@example.com" {
		t.Errorf("expected email 'invited@example.com', got %q", inv.Email)
	}
}

func TestCreateInvitation_Duplicate(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewOrgService(repo, testClock)

	org, _ := svc.CreateOrganization(context.Background(), "user-1", "Test Org")

	_, _ = svc.CreateInvitation(context.Background(), org.ID, "invited@example.com")
	_, err := svc.CreateInvitation(context.Background(), org.ID, "invited@example.com")

	if !errors.Is(err, domain.ErrAlreadyInvited) {
		t.Errorf("expected ErrAlreadyInvited, got %v", err)
	}
}

func TestRenameOrganization(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewOrgService(repo, testClock)

	org, _ := svc.CreateOrganization(context.Background(), "user-1", "Old Name")

	err := svc.RenameOrganization(context.Background(), org.ID, "New Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := svc.GetOrganizationByID(context.Background(), org.ID)
	if updated.Name != "New Name" {
		t.Errorf("expected name 'New Name', got %q", updated.Name)
	}
}

func TestRenameOrganizationAsActor_ForbiddenWithoutRight(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewOrgService(repo, testClock)

	org, _ := svc.CreateOrganization(context.Background(), "owner-1", "Org Name")
	_ = repo.AddMember(context.Background(), domain.OrganizationMember{AccountCoreID: "member-1", OrganizationID: org.ID})

	err := svc.RenameOrganizationAsActor(context.Background(), "member-1", org.ID, "Renamed")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestAcceptInvitation_RejectsMismatchedEmail(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewOrgService(repo, testClock)

	org, _ := svc.CreateOrganization(context.Background(), "owner-1", "Org Name")
	inv, _ := svc.CreateInvitation(context.Background(), org.ID, "invited@example.com")

	_, err := svc.AcceptInvitation(context.Background(), inv.ID, "user-1", "other@example.com")
	if !errors.Is(err, domain.ErrInvitationEmailMismatch) {
		t.Fatalf("expected ErrInvitationEmailMismatch, got %v", err)
	}
}

func TestAcceptInvitation_HappyPath(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewOrgService(repo, testClock)

	org, _ := svc.CreateOrganization(context.Background(), "owner-1", "Org")
	inv, _ := svc.CreateInvitation(context.Background(), org.ID, "new@example.com")

	orgID, err := svc.AcceptInvitation(context.Background(), inv.ID, "new-user", "new@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orgID != org.ID {
		t.Errorf("expected orgID %q, got %q", org.ID, orgID)
	}

	isMember, _ := repo.IsMember(context.Background(), "new-user", org.ID)
	if !isMember {
		t.Error("accepted user should be a member")
	}
}

func TestAcceptInvitation_AlreadyMember(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewOrgService(repo, testClock)

	org, _ := svc.CreateOrganization(context.Background(), "owner-1", "Org")
	inv, _ := svc.CreateInvitation(context.Background(), org.ID, "existing@example.com")

	_ = repo.AddMember(context.Background(), domain.OrganizationMember{AccountCoreID: "existing-user", OrganizationID: org.ID})

	orgID, err := svc.AcceptInvitation(context.Background(), inv.ID, "existing-user", "existing@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orgID != org.ID {
		t.Errorf("expected orgID %q, got %q", org.ID, orgID)
	}

	// Invitation should be deleted
	_, err = svc.FindInvitationByID(context.Background(), inv.ID)
	if !errors.Is(err, domain.ErrInvitationNotFound) {
		t.Error("invitation should have been deleted")
	}
}

func TestCanAccessOrganization(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewOrgService(repo, testClock)

	org, _ := svc.CreateOrganization(context.Background(), "owner-1", "Org")

	can, _ := svc.CanAccessOrganization(context.Background(), "owner-1", org.ID)
	if !can {
		t.Error("owner should have access")
	}

	can, _ = svc.CanAccessOrganization(context.Background(), "stranger", org.ID)
	if can {
		t.Error("stranger should not have access")
	}
}

func TestGetMemberIDs(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewOrgService(repo, testClock)

	org, _ := svc.CreateOrganization(context.Background(), "owner-1", "Org")
	_ = repo.AddMember(context.Background(), domain.OrganizationMember{AccountCoreID: "member-1", OrganizationID: org.ID})

	ids, err := svc.GetMemberIDs(context.Background(), org.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}
	if !idSet["owner-1"] || !idSet["member-1"] {
		t.Errorf("expected owner-1 and member-1 in %v", ids)
	}
}

func TestRemoveUserFromGroupAsActor(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewOrgService(repo, testClock)

	org, _ := svc.CreateOrganization(context.Background(), "owner-1", "Org")
	groups, _ := svc.GetGroups(context.Background(), org.ID)
	var teamGroupID string
	for _, g := range groups {
		if g.Name == "Team Members" {
			teamGroupID = g.ID
			break
		}
	}

	_ = repo.AddMember(context.Background(), domain.OrganizationMember{AccountCoreID: "member-1", OrganizationID: org.ID})
	_ = repo.AddUserToGroup(context.Background(), domain.GroupMember{AccountCoreID: "member-1", GroupID: teamGroupID})

	err := svc.RemoveUserFromGroupAsActor(context.Background(), "owner-1", "member-1", teamGroupID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	isMember, _ := repo.IsGroupMember(context.Background(), "member-1", teamGroupID)
	if isMember {
		t.Error("member should have been removed from group")
	}
}

func TestRemoveUserFromGroupAsActor_Forbidden(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewOrgService(repo, testClock)

	org, _ := svc.CreateOrganization(context.Background(), "owner-1", "Org")
	groups, _ := svc.GetGroups(context.Background(), org.ID)
	var teamGroupID string
	for _, g := range groups {
		if g.Name == "Team Members" {
			teamGroupID = g.ID
			break
		}
	}

	_ = repo.AddMember(context.Background(), domain.OrganizationMember{AccountCoreID: "member-1", OrganizationID: org.ID})
	_ = repo.AddMember(context.Background(), domain.OrganizationMember{AccountCoreID: "member-2", OrganizationID: org.ID})

	err := svc.RemoveUserFromGroupAsActor(context.Background(), "member-1", "member-2", teamGroupID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreateInvitationAsActor(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewOrgService(repo, testClock)

	org, _ := svc.CreateOrganization(context.Background(), "owner-1", "Org")

	inv, err := svc.CreateInvitationAsActor(context.Background(), "owner-1", org.ID, "new@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Email != "new@example.com" {
		t.Errorf("expected email 'new@example.com', got %q", inv.Email)
	}
}

func TestCreateInvitationAsActor_Forbidden(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewOrgService(repo, testClock)

	org, _ := svc.CreateOrganization(context.Background(), "owner-1", "Org")
	_ = repo.AddMember(context.Background(), domain.OrganizationMember{AccountCoreID: "member-1", OrganizationID: org.ID})

	_, err := svc.CreateInvitationAsActor(context.Background(), "member-1", org.ID, "new@example.com")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestAddUserToGroupAsActor_RejectsCrossOrganizationMember(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewOrgService(repo, testClock)

	orgA, _ := svc.CreateOrganization(context.Background(), "owner-a", "Org A")
	orgB, _ := svc.CreateOrganization(context.Background(), "owner-b", "Org B")

	groups, _ := svc.GetGroups(context.Background(), orgA.ID)
	var teamGroupID string
	for _, g := range groups {
		if g.Name == "Team Members" {
			teamGroupID = g.ID
			break
		}
	}
	if teamGroupID == "" {
		t.Fatal("team group missing")
	}

	_ = repo.AddMember(context.Background(), domain.OrganizationMember{
		AccountCoreID:  "foreign-member",
		OrganizationID: orgB.ID,
	})

	err := svc.AddUserToGroupAsActor(context.Background(), "owner-a", "foreign-member", teamGroupID)
	if !errors.Is(err, domain.ErrCrossOrganizationGroupAssignment) {
		t.Fatalf("expected ErrCrossOrganizationGroupAssignment, got %v", err)
	}
}
