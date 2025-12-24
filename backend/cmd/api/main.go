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
	"github.com/NoNiiEa/subShare-Discord/src/easySlip"
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

	slipBaseURL := os.Getenv("EASISLIP_API_URL")
	slipApiKEY := os.Getenv("EASISLIP_API_TOKEN")

    okSlipBaseURL := os.Getenv("SLIPOK_API_URL")
    okSlipApiKEY := os.Getenv("SLIPOK_API_KEY")

	groupRepo := repository.NewGroupRepository(db)
	billRepo := repository.NewBillRepository(db)
	groupService := service.NewGroupService(groupRepo, billRepo)
	easySlipClient := easyslip.NewClient(slipBaseURL, slipApiKEY) 
    okSlipClient := okslip.NewClient(okSlipBaseURL, okSlipApiKEY)
	billService := service.NewBillService(billRepo, groupRepo, easySlipClient, okSlipClient)
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