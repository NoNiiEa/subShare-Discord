package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/NoNiiEa/subShare-Discord/src/models"
	"github.com/NoNiiEa/subShare-Discord/src/exception"
)

type GroupRepository interface {
	Create(ctx context.Context, group *models.Group) (*models.Group, error)
	Update(ctx context.Context, groupId int, g *models.Group) error
	GetById(ctx context.Context, groupId int) (*models.Group, error)
	GetByGuildId(ctx context.Context, guildId string) ([]models.Group, error)
	GetByDueDay(ctx context.Context, dueDay int) ([]models.Group, error)
}

type groupRepo struct {
	db *sql.DB
}

func NewGroupRepository(db *sql.DB) GroupRepository {
	return &groupRepo{db: db}
}

func (r *groupRepo) Create(ctx context.Context, g *models.Group) (*models.Group, error) {
    membersJSON, err := json.Marshal(g.Members)
    if err != nil {
        return nil, err
    }

    paymentJSON, err := json.Marshal(g.Payment)
    if err != nil {
        return nil, err
    }

    const q = `
    INSERT INTO groups (
        name,
        amount,
        amount_per_member,
        due_day,
        members_json,
        discord_guild_id,
        owner_discord_id,
        payment,
        created_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);` 

    res, err := r.db.ExecContext(ctx, q,
        g.Name,
        g.Amount,
        g.AmountPerMember,
        g.DueDay,
        string(membersJSON),
        g.DiscordGuildID,
        g.OwnerDiscordID,
        string(paymentJSON),
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

	_, err = r.db.ExecContext(ctx, q, g.Name, g.Amount, g.AmountPerMember, g.DueDay, string(membersJSON), g.DiscordGuildID, g.OwnerDiscordID, string(paymentJSON), groupId)
	return err
}

func (r *groupRepo) GetById(ctx context.Context, groupId int) (*models.Group, error) {
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
	WHERE id = ?
	`

	row := r.db.QueryRowContext(ctx, q, groupId)
	var (
		g models.Group
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
			return nil, exception.ErrNotFound
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
	g.CreatedAt = t

	return &g, nil
}

func (r *groupRepo) GetByGuildId(ctx context.Context, guildId string) ([]models.Group, error) {
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
	WHERE discord_guild_id = ?
	`

	rows, err := r.db.QueryContext(ctx, q, guildId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Group

	for rows.Next() {
		var (
			g models.Group
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
		g.CreatedAt = t

		result = append(result, g)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *groupRepo) GetByDueDay(ctx context.Context, dueDay int) ([]models.Group, error) {
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
	WHERE due_day = ?
	`

	rows, err := r.db.QueryContext(ctx, q, dueDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Group

	for rows.Next() {
		var (
			g models.Group
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
		g.CreatedAt = t

		result = append(result, g)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
