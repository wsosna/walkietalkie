package ai

import (
	"fmt"
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
func NewService(apiKey, systemPrompt, model string) *Service {
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

// LoadSystemPrompt reads the system prompt from file
func LoadSystemPrompt() (string, error) {
	content, err := os.ReadFile("internal/prompts/system.txt")
	if err != nil {
		return "", fmt.Errorf("failed to read system prompt: %w", err)
	}
	return string(content), nil
}
