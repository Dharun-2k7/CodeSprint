package database

import (
	"context"
	"log"
	"os"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

// InitRedis initializes the optional Redis client.
func InitRedis() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Println("REDIS_URL not set. Running in Redis-less (safe) mode.")
		return
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Printf("Failed to parse REDIS_URL: %v. Running in Redis-less mode.", err)
		return
	}

	client := redis.NewClient(opt)

	// Ping the Redis server to verify the connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Printf("Failed to connect to Redis: %v. Running in Redis-less mode.", err)
		return
	}

	log.Println("Successfully connected to Redis.")
	RedisClient = client
}
