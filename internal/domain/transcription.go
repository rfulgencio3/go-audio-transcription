// Package domain contains the shared types used across all internal packages.
package domain

import "time"

// TranscriptionRecord is the document stored in RavenDB.
// All fields use omitempty so the schema can evolve without migrations.
type TranscriptionRecord struct {
	// ID is auto-assigned by RavenDB as "transcriptions/N-A".
	ID string `json:"Id,omitempty" example:"transcriptions/1-A"`

	// Audio file metadata.
	AudioFilename string `json:"audioFilename" example:"meeting.mp3"`
	FileSizeBytes int64  `json:"fileSizeBytes" example:"2097152"`

	// Transcription result from OpenAI Whisper.
	Transcript    string  `json:"transcript" example:"Hello, this is a test recording..."`
	Language      string  `json:"language,omitempty" example:"en"`
	AudioDuration float64 `json:"audioDuration,omitempty" example:"47.3"`

	// AI analysis from Google Gemini — stored as nested document, schema-free.
	Summary   string   `json:"summary,omitempty"`
	KeyPoints []string `json:"keyPoints,omitempty"`
	Sentiment string   `json:"sentiment,omitempty" example:"positive"`

	// Metadata.
	CreatedAt time.Time `json:"createdAt"`
}
