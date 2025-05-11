package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/rusabd/relay/pkg/db"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	log.Printf("Connecting to MongoDB at %s", os.Getenv("MONGODB_URI"))

	listener, err := db.NewMongoDBListener(ctx, os.Getenv("MONGODB_URI"))
	if err != nil {
		log.Fatal("Error creating MongoDB listener:", err)
	}
	defer listener.Close()

	err = listener.Listen()
	if err != nil {
		log.Fatal("Error listening to MongoDB changes:", err)
	}
}
