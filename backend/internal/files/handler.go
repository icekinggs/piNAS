// Package files — handlers HTTP para operações sobre o filesystem do NAS.
//
// O storage.Jail é a barreira de segurança. Todo path que vem da rede é
// resolvido pelo jail antes de ser usado.
package files

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/pinas/pinas/internal/middleware"
	"github.com/pinas/pinas/internal/storage"
	"github.com/pinas/pinas/pkg/utils"
)

type Handler struct {
	jail *storage.Jail
}

func NewHandler(jail *storage.Jail) *Handler {
	return &Handler{jail: jail}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Get("/download", h.download)
	r.Post("/upload", h.upload)
	r.Post("/folder", h.mkdir)
	r.Patch("/rename", h.rename)
	r.Post("/move", h.move)
	r.Post("/copy", h.copy)
	r.Delete("/", h.delete)
	r.Get("/stat", h.stat)
	r.Get("/usage", h.usage)
}

// userScopedPath aplica escopo de usuário comum: usuários não-admin só acessam o próprio home.
// Admins podem navegar em tudo.
func userScopedPath(r *http.Request, requestedPath string) (string, error) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok {
		return "", errors.New("no claims")
	}
	if claims.Role == "admin" {
		return requestedPath, nil
	}
	home := "/users/" + strings.ToLower(claims.Username)
	clean := path.Clean("/" + strings.TrimPrefix(requestedPath, "/"))
	if clean == home || strings.HasPrefix(clean, home+"/") {
		return clean, nil
	}
	// Usuário comum tentando acessar fora do home.
	return "", errors.New("acesso negado")
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("path")
	if p == "" {
		p = "/"
	}
	scoped, err := userScopedPath(r, p)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	entries, err := h.jail.List(scoped)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrNotFound):
			writeError(w, http.StatusNotFound, "não encontrado")
		case errors.Is(err, storage.ErrEscape):
			writeError(w, http.StatusBadRequest, "path inválido")
		case errors.Is(err, storage.ErrNotDir):
			writeError(w, http.StatusBadRequest, "não é diretório")
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"path":    scoped,
		"entries": entries,
	})
}

func (h *Handler) stat(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("path")
	scoped, err := userScopedPath(r, p)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	e, err := h.jail.Stat(scoped)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("path")
	scoped, err := userScopedPath(r, p)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	f, err := h.jail.Open(scoped)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if fi.IsDir() {
		writeError(w, http.StatusBadRequest, "é diretório")
		return
	}
	// Content-Disposition para forçar download. ServeContent define Content-Type,
	// Content-Length e suporta Range automaticamente.
	w.Header().Set("Content-Disposition", `attachment; filename="`+utils.SafeFilename(fi.Name())+`"`)
	http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
}

// upload aceita multipart com 'path' (diretório destino) e arquivo em 'file'.
// Para uploads grandes/chunked, ver UploadChunk no roadmap (com header Content-Range).
func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	// 32 MB em memória, o resto vai para tmp do tempdir do container.
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "multipart inválido: "+err.Error())
		return
	}
	dir := r.FormValue("path")
	if dir == "" {
		dir = "/"
	}
	scopedDir, err := userScopedPath(r, dir)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "nenhum arquivo enviado")
		return
	}

	results := make([]map[string]interface{}, 0, len(files))
	for _, fh := range files {
		safeName := utils.SafeFilename(fh.Filename)
		dst := path.Join(scopedDir, safeName)

		src, err := fh.Open()
		if err != nil {
			results = append(results, map[string]interface{}{"name": fh.Filename, "ok": false, "error": err.Error()})
			continue
		}
		aw, err := h.jail.CreateAtomic(dst)
		if err != nil {
			src.Close()
			results = append(results, map[string]interface{}{"name": fh.Filename, "ok": false, "error": err.Error()})
			continue
		}
		if _, err := io.Copy(aw, src); err != nil {
			aw.Abort()
			src.Close()
			results = append(results, map[string]interface{}{"name": fh.Filename, "ok": false, "error": err.Error()})
			continue
		}
		src.Close()
		if err := aw.Commit(); err != nil {
			results = append(results, map[string]interface{}{"name": fh.Filename, "ok": false, "error": err.Error()})
			continue
		}
		results = append(results, map[string]interface{}{"name": safeName, "ok": true, "path": dst, "size": fh.Size})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"results": results})
}

type pathBody struct {
	Path string `json:"path"`
}

type renameBody struct {
	Path    string `json:"path"`
	NewName string `json:"new_name"`
}

type moveBody struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (h *Handler) mkdir(w http.ResponseWriter, r *http.Request) {
	var body pathBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	scoped, err := userScopedPath(r, body.Path)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if err := h.jail.MkdirAll(scoped); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": scoped})
}

func (h *Handler) rename(w http.ResponseWriter, r *http.Request) {
	var body renameBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	if body.NewName == "" {
		writeError(w, http.StatusBadRequest, "new_name obrigatório")
		return
	}
	scoped, err := userScopedPath(r, body.Path)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	dst := path.Join(path.Dir(scoped), utils.SafeFilename(body.NewName))
	scopedDst, err := userScopedPath(r, dst)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if err := h.jail.Rename(scoped, scopedDst); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": scopedDst})
}

func (h *Handler) move(w http.ResponseWriter, r *http.Request) {
	var body moveBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	from, err := userScopedPath(r, body.From)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	to, err := userScopedPath(r, body.To)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if err := h.jail.Rename(from, to); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": to})
}

func (h *Handler) copy(w http.ResponseWriter, r *http.Request) {
	var body moveBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	from, err := userScopedPath(r, body.From)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	to, err := userScopedPath(r, body.To)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if err := h.jail.Copy(from, to); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": to})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("path")
	scoped, err := userScopedPath(r, p)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if err := h.jail.Remove(scoped); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": scoped})
}

func (h *Handler) usage(w http.ResponseWriter, r *http.Request) {
	u, err := h.jail.DiskUsage()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
