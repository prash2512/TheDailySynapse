package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"dailysynapse/backend/internal/api"
	"dailysynapse/backend/internal/config"
	"dailysynapse/backend/internal/store"
	"dailysynapse/backend/internal/syncer"
)

func main() {
	cfg := config.Load()

	log.Println("Initializing database...")
	db, err := store.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	storeQueries := store.NewQueries(db)
	feedSyncer := syncer.New(storeQueries)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go feedSyncer.StartBackgroundWorkers(ctx, 5, 15*time.Minute)

	server := api.NewServer(db, feedSyncer)

	log.Printf("Starting server on port %s...", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, server.Routes()); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
