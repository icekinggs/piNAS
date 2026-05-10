// Package middleware — auth JWT, rate-limit, recovery, audit.
package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	pjwt "github.com/pinas/pinas/pkg/jwt"
	"golang.org/x/time/rate"
)

type ctxKey string

const (
	CtxKeyClaims ctxKey = "claims"
)

// FromContext recupera as claims se o request foi autenticado.
func FromContext(ctx context.Context) (*pjwt.Claims, bool) {
	c, ok := ctx.Value(CtxKeyClaims).(*pjwt.Claims)
	return c, ok
}

// Recovery captura panics, loga, devolve 500 sem expor stack ao cliente.
func Recovery(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered",
						"err", rec,
						"path", r.URL.Path,
						"stack", string(debug.Stack()),
					)
					writeJSON(w, http.StatusInternalServerError, map[string]string{
						"error": "erro interno",
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestLogger loga método, path, status, duração.
func RequestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: 200}
			next.ServeHTTP(rec, r)
			log.Info("http",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"dur_ms", time.Since(start).Milliseconds(),
				"ip", clientIP(r),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// JWTAuth exige access token válido em Authorization: Bearer <token>.
// Injeta *pjwt.Claims no contexto.
func JWTAuth(issuer *pjwt.Issuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			tok := ""
			if strings.HasPrefix(h, "Bearer ") {
				tok = strings.TrimPrefix(h, "Bearer ")
			} else {
				tok = websocketProtocolToken(r.Header.Get("Sec-WebSocket-Protocol"))
			}
			if tok == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing token"})
				return
			}
			claims, err := issuer.Parse(tok)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
				return
			}
			ctx := context.WithValue(r.Context(), CtxKeyClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func websocketProtocolToken(header string) string {
	for _, p := range strings.Split(header, ",") {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "pinas.jwt.") {
			return strings.TrimPrefix(p, "pinas.jwt.")
		}
	}
	return ""
}

// RequireAdmin exige role=admin (deve vir após JWTAuth).
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, ok := FromContext(r.Context())
		if !ok || c.Role != "admin" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RateLimitByIP — token bucket por IP, em memória (suficiente para 1 instância).
// Para múltiplas instâncias, mover para Redis. No PiNAS (1 host), in-memory é ótimo.
type ipLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	b        int
}

func NewRateLimiter(rps float64, burst int) func(http.Handler) http.Handler {
	il := &ipLimiter{
		limiters: make(map[string]*rate.Limiter),
		r:        rate.Limit(rps),
		b:        burst,
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			il.mu.Lock()
			lim, ok := il.limiters[ip]
			if !ok {
				lim = rate.NewLimiter(il.r, il.b)
				il.limiters[ip] = lim
			}
			il.mu.Unlock()
			if !lim.Allow() {
				writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limited"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	// Caddy seta X-Real-IP. Se não tiver, cai para RemoteAddr.
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// pega o primeiro
		if i := strings.Index(ip, ","); i >= 0 {
			return strings.TrimSpace(ip[:i])
		}
		return ip
	}
	return r.RemoteAddr
}

// ClientIP é exportado para handlers que querem registrar o IP em audit log.
func ClientIP(r *http.Request) string { return clientIP(r) }

// SecurityHeaders aplica headers de segurança redundantes (Caddy já faz, defesa em profundidade).
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
