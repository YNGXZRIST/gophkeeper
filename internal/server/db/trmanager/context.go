package trmanager

import (
	"context"
)

type txKey struct {
}

// WithTx stores transaction in context.
func WithTx(ctx context.Context, tx *Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// TxFromContext returns transaction from context if present.
func TxFromContext(ctx context.Context) (*Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(*Tx)
	return tx, ok
}
