package repository

import (
	"context"
	"gophkeeper/internal/server/db/conn"
	"gophkeeper/internal/server/db/trmanager"
	"gophkeeper/internal/shared/errors/labelerrors"
	"iter"
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

// queryRowsSeq is the streaming twin of queryRows: it yields each decoded row
// lazily and surfaces the first Query/Scan/Rows failure through the error half
// of the pair, after which iteration stops. The query runs when the consumer
// starts ranging, and the rows are closed when ranging ends.
func queryRowsSeq[T any](ctx context.Context, q trmanager.Querier, label, query string, scan func(scanner) (T, error), args ...any) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var zero T
		rows, err := q.QueryContext(ctx, query, args...)
		if err != nil {
			yield(zero, labelerrors.NewLabelError(label+".Query", err))
			return
		}
		defer rows.Close()

		for rows.Next() {
			v, err := scan(rows)
			if err != nil {
				yield(zero, labelerrors.NewLabelError(label+".Scan", err))
				return
			}
			if !yield(v, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(zero, labelerrors.NewLabelError(label+".Rows", err))
		}
	}
}

type Repositories struct {
	User         *UserRepo
	RefreshToken *RefreshTokenRepo
	Card         *EntryRepo
	Password     *EntryRepo
	Note         *EntryRepo
	File         *FileRepo
}
