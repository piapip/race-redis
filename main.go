package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
)

type CacheData struct {
	Value int
}

func (c *CacheData) String() string {
	return fmt.Sprintf("Value: %d", c.Value)
}

func main() {
	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	normalKey := "normal_counter"
	transactionKey := "transaction_counter"
	versionKey := fmt.Sprintf("%s_version", transactionKey)

	ctx := context.Background()

	startingValue := &CacheData{Value: 0}

	bytes, err := json.Marshal(startingValue)
	if err != nil {
		fmt.Println(err)
	}

	// initiate normal key
	err = redisClient.Set(ctx, normalKey, bytes, 0).Err()
	if err != nil {
		fmt.Println(err)
	}

	// initiate transaction key
	err = redisClient.Set(ctx, transactionKey, bytes, 0).Err()
	if err != nil {
		fmt.Println(err)
	}

	// initiate version key
	err = redisClient.Set(ctx, versionKey, 100, 0).Err()
	if err != nil {
		fmt.Println(err)
	}

	numGoroutines := 10

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
		fmt.Println(err)
	}

	err = json.Unmarshal(finalValBytes, &normalValResult)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("final normal counter = %s\n", normalValResult.String())

	// Check the final transaction value
	var transactionValResult CacheData
	transactionValBytes, err := redisClient.Get(ctx, transactionKey).Bytes()
	if err != nil {
		fmt.Println(err)
	}

	err = json.Unmarshal(transactionValBytes, &transactionValResult)
	if err != nil {
		fmt.Println(err)
	}

	// fmt.Printf("final normal counter = %s\n", transactionValResult.String())

	finalVersion, err := redisClient.Get(ctx, versionKey).Int()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("final normal counter = %s final version = %d\n", transactionValResult.String(), finalVersion)
}

// normalGetAndSet does:
// - Get the data
// - Update the data and set it
func normalGetAndSet(ctx context.Context, redisClient *redis.Client, key string) {
	// Get the data
	res, err := redisClient.Get(ctx, key).Bytes()
	if err != nil {
		fmt.Println(err)
	}

	var result CacheData
	err = json.Unmarshal(res, &result)
	if err != nil {
		fmt.Println(err)
	}

	// Update the data and set it
	result.Value = result.Value + 1

	bytes, err := json.Marshal(result)
	if err != nil {
		fmt.Println(err)
	}

	err = redisClient.Set(ctx, key, bytes, 0).Err()
	if err != nil {
		fmt.Println(err)
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
				fmt.Println(err)
			}

			// Update the data and set it
			result.Value = result.Value + 1

			bytes, err := json.Marshal(result)
			if err != nil {
				fmt.Println(err)
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
				fmt.Println(err)
			}

			// Update the data and set it
			result.Value = result.Value + 1

			bytes, err := json.Marshal(result)
			if err != nil {
				fmt.Println(err)
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
