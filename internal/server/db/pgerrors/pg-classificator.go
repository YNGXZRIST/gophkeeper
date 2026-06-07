// Package pgerrors classifies and wraps PostgreSQL errors.
package pgerrors

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// PGErrorClassification defines retry strategy for postgres errors.
type PGErrorClassification int

const (
	// NonRetriable - операцию не следует повторять
	NonRetriable PGErrorClassification = iota

	// Retriable - операцию можно повторить
	Retriable
)

// PostgresErrorClassifier classifies postgres errors as retriable/non-retriable.
type PostgresErrorClassifier struct{}

// NewPostgresErrorClassifier creates a postgres error classifier.
func NewPostgresErrorClassifier() *PostgresErrorClassifier {
	return &PostgresErrorClassifier{}
}

// Classify классифицирует ошибку и возвращает PGErrorClassification
func (c *PostgresErrorClassifier) Classify(err error) PGErrorClassification {
	if err == nil {
		return NonRetriable
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return СlassifyPgError(pgErr)
	}
	return NonRetriable
}

// СlassifyPgError maps postgres SQLSTATE code to retry classification.
func СlassifyPgError(pgErr *pgconn.PgError) PGErrorClassification {

	switch pgErr.Code {
	// Класс 08 - Ошибки соединения
	case pgerrcode.ConnectionException,
		pgerrcode.ConnectionDoesNotExist,
		pgerrcode.ConnectionFailure:
		return Retriable

	// Класс 40 - Откат транзакции
	case pgerrcode.TransactionRollback, // 40000
		pgerrcode.SerializationFailure, // 40001
		pgerrcode.DeadlockDetected:     // 40P01
		return Retriable

	// Класс 57 - Ошибка оператора
	case pgerrcode.CannotConnectNow: // 57P03
		return Retriable
	}

	// Можно добавить более конкретные проверки с использованием констант pgerrcode
	switch pgErr.Code {
	// Класс 22 - Ошибки данных
	case pgerrcode.DataException,
		pgerrcode.NullValueNotAllowedDataException:
		return NonRetriable

	// Класс 23 - Нарушение ограничений целостности
	case pgerrcode.IntegrityConstraintViolation,
		pgerrcode.RestrictViolation,
		pgerrcode.NotNullViolation,
		pgerrcode.ForeignKeyViolation,
		pgerrcode.UniqueViolation,
		pgerrcode.CheckViolation:
		return NonRetriable

	// Класс 42 - Синтаксические ошибки
	case pgerrcode.SyntaxErrorOrAccessRuleViolation,
		pgerrcode.SyntaxError,
		pgerrcode.UndefinedColumn,
		pgerrcode.UndefinedTable,
		pgerrcode.UndefinedFunction:
		return NonRetriable
	}
	return NonRetriable
}
