package transcription

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// ErrProviderDisabled indicates the transcription provider is unavailable
// because its runtime configuration is missing.
var ErrProviderDisabled = errors.New("transcription provider is disabled")

type disabledTranscriber struct {
	reason string
}

// NewDisabledTranscriber returns a Transcriber that always reports a disabled
// provider error. This keeps the HTTP server bootable in environments where the
// provider credentials are injected later.
func NewDisabledTranscriber(reason string) Transcriber {
	return &disabledTranscriber{reason: reason}
}

func (d *disabledTranscriber) Transcribe(_ context.Context, _ string, _ io.Reader) (Result, error) {
	return Result{}, fmt.Errorf("%w: %s", ErrProviderDisabled, d.reason)
}
