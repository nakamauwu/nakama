package nakama

import "context"

type contextKey string

const contextKeyUser contextKey = "user"

func UserFromContext(ctx context.Context) (User, bool) {
	usr, ok := ctx.Value(contextKeyUser).(User)
	return usr, ok
}

func ContextWithUser(ctx context.Context, usr User) context.Context {
	return context.WithValue(ctx, contextKeyUser, usr)
}
