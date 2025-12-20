package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/NoNiiEa/subShare-Discord/src/exception"
	"github.com/NoNiiEa/subShare-Discord/src/models"
)

type BillRepository interface {
	Create(ctx context.Context, b *models.Bill) error
	Update(ctx context.Context, billId int, b *models.Bill) error
	GetByGuildId(ctx context.Context, guildId string) ([]models.Bill, error)
	GetById(ctx context.Context, billId int) (*models.Bill, error)
	DeleteById(ctx context.Context, billId int) error
	GetByGroupID(ctx context.Context, groupId int) ([]models.Bill, error)
}

type billRepo struct {
	db *sql.DB
}

func NewBillRepository(db *sql.DB) BillRepository {
	return &billRepo{db: db}
}

func (r *billRepo) Create(ctx context.Context, b *models.Bill) error {
	const q = `
	INSERT INTO bills (
		group_id,
		guild_id,
		member_id,
		year,
		month,
		amount_due,
		currency,
		status,
		description,
		proof_json,
		created_at,
		updated_at,
		submitted_at,
		verified_at,
		rejected_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`

	var submittedAt, verifiedAt, rejectedAt interface{}

	if b.SubmittedAt != nil {
		submittedAt = b.SubmittedAt.Format(time.RFC3339)
	} else {
		submittedAt = nil
	}

	if b.VerifiedAt != nil {
		verifiedAt = b.VerifiedAt.Format(time.RFC3339)
	} else {
		verifiedAt = nil
	}

	if b.RejectedAt != nil {
		rejectedAt = b.RejectedAt.Format(time.RFC3339)
	} else {
		rejectedAt = nil
	}

	_, err := r.db.ExecContext(ctx, q,
		b.GroupID,
		b.GuildID,
		b.MemberID,
		b.Year,
		b.Month,
		b.AmountDue,
		b.Currency,
		string(b.Status),
		b.Description,
		b.ProofJSON,
		b.CreatedAt.Format(time.RFC3339),
		b.UpdatedAt.Format(time.RFC3339),
		submittedAt,
		verifiedAt,
		rejectedAt,
	)

	return err
}

func (r *billRepo) Update(ctx context.Context, billId int, b *models.Bill) error {
	if b.UpdatedAt.IsZero() {
		b.UpdatedAt = time.Now().UTC()
	}

	const q = `
	UPDATE bills
	SET
		group_id    = ?,
		guild_id    = ?,
		member_id   = ?,
		year        = ?,
		month       = ?,
		amount_due  = ?,
		currency    = ?,
		status      = ?,
		description = ?,
		proof_json  = ?,
		created_at  = ?,   -- usually unchanged
		updated_at  = ?,
		submitted_at = ?,
		verified_at  = ?,
		rejected_at  = ?
	WHERE id = ?;
	`

	var submittedAt, verifiedAt, rejectedAt interface{}

	if b.SubmittedAt != nil {
		submittedAt = b.SubmittedAt.Format(time.RFC3339)
	} else {
		submittedAt = nil
	}

	if b.VerifiedAt != nil {
		verifiedAt = b.VerifiedAt.Format(time.RFC3339)
	} else {
		verifiedAt = nil
	}

	if b.RejectedAt != nil {
		rejectedAt = b.RejectedAt.Format(time.RFC3339)
	} else {
		rejectedAt = nil
	}

	res, err := r.db.ExecContext(ctx, q,
		b.GroupID,
		b.GuildID,
		b.MemberID,
		b.Year,
		b.Month,
		b.AmountDue,
		b.Currency,
		string(b.Status),
		b.Description,
		b.ProofJSON,
		b.CreatedAt.Format(time.RFC3339),
		b.UpdatedAt.Format(time.RFC3339),
		submittedAt,
		verifiedAt,
		rejectedAt,
		b.ID,
	)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return exception.ErrNotFound
	}

	return nil
}

func (r *billRepo) GetByGuildId(ctx context.Context, guildId string) ([]models.Bill, error) {
	const q = `
	SELECT
		id,
		group_id,
		guild_id,
		member_id,
		year,
		month,
		amount_due,
		currency,
		status,
		description,
		proof_json,
		created_at,
		updated_at,
		submitted_at,
		verified_at,
		rejected_at
	FROM bills
	WHERE guild_id = ?
	`

	rows, err := r.db.QueryContext(ctx, q, guildId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Bill

	for rows.Next() {
		var b models.Bill
		var createdAt, updatedAt string
		var submittedAt, verifiedAt, rejectedAt *string

		if err := rows.Scan(
			&b.ID,
			&b.GroupID,
			&b.GuildID,
			&b.MemberID,
			&b.Year,
			&b.Month,
			&b.AmountDue,
			&b.Currency,
			&b.Status,
			&b.Description,
			&b.ProofJSON,
			&createdAt,
			&updatedAt,
			&submittedAt,
			&verifiedAt,
			&rejectedAt,
		); err != nil {
			return nil, err
		}

		b.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		b.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

		if submittedAt != nil {
			t, _ := time.Parse(time.RFC3339, *submittedAt)
			b.SubmittedAt = &t
		}
		if verifiedAt != nil {
			t, _ := time.Parse(time.RFC3339, *verifiedAt)
			b.VerifiedAt = &t
		}
		if rejectedAt != nil {
			t, _ := time.Parse(time.RFC3339, *rejectedAt)
			b.RejectedAt = &t
		}

		result = append(result, b)
	}

	return result, nil
}

func (r *billRepo) GetById(ctx context.Context, billId int) (*models.Bill, error) {
	const q = `
	SELECT
		id,
		group_id,
		guild_id,
		member_id,
		year,
		month,
		amount_due,
		currency,
		status,
		description,
		proof_json,
		created_at,
		updated_at,
		submitted_at,
		verified_at,
		rejected_at
	FROM bills
	WHERE id = ?
	`

	row := r.db.QueryRowContext(ctx, q, billId)
	var b models.Bill
	var createdAt, updatedAt string
	var submittedAt, verifiedAt, rejectedAt *string

	if err := row.Scan(
		&b.ID,
		&b.GroupID,
		&b.GuildID,
		&b.MemberID,
		&b.Year,
		&b.Month,
		&b.AmountDue,
		&b.Currency,
		&b.Status,
		&b.Description,
		&b.ProofJSON,
		&createdAt,
		&updatedAt,
		&submittedAt,
		&verifiedAt,
		&rejectedAt,
	); err != nil {
		return nil, err
	}

	b.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	b.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	if submittedAt != nil {
		t, _ := time.Parse(time.RFC3339, *submittedAt)
		b.SubmittedAt = &t
	}
	if verifiedAt != nil {
		t, _ := time.Parse(time.RFC3339, *verifiedAt)
		b.VerifiedAt = &t
	}
	if rejectedAt != nil {
		t, _ := time.Parse(time.RFC3339, *rejectedAt)
		b.RejectedAt = &t
	}

	return &b, nil
}

func (r *billRepo) DeleteById(ctx context.Context, billId int) error {
	const q = `
	DELETE FROM bills
	WHERE id = ?;
	`

	res, err := r.db.ExecContext(ctx, q, billId)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return exception.ErrNotFound
	}

	return nil
}

func (r *billRepo) GetByGroupID(ctx context.Context, groupId int) ([]models.Bill, error) {
	const q = `
	SELECT
		id,
		group_id,
		guild_id,
		member_id,
		year,
		month,
		amount_due,
		currency,
		status,
		description,
		proof_json,
		created_at,
		updated_at,
		submitted_at,
		verified_at,
		rejected_at
	FROM bills
	WHERE group_id = ?
	`

	rows, err := r.db.QueryContext(ctx, q, groupId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Bill

	for rows.Next() {
		var b models.Bill
		var createdAt, updatedAt string
		var submittedAt, verifiedAt, rejectedAt *string

		if err := rows.Scan(
			&b.ID,
			&b.GroupID,
			&b.GuildID,
			&b.MemberID,
			&b.Year,
			&b.Month,
			&b.AmountDue,
			&b.Currency,
			&b.Status,
			&b.Description,
			&b.ProofJSON,
			&createdAt,
			&updatedAt,
			&submittedAt,
			&verifiedAt,
			&rejectedAt,
		); err != nil {
			return nil, err
		}

		b.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		b.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

		if submittedAt != nil {
			t, _ := time.Parse(time.RFC3339, *submittedAt)
			b.SubmittedAt = &t
		}
		if verifiedAt != nil {
			t, _ := time.Parse(time.RFC3339, *verifiedAt)
			b.VerifiedAt = &t
		}
		if rejectedAt != nil {
			t, _ := time.Parse(time.RFC3339, *rejectedAt)
			b.RejectedAt = &t
		}

		result = append(result, b)
	}

	return result, nil
}