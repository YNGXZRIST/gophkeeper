// Package syncclient coordinates the per-type synchronizers.
package syncclient

import (
	"context"
	"gophkeeper/internal/client/sync/syncer"
)

type Pool struct {
	syncers []*syncer.Syncer
}

func New(syncers ...*syncer.Syncer) *Pool {
	return &Pool{syncers: syncers}
}

func (p *Pool) SyncAll(ctx context.Context) error {
	for _, s := range p.syncers {
		if err := s.Sync(ctx); err != nil {
			return err
		}
	}
	return nil
}
