package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCardsCRUDAndPaging(t *testing.T) {
	db := newTestDB(t)
	repo := NewEntryRepo(db, TableCard)
	ctx := context.Background()

	var ids []string
	for i := 0; i < 3; i++ {
		c, err := repo.Create(ctx, []byte{byte('a' + i)})
		require.NoError(t, err)
		assert.Equal(t, int64(1), c.Version)
		ids = append(ids, c.ID)
	}

	page1, err := repo.List(ctx, "", 2)
	require.NoError(t, err)
	require.Len(t, page1, 2)
	page2, err := repo.List(ctx, page1[1].ID, 2)
	require.NoError(t, err)
	require.Len(t, page2, 1)

	require.NoError(t, repo.Update(ctx, ids[0], []byte("upd")))
	st := readState(t, db, "cards", ids[0])
	assert.Equal(t, int64(2), st.Version)
	assert.Equal(t, 1, st.Dirty)

	require.NoError(t, repo.Delete(ctx, ids[1]))
	st = readState(t, db, "cards", ids[1])
	assert.Equal(t, 1, st.Deleted)

	list, err := repo.List(ctx, "", 10)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestCardsUpdateClearsConflict(t *testing.T) {
	db := newTestDB(t)
	repo := NewEntryRepo(db, TableCard)
	ctx := context.Background()

	c, err := repo.Create(ctx, []byte("v1"))
	require.NoError(t, err)
	require.NoError(t, repo.MarkConflict(ctx, c.ID, []byte("srv"), 9))
	require.NoError(t, repo.Update(ctx, c.ID, []byte("v2")))

	st := readState(t, db, "cards", c.ID)
	assert.Equal(t, 0, st.Conflict)
	assert.Equal(t, int64(9), st.BaseVersion)
	assert.Nil(t, st.ServerBlob)
}

func TestCardsSyncFlow(t *testing.T) {
	db := newTestDB(t)
	repo := NewEntryRepo(db, TableCard)
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

	row, ok, err := repo.GetRow(ctx, clean.ID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, clean.ID, row.ID)

	_, ok, err = repo.GetRow(ctx, "missing")
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, repo.Upsert(ctx, "srv-1", []byte("blob"), 7))
	st := readState(t, db, "cards", "srv-1")
	assert.Equal(t, int64(7), st.Version)
	assert.Equal(t, 0, st.Dirty)

	require.NoError(t, repo.MarkSynced(ctx, clean.ID, 4))
	st = readState(t, db, "cards", clean.ID)
	assert.Equal(t, int64(4), st.Version)
	assert.Equal(t, 0, st.Dirty)

	require.NoError(t, repo.HardDelete(ctx, "srv-1"))
	assert.False(t, rowExists(t, db, "cards", "srv-1"))
}

func TestCardsConflictResolution(t *testing.T) {
	db := newTestDB(t)
	repo := NewEntryRepo(db, TableCard)
	ctx := context.Background()

	mine, err := repo.Create(ctx, []byte("mine"))
	require.NoError(t, err)
	require.NoError(t, repo.MarkConflict(ctx, mine.ID, []byte("srv"), 11))

	conflicts, err := repo.ListConflicts(ctx)
	require.NoError(t, err)
	require.Len(t, conflicts, 1)
	assert.Equal(t, []byte("srv"), conflicts[0].Server)
	assert.Equal(t, int64(11), conflicts[0].ServerVersion)

	require.NoError(t, repo.ResolveKeepMine(ctx, mine.ID))
	st := readState(t, db, "cards", mine.ID)
	assert.Equal(t, int64(11), st.BaseVersion)
	assert.Equal(t, 1, st.Dirty)
	assert.Equal(t, 0, st.Conflict)

	other, err := repo.Create(ctx, []byte("mine2"))
	require.NoError(t, err)
	require.NoError(t, repo.MarkConflict(ctx, other.ID, []byte("srv2"), 12))
	require.NoError(t, repo.ResolveTakeServer(ctx, other.ID))
	st = readState(t, db, "cards", other.ID)
	assert.Equal(t, int64(12), st.Version)
	assert.Equal(t, 0, st.Dirty)
	assert.Equal(t, 0, st.Conflict)
}
