package main

import (
	"log"
	"os"
	"sync"

	"github.com/ChimeraCoder/anaconda"
)

var api *anaconda.TwitterApi

func init() {
	api = anaconda.NewTwitterApiWithCredentials(
		os.Getenv("ACCESS_TOKEN"),
		os.Getenv("ACCESS_TOKEN_SECRET"),
		os.Getenv("CONSUMER_KEY"),
		os.Getenv("CONSUMER_SECRET"),
	)

	log.Println("init done")
}

func main() {
	for {
		workers := 10
		var wg sync.WaitGroup
		ids := make(chan int64)
		for i := 0; i < workers; i++ {
			go followFollowers(ids, &wg)
		}

		friends := api.GetFriendsIdsAll(nil)

		for f := range friends {
			for _, id := range f.Ids {
				ids <- id
				wg.Add(1)
			}
		}

		close(ids)
		wg.Wait()
	}
}

func followFollowers(ids <-chan int64, wg *sync.WaitGroup) {
	for id := range ids {
		cursor, _ := api.GetFollowersUser(id, nil)

		for _, followID := range cursor.Ids {
			log.Println("following id:", followID)
			api.FollowUserId(followID, nil)
		}

		wg.Done()
	}
}
