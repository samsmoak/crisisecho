package repository

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"crisisecho/internal/apps/location/model"
	mongoRepo "crisisecho/internal/database/abstractrepository/mongodb"
)

type LocationRepository struct {
	*mongoRepo.MongoRepository[model.SavedLocation]
}

func NewLocationRepository(db *mongo.Database) *LocationRepository {
	return &LocationRepository{
		MongoRepository: mongoRepo.NewMongoRepository[model.SavedLocation](db, "saved_locations"),
	}
}

// FindByUserID returns all saved locations belonging to a user.
func (r *LocationRepository) FindByUserID(ctx context.Context, userID primitive.ObjectID) ([]*model.SavedLocation, error) {
	filter := bson.D{{Key: "user_id", Value: userID}}
	results, err := r.FindMany(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("LocationRepository.FindByUserID: %w", err)
	}
	return results, nil
}

// FindByUserIDAndLabel returns a single location matching both user and label.
func (r *LocationRepository) FindByUserIDAndLabel(ctx context.Context, userID primitive.ObjectID, label string) (*model.SavedLocation, error) {
	filter := bson.D{
		{Key: "user_id", Value: userID},
		{Key: "label", Value: label},
	}
	result, err := r.FindOne(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("LocationRepository.FindByUserIDAndLabel: %w", err)
	}
	return result, nil
}

// DeleteByIDAndUserID deletes a location only if it belongs to the given user.
func (r *LocationRepository) DeleteByIDAndUserID(ctx context.Context, id, userID primitive.ObjectID) error {
	filter := bson.D{
		{Key: "_id", Value: id},
		{Key: "user_id", Value: userID},
	}
	res, err := r.Collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("LocationRepository.DeleteByIDAndUserID: %w", err)
	}
	if res.DeletedCount == 0 {
		return fmt.Errorf("LocationRepository.DeleteByIDAndUserID: not found")
	}
	return nil
}
