package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/NoNiiEa/subShare-Discord/src/models"
)

type SlipRepository interface {
	Create(ctx context.Context, s *models.Slip) error
	GetByTransRef(ctx context.Context, transRef string) (*models.Slip, error)
	// WithTx returns a repository bound to the given transaction.
	WithTx(tx DBTX) SlipRepository
}

type slipRepo struct {
	db DBTX
}

func NewSlipRepository(db DBTX) SlipRepository {
	return &slipRepo{db: db}
}

func (r *slipRepo) WithTx(tx DBTX) SlipRepository {
	return &slipRepo{db: tx}
}

func (r *slipRepo) Create(ctx context.Context, s *models.Slip) error {
	const q = `
	INSERT INTO slips (
		transRef,
		submitted_at
	) VALUES (?, ?);
	`

	_, err := r.db.ExecContext(ctx, q, s.TransRef, s.SubmittedAt)
	if err != nil {
		return err
	}

	return nil
}

func (r *slipRepo) GetByTransRef(ctx context.Context, transRef string) (*models.Slip, error) {
	const q = `
	SELECT id, transRef, submitted_at FROM slips
	WHERE transRef = ?
	`

	var slip models.Slip
	row := r.db.QueryRowContext(ctx, q, transRef)
	err := row.Scan(
		&slip.ID,
		&slip.TransRef,
		&slip.SubmittedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &slip, nil
}