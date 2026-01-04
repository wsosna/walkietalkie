package ai

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Service handles all OpenAI operations
type Service struct {
	client       openai.Client
	systemPrompt string
	model        openai.ChatModel
}

// NewService creates a new AI service instance
func NewService(apiKey, model string) *Service {
	systemPrompt, err := loadSystemPrompt()
	if err != nil {
		slog.Error("Failed to load system prompt", "error", err)
		os.Exit(1)
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))

	// Parse model string to ChatModel
	chatModel := openai.ChatModelGPT4oMini // default mini
	if model == "gpt-4o" {
		chatModel = openai.ChatModelGPT4o
	}

	return &Service{
		client:       client,
		systemPrompt: systemPrompt,
		model:        chatModel,
	}
}

// loadSystemPrompt reads the system prompt from file
func loadSystemPrompt() (string, error) {
	content, err := os.ReadFile("internal/prompts/system.txt")
	if err != nil {
		return "", fmt.Errorf("failed to read system prompt: %w", err)
	}
	return string(content), nil
}
