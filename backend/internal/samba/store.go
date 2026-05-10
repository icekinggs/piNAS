// store.go — persistência do desired-state em JSON.
//
// Usamos JSON (não YAML) pra evitar dependência de bibliotecas YAML.
// O script samba-sync.sh no host lê o JSON com `jq` (já instalado pelo bootstrap).
package samba

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const stateFileName = "desired-state.json"

// Store gerencia leitura/escrita atômica do estado em disco.
type Store struct {
	dir  string
	mu   sync.Mutex
}

func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("samba store: %w", err)
	}
	return &Store{dir: dir}, nil
}

func (s *Store) path() string {
	return filepath.Join(s.dir, stateFileName)
}

// Load lê o estado do disco. Se não existe, devolve estado vazio default.
func (s *Store) Load() (*State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return &State{Version: 1, Shares: []Share{}, Users: []User{}}, nil
		}
		return nil, err
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("samba store parse: %w", err)
	}
	if st.Version == 0 {
		st.Version = 1
	}
	if st.Shares == nil {
		st.Shares = []Share{}
	}
	if st.Users == nil {
		st.Users = []User{}
	}
	return &st, nil
}

// Save escreve atomicamente (tmp + rename).
func (s *Store) Save(st *State) error {
	if st == nil {
		return errors.New("samba store: nil state")
	}
	st.Version = 1

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}

	dst := s.path()
	tmp, err := os.CreateTemp(s.dir, ".samba-state-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	// Permissão 0640 — script no host roda como root e lê.
	// Hashes/senhas pendentes ficam aqui temporariamente.
	if err := os.Chmod(tmpPath, 0o640); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, dst)
}
