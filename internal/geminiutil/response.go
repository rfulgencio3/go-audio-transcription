package geminiutil

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
)

// ExtractTextResponse returns the first text part from a Gemini response.
func ExtractTextResponse(resp *genai.GenerateContentResponse) (string, error) {
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}

	raw, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return "", fmt.Errorf("unexpected response part type from Gemini")
	}

	return strings.TrimSpace(string(raw)), nil
}

// UnmarshalJSONTextResponse extracts the first text part, strips optional
// markdown code fences, and unmarshals the remaining JSON payload into dst.
func UnmarshalJSONTextResponse(resp *genai.GenerateContentResponse, dst any) error {
	raw, err := ExtractTextResponse(resp)
	if err != nil {
		return err
	}

	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	if err := json.Unmarshal([]byte(cleaned), dst); err != nil {
		return fmt.Errorf("parsing Gemini JSON response: %w", err)
	}

	return nil
}
