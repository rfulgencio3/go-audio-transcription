package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rfulgencio3/go-audio-transcription/internal/ai"
	"github.com/rfulgencio3/go-audio-transcription/internal/handler"
	"github.com/rfulgencio3/go-audio-transcription/internal/transcription"
)

// --- Mock implementations ---

type mockTranscriber struct {
	result transcription.Result
	err    error
}

func (m *mockTranscriber) Transcribe(_ context.Context, _ string, _ io.Reader) (transcription.Result, error) {
	return m.result, m.err
}

type mockAnalyzer struct {
	result ai.Analysis
	err    error
}

func (m *mockAnalyzer) Analyze(_ context.Context, _ string) (ai.Analysis, error) {
	return m.result, m.err
}

// --- Helpers ---

func newMultipartRequest(t *testing.T, fieldName, filename string, content []byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	if fieldName != "" {
		fw, err := w.CreateFormFile(fieldName, filename)
		if err != nil {
			t.Fatalf("creating form file: %v", err)
		}
		fw.Write(content)
	}
	w.Close()
	req := httptest.NewRequest(http.MethodPost, "/transcribe", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

// --- Tests ---

func TestHandler_Transcribe(t *testing.T) {
	t.Parallel()

	happyTranscriber := &mockTranscriber{
		result: transcription.Result{Text: "hello world", Language: "en", Duration: 5.0},
	}
	happyAnalyzer := &mockAnalyzer{
		result: ai.Analysis{Summary: "A greeting", KeyPoints: []string{"greeting"}, Sentiment: "positive"},
	}

	tests := []struct {
		name           string
		transcriber    transcription.Transcriber
		analyzer       ai.Analyzer
		buildReq       func(t *testing.T) *http.Request
		wantStatusCode int
		wantField      string // JSON field expected in response body
	}{
		{
			name:           "happy path returns 201 with transcript",
			transcriber:    happyTranscriber,
			analyzer:       happyAnalyzer,
			buildReq:       func(t *testing.T) *http.Request { return newMultipartRequest(t, "audio", "test.mp3", []byte("data")) },
			wantStatusCode: http.StatusCreated,
			wantField:      "transcript",
		},
		{
			name:           "missing audio field returns 400",
			transcriber:    happyTranscriber,
			analyzer:       happyAnalyzer,
			buildReq:       func(t *testing.T) *http.Request { return newMultipartRequest(t, "", "", nil) },
			wantStatusCode: http.StatusBadRequest,
			wantField:      "error",
		},
		{
			name:           "transcription failure returns 502",
			transcriber:    &mockTranscriber{err: errors.New("api down")},
			analyzer:       happyAnalyzer,
			buildReq:       func(t *testing.T) *http.Request { return newMultipartRequest(t, "audio", "test.mp3", []byte("data")) },
			wantStatusCode: http.StatusBadGateway,
			wantField:      "error",
		},
		{
			name:           "AI analysis failure still returns 201 with transcript",
			transcriber:    happyTranscriber,
			analyzer:       &mockAnalyzer{err: errors.New("quota exceeded")},
			buildReq:       func(t *testing.T) *http.Request { return newMultipartRequest(t, "audio", "test.mp3", []byte("data")) },
			wantStatusCode: http.StatusCreated,
			wantField:      "transcript",
		},
		{
			name:           "disabled transcriber returns 503",
			transcriber:    transcription.NewDisabledTranscriber("GEMINI_API_KEY is not set"),
			analyzer:       happyAnalyzer,
			buildReq:       func(t *testing.T) *http.Request { return newMultipartRequest(t, "audio", "test.mp3", []byte("data")) },
			wantStatusCode: http.StatusServiceUnavailable,
			wantField:      "error",
		},
		{
			name:           "disabled analyzer still returns 201 with transcript",
			transcriber:    happyTranscriber,
			analyzer:       ai.NewDisabledAnalyzer("GEMINI_API_KEY is not set"),
			buildReq:       func(t *testing.T) *http.Request { return newMultipartRequest(t, "audio", "test.mp3", []byte("data")) },
			wantStatusCode: http.StatusCreated,
			wantField:      "transcript",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := handler.NewHandler(tc.transcriber, tc.analyzer, 25*1024*1024)
			rr := httptest.NewRecorder()
			h.Transcribe(rr, tc.buildReq(t))

			if rr.Code != tc.wantStatusCode {
				t.Errorf("status = %d, want %d; body: %s", rr.Code, tc.wantStatusCode, rr.Body.String())
			}

			var body map[string]any
			if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
				t.Fatalf("decoding response body: %v", err)
			}
			if _, ok := body[tc.wantField]; !ok {
				t.Errorf("response body missing field %q: %v", tc.wantField, body)
			}
			if tc.wantStatusCode == http.StatusCreated {
				if got, ok := body["transcript"]; !ok || got != happyTranscriber.result.Text {
					t.Fatalf("transcript = %v, want %q", got, happyTranscriber.result.Text)
				}
			}
		})
	}
}

func TestHandler_Health(t *testing.T) {
	t.Parallel()

	h := handler.NewHandler(&mockTranscriber{}, &mockAnalyzer{}, 25*1024*1024)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	h.Health(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response body: %v", err)
	}
	if got := body["status"]; got != "ok" {
		t.Fatalf("status body = %v, want ok", got)
	}
}
