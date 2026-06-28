package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesInsertAndList(t *testing.T) {
	db := newTestDB(t)
	repo := NewFilesRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Insert(ctx, "f1", []byte("meta1"), 3, 1))
	require.NoError(t, repo.Insert(ctx, "f2", []byte("meta2"), 5, 1))

	list, err := repo.List(ctx, "", 10)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "f1", list[0].ID)
	assert.Equal(t, []byte("meta1"), list[0].Meta)
	assert.Equal(t, 3, list[0].ChunkCount)
	assert.Equal(t, int64(1), list[0].Version)

	st := readState(t, db, "files", "f1")
	assert.Equal(t, 0, st.Dirty)
	assert.Equal(t, int64(1), st.BaseVersion)

	page, err := repo.List(ctx, "f1", 10)
	require.NoError(t, err)
	require.Len(t, page, 1)
	assert.Equal(t, "f2", page[0].ID)
}

func TestFilesInsertUpsertConflict(t *testing.T) {
	db := newTestDB(t)
	repo := NewFilesRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Insert(ctx, "f1", []byte("meta1"), 3, 1))
	require.NoError(t, repo.Insert(ctx, "f1", []byte("meta2"), 4, 2))

	st := readState(t, db, "files", "f1")
	assert.Equal(t, int64(2), st.Version)
	assert.Equal(t, int64(2), st.BaseVersion)

	list, err := repo.List(ctx, "", 10)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, []byte("meta2"), list[0].Meta)
	assert.Equal(t, 4, list[0].ChunkCount)
}

func TestFilesUpdateMetaAndDelete(t *testing.T) {
	db := newTestDB(t)
	repo := NewFilesRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Insert(ctx, "f1", []byte("meta1"), 3, 1))
	require.NoError(t, repo.UpdateMeta(ctx, "f1", []byte("meta-upd")))

	st := readState(t, db, "files", "f1")
	assert.Equal(t, int64(2), st.Version)
	assert.Equal(t, 1, st.Dirty)

	require.NoError(t, repo.Delete(ctx, "f1"))
	st = readState(t, db, "files", "f1")
	assert.Equal(t, 1, st.Deleted)
	assert.Equal(t, 1, st.Dirty)

	list, err := repo.List(ctx, "", 10)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestFilesUpdateMetaClearsConflict(t *testing.T) {
	db := newTestDB(t)
	repo := NewFilesRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Insert(ctx, "f1", []byte("meta1"), 3, 1))
	require.NoError(t, repo.MarkConflict(ctx, "f1", []byte("srv"), 9))
	require.NoError(t, repo.UpdateMeta(ctx, "f1", []byte("meta2")))

	st := readState(t, db, "files", "f1")
	assert.Equal(t, 0, st.Conflict)
	assert.Equal(t, int64(9), st.BaseVersion)
	assert.Nil(t, st.ServerBlob)
}

func TestFilesSyncFlow(t *testing.T) {
	db := newTestDB(t)
	repo := NewFilesRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Insert(ctx, "clean", []byte("m"), 1, 1))
	require.NoError(t, repo.UpdateMeta(ctx, "clean", []byte("m2")))
	require.NoError(t, repo.Insert(ctx, "conf", []byte("m"), 1, 1))
	require.NoError(t, repo.UpdateMeta(ctx, "conf", []byte("m2")))
	require.NoError(t, repo.MarkConflict(ctx, "conf", []byte("srv"), 5))

	dirty, err := repo.ListDirty(ctx)
	require.NoError(t, err)
	require.Len(t, dirty, 1)
	assert.Equal(t, "clean", dirty[0].ID)

	row, ok, err := repo.GetRow(ctx, "clean")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "clean", row.ID)

	_, ok, err = repo.GetRow(ctx, "missing")
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, repo.Upsert(ctx, "srv-1", []byte("blob"), 7))
	st := readState(t, db, "files", "srv-1")
	assert.Equal(t, int64(7), st.Version)
	assert.Equal(t, 0, st.Dirty)

	require.NoError(t, repo.MarkSynced(ctx, "clean", 4))
	st = readState(t, db, "files", "clean")
	assert.Equal(t, int64(4), st.Version)
	assert.Equal(t, 0, st.Dirty)

	require.NoError(t, repo.HardDelete(ctx, "srv-1"))
	assert.False(t, rowExists(t, db, "files", "srv-1"))
}

func TestFilesConflictResolution(t *testing.T) {
	db := newTestDB(t)
	repo := NewFilesRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Insert(ctx, "f1", []byte("mine"), 1, 1))
	require.NoError(t, repo.MarkConflict(ctx, "f1", []byte("srv"), 11))

	conflicts, err := repo.ListConflicts(ctx)
	require.NoError(t, err)
	require.Len(t, conflicts, 1)
	assert.Equal(t, []byte("mine"), conflicts[0].Local)
	assert.Equal(t, []byte("srv"), conflicts[0].Server)
	assert.Equal(t, int64(11), conflicts[0].ServerVersion)

	require.NoError(t, repo.ResolveKeepMine(ctx, "f1"))
	st := readState(t, db, "files", "f1")
	assert.Equal(t, int64(11), st.BaseVersion)
	assert.Equal(t, 1, st.Dirty)
	assert.Equal(t, 0, st.Conflict)

	require.NoError(t, repo.Insert(ctx, "f2", []byte("mine2"), 1, 1))
	require.NoError(t, repo.MarkConflict(ctx, "f2", []byte("srv2"), 12))
	require.NoError(t, repo.ResolveTakeServer(ctx, "f2"))
	st = readState(t, db, "files", "f2")
	assert.Equal(t, int64(12), st.Version)
	assert.Equal(t, 0, st.Dirty)
	assert.Equal(t, 0, st.Conflict)

	list, err := repo.List(ctx, "", 10)
	require.NoError(t, err)
	for _, f := range list {
		if f.ID == "f2" {
			assert.Equal(t, []byte("srv2"), f.Meta)
		}
	}
}
