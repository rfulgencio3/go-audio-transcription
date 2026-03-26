// Package domain contains the shared types used across all internal packages.
package domain

import "time"

// TranscriptionRecord is the document stored in MongoDB.
// All fields use omitempty so the schema can evolve without migrations.
type TranscriptionRecord struct {
	// ID is assigned by the application and stored as MongoDB _id.
	ID string `json:"Id,omitempty" bson:"_id,omitempty" example:"67e3e8f6b4f54d9f0cf0aa11"`

	// Audio file metadata.
	AudioFilename string `json:"audioFilename" bson:"audioFilename" example:"meeting.mp3"`
	FileSizeBytes int64  `json:"fileSizeBytes" bson:"fileSizeBytes" example:"2097152"`

	// Transcription result from Gemini audio understanding, stored as the full spoken text.
	Transcript    string  `json:"transcript" bson:"transcript" example:"Hello, this is a test recording..."`
	Language      string  `json:"language,omitempty" bson:"language,omitempty" example:"en"`
	AudioDuration float64 `json:"audioDuration,omitempty" bson:"audioDuration,omitempty" example:"47.3"`

	// AI analysis from Google Gemini - stored as nested document, schema-free.
	Summary   string   `json:"summary,omitempty" bson:"summary,omitempty"`
	KeyPoints []string `json:"keyPoints,omitempty" bson:"keyPoints,omitempty"`
	Sentiment string   `json:"sentiment,omitempty" bson:"sentiment,omitempty" example:"positive"`

	// Metadata.
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
}
