// Package syncfiles adapts the local files repository to syncer.Repo.
package syncfiles

import (
	"context"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/sync/syncer"
)

const entity = "files"

type Repo struct {
	files *repository.FilesRepo
	state *repository.SyncStateRepo
}

func NewRepo(files *repository.FilesRepo, state *repository.SyncStateRepo) *Repo {
	return &Repo{files: files, state: state}
}

var _ syncer.Repo = (*Repo)(nil)

func (r *Repo) ListDirty(ctx context.Context) ([]syncer.Row, error) {
	rows, err := r.files.ListDirty(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]syncer.Row, len(rows))
	for i, row := range rows {
		out[i] = toRow(row)
	}
	return out, nil
}

func (r *Repo) Get(ctx context.Context, id string) (syncer.Row, bool, error) {
	row, ok, err := r.files.GetRow(ctx, id)
	if err != nil || !ok {
		return syncer.Row{}, ok, err
	}
	return toRow(row), true, nil
}

func (r *Repo) Upsert(ctx context.Context, id string, data []byte, version int64) error {
	return r.files.Upsert(ctx, id, data, version)
}

func (r *Repo) HardDelete(ctx context.Context, id string) error {
	return r.files.HardDelete(ctx, id)
}

func (r *Repo) MarkSynced(ctx context.Context, id string, version int64) error {
	return r.files.MarkSynced(ctx, id, version)
}

func (r *Repo) MarkConflict(ctx context.Context, id string, serverBlob []byte, serverVersion int64) error {
	return r.files.MarkConflict(ctx, id, serverBlob, serverVersion)
}

func (r *Repo) LastSyncedAt(ctx context.Context) (string, error) {
	return r.state.Cursor(ctx, entity)
}

func (r *Repo) SetLastSyncedAt(ctx context.Context, cursor string) error {
	return r.state.SetCursor(ctx, entity, cursor)
}

func toRow(s repository.SyncRow) syncer.Row {
	return syncer.Row{
		ID:          s.ID,
		Data:        s.Data,
		Version:     s.Version,
		BaseVersion: s.BaseVersion,
		Dirty:       s.Dirty,
		Deleted:     s.Deleted,
	}
}
