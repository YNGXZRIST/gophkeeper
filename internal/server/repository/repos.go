package repository

import (
	"context"
	"gophkeeper/internal/server/db/conn"
	"gophkeeper/internal/server/db/trmanager"
	"gophkeeper/internal/shared/errors/labelerrors"
)

const labelRepository = "Repository"

type repoBase struct{ db *conn.DB }

func (b *repoBase) q(ctx context.Context) trmanager.Querier {
	return trmanager.Resolve(ctx, b.db)
}

type scanner interface{ Scan(dest ...any) error }

// queryRows runs query and decodes every result row with scan. It centralizes
// the rows.Next/Scan/Err boilerplate shared by the list-style queries, wrapping
// each failure under label with the phase (.Query/.Scan/.Rows) appended.
func queryRows[T any](ctx context.Context, q trmanager.Querier, label, query string, scan func(scanner) (T, error), args ...any) ([]T, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, labelerrors.NewLabelError(label+".Query", err)
	}
	defer rows.Close()

	var out []T
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, labelerrors.NewLabelError(label+".Scan", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, labelerrors.NewLabelError(label+".Rows", err)
	}
	return out, nil
}

type Repositories struct {
	User         *UserRepo
	RefreshToken *RefreshTokenRepo
	Card         *EntryRepo
	Password     *EntryRepo
	Note         *EntryRepo
	File         *FileRepo
}
