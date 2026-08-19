package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	pgErrInvalidTextRepresentation = "22P02" // ej. un UUID mal formado en un WHERE id = $1
	pgErrUniqueViolation           = "23505"
)

func isPgError(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
