package nakama

import "context"

type contextKey string

const contextKeyUser contextKey = "user"

func UserFromContext(ctx context.Context) (UserIdentity, bool) {
	usr, ok := ctx.Value(contextKeyUser).(UserIdentity)
	return usr, ok
}

func ContextWithUser(ctx context.Context, usr UserIdentity) context.Context {
	return context.WithValue(ctx, contextKeyUser, usr)
}
