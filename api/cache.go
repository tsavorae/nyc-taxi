package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client

func connectRedis() {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	rdb = redis.NewClient(&redis.Options{
		Addr: addr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("redis ping:", err)
	}

	log.Println("redis connected —", addr)
}

// blacklistToken stores a JWT token ID (jti) in Redis until it expires.
// Used for logout / token revocation.
func blacklistToken(jti string, expiration time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return rdb.Set(ctx, "bl:"+jti, "1", expiration).Err()
}

// isTokenBlacklisted checks if a JWT token ID has been revoked.
func isTokenBlacklisted(jti string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	val, err := rdb.Exists(ctx, "bl:"+jti).Result()
	if err != nil {
		log.Println("redis exists error:", err)
		return false
	}
	return val > 0
}
