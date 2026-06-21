package repository

import (
	"context"
	"database/sql"
	"errors"
	"gophkeeper/internal/server/db/conn"
	"gophkeeper/internal/server/model"
	"gophkeeper/internal/shared/errors/labelerrors"
	"time"
)

type CardRepo struct {
	repoBase
}

const CardListByUserQuery = `SELECT id, user_id, data, version, created_at, updated_at
FROM cards
WHERE user_id = $1 AND deleted_at IS NULL AND ($2::uuid IS NULL OR id > $2::uuid)
ORDER BY id
LIMIT $3 OFFSET $4`

const CardGetByIDQuery = `SELECT id, user_id, data, version, created_at, updated_at
FROM cards
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`

const CardCreateQuery = `INSERT INTO cards(id, user_id, data) VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE SET data = excluded.data, version = cards.version + 1, updated_at = now(), deleted_at = NULL
WHERE cards.user_id = excluded.user_id
RETURNING id, user_id, data, version, created_at, updated_at`

const CardUpdateQuery = `UPDATE cards
SET data = $1, version = version + 1, updated_at = now()
WHERE id = $2 AND user_id = $3 AND version = $4
RETURNING id, user_id, data, version, created_at, updated_at`

const CardDeleteQuery = `UPDATE cards
SET deleted_at = now(), version = version + 1, updated_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`

const CardChangesQuery = `SELECT id, data, version, deleted_at IS NOT NULL, updated_at
FROM cards
WHERE user_id = $1 AND ($2::timestamptz IS NULL OR updated_at > $2::timestamptz)
ORDER BY updated_at`

func NewCardRepo(db *conn.DB) *CardRepo {
	return &CardRepo{repoBase: repoBase{db: db}}
}

// GetByUser returns one chunk of the user's encrypted cards.
func (c CardRepo) GetByUser(ctx context.Context, user *model.User, lastID string, limit, offset int) ([]*model.Card, error) {
	var cursor any
	if lastID != "" {
		cursor = lastID
	}

	rows, err := c.q(ctx).QueryContext(ctx, CardListByUserQuery, user.ID, cursor, limit, offset)
	if err != nil {
		return nil, labelerrors.NewLabelError(labelRepository+".Card.GetByUser.Query", err)
	}
	defer rows.Close()

	var cards []*model.Card
	for rows.Next() {
		var card model.Card
		if err := rows.Scan(
			&card.ID, &card.UserID, &card.Data, &card.Version, &card.CreatedAt, &card.UpdatedAt,
		); err != nil {
			return nil, labelerrors.NewLabelError(labelRepository+".Card.GetByUser.Scan", err)
		}
		cards = append(cards, &card)
	}
	if err := rows.Err(); err != nil {
		return nil, labelerrors.NewLabelError(labelRepository+".Card.GetByUser.Rows", err)
	}
	return cards, nil
}

// GetByID returns a single encrypted card owned by the user.
func (c CardRepo) GetByID(ctx context.Context, user *model.User, id string) (*model.Card, error) {
	var card model.Card
	err := c.q(ctx).QueryRowContext(ctx, CardGetByIDQuery, id, user.ID).
		Scan(&card.ID, &card.UserID, &card.Data, &card.Version, &card.CreatedAt, &card.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrCardNotFound
		}
		return nil, labelerrors.NewLabelError(labelRepository+".Card.GetByID", err)
	}
	return &card, nil
}

// Create stores a new encrypted card for the user.
func (c CardRepo) Create(ctx context.Context, user *model.User, card *model.Card) (*model.Card, error) {
	err := c.q(ctx).QueryRowContext(ctx, CardCreateQuery, card.ID, user.ID, card.Data).
		Scan(&card.ID, &card.UserID, &card.Data, &card.Version, &card.CreatedAt, &card.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrCardNotFound
		}
		return nil, labelerrors.NewLabelError(labelRepository+".Card.Create", err)
	}
	return card, nil
}

// Update overwrites the card's encrypted payload using optimistic versioning.
func (c CardRepo) Update(ctx context.Context, user *model.User, card *model.Card) (*model.Card, error) {
	err := c.q(ctx).QueryRowContext(ctx, CardUpdateQuery, card.Data, card.ID, user.ID, card.Version).
		Scan(&card.ID, &card.UserID, &card.Data, &card.Version, &card.CreatedAt, &card.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrVersionConflict
		}
		return nil, labelerrors.NewLabelError(labelRepository+".Card.Update", err)
	}
	return card, nil
}

// Delete removes a card owned by the user.
func (c CardRepo) Delete(ctx context.Context, user *model.User, id string) error {
	res, err := c.q(ctx).ExecContext(ctx, CardDeleteQuery, id, user.ID)
	if err != nil {
		return labelerrors.NewLabelError(labelRepository+".Card.Delete", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return labelerrors.NewLabelError(labelRepository+".Card.Delete.RowsAffected", err)
	}
	if affected == 0 {
		return model.ErrCardNotFound
	}
	return nil
}

// Changes returns all the user's cards (including tombstones) updated after since.
func (c CardRepo) Changes(ctx context.Context, user *model.User, since time.Time) ([]*model.CardChange, error) {
	var cursor any
	if !since.IsZero() {
		cursor = since
	}

	rows, err := c.q(ctx).QueryContext(ctx, CardChangesQuery, user.ID, cursor)
	if err != nil {
		return nil, labelerrors.NewLabelError(labelRepository+".Card.Changes.Query", err)
	}
	defer rows.Close()

	var changes []*model.CardChange
	for rows.Next() {
		var ch model.CardChange
		if err := rows.Scan(&ch.ID, &ch.Data, &ch.Version, &ch.Deleted, &ch.UpdatedAt); err != nil {
			return nil, labelerrors.NewLabelError(labelRepository+".Card.Changes.Scan", err)
		}
		changes = append(changes, &ch)
	}
	if err := rows.Err(); err != nil {
		return nil, labelerrors.NewLabelError(labelRepository+".Card.Changes.Rows", err)
	}
	return changes, nil
}
