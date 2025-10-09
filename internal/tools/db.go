package tools

import (
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func ConnectDB(databaseURI string) (*mongo.Client, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(databaseURI))
	if err != nil {
		return nil, err
	}

	return client, nil
}
