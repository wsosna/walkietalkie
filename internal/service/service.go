package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"walkietalkie/internal/service/internal/ai"
)

// aiSvc defines the interface for AI operations (useful for testing)
type aiSvc interface {
	SendMessage(ctx context.Context, message, previousResponseID, language string) (string, string, error)
	TranscribeAndSend(ctx context.Context, audioFile io.Reader, filename, previousResponseID, language string) (string, string, error)
	GenerateVoiceResponse(ctx context.Context, text, language string) ([]byte, error)
}

// Service handles business logic for chat operations
type Service struct {
	ai aiSvc
}

// New creates a new service instance
func New(aiService *ai.Service) *Service {
	return &Service{
		ai: aiService,
	}
}

// Response represents a processed chat response
type Response struct {
	Reply      string
	ResponseID string
	AudioData  string // base64-encoded audio
}

// ProcessTextMessage handles text message processing
func (s *Service) ProcessTextMessage(ctx context.Context, message, previousResponseID, language string) (*Response, error) {
	// Validate and map language
	mappedLanguage := validateAndMapLanguage(language)

	slog.Info("Processing text message", "message_length", len(message), "language", mappedLanguage)

	// Get text response from AI
	responseText, responseID, err := s.ai.SendMessage(ctx, message, previousResponseID, mappedLanguage)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Generate audio response
	audioData, err := s.generateAudioResponse(ctx, responseText, mappedLanguage)
	if err != nil {
		slog.Error("Failed to generate voice response", "error", err)
		// Continue without audio - text response is still valuable
	}

	return &Response{
		Reply:      responseText,
		ResponseID: responseID,
		AudioData:  audioData,
	}, nil
}

// ProcessAudioMessage handles audio message processing
func (s *Service) ProcessAudioMessage(ctx context.Context, audioFile io.Reader, filename, previousResponseID, language string) (*Response, error) {
	// Validate and map language
	mappedLanguage := validateAndMapLanguage(language)

	slog.Info("Processing audio message", "filename", filename, "language", mappedLanguage)

	// Transcribe and get text response from AI
	responseText, responseID, err := s.ai.TranscribeAndSend(ctx, audioFile, filename, previousResponseID, mappedLanguage)
	if err != nil {
		return nil, fmt.Errorf("failed to process audio: %w", err)
	}

	// Generate audio response
	audioData, err := s.generateAudioResponse(ctx, responseText, mappedLanguage)
	if err != nil {
		slog.Error("Failed to generate voice response", "error", err)
		// Continue without audio - text response is still valuable
	}

	return &Response{
		Reply:      responseText,
		ResponseID: responseID,
		AudioData:  audioData,
	}, nil
}

// generateAudioResponse generates audio from text and returns base64-encoded data
func (s *Service) generateAudioResponse(ctx context.Context, text, language string) (string, error) {
	audioBytes, err := s.ai.GenerateVoiceResponse(ctx, text, language)
	if err != nil {
		return "", err
	}

	// Encode to base64 for JSON transport
	return base64.StdEncoding.EncodeToString(audioBytes), nil
}

// validateAndMapLanguage validates the language code and returns full language name or default
func validateAndMapLanguage(lang string) string {
	validLanguages := map[string]string{
		"en": "english",
		"pl": "polish",
		"uk": "ukrainian",
		"it": "italian",
		"es": "spanish",
	}

	if mappedLang, ok := validLanguages[strings.ToLower(lang)]; ok {
		return mappedLang
	}

	// Default to English if invalid or empty
	return "english"
}
