// Package syncer reconciles a local secret store with the server. It is generic
// over the secret type: notes, cards, passwords and files all use it through
// the Repo and Client interfaces.
package syncer

import (
	"context"
	"errors"
)

var ErrVersionConflict = errors.New("version conflict")

type Row struct {
	ID          string
	Data        []byte
	Version     int64
	BaseVersion int64
	Dirty       bool
	Deleted     bool
}

type Change struct {
	ID      string
	Data    []byte
	Version int64
	Deleted bool
}

type Repo interface {
	ListDirty(ctx context.Context) ([]Row, error)
	Get(ctx context.Context, id string) (Row, bool, error)
	Upsert(ctx context.Context, id string, data []byte, version int64) error
	HardDelete(ctx context.Context, id string) error
	MarkSynced(ctx context.Context, id string, version int64) error
	MarkConflict(ctx context.Context, id string, serverBlob []byte, serverVersion int64) error
	LastSyncedAt(ctx context.Context) (string, error)
	SetLastSyncedAt(ctx context.Context, cursor string) error
}

type Client interface {
	Changes(ctx context.Context, since string) ([]Change, string, error)
	Create(ctx context.Context, id string, data []byte) (int64, error)
	Update(ctx context.Context, id string, data []byte, version int64) (int64, error)
	Delete(ctx context.Context, id string, version int64) error
}

type Syncer struct {
	repo   Repo
	client Client
}

func New(repo Repo, client Client) *Syncer {
	return &Syncer{repo: repo, client: client}
}

func (s *Syncer) Sync(ctx context.Context) error {
	if err := s.pull(ctx); err != nil {
		return err
	}
	return s.push(ctx)
}

func (s *Syncer) pull(ctx context.Context) error {
	since, err := s.repo.LastSyncedAt(ctx)
	if err != nil {
		return err
	}
	changes, cursor, err := s.client.Changes(ctx, since)
	if err != nil {
		return err
	}
	for _, ch := range changes {
		local, ok, err := s.repo.Get(ctx, ch.ID)
		if err != nil {
			return err
		}
		if ok && local.Dirty {
			if ch.Version != local.BaseVersion {
				if err := s.repo.MarkConflict(ctx, ch.ID, ch.Data, ch.Version); err != nil {
					return err
				}
			}
			continue
		}
		if ch.Deleted {
			if err := s.repo.HardDelete(ctx, ch.ID); err != nil {
				return err
			}
			continue
		}
		if err := s.repo.Upsert(ctx, ch.ID, ch.Data, ch.Version); err != nil {
			return err
		}
	}
	return s.repo.SetLastSyncedAt(ctx, cursor)
}

func (s *Syncer) push(ctx context.Context) error {
	dirty, err := s.repo.ListDirty(ctx)
	if err != nil {
		return err
	}
	for _, row := range dirty {
		err := s.pushRow(ctx, row)
		if errors.Is(err, ErrVersionConflict) {
			if cerr := s.repo.MarkConflict(ctx, row.ID, nil, 0); cerr != nil {
				return cerr
			}
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Syncer) pushRow(ctx context.Context, row Row) error {
	switch {
	case row.Deleted:
		if err := s.client.Delete(ctx, row.ID, row.BaseVersion); err != nil {
			return err
		}
		return s.repo.HardDelete(ctx, row.ID)
	case row.BaseVersion == 0:
		version, err := s.client.Create(ctx, row.ID, row.Data)
		if err != nil {
			return err
		}
		return s.repo.MarkSynced(ctx, row.ID, version)
	default:
		version, err := s.client.Update(ctx, row.ID, row.Data, row.BaseVersion)
		if err != nil {
			return err
		}
		return s.repo.MarkSynced(ctx, row.ID, version)
	}
}
