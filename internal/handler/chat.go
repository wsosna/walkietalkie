package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"walkietalkie/internal/service"
)

// Chat handles HTTP requests for chat endpoint
type Chat struct {
	svc *service.Service
}

// NewChat creates a new chat handler
func NewChat(chatService *service.Service) *Chat {
	return &Chat{
		svc: chatService,
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
	contentType := r.Header.Get("Content-Type")

	var response *service.Response
	var err error

	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Audio input
		response, err = h.handleAudio(ctx, r)
	} else {
		// Text input
		response, err = h.handleText(ctx, r)
	}

	if err != nil {
		slog.Error("Request failed", "error", err)
		http.Error(w, "Request failed", http.StatusInternalServerError)
		return
	}

	// Build and send JSON response
	w.Header().Set("Content-Type", "application/json")
	chatResponse := ChatResponse{
		Reply:      response.Reply,
		ResponseID: response.ResponseID,
		AudioData:  response.AudioData,
	}

	if err := json.NewEncoder(w).Encode(chatResponse); err != nil {
		slog.Error("Error encoding response", "error", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleText processes text-based chat requests
func (h *Chat) handleText(ctx context.Context, r *http.Request) (*service.Response, error) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode request", "error", err)
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if req.Message == "" {
		return nil, fmt.Errorf("message is required")
	}

	return h.svc.ProcessTextMessage(ctx, req.Message, req.PreviousResponseID, req.Language)
}

// handleAudio processes audio-based chat requests
func (h *Chat) handleAudio(ctx context.Context, r *http.Request) (*service.Response, error) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		slog.Error("Failed to parse multipart form", "error", err)
		return nil, fmt.Errorf("invalid multipart form: %w", err)
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		slog.Error("Failed to get audio file", "error", err)
		return nil, fmt.Errorf("audio file missing: %w", err)
	}
	defer file.Close()

	previousResponseID := r.FormValue("previousResponseId")
	language := r.FormValue("language")

	return h.svc.ProcessAudioMessage(ctx, file, header.Filename, previousResponseID, language)
}
