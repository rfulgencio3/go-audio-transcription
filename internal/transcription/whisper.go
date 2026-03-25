package transcription

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/sashabaranov/go-openai"
)

// WhisperTranscriber implements Transcriber using the OpenAI Whisper API.
type WhisperTranscriber struct {
	client *openai.Client
}

// NewWhisperTranscriber creates a WhisperTranscriber with the provided API key.
// The API key is passed explicitly — never read from the environment inside the constructor.
func NewWhisperTranscriber(apiKey string) *WhisperTranscriber {
	return &WhisperTranscriber{
		client: openai.NewClient(apiKey),
	}
}

// Transcribe uploads audio to the OpenAI Whisper API and returns the transcript.
// It respects ctx for cancellation and timeout propagation.
func (t *WhisperTranscriber) Transcribe(ctx context.Context, filename string, audio io.Reader) (Result, error) {
	buf, err := io.ReadAll(audio)
	if err != nil {
		return Result{}, fmt.Errorf("transcription.Transcribe: reading audio bytes: %w", err)
	}

	resp, err := t.client.CreateTranscription(ctx, openai.AudioRequest{
		Model:    openai.Whisper1,
		FilePath: filename,
		Reader:   bytes.NewReader(buf),
	})
	if err != nil {
		return Result{}, fmt.Errorf("transcription.Transcribe: calling OpenAI Whisper API: %w", err)
	}

	return Result{
		Text:     resp.Text,
		Language: resp.Language,
		Duration: float64(resp.Duration),
	}, nil
}
