package utils

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func HasCode(err error, code string) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) && pgErr.Code == code
}
