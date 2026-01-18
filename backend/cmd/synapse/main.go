package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"dailysynapse/backend/internal/api"
	"dailysynapse/backend/internal/config"
	"dailysynapse/backend/internal/judge"
	"dailysynapse/backend/internal/store"
	"dailysynapse/backend/internal/syncer"
	pkgjudge "dailysynapse/backend/pkg/judge"
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

	// Initialize Judge
	var judgeWorker *judge.Worker
	if cfg.GeminiAPIKey != "" {
		log.Println("Initializing Judge (Gemini)...")
		geminiClient, err := pkgjudge.NewGeminiClient(cfg.GeminiAPIKey)
		if err != nil {
			log.Printf("Warning: Failed to initialize Gemini client: %v. Judge will be disabled.", err)
		} else {
			judgeWorker = judge.NewWorker(storeQueries, geminiClient)
		}
	} else {
		log.Println("Warning: GEMINI_API_KEY not set. Judge will be disabled.")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go feedSyncer.StartBackgroundWorkers(ctx, 5, 15*time.Minute)

	if judgeWorker != nil {
		go judgeWorker.Start(ctx)
	}

	server := api.NewServer(db, feedSyncer)

	log.Printf("Starting server on port %s...", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, server.Routes()); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
