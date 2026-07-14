package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/NoNiiEa/subShare-Discord/src/exception"
	"github.com/NoNiiEa/subShare-Discord/src/models"
)

type GroupRepository interface {
	Create(ctx context.Context, group *models.Group) (*models.Group, error)
	Update(ctx context.Context, groupId int, g *models.Group) error
	GetById(ctx context.Context, groupId int) (*models.Group, error)
	GetByGuildId(ctx context.Context, guildId string) ([]models.Group, error)
	GetByDueDay(ctx context.Context, dueDay int) ([]models.Group, error)
	DeleteById(ctx context.Context, groupId int) error
	// WithTx returns a repository bound to the given transaction.
	WithTx(tx DBTX) GroupRepository
}

type groupRepo struct {
	db DBTX
}

func NewGroupRepository(db DBTX) GroupRepository {
	return &groupRepo{db: db}
}

func (r *groupRepo) WithTx(tx DBTX) GroupRepository {
	return &groupRepo{db: tx}
}

const groupColumns = `id, name, amount, amount_per_member, due_day,
	members_json, discord_guild_id, owner_discord_id, payment, created_at`

// marshalGroupJSON serializes the JSON TEXT columns (members + payment).
func marshalGroupJSON(g *models.Group) (members string, payment string, err error) {
	membersJSON, err := json.Marshal(g.Members)
	if err != nil {
		return "", "", err
	}
	paymentJSON, err := json.Marshal(g.Payment)
	if err != nil {
		return "", "", err
	}
	return string(membersJSON), string(paymentJSON), nil
}

// scanGroup reads one group row (10 columns in groupColumns order), decoding the
// embedded members/payment JSON.
func scanGroup(s rowScanner) (*models.Group, error) {
	var (
		g           models.Group
		membersJSON string
		paymentJSON string
		createdAt   string
	)

	if err := s.Scan(
		&g.ID,
		&g.Name,
		&g.Amount,
		&g.AmountPerMember,
		&g.DueDay,
		&membersJSON,
		&g.DiscordGuildID,
		&g.OwnerDiscordID,
		&paymentJSON,
		&createdAt,
	); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(membersJSON), &g.Members); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(paymentJSON), &g.Payment); err != nil {
		return nil, err
	}
	g.CreatedAt = parseTime(createdAt)

	return &g, nil
}

// queryGroups runs a group SELECT with a single-argument WHERE clause.
func (r *groupRepo) queryGroups(ctx context.Context, whereClause string, arg any) ([]models.Group, error) {
	q := `SELECT ` + groupColumns + ` FROM groups WHERE ` + whereClause

	rows, err := r.db.QueryContext(ctx, q, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Group
	for rows.Next() {
		g, err := scanGroup(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *groupRepo) Create(ctx context.Context, g *models.Group) (*models.Group, error) {
	membersJSON, paymentJSON, err := marshalGroupJSON(g)
	if err != nil {
		return nil, err
	}

	const q = `
	INSERT INTO groups (
		name, amount, amount_per_member, due_day, members_json,
		discord_guild_id, owner_discord_id, payment, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`

	res, err := r.db.ExecContext(ctx, q,
		g.Name,
		g.Amount,
		g.AmountPerMember,
		g.DueDay,
		membersJSON,
		g.DiscordGuildID,
		g.OwnerDiscordID,
		paymentJSON,
		g.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	g.ID = int(id)
	return g, nil
}

func (r *groupRepo) Update(ctx context.Context, groupId int, g *models.Group) error {
	membersJSON, paymentJSON, err := marshalGroupJSON(g)
	if err != nil {
		return err
	}

	const q = `
	UPDATE groups SET
		name = ?, amount = ?, amount_per_member = ?, due_day = ?,
		members_json = ?, discord_guild_id = ?, owner_discord_id = ?, payment = ?
	WHERE id = ?
	`

	_, err = r.db.ExecContext(ctx, q,
		g.Name,
		g.Amount,
		g.AmountPerMember,
		g.DueDay,
		membersJSON,
		g.DiscordGuildID,
		g.OwnerDiscordID,
		paymentJSON,
		groupId,
	)
	return err
}

func (r *groupRepo) GetById(ctx context.Context, groupId int) (*models.Group, error) {
	q := `SELECT ` + groupColumns + ` FROM groups WHERE id = ?`
	row := r.db.QueryRowContext(ctx, q, groupId)

	g, err := scanGroup(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, exception.ErrNotFound
		}
		return nil, err
	}
	return g, nil
}

func (r *groupRepo) GetByGuildId(ctx context.Context, guildId string) ([]models.Group, error) {
	return r.queryGroups(ctx, "discord_guild_id = ?", guildId)
}

func (r *groupRepo) GetByDueDay(ctx context.Context, dueDay int) ([]models.Group, error) {
	return r.queryGroups(ctx, "due_day = ?", dueDay)
}

func (r *groupRepo) DeleteById(ctx context.Context, groupId int) error {
	const q = `DELETE FROM groups WHERE id = ?`

	res, err := r.db.ExecContext(ctx, q, groupId)
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
