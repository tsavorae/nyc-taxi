package main

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClient   *mongo.Client
	usersCol      *mongo.Collection
	trainHistCol  *mongo.Collection
)

// User represents a registered user stored in MongoDB.
type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username  string             `bson:"username"      json:"username"`
	Password  string             `bson:"password"      json:"-"`
	CreatedAt time.Time          `bson:"created_at"    json:"created_at"`
}

// TrainRecord stores the result of each training run.
type TrainRecord struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID     primitive.ObjectID `bson:"user_id"       json:"user_id"`
	TotalTrees int                `bson:"total_trees"   json:"total_trees"`
	MAE        float64            `bson:"mae"           json:"mae"`
	RMSE       float64            `bson:"rmse"          json:"rmse"`
	R2         float64            `bson:"r2"            json:"r2"`
	DurTotalMS int64              `bson:"dur_total_ms"  json:"dur_total_ms"`
	PerNode    []NodeInfo         `bson:"per_node"      json:"per_node"`
	CreatedAt  time.Time          `bson:"created_at"    json:"created_at"`
}

func connectMongo() {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://admin:secret@localhost:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("mongo connect:", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("mongo ping:", err)
	}

	mongoClient = client
	db := client.Database("nyctaxi")
	usersCol = db.Collection("users")
	trainHistCol = db.Collection("train_history")

	// Unique index on username
	usersCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	log.Println("mongodb connected — db: nyctaxi")
}

func findUserByUsername(username string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var u User
	err := usersCol.FindOne(ctx, bson.M{"username": username}).Decode(&u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func insertUser(u *User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	u.CreatedAt = time.Now()
	res, err := usersCol.InsertOne(ctx, u)
	if err != nil {
		return err
	}
	u.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func insertTrainRecord(rec *TrainRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rec.CreatedAt = time.Now()
	res, err := trainHistCol.InsertOne(ctx, rec)
	if err != nil {
		return err
	}
	rec.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func listTrainHistory(userID primitive.ObjectID, limit int64) ([]TrainRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit)
	cursor, err := trainHistCol.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []TrainRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	return records, nil
}
