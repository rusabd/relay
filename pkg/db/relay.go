package db

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBRelay struct {
	client *mongo.Client
	ctx    context.Context
}

func NewMongoDBRelay(ctx context.Context, uri string) (*MongoDBRelay, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	return &MongoDBRelay{
		client: client,
		ctx:    ctx,
	}, nil
}

func (m *MongoDBRelay) Close() error {
	return m.client.Disconnect(m.ctx)
}

func (m *MongoDBRelay) Set(namespace, key string, value map[string]any) (primitive.ObjectID, error) {
	collection := m.client.Database("relay").Collection(namespace)
	result, err := collection.InsertOne(m.ctx, map[string]any{
		"key":       key,
		"value":     value,
	})
	if err != nil {
		return primitive.NilObjectID, err
	}
	return result.InsertedID.(primitive.ObjectID), nil
}

func (m *MongoDBRelay) Get(namespace, key string, version primitive.ObjectID) ([]map[string]any, error) {
    collection := m.client.Database("relay").Collection(namespace)
    var result []map[string]any
    cursor, err := collection.Find(m.ctx, bson.M{
        "_id": bson.M{
            "$gt": version,
        },        
        "key":       key,
    })
    if err != nil {
        return nil, err
    }
    defer cursor.Close(m.ctx)
    for cursor.Next(m.ctx) {
        var doc map[string]any
        log.Printf("doc: %v", cursor.Current)
        if err := cursor.Decode(&doc); err != nil {
            return nil, err
        }
        result = append(result, doc)
    }

    if err := cursor.Err(); err != nil {
        return nil, err
    }
    log.Printf("result: %v", result)
    return result, nil
}