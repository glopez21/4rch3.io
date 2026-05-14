package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	mu       sync.RWMutex
	visitors map[string]*rate.Limiter
	r        rate.Limit
	burst    int
}

func NewRateLimiter(r rate.Limit, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*rate.Limiter),
		r:        r,
		burst:    burst,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.RLock()
	limiter, ok := rl.visitors[key]
	rl.mu.RUnlock()
	if ok {
		return limiter
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if limiter, ok := rl.visitors[key]; ok {
		return limiter
	}
	limiter = rate.NewLimiter(rl.r, rl.burst)
	rl.visitors[key] = limiter
	return limiter
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		limiter := rl.getLimiter(host)
		if !limiter.Allow() {
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(10 * time.Minute)
		rl.mu.Lock()
		rl.visitors = make(map[string]*rate.Limiter)
		rl.mu.Unlock()
	}
}
