package auth

import (
	"context"
)

type contextKey string

const userContextKey contextKey = "auth_user"

type User struct {
	ID              string
	Email           string
	Roles           []string
	ActivePartyID   string
	ActivePartyKind string
}

func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(userContextKey).(User)
	return user, ok
}

func MustUserFromContext(ctx context.Context) User {
	user, ok := UserFromContext(ctx)
	if !ok {
		panic("auth: no user in context")
	}
	return user
}

func IsAuthenticated(ctx context.Context) bool {
	_, ok := UserFromContext(ctx)
	return ok
}
