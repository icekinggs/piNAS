// Package database — conexão SQLite, ativação de WAL, migrations.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// Open abre o banco com PRAGMAs adequados a uso em NAS embarcado.
//
// Por que estes PRAGMAs:
//   - WAL: leituras não bloqueiam writes (essencial para listagens enquanto upload acontece)
//   - busy_timeout=5000: writes concorrentes esperam em vez de erro imediato
//   - synchronous=NORMAL: trade-off seguro com WAL (durabilidade vs IO no SD card / USB)
//   - foreign_keys=ON: SQLite desliga FK por padrão, ON é o que esperamos
func Open(path string) (*sqlx.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	dsn := fmt.Sprintf(
		"file:%s?_journal=WAL&_busy_timeout=5000&_synchronous=NORMAL&_foreign_keys=on",
		path,
	)

	db, err := sqlx.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// SQLite não escala writes com múltiplas conexões; pool pequeno.
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)

	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

// RunMigrations executa migrations .sql em ordem alfabética a partir de um diretório.
// Idempotente: cada migration usa CREATE IF NOT EXISTS.
// Evolução futura: tabela schema_migrations + versionamento.
func RunMigrations(ctx context.Context, db *sqlx.DB, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %s: %w", dir, err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if _, err := db.ExecContext(ctx, string(data)); err != nil {
			return fmt.Errorf("exec %s: %w", name, err)
		}
	}
	return nil
}

// Tx executa fn numa transação com rollback automático em erro/panic.
func Tx(ctx context.Context, db *sqlx.DB, fn func(*sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
