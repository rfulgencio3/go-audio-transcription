// Package handler provides HTTP handlers for the audio transcription API.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rfulgencio3/go-audio-transcription/internal/ai"
	"github.com/rfulgencio3/go-audio-transcription/internal/domain"
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
	maxBytes    int64
}

// NewHandler constructs a Handler with all required dependencies injected.
func NewHandler(
	t transcription.Transcriber,
	a ai.Analyzer,
	maxBytes int64,
) *Handler {
	return &Handler{
		transcriber: t,
		analyzer:    a,
		maxBytes:    maxBytes,
	}
}

// Transcribe godoc
//
//	@Summary		Transcribe an audio file
//	@Description	Receives an audio file upload, transcribes the complete spoken content with Google Gemini,
//	@Description	and returns the full transcript. Transcript analysis is best-effort.
//	@Tags			transcription
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			audio	formData	file	true	"Audio file (mp3, mp4, wav, m4a, ogg, webm, flac — max 25MB)"
//	@Success		201		{object}	domain.TranscriptionRecord
//	@Failure		400		{object}	ErrorResponse	"Missing 'audio' field"
//	@Failure		413		{object}	ErrorResponse	"File exceeds size limit"
//	@Failure		503		{object}	ErrorResponse	"Transcription provider is not configured"
//	@Failure		500		{object}	ErrorResponse	"Internal server error"
//	@Failure		502		{object}	ErrorResponse	"Upstream transcription failure from Gemini"
//	@Router			/transcribe [post]
func (h *Handler) Transcribe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Printf("handler.Transcribe: start")

	r.Body = http.MaxBytesReader(w, r.Body, h.maxBytes)
	if err := r.ParseMultipartForm(h.maxBytes); err != nil {
		log.Printf("handler.Transcribe: multipart parse failed: %v", err)
		writeJSON(w, http.StatusRequestEntityTooLarge, ErrorResponse{Error: "file exceeds maximum allowed size"})
		return
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		log.Printf("handler.Transcribe: missing audio field: %v", err)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "audio field is required"})
		return
	}
	defer file.Close()
	log.Printf("handler.Transcribe: received file=%s size=%d", header.Filename, header.Size)

	result, err := h.transcriber.Transcribe(ctx, header.Filename, file)
	if err != nil {
		log.Printf("handler.Transcribe: transcription failed: %v", err)
		if errors.Is(err, transcription.ErrProviderDisabled) {
			writeJSON(w, http.StatusServiceUnavailable, ErrorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusBadGateway, ErrorResponse{Error: fmt.Sprintf("transcription failed: %v", err)})
		return
	}

	record := &domain.TranscriptionRecord{
		AudioFilename: header.Filename,
		FileSizeBytes: header.Size,
		Transcript:    result.Text,
		Language:      result.Language,
		AudioDuration: result.Duration,
		CreatedAt:     time.Now().UTC(),
	}

	if h.analyzer != nil {
		analysis, err := h.analyzer.Analyze(ctx, result.Text)
		if err != nil {
			log.Printf("handler.Transcribe: analysis skipped after transcript capture: %v", err)
		} else {
			record.Summary = analysis.Summary
			record.KeyPoints = analysis.KeyPoints
			record.Sentiment = analysis.Sentiment
		}
	}

	log.Printf("handler.Transcribe: success")
	writeJSON(w, http.StatusCreated, record)
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
