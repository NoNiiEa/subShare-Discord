package database

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func NewDatabase() (*sql.DB, error) {
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "app.db")
db, err := sql.Open("sqlite3", dbPath+"?cache=shared&mode=rwc&_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	// With WAL mode enabled, SQLite supports concurrent readers and a single writer.
	// We can allow multiple open connections to avoid blocking read queries.
	db.SetMaxOpenConns(10)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := CreateTable(db); err != nil {
		return nil, err
	}

	log.Println("Connected to SQLite database!")
	return db, nil
}

// CreateTable applies the schema to db. Exported so tests can build the real
// schema against an in-memory database.
func CreateTable(db *sql.DB) error {
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

	_, err := db.Exec(createGroupsTable)
	if err != nil {
		return err
	}

	const createBillTable = `
	CREATE TABLE IF NOT EXISTS bills (
		id				INTEGER PRIMARY KEY,
		group_id		TEXT NOT NULL,
		guild_id		TEXT NOT NULL,
		member_id		TEXT NOT NULL,
		year			INTEGER NOT NULL,
		month			INTEGER NOT NULL,
		amount_due		INTEGER NOT NULL,
		currency		TEXT NOT NULL,
		status			TEXT NOT NULL,
		description		TEXT NOT NULL,
		proof_json		TEXT NOT NULL,
		created_at      TEXT NOT NULL,
    	updated_at      TEXT NOT NULL,
    	submitted_at    TEXT,
    	verified_at     TEXT,
    	rejected_at     TEXT
	);
	`
	_, err = db.Exec(createBillTable)
	if err != nil {
		return err
	}

	const createSlipTable = `
	CREATE TABLE IF NOT EXISTS slips (
		id 				INTEGER PRIMARY KEY,
		transRef		TEXT NOT NULL UNIQUE,
		submitted_at	TEXT
	);
	`

	_, err = db.Exec(createSlipTable)
	if err != nil {
		return err
	}

	return nil
}
