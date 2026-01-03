package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"walkietalkie/internal/ai"
)

// Chat handles HTTP requests for chat endpoint
type Chat struct {
	svc *ai.Service
}

// NewChat creates a new chat handler
func NewChat(aiService *ai.Service) *Chat {
	return &Chat{
		svc: aiService,
	}
}

// ChatRequest represents the incoming request
type ChatRequest struct {
	Message            string `json:"message,omitempty"`
	PreviousResponseID string `json:"previousResponseId,omitempty"`
	Language           string `json:"language,omitempty"`
}

// ChatResponse represents the outgoing response
type ChatResponse struct {
	Reply      string `json:"reply"`
	ResponseID string `json:"responseId"`
	AudioData  string `json:"audioData,omitempty"` // base64-encoded audio
}

// Handle processes both text and audio chat requests
func (h *Chat) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	var responseText, responseID string
	var err error

	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Audio input
		responseText, responseID, err = h.handleAudio(ctx, r)
	} else {
		// Text input
		responseText, responseID, err = h.handleText(ctx, r)
	}

	if err != nil {
		slog.Error("Request failed", "error", err)
		http.Error(w, "Request failed", http.StatusInternalServerError)
		return
	}

	// Generate audio response
	language := validateAndMapLanguage(r.FormValue("language"))
	audioData, err := h.svc.GenerateVoiceResponse(ctx, responseText, language)
	var audioBase64 string
	if err != nil {
		// Log error but don't fail the request - text response is still valuable
		slog.Error("Failed to generate voice response", "error", err)
	} else {
		// Encode audio bytes to base64
		audioBase64 = encodeBase64(audioData)
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	response := ChatResponse{
		Reply:      responseText,
		ResponseID: responseID,
		AudioData:  audioBase64,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Error encoding response", "error", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleText processes text-based chat requests
func (h *Chat) handleText(ctx context.Context, r *http.Request) (string, string, error) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode request", "error", err)
		return "", "", err
	}

	if req.Message == "" {
		return "", "", fmt.Errorf("message is required")
	}

	// Validate and default language
	language := validateAndMapLanguage(req.Language)

	slog.Info("Text message received", "message_length", len(req.Message), "language", language)
	return h.svc.SendMessage(ctx, req.Message, req.PreviousResponseID, language)
}

// handleAudio processes audio-based chat requests
func (h *Chat) handleAudio(ctx context.Context, r *http.Request) (string, string, error) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		slog.Error("Failed to parse multipart form", "error", err)
		return "", "", err
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		slog.Error("Failed to get audio file", "error", err)
		return "", "", err
	}
	defer file.Close()

	previousResponseID := r.FormValue("previousResponseId")
	language := validateAndMapLanguage(r.FormValue("language"))

	slog.Info("Audio file received", "filename", header.Filename, "size", header.Size, "language", language)
	return h.svc.TranscribeAndSend(ctx, file, header.Filename, previousResponseID, language)
}

// encodeBase64 encodes byte slice to base64 string
func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// validateAndMapLanguage validates the language code and returns a valid code or default
func validateAndMapLanguage(lang string) string {
	validLanguages := map[string]string{
		"en": "english",
		"pl": "polish",
		"uk": "ukrainian",
		"it": "italian",
		"es": "spanish",
	}

	if validLanguages[lang] != "" {
		return validLanguages[lang]
	}

	// Default to English if invalid or empty
	return validLanguages["en"]
}
