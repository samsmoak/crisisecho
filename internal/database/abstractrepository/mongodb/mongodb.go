package mongoRepo

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// MongoRepository is a generic base repository that provides standard CRUD
// operations for any document type T. Concrete repositories embed this struct
// and call NewMongoRepository to initialize it.
type MongoRepository[T any] struct {
	Collection *mongo.Collection
}

// NewMongoRepository creates a new MongoRepository backed by the named collection
// in the provided database.
func NewMongoRepository[T any](db *mongo.Database, collectionName string) *MongoRepository[T] {
	return &MongoRepository[T]{
		Collection: db.Collection(collectionName),
	}
}

// GetByID fetches a single document by its _id field. id may be a hex ObjectID
// string or a plain string — ObjectID parsing is attempted first.
func (r *MongoRepository[T]) GetByID(ctx context.Context, id string) (*T, error) {
	filter, err := idFilter(id)
	if err != nil {
		return nil, fmt.Errorf("MongoRepository.GetByID: %w", err)
	}

	var result T
	if err := r.Collection.FindOne(ctx, filter).Decode(&result); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("MongoRepository.GetByID: document not found")
		}
		return nil, fmt.Errorf("MongoRepository.GetByID: %w", err)
	}
	return &result, nil
}

// GetAll returns every document in the collection.
func (r *MongoRepository[T]) GetAll(ctx context.Context) ([]*T, error) {
	cursor, err := r.Collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("MongoRepository.GetAll: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*T
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("MongoRepository.GetAll decode: %w", err)
	}
	return results, nil
}

// Create inserts doc into the collection. After insertion the struct's _id field
// is set to the InsertedID returned by MongoDB via reflection. The doc must be a
// pointer to a struct that has a field tagged `bson:"_id"`.
func (r *MongoRepository[T]) Create(ctx context.Context, doc *T) error {
	result, err := r.Collection.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("MongoRepository.Create: %w", err)
	}

	// Reflect the InsertedID back onto the struct so callers see the generated _id.
	setIDField(doc, result.InsertedID)
	return nil
}

// Update performs a full document replace for the document with the given id.
func (r *MongoRepository[T]) Update(ctx context.Context, id string, doc *T) error {
	filter, err := idFilter(id)
	if err != nil {
		return fmt.Errorf("MongoRepository.Update: %w", err)
	}

	res, err := r.Collection.ReplaceOne(ctx, filter, doc)
	if err != nil {
		return fmt.Errorf("MongoRepository.Update: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("MongoRepository.Update: document not found")
	}
	return nil
}

// FindOneAndUpdate applies an update pipeline to a single document matching filter
// and returns the updated document. It uses the ReturnDocument=After option so the
// returned value reflects the update.
func (r *MongoRepository[T]) FindOneAndUpdate(ctx context.Context, filter, update bson.D) (*T, error) {
	opts := mongo.FindOneAndUpdateOptions{}
	after := mongo.ReturnDocument(mongo.After)
	opts.ReturnDocument = &after

	var result T
	err := r.Collection.FindOneAndUpdate(ctx, filter, update, &opts).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("MongoRepository.FindOneAndUpdate: document not found")
		}
		return nil, fmt.Errorf("MongoRepository.FindOneAndUpdate: %w", err)
	}
	return &result, nil
}

// Delete removes the document with the given id.
func (r *MongoRepository[T]) Delete(ctx context.Context, id string) error {
	filter, err := idFilter(id)
	if err != nil {
		return fmt.Errorf("MongoRepository.Delete: %w", err)
	}

	res, err := r.Collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("MongoRepository.Delete: %w", err)
	}
	if res.DeletedCount == 0 {
		return fmt.Errorf("MongoRepository.Delete: document not found")
	}
	return nil
}

// FindOne returns the first document that matches filter.
func (r *MongoRepository[T]) FindOne(ctx context.Context, filter bson.D) (*T, error) {
	var result T
	if err := r.Collection.FindOne(ctx, filter).Decode(&result); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("MongoRepository.FindOne: document not found")
		}
		return nil, fmt.Errorf("MongoRepository.FindOne: %w", err)
	}
	return &result, nil
}

// FindMany returns all documents matching filter.
func (r *MongoRepository[T]) FindMany(ctx context.Context, filter bson.D) ([]*T, error) {
	cursor, err := r.Collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("MongoRepository.FindMany: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*T
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("MongoRepository.FindMany decode: %w", err)
	}
	return results, nil
}

// idFilter builds a bson filter for the _id field. It tries to parse id as a
// hex ObjectID first; if that fails it falls back to a plain string filter.
func idFilter(id string) (bson.D, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err == nil {
		return bson.D{{Key: "_id", Value: oid}}, nil
	}
	if id == "" {
		return nil, fmt.Errorf("id must not be empty")
	}
	return bson.D{{Key: "_id", Value: id}}, nil
}

// setIDField uses reflection to write insertedID into the struct field tagged
// `bson:"_id"`. It is called after InsertOne to keep the in-memory struct
// consistent with the persisted document.
func setIDField(doc any, insertedID any) {
	v := reflect.ValueOf(doc)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return
	}
	t := v.Type()
	for i := range t.NumField() {
		field := t.Field(i)
		tag := field.Tag.Get("bson")
		if tag == "_id" || tag == "_id,omitempty" {
			fv := v.Field(i)
			if fv.CanSet() {
				idVal := reflect.ValueOf(insertedID)
				if idVal.Type().AssignableTo(fv.Type()) {
					fv.Set(idVal)
				}
			}
			return
		}
	}
}
