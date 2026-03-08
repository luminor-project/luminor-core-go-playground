package domain

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

func (r Role) String() string {
	return string(r)
}

func ParseRole(s string) (Role, bool) {
	switch s {
	case "user":
		return RoleUser, true
	case "admin":
		return RoleAdmin, true
	default:
		return "", false
	}
}
