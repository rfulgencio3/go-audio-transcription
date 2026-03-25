// Package ai defines the contract for AI-powered text analysis.
package ai

import "context"

// Analysis holds the structured output from the AI model.
type Analysis struct {
	// Summary is a 2-3 sentence overview of the transcript.
	Summary string
	// KeyPoints lists the main topics identified in the transcript.
	KeyPoints []string
	// Sentiment is one of "positive", "neutral", or "negative".
	Sentiment string
}

// Analyzer processes a transcript and returns structured insights.
// All implementations must honour context cancellation.
type Analyzer interface {
	Analyze(ctx context.Context, transcript string) (Analysis, error)
}
