package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func DBinstance() *mongo.Client {

	mongoDb := "mongodb://localhost:27017"
	fmt.Print(mongoDb)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoDb))

	if err = client.Disconnect(ctx); err != nil {
		log.Fatal(err)
	}

	fmt.Println("connected to MongoDB")

	return client
}

var Client *mongo.Client = DBinstance()

func OpenCollection(Client *mongo.Client, collectionName string) *mongo.Collection {
	var collection *mongo.Collection = Client.Database("restaurant").Collection(collectionName)
	return collection
}
