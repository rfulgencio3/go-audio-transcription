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
	"github.com/rfulgencio3/go-audio-transcription/internal/domain"
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

type mockRepository struct {
	saved   *domain.TranscriptionRecord
	listed  []*domain.TranscriptionRecord
	saveErr error
	listErr error
}

func (m *mockRepository) Save(_ context.Context, record *domain.TranscriptionRecord) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	record.ID = "transcriptions/1-A"
	m.saved = record
	return nil
}

func (m *mockRepository) FindByID(_ context.Context, _ string) (*domain.TranscriptionRecord, error) {
	return nil, nil
}

func (m *mockRepository) List(_ context.Context, _, _ int) ([]*domain.TranscriptionRecord, error) {
	return m.listed, m.listErr
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
		repo           *mockRepository
		buildReq       func(t *testing.T) *http.Request
		wantStatusCode int
		wantField      string // JSON field expected in response body
		wantSaved      bool
	}{
		{
			name:           "happy path returns 201 with transcript",
			transcriber:    happyTranscriber,
			analyzer:       happyAnalyzer,
			repo:           &mockRepository{},
			buildReq:       func(t *testing.T) *http.Request { return newMultipartRequest(t, "audio", "test.mp3", []byte("data")) },
			wantStatusCode: http.StatusCreated,
			wantField:      "transcript",
			wantSaved:      true,
		},
		{
			name:           "missing audio field returns 400",
			transcriber:    happyTranscriber,
			analyzer:       happyAnalyzer,
			repo:           &mockRepository{},
			buildReq:       func(t *testing.T) *http.Request { return newMultipartRequest(t, "", "", nil) },
			wantStatusCode: http.StatusBadRequest,
			wantField:      "error",
			wantSaved:      false,
		},
		{
			name:           "transcription failure returns 502",
			transcriber:    &mockTranscriber{err: errors.New("api down")},
			analyzer:       happyAnalyzer,
			repo:           &mockRepository{},
			buildReq:       func(t *testing.T) *http.Request { return newMultipartRequest(t, "audio", "test.mp3", []byte("data")) },
			wantStatusCode: http.StatusBadGateway,
			wantField:      "error",
			wantSaved:      false,
		},
		{
			name:           "AI analysis failure still returns 201 with transcript",
			transcriber:    happyTranscriber,
			analyzer:       &mockAnalyzer{err: errors.New("quota exceeded")},
			repo:           &mockRepository{},
			buildReq:       func(t *testing.T) *http.Request { return newMultipartRequest(t, "audio", "test.mp3", []byte("data")) },
			wantStatusCode: http.StatusCreated,
			wantField:      "transcript",
			wantSaved:      true,
		},
		{
			name:           "disabled transcriber returns 503",
			transcriber:    transcription.NewDisabledTranscriber("GEMINI_API_KEY is not set"),
			analyzer:       happyAnalyzer,
			repo:           &mockRepository{},
			buildReq:       func(t *testing.T) *http.Request { return newMultipartRequest(t, "audio", "test.mp3", []byte("data")) },
			wantStatusCode: http.StatusServiceUnavailable,
			wantField:      "error",
			wantSaved:      false,
		},
		{
			name:           "disabled analyzer still returns 201 with transcript",
			transcriber:    happyTranscriber,
			analyzer:       ai.NewDisabledAnalyzer("GEMINI_API_KEY is not set"),
			repo:           &mockRepository{},
			buildReq:       func(t *testing.T) *http.Request { return newMultipartRequest(t, "audio", "test.mp3", []byte("data")) },
			wantStatusCode: http.StatusCreated,
			wantField:      "transcript",
			wantSaved:      true,
		},
		{
			name:           "storage failure returns 500",
			transcriber:    happyTranscriber,
			analyzer:       happyAnalyzer,
			repo:           &mockRepository{saveErr: errors.New("db unavailable")},
			buildReq:       func(t *testing.T) *http.Request { return newMultipartRequest(t, "audio", "test.mp3", []byte("data")) },
			wantStatusCode: http.StatusInternalServerError,
			wantField:      "error",
			wantSaved:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := handler.NewHandler(tc.transcriber, tc.analyzer, tc.repo, 25*1024*1024)
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
			if tc.wantSaved {
				if tc.repo.saved == nil {
					t.Fatalf("expected record to be saved")
				}
				if tc.repo.saved.Transcript != happyTranscriber.result.Text {
					t.Fatalf("saved transcript = %q, want %q", tc.repo.saved.Transcript, happyTranscriber.result.Text)
				}
			}
		})
	}
}

func TestHandler_ListTranscriptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		repo           *mockRepository
		url            string
		wantStatusCode int
	}{
		{
			name:           "returns 200 with empty list",
			repo:           &mockRepository{listed: []*domain.TranscriptionRecord{}},
			url:            "/transcriptions",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "returns 200 with records",
			repo:           &mockRepository{listed: []*domain.TranscriptionRecord{{ID: "transcriptions/1-A", Transcript: "hello"}}},
			url:            "/transcriptions?limit=5&offset=0",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "storage failure returns 500",
			repo:           &mockRepository{listErr: errors.New("connection lost")},
			url:            "/transcriptions",
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := handler.NewHandler(&mockTranscriber{}, &mockAnalyzer{}, tc.repo, 25*1024*1024)
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rr := httptest.NewRecorder()
			h.ListTranscriptions(rr, req)

			if rr.Code != tc.wantStatusCode {
				t.Errorf("status = %d, want %d; body: %s", rr.Code, tc.wantStatusCode, rr.Body.String())
			}
		})
	}
}

func TestHandler_Health(t *testing.T) {
	t.Parallel()

	h := handler.NewHandler(&mockTranscriber{}, &mockAnalyzer{}, &mockRepository{}, 25*1024*1024)
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
