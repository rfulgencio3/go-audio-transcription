package ai

import (
	"context"
	"errors"
	"fmt"
)

// ErrProviderDisabled indicates the analyzer is unavailable because its
// runtime configuration is missing.
var ErrProviderDisabled = errors.New("analysis provider is disabled")

type disabledAnalyzer struct {
	reason string
}

// NewDisabledAnalyzer returns an Analyzer that always reports a disabled
// provider error. This keeps the HTTP server bootable in environments where the
// provider credentials are injected later.
func NewDisabledAnalyzer(reason string) Analyzer {
	return &disabledAnalyzer{reason: reason}
}

func (d *disabledAnalyzer) Analyze(_ context.Context, _ string) (Analysis, error) {
	return Analysis{}, fmt.Errorf("%w: %s", ErrProviderDisabled, d.reason)
}
