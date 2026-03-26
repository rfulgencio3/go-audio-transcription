package storage

import (
	"context"
	"fmt"
	"log"
	"time"

	ravendb "github.com/ravendb/ravendb-go-client"

	"github.com/rfulgencio3/go-audio-transcription/internal/domain"
)

// RavenDBRepository implements Repository backed by RavenDB.
// The RavenDB Go client does not accept context.Context in session methods,
// so context cancellation is checked explicitly before and after each operation.
type RavenDBRepository struct {
	store        *ravendb.DocumentStore
	databaseName string
}

// NewRavenDBRepository initialises and validates the DocumentStore connection.
// Returns an error if the RavenDB server is unreachable or the store fails to initialise.
func NewRavenDBRepository(urls []string, databaseName string) (*RavenDBRepository, error) {
	log.Printf("storage: initializing RavenDB store urls=%v database=%s", urls, databaseName)
	store := ravendb.NewDocumentStore(urls, databaseName)
	if err := store.Initialize(); err != nil {
		return nil, fmt.Errorf("storage.NewRavenDBRepository: initializing document store: %w", err)
	}
	log.Printf("storage: RavenDB store initialized")
	return &RavenDBRepository{
		store:        store,
		databaseName: databaseName,
	}, nil
}

// Save stores a new TranscriptionRecord and populates its RavenDB-assigned ID.
// Context cancellation is checked before opening the session and after SaveChanges.
func (r *RavenDBRepository) Save(ctx context.Context, record *domain.TranscriptionRecord) error {
	log.Printf("storage.Save: start filename=%s size=%d", record.AudioFilename, record.FileSizeBytes)
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("storage.Save: context cancelled before session: %w", err)
	}

	session, err := r.store.OpenSession("")
	if err != nil {
		return fmt.Errorf("storage.Save: opening RavenDB session: %w", err)
	}
	defer session.Close()
	log.Printf("storage.Save: session opened")

	record.CreatedAt = time.Now().UTC()

	if err := session.Store(record); err != nil {
		return fmt.Errorf("storage.Save: storing document: %w", err)
	}

	if err := session.SaveChanges(); err != nil {
		return fmt.Errorf("storage.Save: saving changes: %w", err)
	}
	log.Printf("storage.Save: save changes completed")

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("storage.Save: context cancelled after save: %w", err)
	}

	log.Printf("storage.Save: success")
	return nil
}

// FindByID retrieves a single TranscriptionRecord by its RavenDB document ID.
func (r *RavenDBRepository) FindByID(ctx context.Context, id string) (*domain.TranscriptionRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("storage.FindByID: context cancelled: %w", err)
	}

	session, err := r.store.OpenSession("")
	if err != nil {
		return nil, fmt.Errorf("storage.FindByID: opening RavenDB session: %w", err)
	}
	defer session.Close()

	var record domain.TranscriptionRecord
	if err := session.Load(&record, id); err != nil {
		return nil, fmt.Errorf("storage.FindByID: loading document %q: %w", id, err)
	}

	return &record, nil
}

// List returns up to limit TranscriptionRecords ordered by CreatedAt descending,
// skipping the first offset records.
func (r *RavenDBRepository) List(ctx context.Context, limit, offset int) ([]*domain.TranscriptionRecord, error) {
	log.Printf("storage.List: start limit=%d offset=%d", limit, offset)
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("storage.List: context cancelled: %w", err)
	}

	session, err := r.store.OpenSession("")
	if err != nil {
		return nil, fmt.Errorf("storage.List: opening RavenDB session: %w", err)
	}
	defer session.Close()
	log.Printf("storage.List: session opened")

	var records []*domain.TranscriptionRecord
	q := session.QueryCollection("TranscriptionRecords")
	q = q.OrderByDescending("CreatedAt")
	q = q.Skip(offset)
	q = q.Take(limit)

	if err := q.GetResults(&records); err != nil {
		return nil, fmt.Errorf("storage.List: querying documents: %w", err)
	}
	log.Printf("storage.List: query completed records=%d", len(records))

	return records, nil
}

// Close shuts down the DocumentStore and releases all connections.
func (r *RavenDBRepository) Close() {
	r.store.Close()
}
