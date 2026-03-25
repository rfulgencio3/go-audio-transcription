package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const analyzePromptTemplate = `You are an audio transcript analyst.
Analyze the following transcript and respond with ONLY valid JSON, no markdown fences.

Transcript:
%s

Required JSON format:
{
  "summary": "2-3 sentence summary",
  "keyPoints": ["point 1", "point 2"],
  "sentiment": "positive|neutral|negative"
}`

// geminiResponse is the expected JSON shape returned by the Gemini model.
type geminiResponse struct {
	Summary   string   `json:"summary"`
	KeyPoints []string `json:"keyPoints"`
	Sentiment string   `json:"sentiment"`
}

// GeminiAnalyzer implements Analyzer using the Google Gemini API.
type GeminiAnalyzer struct {
	client    *genai.Client
	modelName string
}

// NewGeminiAnalyzer creates a GeminiAnalyzer.
// ctx is used only for client construction (one-time TLS/gRPC handshake).
func NewGeminiAnalyzer(ctx context.Context, apiKey, modelName string) (*GeminiAnalyzer, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("ai.NewGeminiAnalyzer: creating genai client: %w", err)
	}
	return &GeminiAnalyzer{
		client:    client,
		modelName: modelName,
	}, nil
}

// Analyze sends the transcript to Gemini and parses the structured JSON response.
// It respects ctx for cancellation and timeout propagation.
func (g *GeminiAnalyzer) Analyze(ctx context.Context, transcript string) (Analysis, error) {
	model := g.client.GenerativeModel(g.modelName)
	prompt := fmt.Sprintf(analyzePromptTemplate, transcript)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return Analysis{}, fmt.Errorf("ai.Analyze: generating content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return Analysis{}, fmt.Errorf("ai.Analyze: empty response from Gemini")
	}

	raw, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return Analysis{}, fmt.Errorf("ai.Analyze: unexpected response part type from Gemini")
	}

	// Gemini may wrap JSON in markdown fences — strip them before unmarshalling.
	cleaned := strings.TrimSpace(string(raw))
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var gr geminiResponse
	if err := json.Unmarshal([]byte(cleaned), &gr); err != nil {
		return Analysis{}, fmt.Errorf("ai.Analyze: parsing Gemini JSON response: %w", err)
	}

	return Analysis{
		Summary:   gr.Summary,
		KeyPoints: gr.KeyPoints,
		Sentiment: gr.Sentiment,
	}, nil
}

// Close releases the underlying gRPC connection.
func (g *GeminiAnalyzer) Close() error {
	return g.client.Close()
}
