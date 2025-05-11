package main

import (
	"context"
	"log"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/joho/godotenv"
	"github.com/rusabd/relay/api"
	"github.com/rusabd/relay/pkg/db"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	log := logrus.New()
	log.Out = os.Stdout
	log.SetFormatter(&logrus.TextFormatter{})

	db, err := db.NewMongoDBRelay(context.Background(), os.Getenv("MONGODB_URI"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = api.SetupRouter(db)
	if err != nil {
		log.Fatal(err)
	}

}
