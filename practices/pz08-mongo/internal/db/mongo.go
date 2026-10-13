package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const timeout = 5 * time.Second

func Connect(ctx context.Context, uri, database string) (*mongo.Client, *mongo.Collection, error) {
	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client, err := mongo.Connect(connectCtx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, nil, fmt.Errorf("connect to MongoDB: %w", err)
	}
	if err := Ping(ctx, client); err != nil {
		_ = Disconnect(client)
		return nil, nil, err
	}
	collection := client.Database(database).Collection("notes")
	indexCtx, indexCancel := context.WithTimeout(ctx, timeout)
	defer indexCancel()
	_, err = collection.Indexes().CreateOne(indexCtx, mongo.IndexModel{
		Keys:    bson.D{{Key: "title", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("unique_title"),
	})
	if err != nil {
		_ = Disconnect(client)
		return nil, nil, fmt.Errorf("create title index: %w", err)
	}
	return client, collection, nil
}

func Ping(ctx context.Context, client *mongo.Client) error {
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		return fmt.Errorf("ping MongoDB: %w", err)
	}
	return nil
}

func Disconnect(client *mongo.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return client.Disconnect(ctx)
}
