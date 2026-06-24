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
	gr, err := errgroup.WithContext(ctx)
	if err != nil {
		return fmt.Errorf("cannot create err group: %w", err)
	}
	for _, s := range p.syncers {
		gr.Go(func() error {
			return s.Sync(ctx)
		})
	}
	errGr := gr.Wait()
	if errGr != nil {
		return fmt.Errorf("sync err: %w", errGr)
	}
	return nil
}
