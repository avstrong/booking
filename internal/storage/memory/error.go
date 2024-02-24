package memory

import "errors"

var (
	ErrTransactionIDNotFoundInCtx = errors.New("no transaction id found in ctx")
	ErrTransactionNotFound        = errors.New("transaction not found")
)
