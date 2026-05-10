package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveKeepsTraversalInsideJail(t *testing.T) {
	j, err := NewJail(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := j.Resolve("../../etc/passwd")
	if err != nil {
		t.Fatalf("expected traversal to be normalized inside jail, got %v", err)
	}
	rel, err := filepath.Rel(j.Root(), resolved)
	if err != nil {
		t.Fatal(err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatalf("resolved path escaped jail: %s", resolved)
	}
}

func TestOpenRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	j, err := NewJail(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.Open("/link.txt"); !errors.Is(err, ErrEscape) {
		t.Fatalf("expected ErrEscape, got %v", err)
	}
}

func TestCreateRejectsSymlinkParentEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "out")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	j, err := NewJail(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.CreateAtomic("/out/file.txt"); !errors.Is(err, ErrEscape) {
		t.Fatalf("expected ErrEscape, got %v", err)
	}
}
