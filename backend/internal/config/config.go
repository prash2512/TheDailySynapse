package config

import (
	"os"

	"github.com/joho/godotenv"
)

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
	paths := []string{".env", "../.env", "../../.env"}
	for _, path := range paths {
		if err := godotenv.Load(path); err == nil {
			break
		}
	}

	return &Config{
		DatabaseURL:  getEnv("DATABASE_URL", DefaultDatabaseURL),
		GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),
		Port:         getEnv("PORT", DefaultPort),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
