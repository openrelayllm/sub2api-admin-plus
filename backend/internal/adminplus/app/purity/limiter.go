package purity

import (
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"strings"
	"sync"
	"time"
)

type publicLimiter struct {
	mu      sync.Mutex
	entries map[string]limitBucket
}

type limitBucket struct {
	Count   int
	ResetAt time.Time
}

func newPublicLimiter() *publicLimiter {
	return &publicLimiter{entries: map[string]limitBucket{}}
}

func (s *Service) enforceRateLimit(clientIP string, apiKey string) error {
	if s == nil || s.limiter == nil {
		return nil
	}
	keyHash := sha256Hex(apiKey)
	return s.limiter.allow(s.currentTime(), clientIP, keyHash)
}

func (l *publicLimiter) allow(now time.Time, clientIP string, keyHash string) error {
	if l == nil {
		return nil
	}
	ipKey := strings.TrimSpace(clientIP)
	if ipKey == "" {
		ipKey = "unknown"
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cleanup(now)
	if !l.allowBucket(now, "ip-hour:"+ipKey, 5, time.Hour) ||
		!l.allowBucket(now, "ip-day:"+ipKey, 20, 24*time.Hour) ||
		!l.allowBucket(now, "key-hour:"+keyHash, 3, time.Hour) {
		return infraerrors.TooManyRequests("PURITY_RATE_LIMITED", "too many purity checks; retry later")
	}
	return nil
}

func (l *publicLimiter) allowBucket(now time.Time, key string, limit int, window time.Duration) bool {
	bucket := l.entries[key]
	if bucket.ResetAt.IsZero() || !now.Before(bucket.ResetAt) {
		bucket = limitBucket{ResetAt: now.Add(window)}
	}
	if bucket.Count >= limit {
		l.entries[key] = bucket
		return false
	}
	bucket.Count++
	l.entries[key] = bucket
	return true
}

func (l *publicLimiter) cleanup(now time.Time) {
	for key, bucket := range l.entries {
		if !bucket.ResetAt.IsZero() && now.After(bucket.ResetAt) {
			delete(l.entries, key)
		}
	}
}
