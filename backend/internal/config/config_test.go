package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any existing env vars
	os.Clearenv()

	cfg := Load()

	if cfg.DatabaseURL != "synapse.db" {
		t.Errorf("DatabaseURL = %v, want synapse.db", cfg.DatabaseURL)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %v, want 8080", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %v, want info", cfg.LogLevel)
	}
	if cfg.SyncInterval != 15*time.Minute {
		t.Errorf("SyncInterval = %v, want 15m", cfg.SyncInterval)
	}
	if cfg.SyncBatchSize != 20 {
		t.Errorf("SyncBatchSize = %v, want 20", cfg.SyncBatchSize)
	}
	if cfg.SyncWorkers != 5 {
		t.Errorf("SyncWorkers = %v, want 5", cfg.SyncWorkers)
	}
	if cfg.ArticleHorizonDays != 120 {
		t.Errorf("ArticleHorizonDays = %v, want 120", cfg.ArticleHorizonDays)
	}
	if cfg.RetentionDays != 30 {
		t.Errorf("RetentionDays = %v, want 30", cfg.RetentionDays)
	}
	if cfg.JudgeInterval != 6*time.Second {
		t.Errorf("JudgeInterval = %v, want 6s", cfg.JudgeInterval)
	}
	if cfg.MaxContentLength != 20000 {
		t.Errorf("MaxContentLength = %v, want 20000", cfg.MaxContentLength)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	os.Setenv("DATABASE_URL", "test.db")
	os.Setenv("PORT", "9000")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("SYNC_INTERVAL", "30m")
	os.Setenv("SYNC_BATCH_SIZE", "10")
	os.Setenv("SYNC_WORKERS", "3")
	os.Setenv("ARTICLE_HORIZON_DAYS", "60")
	os.Setenv("RETENTION_DAYS", "14")
	os.Setenv("JUDGE_INTERVAL", "10s")
	os.Setenv("MAX_CONTENT_LENGTH", "10000")
	os.Setenv("HTTP_TIMEOUT", "5s")
	defer os.Clearenv()

	cfg := Load()

	if cfg.DatabaseURL != "test.db" {
		t.Errorf("DatabaseURL = %v, want test.db", cfg.DatabaseURL)
	}
	if cfg.Port != "9000" {
		t.Errorf("Port = %v, want 9000", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %v, want debug", cfg.LogLevel)
	}
	if cfg.SyncInterval != 30*time.Minute {
		t.Errorf("SyncInterval = %v, want 30m", cfg.SyncInterval)
	}
	if cfg.SyncBatchSize != 10 {
		t.Errorf("SyncBatchSize = %v, want 10", cfg.SyncBatchSize)
	}
	if cfg.SyncWorkers != 3 {
		t.Errorf("SyncWorkers = %v, want 3", cfg.SyncWorkers)
	}
	if cfg.ArticleHorizonDays != 60 {
		t.Errorf("ArticleHorizonDays = %v, want 60", cfg.ArticleHorizonDays)
	}
	if cfg.RetentionDays != 14 {
		t.Errorf("RetentionDays = %v, want 14", cfg.RetentionDays)
	}
	if cfg.JudgeInterval != 10*time.Second {
		t.Errorf("JudgeInterval = %v, want 10s", cfg.JudgeInterval)
	}
	if cfg.MaxContentLength != 10000 {
		t.Errorf("MaxContentLength = %v, want 10000", cfg.MaxContentLength)
	}
	if cfg.HTTPTimeout != 5*time.Second {
		t.Errorf("HTTPTimeout = %v, want 5s", cfg.HTTPTimeout)
	}
}

func TestGetIntEnv_InvalidValue(t *testing.T) {
	os.Setenv("SYNC_BATCH_SIZE", "invalid")
	defer os.Unsetenv("SYNC_BATCH_SIZE")

	cfg := Load()
	// Should fall back to default
	if cfg.SyncBatchSize != 20 {
		t.Errorf("SyncBatchSize = %v, want 20 (default)", cfg.SyncBatchSize)
	}
}

func TestGetDurationEnv_InvalidValue(t *testing.T) {
	os.Setenv("SYNC_INTERVAL", "invalid")
	defer os.Unsetenv("SYNC_INTERVAL")

	cfg := Load()
	// Should fall back to default
	if cfg.SyncInterval != 15*time.Minute {
		t.Errorf("SyncInterval = %v, want 15m (default)", cfg.SyncInterval)
	}
}

