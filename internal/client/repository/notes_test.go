package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotesCreateAndList(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotesRepo(db)
	ctx := context.Background()

	n, err := repo.Create(ctx, []byte("first"))
	require.NoError(t, err)
	assert.NotEmpty(t, n.ID)
	assert.Equal(t, int64(1), n.Version)
	assert.Equal(t, []byte("first"), n.Data)

	st := readState(t, db, "notes", n.ID)
	assert.Equal(t, 1, st.Dirty)
	assert.Equal(t, 0, st.Deleted)
	assert.Equal(t, int64(0), st.BaseVersion)

	list, err := repo.List(ctx, "", 10)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, n.ID, list[0].ID)
}

func TestNotesListKeysetPaging(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotesRepo(db)
	ctx := context.Background()

	var ids []string
	for i := 0; i < 3; i++ {
		n, err := repo.Create(ctx, []byte{byte('a' + i)})
		require.NoError(t, err)
		ids = append(ids, n.ID)
	}

	page1, err := repo.List(ctx, "", 2)
	require.NoError(t, err)
	require.Len(t, page1, 2)

	page2, err := repo.List(ctx, page1[len(page1)-1].ID, 2)
	require.NoError(t, err)
	require.Len(t, page2, 1)

	assert.Greater(t, page2[0].ID, page1[1].ID)
}

func TestNotesListExcludesDeleted(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotesRepo(db)
	ctx := context.Background()

	n, err := repo.Create(ctx, []byte("x"))
	require.NoError(t, err)
	require.NoError(t, repo.Delete(ctx, n.ID))

	list, err := repo.List(ctx, "", 10)
	require.NoError(t, err)
	assert.Empty(t, list)

	st := readState(t, db, "notes", n.ID)
	assert.Equal(t, 1, st.Deleted)
	assert.Equal(t, 1, st.Dirty)
	assert.Equal(t, int64(2), st.Version)
}

func TestNotesUpdate(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotesRepo(db)
	ctx := context.Background()

	n, err := repo.Create(ctx, []byte("v1"))
	require.NoError(t, err)
	require.NoError(t, repo.Update(ctx, n.ID, []byte("v2")))

	st := readState(t, db, "notes", n.ID)
	assert.Equal(t, int64(2), st.Version)
	assert.Equal(t, 1, st.Dirty)

	list, err := repo.List(ctx, "", 10)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, []byte("v2"), list[0].Data)
}

func TestNotesUpdateClearsConflict(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotesRepo(db)
	ctx := context.Background()

	n, err := repo.Create(ctx, []byte("v1"))
	require.NoError(t, err)
	require.NoError(t, repo.MarkConflict(ctx, n.ID, []byte("srv"), 9))

	st := readState(t, db, "notes", n.ID)
	require.Equal(t, 1, st.Conflict)

	require.NoError(t, repo.Update(ctx, n.ID, []byte("v2")))
	st = readState(t, db, "notes", n.ID)
	assert.Equal(t, 0, st.Conflict)
	assert.Nil(t, st.ServerBlob)
	assert.False(t, st.ServerVersion.Valid)

	assert.Equal(t, int64(9), st.BaseVersion)
}

func TestNotesListDirtyExcludesConflict(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotesRepo(db)
	ctx := context.Background()

	clean, err := repo.Create(ctx, []byte("dirty"))
	require.NoError(t, err)
	conflicted, err := repo.Create(ctx, []byte("conf"))
	require.NoError(t, err)
	require.NoError(t, repo.MarkConflict(ctx, conflicted.ID, []byte("srv"), 5))

	dirty, err := repo.ListDirty(ctx)
	require.NoError(t, err)
	require.Len(t, dirty, 1)
	assert.Equal(t, clean.ID, dirty[0].ID)
	assert.True(t, dirty[0].Dirty)
	assert.False(t, dirty[0].Deleted)
}

func TestNotesGetRow(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotesRepo(db)
	ctx := context.Background()

	n, err := repo.Create(ctx, []byte("data"))
	require.NoError(t, err)

	row, ok, err := repo.GetRow(ctx, n.ID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, n.ID, row.ID)
	assert.Equal(t, []byte("data"), row.Data)

	_, ok, err = repo.GetRow(ctx, "missing")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestNotesUpsert(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotesRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Upsert(ctx, "id-1", []byte("server"), 7))
	st := readState(t, db, "notes", "id-1")
	assert.Equal(t, int64(7), st.Version)
	assert.Equal(t, int64(7), st.BaseVersion)
	assert.Equal(t, 0, st.Dirty)
	assert.Equal(t, 0, st.Deleted)

	require.NoError(t, repo.Upsert(ctx, "id-1", []byte("server2"), 8))
	st = readState(t, db, "notes", "id-1")
	assert.Equal(t, int64(8), st.Version)
}

func TestNotesHardDelete(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotesRepo(db)
	ctx := context.Background()

	n, err := repo.Create(ctx, []byte("data"))
	require.NoError(t, err)
	require.NoError(t, repo.HardDelete(ctx, n.ID))
	assert.False(t, rowExists(t, db, "notes", n.ID))
}

func TestNotesMarkSynced(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotesRepo(db)
	ctx := context.Background()

	n, err := repo.Create(ctx, []byte("data"))
	require.NoError(t, err)
	require.NoError(t, repo.MarkSynced(ctx, n.ID, 4))

	st := readState(t, db, "notes", n.ID)
	assert.Equal(t, int64(4), st.Version)
	assert.Equal(t, int64(4), st.BaseVersion)
	assert.Equal(t, 0, st.Dirty)
}

func TestNotesMarkConflict(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotesRepo(db)
	ctx := context.Background()

	n, err := repo.Create(ctx, []byte("data"))
	require.NoError(t, err)
	require.NoError(t, repo.MarkConflict(ctx, n.ID, []byte("srv"), 3))

	st := readState(t, db, "notes", n.ID)
	assert.Equal(t, 1, st.Conflict)
	assert.Equal(t, []byte("srv"), st.ServerBlob)
	assert.Equal(t, int64(3), st.ServerVersion.Int64)
}

func TestNotesListConflicts(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotesRepo(db)
	ctx := context.Background()

	withBlob, err := repo.Create(ctx, []byte("mine"))
	require.NoError(t, err)
	require.NoError(t, repo.MarkConflict(ctx, withBlob.ID, []byte("srv"), 6))

	noBlob, err := repo.Create(ctx, []byte("mine2"))
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "UPDATE notes SET conflict = 1 WHERE id = ?", noBlob.ID)
	require.NoError(t, err)

	conflicts, err := repo.ListConflicts(ctx)
	require.NoError(t, err)
	require.Len(t, conflicts, 1)
	assert.Equal(t, withBlob.ID, conflicts[0].ID)
	assert.Equal(t, []byte("mine"), conflicts[0].Local)
	assert.Equal(t, []byte("srv"), conflicts[0].Server)
	assert.Equal(t, int64(6), conflicts[0].ServerVersion)
}

func TestNotesResolveKeepMine(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotesRepo(db)
	ctx := context.Background()

	n, err := repo.Create(ctx, []byte("mine"))
	require.NoError(t, err)
	require.NoError(t, repo.MarkConflict(ctx, n.ID, []byte("srv"), 11))
	require.NoError(t, repo.ResolveKeepMine(ctx, n.ID))

	st := readState(t, db, "notes", n.ID)
	assert.Equal(t, int64(11), st.BaseVersion)
	assert.Equal(t, 1, st.Dirty)
	assert.Equal(t, 0, st.Conflict)
	assert.Nil(t, st.ServerBlob)
}

func TestNotesResolveTakeServer(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotesRepo(db)
	ctx := context.Background()

	n, err := repo.Create(ctx, []byte("mine"))
	require.NoError(t, err)
	require.NoError(t, repo.MarkConflict(ctx, n.ID, []byte("srv"), 12))
	require.NoError(t, repo.ResolveTakeServer(ctx, n.ID))

	st := readState(t, db, "notes", n.ID)
	assert.Equal(t, int64(12), st.Version)
	assert.Equal(t, int64(12), st.BaseVersion)
	assert.Equal(t, 0, st.Dirty)
	assert.Equal(t, 0, st.Conflict)
	assert.Nil(t, st.ServerBlob)

	row, ok, err := repo.GetRow(ctx, n.ID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, []byte("srv"), row.Data)
}
