package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type CardsRepo struct {
	repoBase
}
type CardRow struct {
	ID      string
	Data    []byte
	Version int64
}

const cardsListQuery = `SELECT id, data, version FROM cards
WHERE deleted = 0 AND (? = '' OR id > ?)
ORDER BY id LIMIT ?`
const cardsInsertQuery = `INSERT INTO cards (id, data, version, dirty, base_version) VALUES (?, ?, 1, 1, 0)`
const cardsUpdateQuery = `UPDATE cards SET data = ?, version = version + 1, dirty = 1,
base_version = CASE WHEN conflict = 1 THEN COALESCE(server_version, base_version) ELSE base_version END,
conflict = 0, server_blob = NULL, server_version = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
const cardsDeleteQuery = `UPDATE cards SET deleted = 1, dirty = 1, version = version + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`

func NewCardsRepo(conn *sql.DB) *CardsRepo {
	return &CardsRepo{repoBase: repoBase{db: conn}}
}
func (r *CardsRepo) List(ctx context.Context, lastID string, limit int) ([]CardRow, error) {
	rows, err := r.db.QueryContext(ctx, cardsListQuery, lastID, lastID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CardRow
	for rows.Next() {
		var n CardRow
		if err := rows.Scan(&n.ID, &n.Data, &n.Version); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
func (r *CardsRepo) Create(ctx context.Context, data []byte) (CardRow, error) {
	id := uuid.NewString()
	_, err := r.db.ExecContext(ctx,
		cardsInsertQuery,
		id, data)
	if err != nil {
		return CardRow{}, err
	}
	return CardRow{ID: id, Data: data, Version: 1}, nil
}
func (r *CardsRepo) Update(ctx context.Context, id string, data []byte) error {
	_, err := r.db.ExecContext(ctx,
		cardsUpdateQuery,
		data, id)
	return err
}

func (r *CardsRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		cardsDeleteQuery,
		id)
	return err
}
