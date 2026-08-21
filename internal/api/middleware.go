package api

import (
	"crypto/subtle"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/cors"
)

// RemoteAuthMiddleware enforces Bearer token authentication for non-loopback requests.
func RemoteAuthMiddleware(authToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If no auth token is configured, allow all requests
			if authToken == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check if caller is loopback
			if isLoopback(r.RemoteAddr) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract token from Authorization header or ?token= query parameter
			token := ""
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			} else if strings.HasPrefix(authHeader, "bearer ") {
				token = strings.TrimPrefix(authHeader, "bearer ")
			} else {
				token = r.URL.Query().Get("token")
			}

			// Constant-time token verification
			if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(authToken)) != 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"detail": "Remote access requires an access token.",
					"auth":   "token",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddleware configures cross-origin resource sharing headers.
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	origins := []string{"*"}
	if len(allowedOrigins) > 0 {
		origins = allowedOrigins
	}

	return cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Origin", "X-Requested-With"},
		ExposedHeaders:   []string{"Link", "Content-Length"},
		AllowCredentials: true,
		MaxAge:           300,
	})
}

// isLoopback checks if an IP address string belongs to the loopback interface.
func isLoopback(remoteAddr string) bool {
	if remoteAddr == "" {
		return true
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	if host == "" || host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "::ffff:127.0.0.1" {
		return true
	}

	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return true
	}

	return false
}

// ResponseWriterWrapper wraps http.ResponseWriter to capture status code for logging.
type ResponseWriterWrapper struct {
	http.ResponseWriter
	StatusCode int
}

func (rw *ResponseWriterWrapper) WriteHeader(code int) {
	rw.StatusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *ResponseWriterWrapper) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// RequestLogger creates a simple logging middleware.
func RequestLogger(logger func(string, ...interface{})) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &ResponseWriterWrapper{ResponseWriter: w, StatusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)
			if logger != nil && !strings.HasPrefix(r.URL.Path, "/_astro") && !strings.HasPrefix(r.URL.Path, "/assets") {
				duration := time.Since(start)
				logger("%s %s %d %s", r.Method, r.URL.Path, wrapped.StatusCode, duration)
			}
		})
	}
}
