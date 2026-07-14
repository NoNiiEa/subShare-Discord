package repository

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// DBTX is the subset of *sql.DB / *sql.Tx used by the repositories, allowing a
// repository to run either directly against the pool or inside a transaction
// (see the WithTx methods).
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows so scan helpers can be
// shared between single-row and multi-row queries.
type rowScanner interface {
	Scan(dest ...any) error
}

// nullableTime formats a non-nil time as RFC3339 for use as a SQL argument,
// returning nil for a nil pointer (so the column is stored as NULL).
func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}

// parseTime parses a stored RFC3339 timestamp, logging (rather than failing) on
// malformed data and returning the zero time.
func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		log.Printf("[repository] failed to parse time %q: %v", s, err)
		return time.Time{}
	}
	return t
}

// parseNullableTime parses an optional stored RFC3339 timestamp.
func parseNullableTime(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t := parseTime(*s)
	return &t
}
