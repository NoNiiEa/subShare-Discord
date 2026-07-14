// cmd/api/main.go
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/NoNiiEa/subShare-Discord/src/api"
	"github.com/NoNiiEa/subShare-Discord/src/api/handlers"
	"github.com/NoNiiEa/subShare-Discord/src/database"
	"github.com/NoNiiEa/subShare-Discord/src/okslip"
	"github.com/NoNiiEa/subShare-Discord/src/repository"
	"github.com/NoNiiEa/subShare-Discord/src/service"
	"github.com/joho/godotenv"
)

func main() {
    if os.Getenv("APP_ENV") != "production" {
        err := godotenv.Load("config/.env")
        if err != nil {
            log.Println("No .env file found, relying on system environment variables")
        }
    }
	
	db, err := database.NewDatabase()
	if err != nil {
		log.Fatal(err)
	}

	loc, err := time.LoadLocation("Asia/Bangkok")
    if err != nil {
        log.Fatalf("Critical error: could not load timezone: %v", err)
    }

    time.Local = loc

	okSlipBaseURL := os.Getenv("SLIPOK_API_URL")
	okSlipApiKEY := os.Getenv("SLIPOK_API_KEY")

	groupRepo := repository.NewGroupRepository(db)
	billRepo := repository.NewBillRepository(db)
	slipRepo := repository.NewSlipRepository(db)
	okSlipClient := okslip.NewClient(okSlipBaseURL, okSlipApiKEY)

	groupService := service.NewGroupService(db, groupRepo, billRepo)
	billService := service.NewBillService(db, billRepo, groupRepo, slipRepo, okSlipClient)
	groupHandler := handlers.NewGroupHandler(groupService)
	billHandler := handlers.NewBillHandler(billService)

	router := api.NewRouter(groupHandler, billHandler)

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