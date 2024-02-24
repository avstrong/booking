package memory

import "context"

type contextKey string

const transactionKey contextKey = "storageTransactionID"

func withTransactionID(ctx context.Context, trxID string) context.Context {
	return context.WithValue(ctx, transactionKey, trxID)
}

func transactionIDFromContext(ctx context.Context) (string, bool) {
	trxID, ok := ctx.Value(transactionKey).(string)

	return trxID, ok
}
