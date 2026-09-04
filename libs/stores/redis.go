package stores

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
)

var (
	Ctx = context.Background()
	Rdb *redis.Client
)

func InitRedis() {
	client, err := OpenRedis()
	if err != nil {
		log.Fatalf("Failed to connect Redis: %v", err)
	}
	Rdb = client
	log.Println("[REDIS] Connected OK")
}

func OpenRedis() (*redis.Client, error) {
	db, _ := strconv.Atoi(os.Getenv("REDIS_DB"))
	Rdb = redis.NewClient(&redis.Options{
		Addr:     envOr("REDIS_ADDR", "localhost:6379"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	})
	if err := Rdb.Ping(Ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping Redis: %w", err)
	}
	return Rdb, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
