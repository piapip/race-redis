package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/go-redsync/redsync/v4"
	"github.com/redis/go-redis/v9"
)

const (
	normalKey              = "normal_counter"
	transactionKey         = "transaction_counter"
	transactionKeyWithLock = "transaction_counter_lock"
	numGoroutines          = 10
)

var (
	ctx        = context.Background()
	versionKey = fmt.Sprintf("%s_version", transactionKey)
)

type CacheData struct {
	Value int
}

func (c *CacheData) String() string {
	return fmt.Sprintf("Value: %d", c.Value)
}

func main() {
	standAloneRedis()
	// redisCluster := redis.NewClusterClient(&redis.ClusterOptions{
	// 	// Check the CLUSTER SLOTS
	// 	Addrs: []string{
	// 		":7000",
	// 		":7004",
	// 	},
	// })

	// pool := goredis.NewPool(redisCluster)

	// // Create an instance of redisync to be used to obtain a mutual exclusion lock.
	// redsyncInstance := redsync.New(pool)

	// // Obtain a new mutex by using the same name for all instances wanting the same lock.
	// mutexname := "my-global-mutex"
	// mutex := redsyncInstance.NewMutex(mutexname)

	// // Initialize Redis cluster
	// startingValue := &CacheData{Value: 0}

	// initiateCluster(redisCluster, startingValue)

	// var wg sync.WaitGroup
	// wg.Add(numGoroutines)

	// for i := 0; i < numGoroutines; i++ {
	// 	go func(id int) {
	// 		defer wg.Done()

	// 		transactionWithLock(ctx, redisCluster, mutex, transactionKeyWithLock)
	// 	}(i)
	// }

	// wg.Wait()

	// var finalValResult CacheData
	// finalValBytes, err := redisCluster.Get(ctx, transactionKeyWithLock).Bytes()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// err = json.Unmarshal(finalValBytes, &finalValResult)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Printf("final counter = %s\n", finalValResult.String())
}

func standAloneRedis() {
	// Initialize Redis client
	startingValue := &CacheData{Value: 0}

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	err := initiateClient(redisClient, startingValue)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			normalGetAndSet(ctx, redisClient, normalKey)
		}(i)
	}

	wg.Wait()

	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			// transactionGetAndSet(ctx, redisClient, transactionKey)
			transactionGetAndSetWithVersion(ctx, redisClient, transactionKey)
		}(i)
	}

	wg.Wait()

	fmt.Printf("Starting normal counter: %s (Value increment by 1)\n", startingValue.String())
	fmt.Printf("Starting version 0 (Version increment by 1)\n")
	fmt.Println()

	fmt.Printf("After %d cycle:\n", numGoroutines)
	// Check the final normal value
	var normalValResult CacheData
	finalValBytes, err := redisClient.Get(ctx, normalKey).Bytes()
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(finalValBytes, &normalValResult)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("final normal counter = %s\n", normalValResult.String())

	// Check the final transaction value
	var transactionValResult CacheData
	transactionValBytes, err := redisClient.Get(ctx, transactionKey).Bytes()
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(transactionValBytes, &transactionValResult)
	if err != nil {
		log.Fatal(err)
	}

	// fmt.Printf("final counter = %s\n", transactionValResult.String())

	finalVersion, err := redisClient.Get(ctx, versionKey).Int()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("final counter = %s final version = %d\n", transactionValResult.String(), finalVersion)
}

func initiateCluster(redisCluster *redis.ClusterClient, startingValue *CacheData) error {
	bytes, err := json.Marshal(startingValue)
	if err != nil {
		return err
	}

	// initiate key
	err = redisCluster.Set(ctx, transactionKeyWithLock, bytes, 0).Err()
	if err != nil {
		return err
	}

	return nil
}

func initiateClient(redisClient *redis.Client, startingValue *CacheData) error {
	bytes, err := json.Marshal(startingValue)
	if err != nil {
		return err
	}

	// initiate normal key
	err = redisClient.Set(ctx, normalKey, bytes, 0).Err()
	if err != nil {
		return err
	}

	// initiate transaction key
	err = redisClient.Set(ctx, transactionKey, bytes, 0).Err()
	if err != nil {
		return err
	}

	// initiate version key
	err = redisClient.Set(ctx, versionKey, 100, 0).Err()
	if err != nil {
		return err
	}

	return nil
}

// normalGetAndSet does:
// - Get the data
// - Update the data and set it
func normalGetAndSet(ctx context.Context, redisClient *redis.Client, key string) {
	// Get the data
	res, err := redisClient.Get(ctx, key).Bytes()
	if err != nil {
		log.Fatal(err)
	}

	var result CacheData
	err = json.Unmarshal(res, &result)
	if err != nil {
		log.Fatal(err)
	}

	// Update the data and set it
	result.Value = result.Value + 1

	bytes, err := json.Marshal(result)
	if err != nil {
		log.Fatal(err)
	}

	err = redisClient.Set(ctx, key, bytes, 0).Err()
	if err != nil {
		log.Fatal(err)
	}
}

func transactionGetAndSet(ctx context.Context, redisClient *redis.Client, key string) {
	for {
		err := redisClient.Watch(ctx, func(tx *redis.Tx) error {
			// Get the data
			res, err := tx.Get(ctx, key).Bytes()
			if err != nil && err != redis.Nil {
				return err
			}

			var result CacheData
			err = json.Unmarshal(res, &result)
			if err != nil {
				log.Fatal(err)
			}

			// Update the data and set it
			result.Value = result.Value + 1

			bytes, err := json.Marshal(result)
			if err != nil {
				log.Fatal(err)
			}

			_, err = tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
				p.Set(ctx, key, bytes, 0)

				return nil
			})

			return err
		}, key)
		if err == nil {
			return
		}
	}
}

func transactionGetAndSetWithVersion(ctx context.Context, redisClient *redis.Client, key string) {
	versionKey := fmt.Sprintf("%s_version", key)
	for {
		err := redisClient.Watch(ctx, func(tx *redis.Tx) error {
			// Get the data
			res, err := tx.Get(ctx, key).Bytes()
			if err != nil && err != redis.Nil {
				return err
			}

			var result CacheData
			err = json.Unmarshal(res, &result)
			if err != nil {
				log.Fatal(err)
			}

			// Update the data and set it
			result.Value = result.Value + 1

			bytes, err := json.Marshal(result)
			if err != nil {
				log.Fatal(err)
			}

			_, err = tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
				p.Set(ctx, key, bytes, 0)

				// Update the version to it will stop other transactions.
				p.Incr(ctx, versionKey)

				return nil
			})

			return err
		}, versionKey)
		if err == nil {
			return
		}
	}
}

func transactionWithLock(
	ctx context.Context,
	redisClusterClient *redis.ClusterClient,
	mutex *redsync.Mutex,
	key string,
) {
	if err := mutex.Lock(); err != nil {
		log.Fatal(err)
	}

	res, err := redisClusterClient.Get(ctx, key).Bytes()
	if err != nil {
		log.Fatal(err)
	}

	var result CacheData
	err = json.Unmarshal(res, &result)
	if err != nil {
		log.Fatal(err)
	}

	// Update the data and set it
	result.Value = result.Value + 1

	bytes, err := json.Marshal(result)
	if err != nil {
		log.Fatal(err)
	}

	err = redisClusterClient.Set(ctx, key, bytes, 0).Err()
	if err != nil {
		log.Fatal(err)
	}

	if ok, err := mutex.Unlock(); !ok || err != nil {
		log.Fatal("unlock failed")
	}
}
