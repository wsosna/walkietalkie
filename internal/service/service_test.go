package service

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"testing"
)

// mockAIService implements the AI service interface for testing
type mockAIService struct {
	sendMessageFunc           func(ctx context.Context, message, previousResponseID, language string) (string, string, error)
	transcribeAndSendFunc     func(ctx context.Context, audioFile io.Reader, filename, previousResponseID, language string) (string, string, error)
	generateVoiceResponseFunc func(ctx context.Context, text, language string) ([]byte, error)
}

func (m *mockAIService) SendMessage(ctx context.Context, message, previousResponseID, language string) (string, string, error) {
	if m.sendMessageFunc != nil {
		return m.sendMessageFunc(ctx, message, previousResponseID, language)
	}
	return "", "", errors.New("not implemented")
}

func (m *mockAIService) TranscribeAndSend(ctx context.Context, audioFile io.Reader, filename, previousResponseID, language string) (string, string, error) {
	if m.transcribeAndSendFunc != nil {
		return m.transcribeAndSendFunc(ctx, audioFile, filename, previousResponseID, language)
	}
	return "", "", errors.New("not implemented")
}

func (m *mockAIService) GenerateVoiceResponse(ctx context.Context, text, language string) ([]byte, error) {
	if m.generateVoiceResponseFunc != nil {
		return m.generateVoiceResponseFunc(ctx, text, language)
	}
	return nil, errors.New("not implemented")
}

// TestProcessTextMessage_Success tests the happy path for text message processing
func TestProcessTextMessage_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockAudio := []byte("fake audio data")
	expectedReply := "Hello! How can I help you?"
	expectedResponseID := "response-123"
	expectedAudioBase64 := base64.StdEncoding.EncodeToString(mockAudio)

	mockAI := &mockAIService{
		sendMessageFunc: func(ctx context.Context, message, previousResponseID, language string) (string, string, error) {
			// Verify inputs
			if message != "Hi there" {
				t.Errorf("Expected message 'Hi there', got '%s'", message)
			}
			if previousResponseID != "prev-123" {
				t.Errorf("Expected previousResponseID 'prev-123', got '%s'", previousResponseID)
			}
			if language != "english" {
				t.Errorf("Expected language 'english', got '%s'", language)
			}
			return expectedReply, expectedResponseID, nil
		},
		generateVoiceResponseFunc: func(ctx context.Context, text, language string) ([]byte, error) {
			// Verify inputs
			if text != expectedReply {
				t.Errorf("Expected text '%s', got '%s'", expectedReply, text)
			}
			if language != "english" {
				t.Errorf("Expected language 'english', got '%s'", language)
			}
			return mockAudio, nil
		},
	}

	svc := &Service{ai: mockAI}

	// Act
	response, err := svc.ProcessTextMessage(ctx, "Hi there", "prev-123", "en")

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Reply != expectedReply {
		t.Errorf("Expected reply '%s', got '%s'", expectedReply, response.Reply)
	}

	if response.ResponseID != expectedResponseID {
		t.Errorf("Expected responseID '%s', got '%s'", expectedResponseID, response.ResponseID)
	}

	if response.AudioData != expectedAudioBase64 {
		t.Errorf("Expected audioData '%s', got '%s'", expectedAudioBase64, response.AudioData)
	}
}

// TestProcessTextMessage_AllLanguages tests language mapping for all supported languages
func TestProcessTextMessage_AllLanguages(t *testing.T) {
	testCases := []struct {
		langCode     string
		expectedLang string
	}{
		{"en", "english"},
		{"pl", "polish"},
		{"uk", "ukrainian"},
		{"it", "italian"},
		{"es", "spanish"},
		{"", "english"},        // empty defaults to english
		{"invalid", "english"}, // invalid defaults to english
	}

	for _, tc := range testCases {
		t.Run(tc.langCode, func(t *testing.T) {
			ctx := context.Background()
			var capturedLanguage string

			mockAI := &mockAIService{
				sendMessageFunc: func(ctx context.Context, message, previousResponseID, language string) (string, string, error) {
					capturedLanguage = language
					return "reply", "id-123", nil
				},
				generateVoiceResponseFunc: func(ctx context.Context, text, language string) ([]byte, error) {
					return []byte("audio"), nil
				},
			}

			svc := &Service{ai: mockAI}
			_, err := svc.ProcessTextMessage(ctx, "test", "", tc.langCode)

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if capturedLanguage != tc.expectedLang {
				t.Errorf("Expected language '%s', got '%s'", tc.expectedLang, capturedLanguage)
			}
		})
	}
}

// TestProcessTextMessage_ContinuesWithoutAudio tests that text response is returned even if TTS fails
func TestProcessTextMessage_ContinuesWithoutAudio(t *testing.T) {
	// Arrange
	ctx := context.Background()
	expectedReply := "Hello!"
	expectedResponseID := "response-456"

	mockAI := &mockAIService{
		sendMessageFunc: func(ctx context.Context, message, previousResponseID, language string) (string, string, error) {
			return expectedReply, expectedResponseID, nil
		},
		generateVoiceResponseFunc: func(ctx context.Context, text, language string) ([]byte, error) {
			return nil, errors.New("TTS service unavailable")
		},
	}

	svc := &Service{ai: mockAI}

	// Act
	response, err := svc.ProcessTextMessage(ctx, "Hello", "", "en")

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Reply != expectedReply {
		t.Errorf("Expected reply '%s', got '%s'", expectedReply, response.Reply)
	}

	if response.ResponseID != expectedResponseID {
		t.Errorf("Expected responseID '%s', got '%s'", expectedResponseID, response.ResponseID)
	}

	// Audio should be empty string when TTS fails
	if response.AudioData != "" {
		t.Errorf("Expected empty audioData, got '%s'", response.AudioData)
	}
}

// TestProcessAudioMessage_Success tests the happy path for audio message processing
func TestProcessAudioMessage_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	audioFile := strings.NewReader("fake audio bytes")
	filename := "recording.webm"
	mockResponseAudio := []byte("response audio data")
	expectedReply := "I heard you!"
	expectedResponseID := "audio-response-789"
	expectedAudioBase64 := base64.StdEncoding.EncodeToString(mockResponseAudio)

	mockAI := &mockAIService{
		transcribeAndSendFunc: func(ctx context.Context, audioFile io.Reader, fname, previousResponseID, language string) (string, string, error) {
			// Verify inputs
			if fname != filename {
				t.Errorf("Expected filename '%s', got '%s'", filename, fname)
			}
			if previousResponseID != "prev-audio-123" {
				t.Errorf("Expected previousResponseID 'prev-audio-123', got '%s'", previousResponseID)
			}
			if language != "polish" {
				t.Errorf("Expected language 'polish', got '%s'", language)
			}
			return expectedReply, expectedResponseID, nil
		},
		generateVoiceResponseFunc: func(ctx context.Context, text, language string) ([]byte, error) {
			if text != expectedReply {
				t.Errorf("Expected text '%s', got '%s'", expectedReply, text)
			}
			if language != "polish" {
				t.Errorf("Expected language 'polish', got '%s'", language)
			}
			return mockResponseAudio, nil
		},
	}

	svc := &Service{ai: mockAI}

	// Act
	response, err := svc.ProcessAudioMessage(ctx, audioFile, filename, "prev-audio-123", "pl")

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Reply != expectedReply {
		t.Errorf("Expected reply '%s', got '%s'", expectedReply, response.Reply)
	}

	if response.ResponseID != expectedResponseID {
		t.Errorf("Expected responseID '%s', got '%s'", expectedResponseID, response.ResponseID)
	}

	if response.AudioData != expectedAudioBase64 {
		t.Errorf("Expected audioData '%s', got '%s'", expectedAudioBase64, response.AudioData)
	}
}

// TestProcessAudioMessage_ContinuesWithoutResponseAudio tests that text is returned even if TTS fails
func TestProcessAudioMessage_ContinuesWithoutResponseAudio(t *testing.T) {
	// Arrange
	ctx := context.Background()
	audioFile := strings.NewReader("audio data")
	expectedReply := "Transcribed text"
	expectedResponseID := "audio-id-999"

	mockAI := &mockAIService{
		transcribeAndSendFunc: func(ctx context.Context, audioFile io.Reader, filename, previousResponseID, language string) (string, string, error) {
			return expectedReply, expectedResponseID, nil
		},
		generateVoiceResponseFunc: func(ctx context.Context, text, language string) ([]byte, error) {
			return nil, errors.New("TTS generation failed")
		},
	}

	svc := &Service{ai: mockAI}

	// Act
	response, err := svc.ProcessAudioMessage(ctx, audioFile, "test.webm", "", "en")

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Reply != expectedReply {
		t.Errorf("Expected reply '%s', got '%s'", expectedReply, response.Reply)
	}

	if response.ResponseID != expectedResponseID {
		t.Errorf("Expected responseID '%s', got '%s'", expectedResponseID, response.ResponseID)
	}

	// Audio should be empty when TTS fails
	if response.AudioData != "" {
		t.Errorf("Expected empty audioData, got '%s'", response.AudioData)
	}
}

// TestValidateAndMapLanguage tests the language validation helper
func TestValidateAndMapLanguage(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"en", "english"},
		{"pl", "polish"},
		{"uk", "ukrainian"},
		{"it", "italian"},
		{"es", "spanish"},
		{"", "english"},
		{"invalid", "english"},
		{"PL", "polish"}, // uppercase should work
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := validateAndMapLanguage(tc.input)
			if result != tc.expected {
				t.Errorf("Input '%s': expected '%s', got '%s'", tc.input, tc.expected, result)
			}
		})
	}
}
