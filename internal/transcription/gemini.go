package transcription

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/rfulgencio3/go-audio-transcription/internal/geminiutil"
	"google.golang.org/api/option"
)

const transcribePrompt = `Transcribe all spoken content from the audio and return only valid JSON.
Do not summarize, omit, or rewrite. Preserve the original order of speech.
Include all words that are intelligible, including repetitions, hesitations, numbers, names, and short filler words when they are audible.

Required JSON format:
{
  "transcript": "full verbatim transcript with all spoken details",
  "language": "ISO 639-1 language code when confidently known, otherwise empty string"
}`

type geminiTranscriptionResponse struct {
	Transcript string `json:"transcript"`
	Language   string `json:"language"`
}

// GeminiTranscriber implements Transcriber using the Gemini API audio input flow.
type GeminiTranscriber struct {
	client    *genai.Client
	modelName string
}

// NewGeminiTranscriber creates a GeminiTranscriber.
func NewGeminiTranscriber(ctx context.Context, apiKey, modelName string) (*GeminiTranscriber, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("transcription.NewGeminiTranscriber: creating genai client: %w", err)
	}

	return &GeminiTranscriber{
		client:    client,
		modelName: modelName,
	}, nil
}

// Transcribe uploads audio to Gemini and returns the transcript.
// The duration is left unset because the Gemini Files API does not guarantee
// audio duration metadata in the response used by this application.
func (t *GeminiTranscriber) Transcribe(ctx context.Context, filename string, audio io.Reader) (Result, error) {
	buf, err := io.ReadAll(audio)
	if err != nil {
		return Result{}, fmt.Errorf("transcription.Transcribe: reading audio bytes: %w", err)
	}

	mimeType := detectAudioMIMEType(filename, buf)
	uploaded, err := t.client.UploadFile(ctx, "", bytes.NewReader(buf), &genai.UploadFileOptions{
		DisplayName: filepath.Base(filename),
		MIMEType:    mimeType,
	})
	if err != nil {
		return Result{}, fmt.Errorf("transcription.Transcribe: uploading audio to Gemini: %w", err)
	}
	defer func() {
		_ = t.client.DeleteFile(ctx, uploaded.Name)
	}()

	model := t.client.GenerativeModel(t.modelName)
	model.ResponseMIMEType = "application/json"
	model.ResponseSchema = &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"transcript": {Type: genai.TypeString},
			"language":   {Type: genai.TypeString},
		},
		Required: []string{"transcript"},
	}

	resp, err := model.GenerateContent(
		ctx,
		genai.Text(transcribePrompt),
		genai.FileData{URI: uploaded.URI, MIMEType: uploaded.MIMEType},
	)
	if err != nil {
		return Result{}, fmt.Errorf("transcription.Transcribe: calling Gemini API: %w", err)
	}

	var parsed geminiTranscriptionResponse
	if err := geminiutil.UnmarshalJSONTextResponse(resp, &parsed); err != nil {
		return Result{}, fmt.Errorf("transcription.Transcribe: %w", err)
	}

	parsed.Transcript = strings.TrimSpace(parsed.Transcript)
	parsed.Language = strings.TrimSpace(parsed.Language)
	if parsed.Transcript == "" {
		return Result{}, fmt.Errorf("transcription.Transcribe: Gemini returned an empty transcript")
	}

	return Result{
		Text:     parsed.Transcript,
		Language: parsed.Language,
		Duration: 0,
	}, nil
}

// Close releases the underlying Gemini client.
func (t *GeminiTranscriber) Close() error {
	return t.client.Close()
}

func detectAudioMIMEType(filename string, data []byte) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".mp3":
		return "audio/mpeg"
	case ".mp4", ".m4a":
		return "audio/mp4"
	case ".wav":
		return "audio/wav"
	case ".webm":
		return "audio/webm"
	case ".ogg":
		return "audio/ogg"
	case ".flac":
		return "audio/flac"
	}

	if byExt := mime.TypeByExtension(filepath.Ext(filename)); byExt != "" {
		return byExt
	}

	if len(data) == 0 {
		return "application/octet-stream"
	}

	sniffLen := min(len(data), 512)
	return http.DetectContentType(data[:sniffLen])
}
