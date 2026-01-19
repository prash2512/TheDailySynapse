package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL  string
	GeminiAPIKey string
	Port         string
	LogLevel     string

	SyncInterval       time.Duration
	SyncBatchSize      int
	SyncWorkers        int
	ArticleHorizonDays int
	RetentionDays      int
	HTTPTimeout        time.Duration

	JudgeInterval    time.Duration
	MaxContentLength int
}

func Load() *Config {
	paths := []string{".env", "../.env", "../../.env"}
	for _, path := range paths {
		if err := godotenv.Overload(path); err == nil {
			break
		}
	}

	return &Config{
		DatabaseURL:  getEnv("DATABASE_URL", "synapse.db"),
		GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),
		Port:         getEnv("PORT", "8080"),
		LogLevel:     getEnv("LOG_LEVEL", "info"),

		SyncInterval:       getDurationEnv("SYNC_INTERVAL", 15*time.Minute),
		SyncBatchSize:      getIntEnv("SYNC_BATCH_SIZE", 20),
		SyncWorkers:        getIntEnv("SYNC_WORKERS", 5),
		ArticleHorizonDays: getIntEnv("ARTICLE_HORIZON_DAYS", 7),
		RetentionDays:      getIntEnv("RETENTION_DAYS", 30),
		HTTPTimeout:        getDurationEnv("HTTP_TIMEOUT", 10*time.Second),

		JudgeInterval:    getDurationEnv("JUDGE_INTERVAL", 6*time.Second),
		MaxContentLength: getIntEnv("MAX_CONTENT_LENGTH", 20000),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return fallback
}
