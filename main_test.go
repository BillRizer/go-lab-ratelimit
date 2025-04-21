package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter(t *testing.T) {
	redisAddr := os.Getenv("TEST_REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisClient = redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   0,
	})
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		t.Skipf("Teste skipped: Redis não disponível em %s: %v", redisAddr, err)
		return
	}
	rateLimitIP = 2
	rateLimitToken = 3
	blockTime = 1 * time.Second
	cleanupRedisKeys(t)
	defer cleanupRedisKeys(t)
	tests := []struct {
		name           string
		ip             string
		token          string
		expectedStatus int
		requests       int
	}{
		{
			name:           "IP baseado: dentro do limite",
			ip:             "192.168.1.1",
			token:          "",
			expectedStatus: http.StatusOK,
			requests:       1,
		},
		{
			name:           "IP baseado: no limite",
			ip:             "192.168.1.2",
			token:          "",
			expectedStatus: http.StatusOK,
			requests:       2,
		},
		{
			name:           "IP baseado: excedeu limite",
			ip:             "192.168.1.3",
			token:          "",
			expectedStatus: http.StatusTooManyRequests,
			requests:       3,
		},
		{
			name:           "Token baseado: dentro do limite",
			ip:             "192.168.1.4",
			token:          "test-token-1",
			expectedStatus: http.StatusOK,
			requests:       3,
		},
		{
			name:           "Token baseado: excedeu limite",
			ip:             "192.168.1.5",
			token:          "test-token-2",
			expectedStatus: http.StatusTooManyRequests,
			requests:       4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupSpecificKeys(t, tt.ip, tt.token)
			var lastStatusCode int
			for i := 0; i < tt.requests; i++ {
				req, err := http.NewRequest("GET", "/", nil)
				if err != nil {
					t.Fatalf("Erro ao criar request: %v", err)
				}
				req.RemoteAddr = tt.ip
				if tt.token != "" {
					req.Header.Set("API_KEY", tt.token)
				}
				rr := httptest.NewRecorder()
				rateLimiter(rr, req)
				lastStatusCode = rr.Code
			}
			assert.Equal(t, tt.expectedStatus, lastStatusCode,
				"Status code esperado %d, recebido %d", tt.expectedStatus, lastStatusCode)
		})
	}

	t.Run("Reset após estar expirado", func(t *testing.T) {
		ip := "192.168.1.6"
		key := "rate_limter:ip:" + ip

		redisClient.Del(ctx, key)

		oldBlockTime := blockTime
		blockTime = 1 * time.Second
		defer func() { blockTime = oldBlockTime }()

		for i := 0; i < rateLimitIP; i++ {
			req, _ := http.NewRequest("GET", "/", nil)
			req.RemoteAddr = ip
			rr := httptest.NewRecorder()
			rateLimiter(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
		}
		req, _ := http.NewRequest("GET", "/", nil)
		req.RemoteAddr = ip

		rr := httptest.NewRecorder()
		rateLimiter(rr, req)
		assert.Equal(t, http.StatusTooManyRequests, rr.Code)
		time.Sleep(blockTime + 100*time.Millisecond)
		req, _ = http.NewRequest("GET", "/", nil)
		req.RemoteAddr = ip
		rr = httptest.NewRecorder()
		rateLimiter(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

func cleanupRedisKeys(t *testing.T) {
	keys, err := redisClient.Keys(ctx, "rate_limter:*").Result()
	if err != nil {
		t.Logf("Erro ao buscar chaves para limpeza: %v", err)
		return
	}
	if len(keys) > 0 {
		_, err = redisClient.Del(ctx, keys...).Result()
		if err != nil {
			t.Logf("Erro ao limpar chaves: %v", err)
		}
	}
}

func cleanupSpecificKeys(t *testing.T, ip, token string) {
	var keys []string
	if token != "" {
		keys = append(keys, "rate_limter:token:"+token)
	}
	if ip != "" {
		keys = append(keys, "rate_limter:ip:"+ip)
	}
	if len(keys) > 0 {
		_, err := redisClient.Del(ctx, keys...).Result()
		if err != nil {
			t.Logf("Erro ao limpar chaves específicas: %v", err)
		}
	}
}
