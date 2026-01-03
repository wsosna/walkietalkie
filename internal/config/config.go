package config

import (
	"log"
	"os"
)

// Config holds the application configuration
type Config struct {
	OpenAIAPIKey string
	Model        string
	Port         string
}

// Load loads configuration from environment variables
func Load() Config {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	model := os.Getenv("OPENAI_MODEL") // default set by service
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // default port
	}

	return Config{
		OpenAIAPIKey: apiKey,
		Model:        model,
		Port:         port,
	}
}
