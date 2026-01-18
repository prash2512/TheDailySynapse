package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dailysynapse/backend/internal/api"
	"dailysynapse/backend/internal/config"
	"dailysynapse/backend/internal/judge"
	"dailysynapse/backend/internal/logging"
	"dailysynapse/backend/internal/store"
	"dailysynapse/backend/internal/syncer"
	pkgjudge "dailysynapse/backend/pkg/judge"
)

func main() {
	cfg := config.Load()
	logger := logging.New(cfg.LogLevel)

	logger.Info("initializing database")
	db, err := store.Open(cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	storeQueries := store.NewQueries(db)
	feedSyncer := syncer.New(storeQueries, cfg, logger)

	var judgeWorker *judge.Worker
	if cfg.GeminiAPIKey != "" {
		logger.Info("initializing judge")
		geminiClient, err := pkgjudge.NewGeminiClient(cfg.GeminiAPIKey, cfg.MaxContentLength)
		if err != nil {
			logger.Warn("failed to initialize gemini client, judge disabled", "error", err)
		} else {
			judgeWorker = judge.NewWorker(storeQueries, geminiClient, cfg, logger)
		}
	} else {
		logger.Warn("GEMINI_API_KEY not set, judge disabled")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go feedSyncer.StartBackgroundWorkers(ctx)

	if judgeWorker != nil {
		go judgeWorker.Start(ctx)
	}

	server := api.NewServer(db, feedSyncer, logger)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      server.Routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("starting server", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", "error", err)
	}

	logger.Info("shutdown complete")
}
