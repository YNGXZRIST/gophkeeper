package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncStateCursor(t *testing.T) {
	db := newTestDB(t)
	repo := NewSyncStateRepo(db)
	ctx := context.Background()

	cur, err := repo.Cursor(ctx, "notes")
	require.NoError(t, err)
	assert.Equal(t, "", cur)

	require.NoError(t, repo.SetCursor(ctx, "notes", "cursor-1"))
	cur, err = repo.Cursor(ctx, "notes")
	require.NoError(t, err)
	assert.Equal(t, "cursor-1", cur)

	require.NoError(t, repo.SetCursor(ctx, "notes", "cursor-2"))
	cur, err = repo.Cursor(ctx, "notes")
	require.NoError(t, err)
	assert.Equal(t, "cursor-2", cur)

	require.NoError(t, repo.SetCursor(ctx, "cards", "c-cur"))
	cur, err = repo.Cursor(ctx, "cards")
	require.NoError(t, err)
	assert.Equal(t, "c-cur", cur)
	cur, err = repo.Cursor(ctx, "notes")
	require.NoError(t, err)
	assert.Equal(t, "cursor-2", cur)
}
