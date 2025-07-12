package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

var redisClient *redis.Client
var ctx = context.Background()

var rateLimitIP int
var rateLimitToken int
var blockTime time.Duration

func Init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
		DB:   0,
	})

	rateLimitIP = getEnvInt("RATE_LIMIT_IP", 5)
	rateLimitToken = getEnvInt("RATE_LIMIT_TOKEN", 10)
	blockTime = time.Duration(getEnvInt("BLOCK_TIME_SECONDS", 300)) * time.Second
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := getEnv(key, fmt.Sprintf("%d", fallback))
	intValue, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("Error parsing environment variable %s: %v", key, err)
	}
	return intValue
}

func rateLimiter(w http.ResponseWriter, r *http.Request) {
	ip := r.RemoteAddr
	token := r.Header.Get("API_KEY")

	var key string
	var limit int

	if token != "" {
		key = fmt.Sprintf("rate_limiter:token:%s", token)
		limit = rateLimitToken
	} else {
		key = fmt.Sprintf("rate_limiter:ip:%s", ip)
		limit = rateLimitIP
	}

	count, err := redisClient.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		http.Error(w, "Error checking rate limit", http.StatusInternalServerError)
		return
	}

	if count >= limit {
		http.Error(w, "You have reached the maximum number of requests or actions allowed within a certain time frame", http.StatusTooManyRequests)
		return
	}

	_, err = redisClient.Incr(ctx, key).Result()
	if err != nil {
		http.Error(w, "Error incrementing rate limit", http.StatusInternalServerError)
		return
	}

	redisClient.Expire(ctx, key, 1*time.Second)

	redisClient.Expire(ctx, key, blockTime)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Request allowed"))
}

func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rateLimiter(w, r)
		next.ServeHTTP(w, r)
	})
}

func main() {
	r := mux.NewRouter()
	r.Use(rateLimitMiddleware)
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, world!"))
	})
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
