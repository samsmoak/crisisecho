package repository

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"crisisecho/internal/apps/user/model"
	mongoRepo "crisisecho/internal/database/abstractrepository/mongodb"
)

// UserRepository provides data access for the users collection.
type UserRepository struct {
	*mongoRepo.MongoRepository[model.User]
}

// NewUserRepository constructs a UserRepository backed by the "users" collection.
func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{
		MongoRepository: mongoRepo.NewMongoRepository[model.User](db, "users"),
	}
}

// FindByFirebaseUID returns the user with the given Firebase UID, or nil if not found.
func (r *UserRepository) FindByFirebaseUID(ctx context.Context, uid string) (*model.User, error) {
	filter := bson.D{{Key: "firebase_uid", Value: uid}}
	result, err := r.FindOne(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("UserRepository.FindByFirebaseUID: %w", err)
	}
	return result, nil
}

// UpsertByFirebaseUID finds a user by UID or inserts a new one.
// Uses $setOnInsert so existing users are not overwritten on every login.
func (r *UserRepository) UpsertByFirebaseUID(ctx context.Context, user *model.User) (*model.User, error) {
	filter := bson.D{{Key: "firebase_uid", Value: user.FirebaseUID}}
	update := bson.D{
		{Key: "$setOnInsert", Value: user},
	}
	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var result model.User
	err := r.Collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("UserRepository.UpsertByFirebaseUID: %w", err)
	}
	return &result, nil
}

// UpdateProfile applies a partial update to the user matching the given Firebase UID.
func (r *UserRepository) UpdateProfile(ctx context.Context, uid string, update bson.D) (*model.User, error) {
	filter := bson.D{{Key: "firebase_uid", Value: uid}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var result model.User
	err := r.Collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("UserRepository.UpdateProfile: %w", err)
	}
	return &result, nil
}
