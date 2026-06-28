// Package syncclient coordinates the per-type synchronizers.
package syncclient

import (
	"context"
	"fmt"
	"gophkeeper/internal/client/sync/syncer"

	"golang.org/x/sync/errgroup"
)

type Pool struct {
	syncers []*syncer.Syncer
}

func New(syncers ...*syncer.Syncer) *Pool {
	return &Pool{syncers: syncers}
}

func (p *Pool) SyncAll(ctx context.Context) error {
	gr, grCtx := errgroup.WithContext(ctx)
	for _, s := range p.syncers {
		gr.Go(func() error {
			return s.Sync(grCtx)
		})
	}
	if err := gr.Wait(); err != nil {
		return fmt.Errorf("sync err: %w", err)
	}
	return nil
}
