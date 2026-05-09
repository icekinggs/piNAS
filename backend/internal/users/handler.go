// Package users — handlers HTTP. Maioria exige role admin (roteado fora).
package users

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/pinas/pinas/internal/middleware"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/{id}", h.get)
	r.Patch("/{id}", h.update)
	r.Delete("/{id}", h.delete)
	r.Post("/{id}/password", h.changePassword)
}

type View struct {
	ID         int64  `json:"id"`
	Username   string `json:"username"`
	Role       string `json:"role"`
	HomePath   string `json:"home_path"`
	QuotaBytes int64  `json:"quota_bytes"`
	Disabled   bool   `json:"disabled"`
	CreatedAt  int64  `json:"created_at"`
}

func toView(u *User) View {
	return View{
		ID: u.ID, Username: u.Username, Role: u.Role,
		HomePath: u.HomePath, QuotaBytes: u.QuotaBytes,
		Disabled: u.Disabled, CreatedAt: u.CreatedAt,
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	views := make([]View, 0, len(list))
	for i := range list {
		views = append(views, toView(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"users": views})
}

type createBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Quota    int64  `json:"quota_bytes"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var body createBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	u, err := h.svc.Create(r.Context(), CreateInput{
		Username: body.Username, Password: body.Password,
		Role: body.Role, Quota: body.Quota,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrUsernameTaken):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrInvalidUsername), errors.Is(err, ErrInvalidPassword):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusCreated, toView(u))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}
	u, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toView(u))
}

type updateBody struct {
	Disabled *bool   `json:"disabled,omitempty"`
	Role     *string `json:"role,omitempty"`
	Quota    *int64  `json:"quota_bytes,omitempty"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body updateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	if body.Disabled != nil {
		if err := h.svc.SetDisabled(r.Context(), id, *body.Disabled); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	u, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toView(u))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}
	// Não permita auto-delete do último admin.
	claims, _ := middleware.FromContext(r.Context())
	if claims != nil && claims.UserID == id {
		writeError(w, http.StatusBadRequest, "não pode se autodeletar")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

type passwordBody struct {
	NewPassword string `json:"new_password"`
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body passwordBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	if err := h.svc.ChangePassword(r.Context(), id, body.NewPassword); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
