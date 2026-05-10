// handler.go — rotas HTTP do módulo Samba.
package samba

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes registra as rotas. Espera estar atrás de middleware admin.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/state", h.getState)

	r.Get("/shares", h.listShares)
	r.Post("/shares", h.createShare)
	r.Get("/shares/{name}", h.getShare)
	r.Put("/shares/{name}", h.updateShare)
	r.Delete("/shares/{name}", h.deleteShare)

	r.Get("/users", h.listUsers)
	r.Post("/users", h.createUser)
	r.Post("/users/{name}/password", h.setUserPassword)
	r.Patch("/users/{name}", h.updateUser)
	r.Delete("/users/{name}", h.deleteUser)
}

// ----- shares -----

func (h *Handler) getState(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.GetState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Remove hashes / senhas pendentes antes de devolver.
	for i := range st.Users {
		st.Users[i].NTHash = ""
	}
	writeJSON(w, http.StatusOK, st)
}

func (h *Handler) listShares(w http.ResponseWriter, r *http.Request) {
	shares, err := h.svc.ListShares()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"shares": shares})
}

func (h *Handler) getShare(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	sh, err := h.svc.GetShare(name)
	if err != nil {
		if errors.Is(err, ErrShareNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sh)
}

func (h *Handler) createShare(w http.ResponseWriter, r *http.Request) {
	var sh Share
	if err := json.NewDecoder(r.Body).Decode(&sh); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	if err := h.svc.CreateShare(sh); err != nil {
		switch {
		case errors.Is(err, ErrShareExists):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrInvalidName), errors.Is(err, ErrInvalidPath), errors.Is(err, ErrReservedName):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusCreated, sh)
}

func (h *Handler) updateShare(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var sh Share
	if err := json.NewDecoder(r.Body).Decode(&sh); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	if sh.Name == "" {
		sh.Name = name
	}
	if err := h.svc.UpdateShare(name, sh); err != nil {
		switch {
		case errors.Is(err, ErrShareNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrInvalidName), errors.Is(err, ErrInvalidPath):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, sh)
}

func (h *Handler) deleteShare(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := h.svc.DeleteShare(name); err != nil {
		if errors.Is(err, ErrShareNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// ----- users -----

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.svc.ListUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"users": users})
}

type createUserBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var body createUserBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	if err := h.svc.CreateUser(CreateUserInput{Username: body.Username, Password: body.Password}); err != nil {
		switch {
		case errors.Is(err, ErrUserExists):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrInvalidName):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"username": body.Username})
}

type setPasswordBody struct {
	Password string `json:"password"`
}

func (h *Handler) setUserPassword(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var body setPasswordBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	if err := h.svc.SetUserPassword(name, body.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type updateUserBody struct {
	Disabled *bool `json:"disabled,omitempty"`
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var body updateUserBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	if body.Disabled != nil {
		if err := h.svc.SetUserDisabled(name, *body.Disabled); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := h.svc.DeleteUser(name); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// ----- helpers -----

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
