package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/joho/godotenv"

	httpserver "github.com/NoNiiEa/subShare-Discord/source/api"
	"github.com/NoNiiEa/subShare-Discord/source/bill"
	"github.com/NoNiiEa/subShare-Discord/source/database"
	"github.com/NoNiiEa/subShare-Discord/source/group"
	"github.com/NoNiiEa/subShare-Discord/source/billVer"
)

func main() {
	// Load env from config/.env (ignore error in prod if you use real env vars)
	if err := godotenv.Load("config/.env"); err != nil {
		log.Printf("warning: could not load config/.env: %v", err)
	}

	// Determine DB path
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		// default: ./data/subshare.db
		if err := os.MkdirAll("data", 0o755); err != nil {
			log.Fatalf("failed to create data dir: %v", err)
		}
		dbPath = filepath.Join("data", "subshare.db")
	} else {
		// ensure directory for custom DB_PATH exists
		dbDir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dbDir, 0o755); err != nil {
			log.Fatalf("failed to create db dir %s: %v", dbDir, err)
		}
	}

	// Open SQLite DB
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("failed to open sqlite db: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	ctx := context.Background()

	sqlStore := database.NewSQLiteStore(db)
	if err := sqlStore.InitSchema(ctx); err != nil {
		log.Fatalf("failed to init schema: %v", err)
	}

	groupSvc := group.NewService(sqlStore)
	billSvc := bill.NewService(sqlStore)
	billVerSvc := billver.NewService(sqlStore, nil,  os.Getenv("EASISLIP_API_URL"), os.Getenv("EASISLIP_API_TOKEN"),)

	server := httpserver.NewServer(groupSvc, billSvc, billVerSvc)

	startDailyPaymentReset(ctx, groupSvc)

	// Determine port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	log.Printf("subShare API listening on %s (db: %s)", addr, dbPath)

	if err := http.ListenAndServe(addr, server); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func startDailyPaymentReset(ctx context.Context, svc *group.Service) {
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()

		var lastDay int = -1
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				day := now.Day()

				if day == lastDay {
					continue
				}
				lastDay = day

				if err := svc.ResetPaymentForDueday(ctx, day); err != nil {
					log.Printf("error resetting payments for day %d: %v", day, err)
				} else {
					log.Printf("reset payments for due_day=%d", day)
				}
			}
		}
	}()
}
