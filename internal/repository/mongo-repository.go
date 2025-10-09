package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/acertainpoggerman/online-exam-system/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type mongoRepository struct {
	client       *mongo.Client
	databaseName string
}

func NewMongoRepository(c *mongo.Client, name string) *mongoRepository {
	return &mongoRepository{
		client:       c,
		databaseName: name,
	}
}

func (r *mongoRepository) DB() *mongo.Database {
	return r.client.Database(r.databaseName)
}

// [User Methods]

func (r *mongoRepository) CreateUser(ctx context.Context, user models.User) (string, error) {

	coll := r.DB().Collection(CollectionNameUsers)

	result, err := coll.InsertOne(ctx, user)
	if err != nil {
		return "", err
	}

	userID, ok := result.InsertedID.(bson.ObjectID)
	if !ok {
		return "", fmt.Errorf("failed to convert to object ID")
	}

	return userID.Hex(), nil
}

func (r *mongoRepository) FindUsers(ctx context.Context) ([]models.User, error) {
	return []models.User{}, NotImplementedError("FindUsers")
}

func (r *mongoRepository) FindUserByID(ctx context.Context, userID string) (*models.User, error) {

	coll := r.DB().Collection(CollectionNameUsers)

	id, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	var result models.User
	filter := bson.D{{
		Key: "$and",
		Value: bson.A{
			// Checking for ID
			bson.D{{Key: "_id", Value: id}},
			// Checking for Logical Deletion
			bson.D{{Key: "$or", Value: bson.A{
				bson.D{{Key: "deleted_at", Value: nil}},
				bson.D{{Key: "deleted_at", Value: bson.D{{Key: "$gt", Value: time.Now()}}}},
			}}},
		},
	}}

	err = coll.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *mongoRepository) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {

	coll := r.DB().Collection(CollectionNameUsers)

	var result models.User
	filter := bson.D{{Key: "email", Value: email}}

	err := coll.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *mongoRepository) DeleteUserByID(ctx context.Context, userID string) error {

	coll := r.DB().Collection(CollectionNameUsers)

	id, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	update := bson.D{{
		Key: "$set",
		Value: bson.D{{
			Key:   "deleted_at",
			Value: time.Now(),
		}},
	}}

	_, err = coll.UpdateByID(ctx, id, update)
	return err
}

// [Script Methods]

func (r *mongoRepository) CreateScript(ctx context.Context, script models.Script) (*models.Script, error) {

	coll := r.DB().Collection(CollectionNameScripts)

	result, err := coll.InsertOne(ctx, script)
	if err != nil {
		return nil, err
	}

	scriptID, ok := result.InsertedID.(bson.ObjectID)
	if !ok {
		return nil, fmt.Errorf("failed to convert to object ID")
	}

	var created models.Script
	filter := bson.D{{Key: "_id", Value: scriptID}}

	err = coll.FindOne(ctx, filter).Decode(&created)
	if err != nil {
		return nil, err
	}

	return &created, nil
}

func (r *mongoRepository) FindScripts(ctx context.Context) ([]models.Script, error) {
	return nil, NotImplementedError("GetScripts")
}

func (r *mongoRepository) FindScriptByID(ctx context.Context, scriptID string) (*models.Script, error) {

	coll := r.DB().Collection(CollectionNameScripts)

	id, err := bson.ObjectIDFromHex(scriptID)
	if err != nil {
		return nil, err
	}

	var result models.Script
	filter := bson.D{{
		Key: "$and",
		Value: bson.A{
			// Checking for ID
			bson.D{{Key: "_id", Value: id}},
			// Checking for Logical Deletion
			bson.D{{Key: "$or", Value: bson.A{
				bson.D{{Key: "deleted_at", Value: nil}},
				bson.D{{Key: "deleted_at", Value: bson.D{{Key: "$gt", Value: time.Now()}}}},
			}}},
		},
	}}

	err = coll.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *mongoRepository) DeleteScriptByID(ctx context.Context, scriptID string) error {

	coll := r.DB().Collection(CollectionNameScripts)

	id, err := bson.ObjectIDFromHex(scriptID)
	if err != nil {
		return err
	}

	update := bson.D{{
		Key: "$set",
		Value: bson.D{{
			Key:   "deleted_at",
			Value: time.Now(),
		}},
	}}

	_, err = coll.UpdateByID(ctx, id, update)
	return err
}

func (r *mongoRepository) UpdateScript(ctx context.Context, script models.Script) (*models.Script, error) {

	coll := r.DB().Collection(CollectionNameScripts)

	update := bson.D{{
		Key: "$set",
		Value: bson.D{
			{
				Key:   "title",
				Value: script.Title,
			},
			{
				Key:   "description",
				Value: script.Title,
			},

			{
				Key:   "questions",
				Value: script.Questions,
			},
			{
				Key:   "modified_at",
				Value: time.Now(),
			},
		},
	}}

	_, err := coll.UpdateByID(ctx, script.ID, update)
	if err != nil {
		return nil, err
	}

	var updated models.Script
	filter := bson.D{{Key: "_id", Value: script.ID}}

	err = coll.FindOne(ctx, filter).Decode(&updated)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return &updated, nil
}

// [Session Methods]

func (r *mongoRepository) CreateSession(ctx context.Context, session models.Session) (*models.Session, error) {

	coll := r.DB().Collection(CollectionNameSessions)

	result, err := coll.InsertOne(ctx, session)
	if err != nil {
		return nil, err
	}

	sessionID, ok := result.InsertedID.(bson.ObjectID)
	if !ok {
		return nil, fmt.Errorf("failed to convert to object ID")
	}

	var created models.Session
	filter := bson.D{{Key: "_id", Value: sessionID}}

	err = coll.FindOne(ctx, filter).Decode(&created)
	if err != nil {
		return nil, err
	}

	return &created, nil
}

func (r *mongoRepository) FindSessions(ctx context.Context) ([]models.Session, error) {
	return nil, NotImplementedError("FindSessions")
}

func (r *mongoRepository) FindSessionByID(ctx context.Context, sessionID string) (*models.Session, error) {

	coll := r.DB().Collection(CollectionNameSessions)

	id, err := bson.ObjectIDFromHex(sessionID)
	if err != nil {
		return nil, err
	}

	var result models.Session
	filter := bson.D{{
		Key: "$and",
		Value: bson.A{
			// Checking for ID
			bson.D{{Key: "_id", Value: id}},
			// Checking for Logical Deletion
			bson.D{{Key: "$or", Value: bson.A{
				bson.D{{Key: "deleted_at", Value: nil}},
				bson.D{{Key: "deleted_at", Value: bson.D{{Key: "$gt", Value: time.Now()}}}},
			}}},
		},
	}}

	err = coll.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *mongoRepository) DeleteSessionByID(ctx context.Context, sessionID string) error {

	coll := r.DB().Collection(CollectionNameSessions)

	id, err := bson.ObjectIDFromHex(sessionID)
	if err != nil {
		return err
	}

	update := bson.D{{
		Key: "$set",
		Value: bson.D{{
			Key:   "deleted_at",
			Value: time.Now(),
		}},
	}}

	_, err = coll.UpdateByID(ctx, id, update)
	return err
}

func (r *mongoRepository) UpdateSession(ctx context.Context, session models.Session) (*models.Session, error) {

	coll := r.DB().Collection(CollectionNameSessions)

	update := bson.D{{
		Key: "$set",
		Value: bson.D{
			{
				Key:   "status",
				Value: session.Status,
			},
			{
				Key:   "start_time",
				Value: session.StartTime,
			},
			{
				Key:   "duration_mins",
				Value: session.DurationMins,
			},
			{
				Key:   "grace_duration_mins",
				Value: session.GraceDurationMins,
			},
			{
				Key:   "modified_at",
				Value: time.Now(),
			},
		},
	}}

	_, err := coll.UpdateByID(ctx, session.ID, update)
	if err != nil {
		return nil, err
	}

	var updated models.Session
	filter := bson.D{{Key: "_id", Value: session.ID}}

	err = coll.FindOne(ctx, filter).Decode(&updated)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return &updated, nil
}
