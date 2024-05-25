package auth

import (
	"context"

	"github.com/nakamauwu/nakama/types"
)

type ctxKey struct{ name string }

var ctxKeyUser = &ctxKey{"user-ctx-key"}

func ContextWithUser(ctx context.Context, user types.User) context.Context {
	return context.WithValue(ctx, ctxKeyUser, user)
}

func UserFromContext(ctx context.Context) (types.User, bool) {
	user, ok := ctx.Value(ctxKeyUser).(types.User)
	return user, ok
}
