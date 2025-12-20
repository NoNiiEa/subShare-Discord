// cmd/api/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/NoNiiEa/subShare-Discord/src/api"
	"github.com/NoNiiEa/subShare-Discord/src/api/handlers"
	"github.com/NoNiiEa/subShare-Discord/src/database"
	"github.com/NoNiiEa/subShare-Discord/src/easySlip"
	"github.com/NoNiiEa/subShare-Discord/src/repository"
	"github.com/NoNiiEa/subShare-Discord/src/service"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load("config/.env"); err != nil {
		log.Printf("warning: could not load config/.env: %v", err)
	}
	
	db, err := database.NewDatabase()
	if err != nil {
		log.Fatal(err)
	}

	slipBaseURL := os.Getenv("EASISLIP_API_URL")
	slipApiKEY := os.Getenv("EASISLIP_API_TOKEN")

	groupRepo := repository.NewGroupRepository(db)
	billRepo := repository.NewBillRepository(db)
	groupService := service.NewGroupService(groupRepo, billRepo)
	easySlipClient := easyslip.NewClient(slipBaseURL, slipApiKEY) 
	billService := service.NewBillService(billRepo, groupRepo, easySlipClient)
	groupHandler := handlers.NewGroupHandler(groupService)
	billHandler := handlers.NewBillHandler(billService)

	router := api.NewRouter(groupHandler, billHandler)

	startDailyPaymentReset(context.Background(), groupService)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	log.Printf("subShare API listening on %s ", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func startDailyPaymentReset(ctx context.Context, svc service.GroupService) {
    go func() {
        // Ticker: Check every hour (or every 30 mins to be safe)
        ticker := time.NewTicker(time.Hour) 
        defer ticker.Stop()

        // Initialize lastDay to current day so we don't run immediately on startup
        // (Unless you WANT it to run on startup? If so, set to -1)
        var lastDay = time.Now().Day()

        for {
            select {
            case <-ctx.Done():
                log.Println("Stopping payment reset worker...")
                return

            case <-ticker.C:
                now := time.Now()
                day := now.Day()

                // If we already ran today, skip
                if day == lastDay {
                    continue
                }

                // Update lastDay immediately so we don't retry if the DB is slow
                lastDay = day

                log.Printf("Starting bill cycle for day %d", day)

                // 1. Get groups due today
                groups, err := svc.GetByDueDay(ctx, day)
                if err != nil {
                    log.Printf("Error fetching groups for day %d: %v", day, err)
                    continue
                }

                // 2. Process them
                for _, g := range groups { // Removed parentheses
                    if err := svc.CreateBillCycle(ctx, &g); err != nil {
                        // Log error but continue loop for other groups
                        log.Printf("Failed to create bill cycle for Group ID %d: %v", g.ID, err)
                    }
                }

                log.Printf("Completed bill cycle for day %d", day)
            }
        }
    }() // <--- FIXED: Added executing parentheses
}