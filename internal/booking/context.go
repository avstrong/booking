package booking

import "context"

type contextKey string

const idempotencyKey contextKey = "idempotencyKey"

func NewContextWithIdempotencyKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, idempotencyKey, key)
}

func IdempotencyKeyFromContext(ctx context.Context) (string, bool) {
	key, ok := ctx.Value(idempotencyKey).(string)

	return key, ok
}
