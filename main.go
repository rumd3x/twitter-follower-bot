package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/ChimeraCoder/anaconda"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var api *anaconda.TwitterApi
var collection *mongo.Collection

type dbID struct {
	Value int64 `bson:"value"`
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	api = anaconda.NewTwitterApiWithCredentials(
		os.Getenv("TWITTER_ACCESS_TOKEN"),
		os.Getenv("TWITTER_ACCESS_TOKEN_SECRET"),
		os.Getenv("TWITTER_CONSUMER_KEY"),
		os.Getenv("TWITTER_CONSUMER_SECRET"),
	)

	client, err := mongo.Connect(ctx(10), options.Client().ApplyURI(os.Getenv("MONGODB_URI")))

	if err != nil {
		log.Fatal(err)
	}

	collection = client.Database("twitter").Collection("following")
	go syncronizer()

	log.Println("init done")
}

func main() {
	for {
		workers := 5
		var wg sync.WaitGroup
		ids := make(chan int64)
		for i := 0; i < workers; i++ {
			go followFollowers(ids, &wg)
		}

		log.Println("fetching initial data")

		v := url.Values{}
		v.Set("count", "5000")
		friends := api.GetFriendsIdsAll(v)

		for f := range friends {
			for _, id := range f.Ids {
				wg.Add(1)
				ids <- id
			}
		}

		close(ids)
		wg.Wait()
	}
}

func followFollowers(ids <-chan int64, wgParent *sync.WaitGroup) {
	for id := range ids {
		var wg sync.WaitGroup

		var v = url.Values{}
		v.Set("user_id", strconv.FormatInt(id, 10))
		v.Set("count", "5000")
		followIDs := api.GetFollowersIdsAll(v)

		for follows := range followIDs {
			for _, followID := range follows.Ids {
				if isOnDB(followID) {
					log.Println("duplicate:", followID)
					continue
				}

				wg.Add(1)
				go followUser(followID, &wg)
			}
		}

		wg.Wait()
		wgParent.Done()
	}
}

func followUser(id int64, wg *sync.WaitGroup) {
	defer wg.Done()
	u, err := api.FollowUserId(id, nil)

	if err != nil {
		log.Println("failed:", id)
		return
	}

	log.Println("followed:", u.ScreenName)
	insertDB(id)
}

func syncronizer() {
	for {
		log.Println("starting db sync")
		ctx := ctx(10)

		v := url.Values{}
		v.Set("count", "5000")
		fChan := api.GetFriendsIdsAll(v)

		friends := []int64{}
		for fPage := range fChan {
			for _, f := range fPage.Ids {
				friends = append(friends, f)
				if !isOnDB(f) {
					insertDB(f)
				}
			}
		}

		cur, _ := collection.Find(ctx, bson.M{})

		results := []dbID{}
		cur.All(ctx, &results)

		for _, f := range results {
			if !inSlice(friends, f.Value) {
				collection.DeleteOne(ctx, f)
			}
		}

		log.Println("db sync completed")
		time.Sleep(5 * time.Minute)
	}
}

// db stuff

func isOnDB(id int64) bool {
	var data bson.M
	collection.FindOne(ctx(5), bson.M{"value": id}).Decode(&data)
	return (len(data) > 0)
}

func insertDB(id int64) {
	value := bson.M{"value": id}
	collection.InsertOne(ctx(5), value)
}

// utils

func ctx(t time.Duration) context.Context {
	ctx, _ := context.WithTimeout(context.Background(), t*time.Second)
	return ctx
}

func inSlice(slice []int64, val int64) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
