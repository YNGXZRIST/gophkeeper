// Package syncnotes adapts the local notes repository to syncer.Repo.
package syncnotes

import (
	"context"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/sync/syncer"
)

const entity = "notes"

type Repo struct {
	notes *repository.NotesRepo
	state *repository.SyncStateRepo
}

func NewRepo(notes *repository.NotesRepo, state *repository.SyncStateRepo) *Repo {
	return &Repo{notes: notes, state: state}
}

var _ syncer.Repo = (*Repo)(nil)

func (r *Repo) ListDirty(ctx context.Context) ([]syncer.Row, error) {
	rows, err := r.notes.ListDirty(ctx)
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
	row, ok, err := r.notes.GetRow(ctx, id)
	if err != nil || !ok {
		return syncer.Row{}, ok, err
	}
	return toRow(row), true, nil
}

func (r *Repo) Upsert(ctx context.Context, id string, data []byte, version int64) error {
	return r.notes.Upsert(ctx, id, data, version)
}

func (r *Repo) HardDelete(ctx context.Context, id string) error {
	return r.notes.HardDelete(ctx, id)
}

func (r *Repo) MarkSynced(ctx context.Context, id string, version int64) error {
	return r.notes.MarkSynced(ctx, id, version)
}

func (r *Repo) MarkConflict(ctx context.Context, id string, serverBlob []byte, serverVersion int64) error {
	return r.notes.MarkConflict(ctx, id, serverBlob, serverVersion)
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
