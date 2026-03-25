// Package storage defines the persistence contract for transcription records.
package storage

import (
	"context"

	"github.com/rfulgencio3/go-audio-transcription/internal/domain"
)

// Repository persists and retrieves TranscriptionRecords.
// Context is carried through all operations for deadline propagation
// at the HTTP/network layer.
type Repository interface {
	// Save stores a new TranscriptionRecord, populating its ID after insert.
	Save(ctx context.Context, record *domain.TranscriptionRecord) error

	// FindByID retrieves a single record by its document ID.
	FindByID(ctx context.Context, id string) (*domain.TranscriptionRecord, error)

	// List returns records ordered by CreatedAt descending.
	List(ctx context.Context, limit, offset int) ([]*domain.TranscriptionRecord, error)
}
