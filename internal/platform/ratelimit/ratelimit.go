// Package ratelimit provides IP-based rate limiting middleware for HTTP handlers.
package ratelimit

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// IPRateLimiter tracks request counts per IP address.
type IPRateLimiter struct {
	mu          sync.RWMutex
	requests    map[string][]time.Time // IP -> list of request times
	maxRequests int                    // Maximum requests allowed
	window      time.Duration          // Time window for counting
}

// NewIPRateLimiter creates a new rate limiter.
// maxRequests: maximum number of requests allowed per window
// window: time duration for the rate limit window
func NewIPRateLimiter(maxRequests int, window time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		requests:    make(map[string][]time.Time),
		maxRequests: maxRequests,
		window:      window,
	}
}

// isAllowed checks if the given IP can make a request.
func (rl *IPRateLimiter) isAllowed(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Get existing requests for this IP and filter out old ones
	var recent []time.Time
	for _, t := range rl.requests[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	// Check if under limit
	if len(recent) >= rl.maxRequests {
		rl.requests[ip] = recent
		return false
	}

	// Add current request
	recent = append(recent, now)
	rl.requests[ip] = recent
	return true
}

// getClientIP extracts the client IP from the request.
// It respects X-Forwarded-For and X-Real-IP headers for proxied requests.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (common for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, use the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// Check X-Real-IP header (Nginx, etc.)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr might not have a port
		return r.RemoteAddr
	}
	return ip
}

// Middleware returns an HTTP middleware that applies rate limiting to password reset endpoints.
func Middleware(limiter *IPRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only apply rate limiting to password reset endpoints
			if !isPasswordResetEndpoint(r.URL.Path, r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			ip := getClientIP(r)
			if !limiter.isAllowed(ip) {
				w.Header().Set("Retry-After", "900") // 15 minutes in seconds
				http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isPasswordResetEndpoint checks if the request is for a password reset endpoint.
func isPasswordResetEndpoint(path, method string) bool {
	// Check POST endpoints (the ones that actually perform actions)
	if method == http.MethodPost {
		return path == "/forgot-password" || path == "/reset-password"
	}
	return false
}
