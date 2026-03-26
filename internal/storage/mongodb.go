package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/rfulgencio3/go-audio-transcription/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	connectTimeout = 15 * time.Second
	closeTimeout   = 5 * time.Second
)

// MongoRepository implements Repository backed by MongoDB.
type MongoRepository struct {
	client     *mongo.Client
	collection *mongo.Collection
}

// NewMongoRepository initializes the MongoDB client, validates connectivity,
// and ensures the query index used by list operations exists.
func NewMongoRepository(uri, databaseName, collectionName string) (*MongoRepository, error) {
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("storage.NewMongoRepository: connecting mongo client: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("storage.NewMongoRepository: pinging mongo client: %w", err)
	}

	collection := client.Database(databaseName).Collection(collectionName)
	if _, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "createdAt", Value: -1}},
	}); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("storage.NewMongoRepository: creating index: %w", err)
	}

	return &MongoRepository{
		client:     client,
		collection: collection,
	}, nil
}

// Save stores a new TranscriptionRecord and populates its ID before insert.
func (r *MongoRepository) Save(ctx context.Context, record *domain.TranscriptionRecord) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("storage.Save: context cancelled before insert: %w", err)
	}

	if record.ID == "" {
		record.ID = primitive.NewObjectID().Hex()
	}
	record.CreatedAt = time.Now().UTC()

	if _, err := r.collection.InsertOne(ctx, record); err != nil {
		return fmt.Errorf("storage.Save: inserting document: %w", err)
	}

	return nil
}

// FindByID retrieves a single TranscriptionRecord by its MongoDB document ID.
func (r *MongoRepository) FindByID(ctx context.Context, id string) (*domain.TranscriptionRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("storage.FindByID: context cancelled: %w", err)
	}

	var record domain.TranscriptionRecord
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&record)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("storage.FindByID: querying document %q: %w", id, err)
	}

	return &record, nil
}

// List returns up to limit TranscriptionRecords ordered by CreatedAt descending,
// skipping the first offset records.
func (r *MongoRepository) List(ctx context.Context, limit, offset int) ([]*domain.TranscriptionRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("storage.List: context cancelled: %w", err)
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(int64(offset)).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, fmt.Errorf("storage.List: querying documents: %w", err)
	}
	defer cursor.Close(ctx)

	var records []*domain.TranscriptionRecord
	for cursor.Next(ctx) {
		var record domain.TranscriptionRecord
		if err := cursor.Decode(&record); err != nil {
			return nil, fmt.Errorf("storage.List: decoding document: %w", err)
		}
		records = append(records, &record)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("storage.List: iterating cursor: %w", err)
	}

	return records, nil
}

// Close shuts down the MongoDB client and releases all connections.
func (r *MongoRepository) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), closeTimeout)
	defer cancel()
	_ = r.client.Disconnect(ctx)
}
