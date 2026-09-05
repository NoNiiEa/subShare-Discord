package repository

import (
	"context"
	"strings"
	"time"

	"github.com/NoNiiEa/subShare-Discord/src/exception"
	"github.com/NoNiiEa/subShare-Discord/src/models"
)

type BillRepository interface {
	Create(ctx context.Context, b *models.Bill) error
	Update(ctx context.Context, billId int, b *models.Bill) error
	GetByGuildId(ctx context.Context, guildId string) ([]models.Bill, error)
	GetById(ctx context.Context, billId int) (*models.Bill, error)
	GetByIds(ctx context.Context, ids []int) ([]models.Bill, error)
	DeleteById(ctx context.Context, billId int) error
	GetByGroupID(ctx context.Context, groupId int) ([]models.Bill, error)
	// WithTx returns a repository bound to the given transaction.
	WithTx(tx DBTX) BillRepository
}

type billRepo struct {
	db DBTX
}

func NewBillRepository(db DBTX) BillRepository {
	return &billRepo{db: db}
}

func (r *billRepo) WithTx(tx DBTX) BillRepository {
	return &billRepo{db: tx}
}

const billColumns = `id, group_id, guild_id, member_id, year, month,
	amount_due, currency, status, description, proof_json,
	created_at, updated_at, submitted_at, verified_at, rejected_at`

// scanBill reads one bill row (16 columns in billColumns order).
func scanBill(s rowScanner) (*models.Bill, error) {
	var b models.Bill
	var createdAt, updatedAt string
	var submittedAt, verifiedAt, rejectedAt *string

	if err := s.Scan(
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

	b.CreatedAt = parseTime(createdAt)
	b.UpdatedAt = parseTime(updatedAt)
	b.SubmittedAt = parseNullableTime(submittedAt)
	b.VerifiedAt = parseNullableTime(verifiedAt)
	b.RejectedAt = parseNullableTime(rejectedAt)

	return &b, nil
}

// queryBills runs a bill SELECT with the given WHERE clause and arguments.
func (r *billRepo) queryBills(ctx context.Context, whereClause string, args ...any) ([]models.Bill, error) {
	q := `SELECT ` + billColumns + ` FROM bills WHERE ` + whereClause

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Bill
	for rows.Next() {
		b, err := scanBill(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *billRepo) Create(ctx context.Context, b *models.Bill) error {
	const q = `
	INSERT INTO bills (
		group_id, guild_id, member_id, year, month, amount_due, currency,
		status, description, proof_json, created_at, updated_at,
		submitted_at, verified_at, rejected_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`

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
		nullableTime(b.SubmittedAt),
		nullableTime(b.VerifiedAt),
		nullableTime(b.RejectedAt),
	)
	if err != nil {
		return err
	}

	// Assign the generated id back, so callers that need to refer to the bill
	// they just created (e.g. a balance-remaining bill) do not have to search
	// for it afterwards.
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	b.ID = int(id)

	return nil
}

func (r *billRepo) Update(ctx context.Context, billId int, b *models.Bill) error {
	if b.UpdatedAt.IsZero() {
		b.UpdatedAt = time.Now().UTC()
	}

	const q = `
	UPDATE bills SET
		group_id = ?, guild_id = ?, member_id = ?, year = ?, month = ?,
		amount_due = ?, currency = ?, status = ?, description = ?, proof_json = ?,
		created_at = ?, updated_at = ?, submitted_at = ?, verified_at = ?, rejected_at = ?
	WHERE id = ?;
	`

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
		nullableTime(b.SubmittedAt),
		nullableTime(b.VerifiedAt),
		nullableTime(b.RejectedAt),
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
	return r.queryBills(ctx, "guild_id = ?", guildId)
}

func (r *billRepo) GetByGroupID(ctx context.Context, groupId int) ([]models.Bill, error) {
	return r.queryBills(ctx, "group_id = ?", groupId)
}

// GetByIds returns the bills matching ids. Missing ids are simply absent from
// the result, so callers that require every id to exist must check the length.
func (r *billRepo) GetByIds(ctx context.Context, ids []int) ([]models.Bill, error) {
	if len(ids) == 0 {
		return []models.Bill{}, nil
	}

	placeholders := strings.Repeat("?,", len(ids)-1) + "?"
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	return r.queryBills(ctx, "id IN ("+placeholders+")", args...)
}

func (r *billRepo) GetById(ctx context.Context, billId int) (*models.Bill, error) {
	q := `SELECT ` + billColumns + ` FROM bills WHERE id = ?`
	row := r.db.QueryRowContext(ctx, q, billId)
	return scanBill(row)
}

func (r *billRepo) DeleteById(ctx context.Context, billId int) error {
	const q = `DELETE FROM bills WHERE id = ?;`

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
