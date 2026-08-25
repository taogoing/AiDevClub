package mcpserver

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"aidevclub/internal/app"
	"aidevclub/internal/platform"
)

func NewHandler(deps Dependencies, cfg *platform.Config, limiter *platform.RateLimiter, infra *app.Infrastructure, logger *slog.Logger) http.Handler {
	sdkHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			actor := actorFromRequest(r)
			return newMCPServer(deps, actor, cfg)
		},
		&mcp.StreamableHTTPOptions{
			Stateless:                  true,
			JSONResponse:               true,
			PropagateRequestCancellation: true,
			MaxRequestBodyBytes:        cfg.MCPMaxBodyBytes,
		},
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealthz)
	mux.HandleFunc("/readyz", handleReadyz(infra))
	mux.Handle("/mcp", sdkHandler)

	return buildMiddlewareChain(mux, cfg, limiter, logger)
}

func buildMiddlewareChain(next http.Handler, cfg *platform.Config, limiter *platform.RateLimiter, logger *slog.Logger) http.Handler {
	handler := next
	handler = requestTimeout(handler, cfg.MCPRequestTimeout)
	handler = rateLimitMiddleware(handler, limiter, cfg)
	handler = bearerAuthMiddleware(handler, cfg.JWTSecret)
	handler = originMiddleware(handler, cfg.MCPAllowedOrigins)
	handler = recoveryMiddleware(handler, logger)
	handler = requestIDMiddleware(handler)
	return handler
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func handleReadyz(infra *app.Infrastructure) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := infra.Ping(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"error"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
}

func actorFromRequest(r *http.Request) Actor {
	if actor, ok := r.Context().Value(actorContextKey{}).(Actor); ok {
		return actor
	}
	return Actor{}
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = generateRequestID()
		}
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r)
	})
}

func recoveryMiddleware(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered", "panic", rec, "path", r.URL.Path)
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func originMiddleware(next http.Handler, allowedOrigins []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}
		if len(allowedOrigins) == 0 {
			http.Error(w, `{"error":"origin not allowed"}`, http.StatusForbidden)
			return
		}
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				next.ServeHTTP(w, r)
				return
			}
		}
		http.Error(w, `{"error":"origin not allowed"}`, http.StatusForbidden)
	})
}

func bearerAuthMiddleware(next http.Handler, jwtSecret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			ctx := context.WithValue(r.Context(), actorContextKey{}, Actor{})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, `{"error":"invalid authorization header"}`, http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(auth, prefix)
		userID, err := platform.ParseAccessToken(jwtSecret, token)
		if err != nil {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), actorContextKey{}, Actor{
			UserID:        userID,
			Authenticated: true,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func rateLimitMiddleware(next http.Handler, limiter *platform.RateLimiter, cfg *platform.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" {
			next.ServeHTTP(w, r)
			return
		}
		actor := actorFromRequest(r)
		var key string
		if actor.Authenticated {
			key = "mcp:user:" + uintToString(actor.UserID)
		} else {
			key = "mcp:ip:" + clientIP(r)
		}
		allowed, err := limiter.Allow(r.Context(), key)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		if !allowed {
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestTimeout(next http.Handler, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" {
			next.ServeHTTP(w, r)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	idx := strings.LastIndex(r.RemoteAddr, ":")
	if idx >= 0 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}

func uintToString(n uint) string {
	return strings.TrimRight(strings.TrimLeft(strings.Replace(
		strings.Repeat("0", 20)+string(rune(n)), "0", "", 1), ""), "")
}

var requestIDCounter uint64

func generateRequestID() string {
	requestIDCounter++
	return "mcp-" + uintToString(uint(requestIDCounter))
}
