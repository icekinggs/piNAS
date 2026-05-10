// Package storage — camada de I/O com jail de path.
//
// PRINCÍPIO DE SEGURANÇA CRÍTICO:
// Toda operação de filesystem deve passar por Resolve(), que:
//  1. Limpa o path (remove . e ..)
//  2. Junta com a raiz (DataDir) e converte pra absoluto
//  3. Verifica se o resultado AINDA está dentro de DataDir
//
// Sem isso, um cliente pode mandar "../../etc/passwd" e ler o sistema.
package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrEscape   = errors.New("storage: path tenta sair do jail")
	ErrNotFound = errors.New("storage: caminho não encontrado")
	ErrNotDir   = errors.New("storage: não é diretório")
	ErrIsDir    = errors.New("storage: é diretório")
	ErrConflict = errors.New("storage: destino já existe")
)

type Jail struct {
	root string // path absoluto, limpo
}

func NewJail(root string) (*Jail, error) {
	abs, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, err
	}
	return &Jail{root: real}, nil
}

// Root devolve o diretório raiz absoluto.
func (j *Jail) Root() string {
	return j.root
}

// Resolve converte um virtualPath (ex: "/users/joao/foto.jpg") em path absoluto
// no host, garantindo que o resultado está dentro do jail.
//
// Aceita "" e "/" como raiz. Não importa se virtualPath começa com / ou não.
func (j *Jail) Resolve(virtualPath string) (string, error) {
	clean := filepath.Clean("/" + strings.TrimPrefix(virtualPath, "/"))
	// Resultado de Clean começa com '/' e nunca contém '..' a menos que seja "/.." ou "/"
	if strings.HasPrefix(clean, "/..") {
		return "", ErrEscape
	}
	abs := filepath.Join(j.root, clean)

	// Defesa em profundidade: re-checa após join.
	rel, err := filepath.Rel(j.root, abs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrEscape
	}
	return abs, nil
}

// Virtualize devolve o virtualPath relativo ao jail (sempre começa com /).
func (j *Jail) ensureInside(abs string) error {
	rel, err := filepath.Rel(j.root, abs)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ErrEscape
	}
	return nil
}

func (j *Jail) resolveExisting(virtualPath string) (string, error) {
	abs, err := j.Resolve(virtualPath)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotFound
		}
		return "", err
	}
	if err := j.ensureInside(real); err != nil {
		return "", err
	}
	return real, nil
}

func (j *Jail) validateExistingInside(virtualPath string) (string, error) {
	abs, err := j.Resolve(virtualPath)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotFound
		}
		return "", err
	}
	if err := j.ensureInside(real); err != nil {
		return "", err
	}
	return abs, nil
}

func (j *Jail) resolveCreateParent(virtualPath string) (string, error) {
	abs, err := j.Resolve(virtualPath)
	if err != nil {
		return "", err
	}
	if err := j.ensureNearestExistingInside(abs); err != nil {
		return "", err
	}
	return abs, nil
}

func (j *Jail) ensureNearestExistingInside(abs string) error {
	cur := abs
	for {
		if _, err := os.Lstat(cur); err == nil {
			real, err := filepath.EvalSymlinks(cur)
			if err != nil {
				return err
			}
			return j.ensureInside(real)
		} else if !os.IsNotExist(err) {
			return err
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return ErrEscape
		}
		cur = parent
	}
}

func (j *Jail) Virtualize(absPath string) (string, error) {
	rel, err := filepath.Rel(j.root, absPath)
	if err != nil {
		return "", err
	}
	if rel == "." {
		return "/", nil
	}
	return "/" + filepath.ToSlash(rel), nil
}

// Entry é um item de diretório.
type Entry struct {
	Name    string `json:"name"`
	Path    string `json:"path"` // virtual
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mtime"` // unix seconds
	Mode    string `json:"mode"`  // string como "drwxr-xr-x"
}

func (j *Jail) List(virtualDir string) ([]Entry, error) {
	abs, err := j.resolveExisting(virtualDir)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, ErrNotDir
	}

	dirents, err := os.ReadDir(abs)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(dirents))
	for _, de := range dirents {
		fi, err := de.Info()
		if err != nil {
			continue
		}
		vpath, _ := j.Virtualize(filepath.Join(abs, de.Name()))
		out = append(out, Entry{
			Name:    de.Name(),
			Path:    vpath,
			IsDir:   de.IsDir(),
			Size:    fi.Size(),
			ModTime: fi.ModTime().Unix(),
			Mode:    fi.Mode().String(),
		})
	}
	return out, nil
}

// Stat devolve metadata de um path.
func (j *Jail) Stat(virtualPath string) (Entry, error) {
	abs, err := j.resolveExisting(virtualPath)
	if err != nil {
		return Entry{}, err
	}
	fi, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return Entry{}, ErrNotFound
		}
		return Entry{}, err
	}
	vpath, _ := j.Virtualize(abs)
	return Entry{
		Name:    fi.Name(),
		Path:    vpath,
		IsDir:   fi.IsDir(),
		Size:    fi.Size(),
		ModTime: fi.ModTime().Unix(),
		Mode:    fi.Mode().String(),
	}, nil
}

// MkdirAll cria diretório (e intermediários) com permissão 0o750.
func (j *Jail) MkdirAll(virtualPath string) error {
	abs, err := j.resolveCreateParent(virtualPath)
	if err != nil {
		return err
	}
	return os.MkdirAll(abs, 0o750)
}

// Open abre um arquivo para leitura.
func (j *Jail) Open(virtualPath string) (*os.File, error) {
	abs, err := j.resolveExisting(virtualPath)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

// CreateAtomic devolve um writer que escreve em arquivo temporário.
// O caller deve chamar Commit() para mover atomicamente para o destino,
// ou Abort() para descartar.
type AtomicWriter struct {
	tmp     *os.File
	tmpPath string
	dst     string
	closed  bool
}

func (j *Jail) CreateAtomic(virtualPath string) (*AtomicWriter, error) {
	abs, err := j.resolveCreateParent(virtualPath)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(abs), ".pinas-tmp-*")
	if err != nil {
		return nil, err
	}
	return &AtomicWriter{tmp: tmp, tmpPath: tmp.Name(), dst: abs}, nil
}

func (a *AtomicWriter) Write(p []byte) (int, error) {
	return a.tmp.Write(p)
}

func (a *AtomicWriter) Commit() error {
	if a.closed {
		return errors.New("AtomicWriter já fechado")
	}
	a.closed = true
	if err := a.tmp.Sync(); err != nil {
		_ = os.Remove(a.tmpPath)
		return err
	}
	if err := a.tmp.Close(); err != nil {
		_ = os.Remove(a.tmpPath)
		return err
	}
	if err := os.Rename(a.tmpPath, a.dst); err != nil {
		_ = os.Remove(a.tmpPath)
		return err
	}
	return nil
}

func (a *AtomicWriter) Abort() {
	if a.closed {
		return
	}
	a.closed = true
	_ = a.tmp.Close()
	_ = os.Remove(a.tmpPath)
}

// Remove apaga arquivo ou diretório (recursivo se diretório).
func (j *Jail) Remove(virtualPath string) error {
	abs, err := j.validateExistingInside(virtualPath)
	if err != nil {
		return err
	}
	// Não permite remover a raiz.
	if abs == j.root {
		return errors.New("storage: não é possível remover a raiz")
	}
	if err := os.RemoveAll(abs); err != nil {
		return err
	}
	return nil
}

// Rename renomeia/move dentro do jail.
func (j *Jail) Rename(srcVirtual, dstVirtual string) error {
	src, err := j.validateExistingInside(srcVirtual)
	if err != nil {
		return err
	}
	dst, err := j.resolveCreateParent(dstVirtual)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dst); err == nil {
		return ErrConflict
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

// Copy faz cópia de arquivo (simples, sem progresso).
func (j *Jail) Copy(srcVirtual, dstVirtual string) error {
	src, err := j.resolveExisting(srcVirtual)
	if err != nil {
		return err
	}
	dst, err := j.resolveCreateParent(dstVirtual)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dst); err == nil {
		return ErrConflict
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(dst)
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		_ = os.Remove(dst)
		return err
	}
	return out.Close()
}

// DiskUsage devolve total/usado/livre do filesystem onde o jail vive.
type Usage struct {
	TotalBytes int64 `json:"total"`
	UsedBytes  int64 `json:"used"`
	FreeBytes  int64 `json:"free"`
}

// FormatBytes formata bytes em humano (KiB/MiB/GiB).
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
