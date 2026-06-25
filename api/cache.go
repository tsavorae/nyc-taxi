package main

import (
	"context"
	"fmt"
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

// ─── Prediction cache ────────────────────────────────────────────────────────
// Stores precalculated predictions so repeated requests with the same features
// are served from Redis instead of re-running the forest.

const predCacheTTL = 1 * time.Hour

// predictionKey builds a deterministic Redis key from the request features.
func predictionKey(f [4]float64) string {
	return fmt.Sprintf("pred:%.4f:%.4f:%.4f:%.4f", f[0], f[1], f[2], f[3])
}

// getCachedPrediction returns a cached prediction and true if present.
func getCachedPrediction(f [4]float64) (float64, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	val, err := rdb.Get(ctx, predictionKey(f)).Float64()
	if err == redis.Nil {
		return 0, false
	}
	if err != nil {
		log.Println("redis get prediction error:", err)
		return 0, false
	}
	return val, true
}

// setCachedPrediction stores a prediction result with TTL.
func setCachedPrediction(f [4]float64, pred float64) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Set(ctx, predictionKey(f), pred, predCacheTTL).Err(); err != nil {
		log.Println("redis set prediction error:", err)
	}
}

// invalidatePredictionCache clears all cached predictions.
// Called after training, since a new model invalidates old predictions.
func invalidatePredictionCache() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	iter := rdb.Scan(ctx, 0, "pred:*", 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if len(keys) > 0 {
		if err := rdb.Del(ctx, keys...).Err(); err != nil {
			log.Println("redis invalidate cache error:", err)
		} else {
			log.Printf("invalidated %d cached predictions\n", len(keys))
		}
	}
}
