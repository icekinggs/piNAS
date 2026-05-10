// Package auth — handlers HTTP.
package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/pinas/pinas/internal/middleware"
	"github.com/pinas/pinas/internal/users"
)

type Handler struct {
	svc       *Service
	users     users.Repository
	cookieSecure bool
}

func NewHandler(svc *Service, users users.Repository, cookieSecure bool) *Handler {
	return &Handler{svc: svc, users: users, cookieSecure: cookieSecure}
}

func (h *Handler) Routes(r chi.Router) {
	r.Post("/login", h.login)
	r.Post("/refresh", h.refresh)
	r.Post("/logout", h.logout)
}

// MeHandler é o handler direto de GET /auth/me.
// Registrado fora do r.Route("/auth", ...) porque o chi não permite Mount() duplicado.
func (h *Handler) MeHandler(w http.ResponseWriter, r *http.Request) {
	h.me(w, r)
}

const refreshCookieName = "pinas_refresh"

type loginBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResp struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	User        UserView  `json:"user"`
}

type UserView struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	HomePath string `json:"home_path"`
}

func toView(u *users.User) UserView {
	return UserView{ID: u.ID, Username: u.Username, Role: u.Role, HomePath: u.HomePath}
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var body loginBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	ip := middleware.ClientIP(r)
	ua := r.UserAgent()

	res, u, err := h.svc.Login(r.Context(), body.Username, body.Password, ip, ua)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "credenciais inválidas")
		case errors.Is(err, ErrUserDisabled):
			writeError(w, http.StatusForbidden, "usuário desativado")
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	h.setRefreshCookie(w, res.RefreshToken, res.ExpiresAt)
	writeJSON(w, http.StatusOK, loginResp{
		AccessToken: res.AccessToken,
		ExpiresAt:   res.ExpiresAt,
		User:        toView(u),
	})
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "sem refresh token")
		return
	}
	ip := middleware.ClientIP(r)
	ua := r.UserAgent()

	res, u, err := h.svc.Refresh(r.Context(), cookie.Value, ip, ua)
	if err != nil {
		// Limpa cookie em caso de erro.
		h.clearRefreshCookie(w)
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.setRefreshCookie(w, res.RefreshToken, res.ExpiresAt)
	writeJSON(w, http.StatusOK, loginResp{
		AccessToken: res.AccessToken,
		ExpiresAt:   res.ExpiresAt,
		User:        toView(u),
	})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err == nil {
		_ = h.svc.Logout(r.Context(), cookie.Value)
	}
	h.clearRefreshCookie(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no auth")
		return
	}
	u, err := h.users.GetByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user não encontrado")
		return
	}
	writeJSON(w, http.StatusOK, toView(u))
}

func (h *Handler) setRefreshCookie(w http.ResponseWriter, token string, exp time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/api/v1/auth", // só vai no /auth/*
		Expires:  exp,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *Handler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
