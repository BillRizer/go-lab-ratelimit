package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type RedisMock struct {
	mock.Mock
}

func (r *RedisMock) Get(ctx context.Context, key string) *redis.StringCmd {
	args := r.Called(ctx, key)
	return args.Get(0).(*redis.StringCmd)
}

func (r *RedisMock) Incr(ctx context.Context, key string) *redis.IntCmd {
	args := r.Called(ctx, key)
	return args.Get(0).(*redis.IntCmd)
}

func (r *RedisMock) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	args := r.Called(ctx, key, expiration)
	return args.Get(0).(*redis.BoolCmd)
}

func TestInit(t *testing.T) {
	os.Setenv("RATE_LIMIT_IP", "10")
	os.Setenv("RATE_LIMIT_TOKEN", "20")
	os.Setenv("BLOCK_TIME", "600")
	Init()
	assert.Equal(t, 10, rateLimitIP)
	assert.Equal(t, 20, rateLimitToken)
	assert.Equal(t, 600*time.Second, blockTime)
}

func TestRateLimiterWithIP(t *testing.T) {
	redisMock := new(RedisMock)
	redisMock.On("Get", context.Background(), "rate_limiter:ip:127.0.0.1").Return(&redis.StringCmd{})
	redisMock.On("Incr", context.Background(), "rate_limiter:ip:127.0.0.1").Return(&redis.IntCmd{})
	redisMock.On("Expire", context.Background(), "rate_limiter:ip:127.0.0.1", mock.Anything).Return(&redis.BoolCmd{})
	handler := rateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Request allowed"))
	}))

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.RemoteAddr = "127.0.0.1"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "Request allowed", rr.Body.String())
	redisMock.AssertExpectations(t)
}


func TestRateLimiterWithToken(t *testing.T) {
	redisMock := new(RedisMock)
	redisMock.On("Get", context.Background(), "rate_limiter:token:my-api-key").Return(&redis.StringCmd{})
	redisMock.On("Incr", context.Background(), "rate_limiter:token:my-api-key").Return(&redis.IntCmd{})
	redisMock.On("Expire", context.Background(), "rate_limiter:token:my-api-key", mock.Anything).Return(&redis.BoolCmd{})
	handler := rateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Request allowed"))
	}))

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("API_KEY", "my-api-key")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "Request allowed", rr.Body.String())
	redisMock.AssertExpectations(t)
}

func TestRateLimiterLimitExceeded(t *testing.T) {
	redisMock := new(RedisMock)
	redisMock.On("Get", context.Background(), "rate_limiter:ip:127.0.0.1").Return(&redis.StringCmd{}).Once()
	handler := rateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Request allowed"))
	}))
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RemoteAddr = "127.0.0.1"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
	assert.Equal(t, "You have reached the maximum number of requests or actions allowed within a certain time frame", rr.Body.String())
	redisMock.AssertExpectations(t)
}
