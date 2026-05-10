// Package api — composição final do router HTTP.
package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/pinas/pinas/internal/auth"
	"github.com/pinas/pinas/internal/files"
	"github.com/pinas/pinas/internal/middleware"
	"github.com/pinas/pinas/internal/samba"
	"github.com/pinas/pinas/internal/system"
	"github.com/pinas/pinas/internal/users"
	"github.com/pinas/pinas/internal/websocket"
	pjwt "github.com/pinas/pinas/pkg/jwt"
)

type Deps struct {
	Logger        *slog.Logger
	JWTIssuer     *pjwt.Issuer
	AuthHandler   *auth.Handler
	UsersHandler  *users.Handler
	FilesHandler  *files.Handler
	SystemHandler *system.Handler
	SambaHandler  *samba.Handler
	WSHub         *websocket.Hub
	AllowedOrigin string
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()

	// Middleware base.
	r.Use(chimw.RealIP)
	r.Use(middleware.Recovery(d.Logger))
	r.Use(middleware.RequestLogger(d.Logger))
	r.Use(middleware.SecurityHeaders)
	r.Use(chimw.Timeout(60 * time.Second))

	// CORS — restrito ao frontend.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{d.AllowedOrigin},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health (público — usado pelo healthcheck do Docker).
	r.Get("/api/v1/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Auth — login/refresh/logout públicos, com rate limit agressivo.
		r.Route("/auth", func(r chi.Router) {
			r.Use(middleware.NewRateLimiter(2, 5)) // 2 req/s, burst 5
			d.AuthHandler.Routes(r)
		})

		// Tudo abaixo exige JWT válido.
		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth(d.JWTIssuer))

			// /me — registrado direto, NÃO via r.Route("/auth")
			// porque o chi não permite Mount() duplicado no mesmo path
			// (já temos r.Route("/auth", ...) acima para login/refresh/logout).
			r.Get("/auth/me", d.AuthHandler.MeHandler)
			r.Post("/auth/password", d.AuthHandler.PasswordHandler)

			// /files — escopado por usuário (admin vê tudo, usuário vê home).
			r.Route("/files", func(r chi.Router) {
				d.FilesHandler.Routes(r)
			})

			// /system/stats — disponível para todos os logados (read-only).
			r.Route("/system", func(r chi.Router) {
				r.Get("/stats", d.SystemHandler.Stats)
			})

			// /users — só admin.
			r.Route("/users", func(r chi.Router) {
				r.Use(middleware.RequireAdmin)
				d.UsersHandler.Routes(r)
			})

			// /samba — só admin (gerencia shares e usuários SMB).
			r.Route("/samba", func(r chi.Router) {
				r.Use(middleware.RequireAdmin)
				d.SambaHandler.Routes(r)
			})

			// WebSocket.
			r.Get("/ws", d.WSHub.ServeHTTP)
		})
	})

	return r
}
