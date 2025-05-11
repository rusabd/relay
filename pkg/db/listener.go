package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBListener struct {
	client *mongo.Client
	ctx    context.Context
}

func NewMongoDBListener(ctx context.Context, url string) (*MongoDBListener, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(url))
	if err != nil {
		return nil, err
	}
	return &MongoDBListener{
		client: client,
		ctx:    ctx,
	}, nil
}

func (m *MongoDBListener) Close() error {
	if err := m.client.Disconnect(m.ctx); err != nil {
		return fmt.Errorf("failed to disconnect from MongoDB: %v", err)
	}
	return nil
}
func (m *MongoDBListener) Listen() error {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	log.Printf("Connecting to MongoDB at %s", os.Getenv("MONGODB_URI"))

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGODB_URI")))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	go func() {
		for {
			err := pingMongoDB(ctx, client)
			if err != nil {
				log.Println("Error pinging MongoDB:", err)
			}
			// sleep for 5 second to allow the ping to be processed
			time.Sleep(1 * time.Second)
		}
	}()

	// Select collection
	collection := client.Database("testdb").Collection("testcollection")

	// Start change stream
	stream, err := collection.Watch(ctx, mongo.Pipeline{})
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close(ctx)

	fmt.Println("Listening for changes...")

	// Iterate change stream
	for stream.Next(ctx) {
		var event bson.M
		if err := stream.Decode(&event); err != nil {
			log.Println("Decode error:", err)
			continue
		}
		fmt.Printf("Change detected: %v\n", event)
	}

	return stream.Err()
}

func pingMongoDB(ctx context.Context, client *mongo.Client) error {
	// insert a ping object to the database
	collection := client.Database("testdb").Collection("testcollection")
	ping := bson.M{"ping": 1}
	_, err := collection.InsertOne(ctx, ping)
	if err != nil {
		return fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	return nil
}
