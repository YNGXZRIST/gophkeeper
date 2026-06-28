package repository

import (
	"database/sql"
)

type repoBase struct{ db *sql.DB }

type Repositories struct {
	Session *SessionRepo
}
