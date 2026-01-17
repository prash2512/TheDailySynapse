package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Default configuration values
const (
	DefaultDatabaseURL = "synapse.db"
	DefaultPort        = "8080"
)

type Config struct {
	DatabaseURL  string
	GeminiAPIKey string
	Port         string
}

func Load() *Config {
	// Try loading .env from the project root
	// We check common locations relative to the binary/execution path
	paths := []string{".env", "../.env", "../../.env"}
	for _, path := range paths {
		if err := godotenv.Load(path); err == nil {
			break
		}
	}

	return &Config{
		DatabaseURL:  getEnv("DATABASE_URL", DefaultDatabaseURL),
		GeminiAPIKey: getEnv("GEMINI_API_KEY", ""), // No default for API key, it is required
		Port:         getEnv("PORT", DefaultPort),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
