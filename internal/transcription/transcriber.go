// Package transcription defines the contract for audio-to-text conversion.
package transcription

import (
	"context"
	"io"
)

// Result holds the output of a transcription operation.
type Result struct {
	// Text is the full transcribed text.
	Text string
	// Language is the detected language code (e.g. "en").
	Language string
	// Duration is the audio length in seconds.
	Duration float64
}

// Transcriber converts audio data into text.
// All implementations must honour context cancellation.
type Transcriber interface {
	Transcribe(ctx context.Context, filename string, audio io.Reader) (Result, error)
}
