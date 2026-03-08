# Organization Book

## Domain Model

### Organization

A workspace/team that users belong to. Each user can own and be a member of multiple organizations.

**Fields:**

- `ID` — UUID
- `OwningUsersID` — UUID of the account that created the organization (cross-vertical reference, no FK)
- `Name` — Optional display name (max 256 chars)
- `CreatedAt` — Timestamp

### Organization Member

Tracks user membership in organizations via a join table.

**Fields:**

- `AccountCoreID` — UUID (cross-vertical reference, no FK)
- `OrganizationID` — UUID (FK to organizations)

### Group

Permission groups within an organization. Two default groups are created automatically:

1. **Administrators** — Has `full_access`, owner is added here
2. **Team Members** — Default group for new members, no special access rights

**Fields:**

- `ID` — UUID
- `OrganizationID` — UUID (FK to organizations)
- `Name` — Group name
- `AccessRights` — Array of access right strings
- `IsDefaultForNewMembers` — Boolean

### Access Rights

```
full_access                           — All permissions
edit_organization_name                — Can rename the organization
invite_organization_members           — Can send invitations
see_organization_groups_and_members   — Can view groups and member lists
move_organization_members_into_groups — Can manage group membership
```

The organization owner always has all access rights regardless of group membership.

### Invitation

Email-based invitations to join an organization.

**Fields:**

- `ID` — UUID
- `OrganizationID` — UUID (FK to organizations)
- `Email` — Invited email address
- `CreatedAt` — Timestamp

## Workflows

### User Registration

1. User creates account via sign-up form
2. `AccountCreatedEvent` is dispatched
3. Organization subscriber creates a default organization (name is empty → displays as "My Organization")
4. `ActiveOrgChangedEvent` is dispatched
5. Account subscriber sets the new org as the user's active organization

### Invitation Flow

1. Organization owner sends invitation via email address
2. Invitation is stored in database
3. Invited user receives link (email sending is a placeholder)
4. User visits invitation URL → sees acceptance page
5. User accepts with the same authenticated account email as the invitation
6. User is added as member and assigned to the default group, invitation is deleted, and the org becomes active

### Organization Switching

Users with multiple organizations can switch their active organization from the organization dashboard.
