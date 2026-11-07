package httpapi

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type limitWindow struct {
	start time.Time
	count int
}

type RateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	now      func() time.Time
	clients  map[string]limitWindow
	requests uint64
}

func NewRateLimiter(limit int, window time.Duration, now func() time.Time) *RateLimiter {
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	if now == nil {
		now = time.Now
	}
	return &RateLimiter{limit: limit, window: window, now: now, clients: make(map[string]limitWindow)}
}

func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/matches") {
			next.ServeHTTP(w, r)
			return
		}
		key := clientKey(r.RemoteAddr)
		now := l.now()
		l.mu.Lock()
		l.requests++
		if l.requests%256 == 0 {
			for client, previous := range l.clients {
				if now.Sub(previous.start) >= 2*l.window {
					delete(l.clients, client)
				}
			}
		}
		entry := l.clients[key]
		if entry.start.IsZero() || now.Sub(entry.start) >= l.window {
			entry = limitWindow{start: now}
		}
		if entry.count >= l.limit {
			l.clients[key] = entry
			l.mu.Unlock()
			retryAfter := int(l.window.Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			writeError(w, http.StatusTooManyRequests, "rate_limited", "too many requests; retry later")
			return
		}
		entry.count++
		l.clients[key] = entry
		l.mu.Unlock()
		next.ServeHTTP(w, r)
	})
}

func clientKey(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}
