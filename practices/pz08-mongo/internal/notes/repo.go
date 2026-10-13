package notes

import (
	"context"
	"errors"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Store interface {
	Create(context.Context, string, string) (Note, error)
	List(context.Context, string, int64, int64) ([]Note, error)
	Get(context.Context, primitive.ObjectID) (Note, error)
	Update(context.Context, primitive.ObjectID, Update) (Note, error)
	Delete(context.Context, primitive.ObjectID) error
}

type Repo struct{ collection *mongo.Collection }

func NewRepo(collection *mongo.Collection) *Repo { return &Repo{collection: collection} }

func (r *Repo) Create(ctx context.Context, title, content string) (Note, error) {
	now := time.Now().UTC()
	note := Note{ID: primitive.NewObjectID(), Title: title, Content: content, CreatedAt: now, UpdatedAt: now}
	_, err := r.collection.InsertOne(ctx, note)
	if err != nil {
		return Note{}, mapError(err)
	}
	return note, nil
}

func (r *Repo) List(ctx context.Context, q string, limit, skip int64) ([]Note, error) {
	filter := bson.M{}
	if q != "" {
		filter["title"] = bson.M{"$regex": regexp.QuoteMeta(q), "$options": "i"}
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}).SetLimit(limit).SetSkip(skip)
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	notes := make([]Note, 0)
	if err := cursor.All(ctx, &notes); err != nil {
		return nil, err
	}
	return notes, nil
}

func (r *Repo) Get(ctx context.Context, id primitive.ObjectID) (Note, error) {
	var note Note
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&note)
	if err != nil {
		return Note{}, mapError(err)
	}
	return note, nil
}

func (r *Repo) Update(ctx context.Context, id primitive.ObjectID, patch Update) (Note, error) {
	set := bson.M{"updatedAt": time.Now().UTC()}
	if patch.Title != nil {
		set["title"] = *patch.Title
	}
	if patch.Content != nil {
		set["content"] = *patch.Content
	}
	var note Note
	err := r.collection.FindOneAndUpdate(ctx, bson.M{"_id": id}, bson.M{"$set": set}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&note)
	if err != nil {
		return Note{}, mapError(err)
	}
	return note, nil
}

func (r *Repo) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func mapError(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrNotFound
	}
	if mongo.IsDuplicateKeyError(err) {
		return ErrDuplicateTitle
	}
	return err
}
