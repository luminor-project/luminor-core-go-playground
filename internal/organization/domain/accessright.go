package domain

type AccessRight string

const (
	AccessRightFullAccess                    AccessRight = "full_access"
	AccessRightEditOrganizationName          AccessRight = "edit_organization_name"
	AccessRightInviteOrganizationMembers     AccessRight = "invite_organization_members"
	AccessRightSeeOrganizationGroups         AccessRight = "see_organization_groups_and_members"
	AccessRightMoveOrganizationMembersGroups AccessRight = "move_organization_members_into_groups"
)

func (a AccessRight) String() string {
	return string(a)
}

// AllAccessRights returns all available access rights for admin groups.
func AllAccessRights() []AccessRight {
	return []AccessRight{
		AccessRightFullAccess,
		AccessRightEditOrganizationName,
		AccessRightInviteOrganizationMembers,
		AccessRightSeeOrganizationGroups,
		AccessRightMoveOrganizationMembersGroups,
	}
}
