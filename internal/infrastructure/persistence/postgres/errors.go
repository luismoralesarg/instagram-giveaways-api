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

// uniqueViolationConstraint devuelve el nombre del constraint/índice que
// disparó una violación de unicidad, para distinguir entre varias reglas
// que conviven en la misma tabla (ej. campaigns tiene dos).
func uniqueViolationConstraint(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
		return pgErr.ConstraintName, true
	}
	return "", false
}
