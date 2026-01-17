package main

import (
	"log"

	"dailysynapse/backend/internal/store"
)

func main() {
	log.Println("Attempting to connect to database and run migrations...")

	db, err := store.Open("synapse.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("Database connection successful. Migrations applied if needed.")
}
