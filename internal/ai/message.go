package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/responses"
)

// SendMessage sends a text message to OpenAI and returns the response
func (s *Service) SendMessage(ctx context.Context, message, previousResponseID, language string) (string, string, error) {
	// Add system prompt for first message only - previousResponseID is saved in cookie per language
	if previousResponseID == "" {
		message = fmt.Sprintf("%s\n\nUser: %s", replacePlaceholders(s.systemPrompt, language), message)
	}
	params := responses.ResponseNewParams{
		Model:           s.model,
		Temperature:     openai.Float(0.4),
		MaxOutputTokens: openai.Int(10240),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(message),
		},
		Store: openai.Bool(true),
	}

	if previousResponseID != "" {
		params.PreviousResponseID = openai.String(previousResponseID)
	}

	resp, err := s.client.Responses.New(ctx, params)
	if err != nil {
		return "", "", fmt.Errorf("OpenAI API error: %w", err)
	}

	return resp.OutputText(), resp.ID, nil
}

func replacePlaceholders(systemPrompt, language string) string {
	return strings.ReplaceAll(systemPrompt, "$LANGUAGE", language)
}
