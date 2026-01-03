package ai

import (
	"context"
	"fmt"
	"io"

	"github.com/openai/openai-go"
)

// GenerateVoiceResponse converts text to speech using OpenAI TTS
func (s *Service) GenerateVoiceResponse(ctx context.Context, text, language string) ([]byte, error) {
	response, err := s.client.Audio.Speech.New(ctx, openai.AudioSpeechNewParams{
		Model:        openai.SpeechModelGPT4oMiniTTS,
		Voice:        openai.AudioSpeechNewParamsVoiceAlloy,
		Input:        text,
		Instructions: openai.Opt(fmt.Sprintf("speak in %s, use this language accent to generate audio", language)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate speech: %w", err)
	}
	defer response.Body.Close()

	audioData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio data: %w", err)
	}

	return audioData, nil
}
