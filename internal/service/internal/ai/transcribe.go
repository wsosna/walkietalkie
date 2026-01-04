package ai

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/openai/openai-go"
)

// Transcribe converts audio to text using Whisper
func (s *Service) Transcribe(ctx context.Context, audioFile io.Reader, filename string) (string, error) {
	contentType := getContentType(filename)

	resp, err := s.client.Audio.Transcriptions.New(
		ctx,
		openai.AudioTranscriptionNewParams{
			File:  openai.File(audioFile, filename, contentType),
			Model: openai.AudioModelWhisper1,
		},
	)
	if err != nil {
		return "", fmt.Errorf("whisper API error: %w", err)
	}

	return resp.Text, nil
}

// TranscribeAndSend transcribes audio then sends as chat message
func (s *Service) TranscribeAndSend(ctx context.Context, audioFile io.Reader, filename, previousResponseID, language string) (string, string, error) {
	text, err := s.Transcribe(ctx, audioFile, filename)
	if err != nil {
		return "", "", fmt.Errorf("transcription failed: %w", err)
	}

	return s.SendMessage(ctx, text, previousResponseID, language)
}

// getContentType determines the MIME type based on file extension
func getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp3":
		return "audio/mpeg"
	case ".mp4":
		return "audio/mp4"
	case ".wav":
		return "audio/wav"
	case ".webm":
		return "audio/webm"
	case ".m4a":
		return "audio/m4a"
	case ".ogg":
		return "audio/ogg"
	default:
		return "audio/webm"
	}
}
