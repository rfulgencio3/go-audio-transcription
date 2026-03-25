// Package handler provides HTTP handlers for the audio transcription API.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/rfulgencio3/go-audio-transcription/internal/ai"
	"github.com/rfulgencio3/go-audio-transcription/internal/domain"
	"github.com/rfulgencio3/go-audio-transcription/internal/storage"
	"github.com/rfulgencio3/go-audio-transcription/internal/transcription"
)

// ErrorResponse represents a standardized error returned by the API.
// @Description Standardized error response
type ErrorResponse struct {
	Error string `json:"error" example:"audio field is required"`
}

// HealthResponse represents the service health status.
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// Handler orchestrates the transcription pipeline for HTTP requests.
type Handler struct {
	transcriber transcription.Transcriber
	analyzer    ai.Analyzer
	repo        storage.Repository
	maxBytes    int64
}

// NewHandler constructs a Handler with all required dependencies injected.
func NewHandler(
	t transcription.Transcriber,
	a ai.Analyzer,
	r storage.Repository,
	maxBytes int64,
) *Handler {
	return &Handler{
		transcriber: t,
		analyzer:    a,
		repo:        r,
		maxBytes:    maxBytes,
	}
}

// Transcribe godoc
//
//	@Summary		Transcribe an audio file
//	@Description	Receives an audio file upload, transcribes and analyzes it with Google Gemini,
//	@Description	and persists the result in RavenDB.
//	@Tags			transcription
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			audio	formData	file	true	"Audio file (mp3, mp4, wav, m4a, ogg, webm, flac — max 25MB)"
//	@Success		201		{object}	domain.TranscriptionRecord
//	@Failure		400		{object}	ErrorResponse	"Missing 'audio' field"
//	@Failure		413		{object}	ErrorResponse	"File exceeds MAX_UPLOAD_BYTES"
//	@Failure		503		{object}	ErrorResponse	"Required AI provider is not configured"
//	@Failure		500		{object}	ErrorResponse	"Internal server error or RavenDB failure"
//	@Failure		502		{object}	ErrorResponse	"Upstream API failure from Gemini"
//	@Router			/transcribe [post]
func (h *Handler) Transcribe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	r.Body = http.MaxBytesReader(w, r.Body, h.maxBytes)
	if err := r.ParseMultipartForm(h.maxBytes); err != nil {
		writeJSON(w, http.StatusRequestEntityTooLarge, ErrorResponse{Error: "file exceeds maximum allowed size"})
		return
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "audio field is required"})
		return
	}
	defer file.Close()

	result, err := h.transcriber.Transcribe(ctx, header.Filename, file)
	if err != nil {
		if errors.Is(err, transcription.ErrProviderDisabled) {
			writeJSON(w, http.StatusServiceUnavailable, ErrorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusBadGateway, ErrorResponse{Error: fmt.Sprintf("transcription failed: %v", err)})
		return
	}

	analysis, err := h.analyzer.Analyze(ctx, result.Text)
	if err != nil {
		if errors.Is(err, ai.ErrProviderDisabled) {
			writeJSON(w, http.StatusServiceUnavailable, ErrorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusBadGateway, ErrorResponse{Error: fmt.Sprintf("AI analysis failed: %v", err)})
		return
	}

	record := &domain.TranscriptionRecord{
		AudioFilename: header.Filename,
		FileSizeBytes: header.Size,
		Transcript:    result.Text,
		Language:      result.Language,
		AudioDuration: result.Duration,
		Summary:       analysis.Summary,
		KeyPoints:     analysis.KeyPoints,
		Sentiment:     analysis.Sentiment,
	}

	if err := h.repo.Save(ctx, record); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to persist transcription"})
		return
	}

	writeJSON(w, http.StatusCreated, record)
}

// ListTranscriptions godoc
//
//	@Summary		List stored transcriptions
//	@Description	Returns paginated transcriptions ordered by creation date descending.
//	@Tags			transcription
//	@Produce		json
//	@Param			limit	query		int	false	"Number of records to return (default 20)"
//	@Param			offset	query		int	false	"Pagination offset (default 0)"
//	@Success		200		{array}		domain.TranscriptionRecord
//	@Failure		500		{object}	ErrorResponse
//	@Router			/transcriptions [get]
func (h *Handler) ListTranscriptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := parseQueryInt(r, "limit", 20)
	offset := parseQueryInt(r, "offset", 0)

	records, err := h.repo.List(ctx, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to list transcriptions"})
		return
	}

	writeJSON(w, http.StatusOK, records)
}

// Health godoc
//
//	@Summary		Service health
//	@Description	Returns 200 when the HTTP service is running.
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	HealthResponse
//	@Router			/health [get]
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func parseQueryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return defaultVal
	}
	return n
}
