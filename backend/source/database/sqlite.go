package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/NoNiiEa/subShare-Discord/source/group"
	"github.com/NoNiiEa/subShare-Discord/source/bill"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) InitSchema(ctx context.Context) error {
	const createGroupsTable = `
CREATE TABLE IF NOT EXISTS groups (
    id                INTEGER PRIMARY KEY,
    name              TEXT NOT NULL,
    amount            REAL NOT NULL,
	amount_per_member REAL NOT NULL,
    due_day           INTEGER NOT NULL,
    members_json      TEXT NOT NULL,
    discord_guild_id  TEXT NOT NULL,
    owner_discord_id  TEXT NOT NULL,
	payment			  TEXT NOT NULL,
    created_at        TEXT NOT NULL
);`
	_, err := s.db.ExecContext(ctx, createGroupsTable)
	if err != nil {
		return err
	}

	const createBillsTable = `
	CREATE TABLE IF NOT EXISTS bills (
		id               INTEGER PRIMARY KEY,
    	group_id         INTEGER NOT NULL,
 		member_id        TEXT NOT NULL,         -- Discord user ID
    	year             INTEGER NOT NULL,      -- e.g. 2026
    	month            INTEGER NOT NULL,      -- 1-12

    	amount_due       REAL NOT NULL,
    	amount_paid      REAL NOT NULL DEFAULT 0,
    	currency         TEXT NOT NULL,         -- e.g. "THB"
    	status           TEXT NOT NULL,         -- pending/submitted/verified/rejected/canceled

    	description      TEXT,

    	proof_json	     TEXT,

    	created_at       TEXT NOT NULL,
    	updated_at       TEXT NOT NULL,
    	submitted_at     TEXT,
    	verified_at      TEXT,
    	rejected_at      TEXT

	);
	`

	_, err = s.db.ExecContext(ctx, createBillsTable)
	if err != nil {
		return err
	}

	return nil
}

func (s *SQLiteStore) NextGroupID(ctx context.Context) (int64, error) {
	const q = `SELECT COALESCE(MAX(id), 0) + 1 FROM groups;`

	var nextID int64
	if err := s.db.QueryRowContext(ctx, q).Scan(&nextID); err != nil {
		return 0, err
	}
	return nextID, nil
}

func (s *SQLiteStore) SaveGroup(ctx context.Context, g group.Group) error {
	membersJSON, err := json.Marshal(g.Members)
	if err != nil {
		return err
	}

	paymentJSON, err := json.Marshal(g.Payment)
	if err != nil {
		return err
	}

	const q = `
INSERT INTO groups (
    id,
    name,
    amount,
	amount_per_member,
    due_day,
    members_json,
    discord_guild_id,
    owner_discord_id,
	payment,
    created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

	_, err = s.db.ExecContext(ctx, q,
		g.ID,
		g.Name,
		g.Amount,
		g.AmountPerMember,
		g.DueDay,
		string(membersJSON),
		g.DiscordGuildID,
		g.OwnerDiscordID,
		string(paymentJSON),
		g.CreateAt.Format(time.RFC3339),
	)
	return err
}

var ErrNotFound = errors.New("store: not found")

func (s *SQLiteStore) GetGroup(ctx context.Context, id int64) (*group.Group, error) {
	const q = `
SELECT
    id,
    name,
    amount,
	amount_per_member,
    due_day,
    members_json,
    discord_guild_id,
    owner_discord_id,
	payment,
    created_at
FROM groups
WHERE id = ?;`

	row := s.db.QueryRowContext(ctx, q, id)
	var (
		g group.Group
		membersJSON string
		paymentJSON string
		createdAtStr string
	)

	if err := row.Scan(
		&g.ID,
		&g.Name,
		&g.Amount,
		&g.AmountPerMember,
		&g.DueDay,
		&membersJSON,
		&g.DiscordGuildID,
		&g.OwnerDiscordID,
		&paymentJSON,
		&createdAtStr,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if err := json.Unmarshal([]byte(membersJSON), &g.Members); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(paymentJSON), &g.Payment); err != nil {
		return nil, err
	}

	t, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, err
	}
	g.CreateAt = t

	return &g, nil
}

func (s *SQLiteStore) DeleteGroup(ctx context.Context, id int64) error {
	const q = `DELETE FROM groups
	WHERE id = ?`

	_, err := s.db.ExecContext(ctx, q, id)
	return err
}

func (s *SQLiteStore) UpdateGroup(ctx context.Context, id int64, g group.Group) error {
	membersJSON, err := json.Marshal(g.Members)
	if err != nil {
		return err
	}
	paymentJSON, err := json.Marshal(g.Payment)
	if err != nil {
		return err
	}

	const q = `
	UPDATE groups
	SET 
    name = ?,
    amount = ?,
	amount_per_member = ?,
    due_day = ?,
    members_json = ?,
    discord_guild_id = ?,
    owner_discord_id = ?,
	payment = ?
	WHERE id = ?
	`

	_, err = s.db.ExecContext(ctx, q, g.Name, g.Amount, g.AmountPerMember, g.DueDay, string(membersJSON), g.DiscordGuildID, g.OwnerDiscordID, string(paymentJSON), id)
	return err
}

func (s *SQLiteStore) GetGroupByDueday(ctx context.Context, dueDay int) ([]group.Group, error){
	const q = `
	SELECT
    id,
    name,
    amount,
	amount_per_member,
    due_day,
    members_json,
    discord_guild_id,
    owner_discord_id,
	payment,
    created_at
	FROM groups
	WHERE due_day = ?;
	`

	rows, err := s.db.QueryContext(ctx, q, dueDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []group.Group

	for rows.Next() {
		var (
			g group.Group
			membersJSON string
			paymentJSON string
			createAtStr string
		)

		if err := rows.Scan(
			&g.ID,
			&g.Name,
			&g.Amount,
			&g.AmountPerMember,
			&g.DueDay,
			&membersJSON,
			&g.DiscordGuildID,
			&g.OwnerDiscordID,
			&paymentJSON,
			&createAtStr,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(membersJSON), &g.Members); err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(paymentJSON), &g.Payment); err != nil {
			return nil, err
		}

		t, err := time.Parse(time.RFC3339, createAtStr)
		if err != nil {
			return nil, err
		}
		g.CreateAt = t

		result = append(result, g)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *SQLiteStore) SaveBill(ctx context.Context, b bill.Bill) error {
	const q = `
INSERT INTO bills (
    id,
    group_id,
    member_id,
    year,
    month,
    amount_due,
    amount_paid,
    currency,
    status,
    description,
    proof_json,
    created_at,
    updated_at,
    submitted_at,
    verified_at,
    rejected_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
`

	// handle nullable times
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

	_, err := s.db.ExecContext(ctx, q,
		b.ID,
		b.GroupID,
		b.MemberID,
		b.Year,
		b.Month,
		b.AmountDue,
		b.AmountPaid,
		b.Currency,
		string(b.Status), // or b.Status if it's already string
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

func (s *SQLiteStore) NextBillID(ctx context.Context) (int64, error) {
	const q = `SELECT COALESCE(MAX(id), 0) + 1 FROM bills;`

	var nextID int64
	if err := s.db.QueryRowContext(ctx, q).Scan(&nextID); err != nil {
		return 0, err
	}
	return nextID, nil
}

func (s *SQLiteStore) GetBillByID(ctx context.Context, id int64) (*bill.Bill, error) {
	const q = `
SELECT
    id,
    group_id,
    member_id,
    year,
    month,
    amount_due,
    amount_paid,
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
WHERE id = ?;
`

	row := s.db.QueryRowContext(ctx, q, id)

	var b bill.Bill
	var createdAt, updatedAt string
	var submittedAt, verifiedAt, rejectedAt *string

	err := row.Scan(
		&b.ID,
		&b.GroupID,
		&b.MemberID,
		&b.Year,
		&b.Month,
		&b.AmountDue,
		&b.AmountPaid,
		&b.Currency,
		&b.Status,
		&b.Description,
		&b.ProofJSON,
		&createdAt,
		&updatedAt,
		&submittedAt,
		&verifiedAt,
		&rejectedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Convert timestamps
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

func (s *SQLiteStore) GetBillsByGroupAndMember(ctx context.Context, groupID int64, memberID string) ([]bill.Bill, error) {
	const q = `
SELECT
    id,
    group_id,
    member_id,
    year,
    month,
    amount_due,
    amount_paid,
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
WHERE group_id = ? AND member_id = ?
ORDER BY year DESC, month DESC;
`

	rows, err := s.db.QueryContext(ctx, q, groupID, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []bill.Bill

	for rows.Next() {
		var b bill.Bill
		var createdAt, updatedAt string
		var submittedAt, verifiedAt, rejectedAt *string

		if err := rows.Scan(
			&b.ID,
			&b.GroupID,
			&b.MemberID,
			&b.Year,
			&b.Month,
			&b.AmountDue,
			&b.AmountPaid,
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

	if len(result) == 0 {
		return nil, ErrNotFound
	}

	return result, nil
}

func (s *SQLiteStore) GetBillByGroupMemberCycle(ctx context.Context, groupID int64, memberID string, year, month int) (*bill.Bill, error) {
	const q = `
SELECT
    id,
    group_id,
    member_id,
    year,
    month,
    amount_due,
    amount_paid,
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
WHERE group_id = ? AND member_id = ? AND year = ? AND month = ?
LIMIT 1;
`

	row := s.db.QueryRowContext(ctx, q, groupID, memberID, year, month)

	var b bill.Bill
	var createdAt, updatedAt string
	var submittedAt, verifiedAt, rejectedAt *string

	err := row.Scan(
		&b.ID,
		&b.GroupID,
		&b.MemberID,
		&b.Year,
		&b.Month,
		&b.AmountDue,
		&b.AmountPaid,
		&b.Currency,
		&b.Status,
		&b.Description,
		&b.ProofJSON,
		&createdAt,
		&updatedAt,
		&submittedAt,
		&verifiedAt,
		&rejectedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
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

func (s *SQLiteStore) GetBillsByMemberID(ctx context.Context, memberID string) ([]bill.Bill, error) {
	const q = `
SELECT
    id,
    group_id,
    member_id,
    year,
    month,
    amount_due,
    amount_paid,
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
WHERE member_id = ?
ORDER BY year DESC, month DESC;
`

	rows, err := s.db.QueryContext(ctx, q, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []bill.Bill

	for rows.Next() {
		var b bill.Bill
		var createdAt, updatedAt string
		var submittedAt, verifiedAt, rejectedAt *string

		if err := rows.Scan(
			&b.ID,
			&b.GroupID,
			&b.MemberID,
			&b.Year,
			&b.Month,
			&b.AmountDue,
			&b.AmountPaid,
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

	if len(result) == 0 {
		return nil, ErrNotFound
	}

	return result, nil
}

func (s *SQLiteStore) GetBillsByGroupID(ctx context.Context, groupID int64) ([]bill.Bill, error) {
	const q = `
SELECT
    id,
    group_id,
    member_id,
    year,
    month,
    amount_due,
    amount_paid,
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
ORDER BY year DESC, month DESC, member_id ASC;
`

	rows, err := s.db.QueryContext(ctx, q, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []bill.Bill

	for rows.Next() {
		var b bill.Bill
		var createdAt, updatedAt string
		var submittedAt, verifiedAt, rejectedAt *string

		if err := rows.Scan(
			&b.ID,
			&b.GroupID,
			&b.MemberID,
			&b.Year,
			&b.Month,
			&b.AmountDue,
			&b.AmountPaid,
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

	if len(result) == 0 {
		return nil, ErrNotFound
	}

	return result, nil
}

func (s *SQLiteStore) UpdateBill(ctx context.Context, b bill.Bill) (*bill.Bill, error) {
	// ensure UpdatedAt is set
	if b.UpdatedAt.IsZero() {
		b.UpdatedAt = time.Now().UTC()
	}

	const q = `
UPDATE bills
SET
    group_id    = ?,
    member_id   = ?,
    year        = ?,
    month       = ?,
    amount_due  = ?,
    amount_paid = ?,
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

	// nullable timestamps
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

	res, err := s.db.ExecContext(ctx, q,
		b.GroupID,
		b.MemberID,
		b.Year,
		b.Month,
		b.AmountDue,
		b.AmountPaid,
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
		return nil, err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrNotFound
	}

	return &b, nil
}
