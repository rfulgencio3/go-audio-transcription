package storage

import (
	"context"
	"fmt"
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
	store := ravendb.NewDocumentStore(urls, databaseName)
	if err := store.Initialize(); err != nil {
		return nil, fmt.Errorf("storage.NewRavenDBRepository: initializing document store: %w", err)
	}
	return &RavenDBRepository{
		store:        store,
		databaseName: databaseName,
	}, nil
}

// Save stores a new TranscriptionRecord and populates its RavenDB-assigned ID.
// Context cancellation is checked before opening the session and after SaveChanges.
func (r *RavenDBRepository) Save(ctx context.Context, record *domain.TranscriptionRecord) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("storage.Save: context cancelled before session: %w", err)
	}

	session, err := r.store.OpenSession("")
	if err != nil {
		return fmt.Errorf("storage.Save: opening RavenDB session: %w", err)
	}
	defer session.Close()

	record.CreatedAt = time.Now().UTC()

	if err := session.Store(record); err != nil {
		return fmt.Errorf("storage.Save: storing document: %w", err)
	}

	if err := session.SaveChanges(); err != nil {
		return fmt.Errorf("storage.Save: saving changes: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("storage.Save: context cancelled after save: %w", err)
	}

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
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("storage.List: context cancelled: %w", err)
	}

	session, err := r.store.OpenSession("")
	if err != nil {
		return nil, fmt.Errorf("storage.List: opening RavenDB session: %w", err)
	}
	defer session.Close()

	var records []*domain.TranscriptionRecord
	q := session.QueryCollection("TranscriptionRecords")
	q = q.OrderByDescending("CreatedAt")
	q = q.Skip(offset)
	q = q.Take(limit)

	if err := q.GetResults(&records); err != nil {
		return nil, fmt.Errorf("storage.List: querying documents: %w", err)
	}

	return records, nil
}

// Close shuts down the DocumentStore and releases all connections.
func (r *RavenDBRepository) Close() {
	r.store.Close()
}
