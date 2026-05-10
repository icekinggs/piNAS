// Package users — gerenciamento de usuários (CRUD), repositório e serviço.
package users

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/pinas/pinas/pkg/argon2id"
)

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

var (
	ErrNotFound        = errors.New("users: não encontrado")
	ErrUsernameTaken   = errors.New("users: nome de usuário já existe")
	ErrInvalidUsername = errors.New("users: username inválido (3-32 chars, letras/números/_)")
	ErrInvalidPassword = errors.New("users: senha muito curta (min. 8 chars)")
	ErrInvalidRole     = errors.New("users: role invalida")
	ErrInvalidQuota    = errors.New("users: quota invalida")
	ErrLastAdmin       = errors.New("users: nao e possivel remover, desativar ou rebaixar o ultimo admin")
)

type User struct {
	ID           int64  `db:"id"            json:"id"`
	Username     string `db:"username"      json:"username"`
	PasswordHash string `db:"password_hash" json:"-"`
	Role         string `db:"role"          json:"role"`
	HomePath     string `db:"home_path"     json:"home_path"`
	QuotaBytes   int64  `db:"quota_bytes"   json:"quota_bytes"`
	Disabled     bool   `db:"disabled"      json:"disabled"`
	CreatedAt    int64  `db:"created_at"    json:"created_at"`
	UpdatedAt    int64  `db:"updated_at"    json:"updated_at"`
}

// Repository abstrai persistência (Repository Pattern).
// Trocar SQLite por Postgres é trivial: nova impl. desta interface.
type Repository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	List(ctx context.Context) ([]User, error)
	Update(ctx context.Context, u *User) error
	UpdatePassword(ctx context.Context, id int64, hash string) error
	Delete(ctx context.Context, id int64) error
	Count(ctx context.Context) (int64, error)
	CountAdmins(ctx context.Context) (int64, error)
}

type sqliteRepo struct{ db *sqlx.DB }

func NewRepository(db *sqlx.DB) Repository {
	return &sqliteRepo{db: db}
}

func (r *sqliteRepo) Create(ctx context.Context, u *User) error {
	now := time.Now().Unix()
	u.CreatedAt = now
	u.UpdatedAt = now
	res, err := r.db.ExecContext(ctx, `
        INSERT INTO users(username, password_hash, role, home_path, quota_bytes, disabled, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		u.Username, u.PasswordHash, u.Role, u.HomePath, u.QuotaBytes, boolToInt(u.Disabled), u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrUsernameTaken
		}
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	u.ID = id
	return nil
}

func (r *sqliteRepo) GetByID(ctx context.Context, id int64) (*User, error) {
	var u User
	if err := r.db.GetContext(ctx, &u, `SELECT * FROM users WHERE id = ?`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *sqliteRepo) GetByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	if err := r.db.GetContext(ctx, &u, `SELECT * FROM users WHERE username = ? COLLATE NOCASE`, username); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *sqliteRepo) List(ctx context.Context) ([]User, error) {
	var users []User
	if err := r.db.SelectContext(ctx, &users, `SELECT * FROM users ORDER BY username`); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *sqliteRepo) Update(ctx context.Context, u *User) error {
	u.UpdatedAt = time.Now().Unix()
	_, err := r.db.ExecContext(ctx, `
        UPDATE users SET role=?, home_path=?, quota_bytes=?, disabled=?, updated_at=? WHERE id=?`,
		u.Role, u.HomePath, u.QuotaBytes, boolToInt(u.Disabled), u.UpdatedAt, u.ID,
	)
	return err
}

func (r *sqliteRepo) UpdatePassword(ctx context.Context, id int64, hash string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET password_hash=?, updated_at=? WHERE id=?`,
		hash, time.Now().Unix(), id)
	return err
}

func (r *sqliteRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id=?`, id)
	return err
}

func (r *sqliteRepo) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.GetContext(ctx, &n, `SELECT COUNT(*) FROM users`)
	return n, err
}

func (r *sqliteRepo) CountAdmins(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.GetContext(ctx, &n, `SELECT COUNT(*) FROM users WHERE role = ? AND disabled = 0`, RoleAdmin)
	return n, err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// ----- Service layer -----

type CreateInput struct {
	Username string
	Password string
	Role     string
	Quota    int64
}

type UpdateInput struct {
	Role     *string
	Quota    *int64
	Disabled *bool
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*User, error) {
	if err := ValidateUsername(in.Username); err != nil {
		return nil, err
	}
	if len(in.Password) < 8 {
		return nil, ErrInvalidPassword
	}
	role := in.Role
	if role != RoleAdmin {
		role = RoleUser
	}
	hash, err := argon2id.Hash(in.Password)
	if err != nil {
		return nil, err
	}
	u := &User{
		Username:     in.Username,
		PasswordHash: hash,
		Role:         role,
		HomePath:     "/users/" + strings.ToLower(in.Username),
		QuotaBytes:   in.Quota,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Service) Get(ctx context.Context, id int64) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]User, error) {
	return s.repo.List(ctx)
}

func (s *Service) ChangePassword(ctx context.Context, id int64, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrInvalidPassword
	}
	hash, err := argon2id.Hash(newPassword)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, id, hash)
}

func (s *Service) Update(ctx context.Context, id int64, in UpdateInput) (*User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	nextRole := u.Role
	nextDisabled := u.Disabled
	nextQuota := u.QuotaBytes

	if in.Role != nil {
		if *in.Role != RoleAdmin && *in.Role != RoleUser {
			return nil, ErrInvalidRole
		}
		nextRole = *in.Role
	}
	if in.Disabled != nil {
		nextDisabled = *in.Disabled
	}
	if in.Quota != nil {
		if *in.Quota < 0 {
			return nil, ErrInvalidQuota
		}
		nextQuota = *in.Quota
	}

	if u.Role == RoleAdmin && (nextRole != RoleAdmin || nextDisabled) {
		if err := s.ensureAnotherActiveAdmin(ctx); err != nil {
			return nil, err
		}
	}

	u.Role = nextRole
	u.Disabled = nextDisabled
	u.QuotaBytes = nextQuota
	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if u.Role == RoleAdmin {
		if err := s.ensureAnotherActiveAdmin(ctx); err != nil {
			return err
		}
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) SetDisabled(ctx context.Context, id int64, disabled bool) error {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if disabled && u.Role == RoleAdmin {
		if err := s.ensureAnotherActiveAdmin(ctx); err != nil {
			return err
		}
	}
	u.Disabled = disabled
	return s.repo.Update(ctx, u)
}

func (s *Service) ensureAnotherActiveAdmin(ctx context.Context) error {
	n, err := s.repo.CountAdmins(ctx)
	if err != nil {
		return err
	}
	if n <= 1 {
		return ErrLastAdmin
	}
	return nil
}

// ValidateUsername restringe a [a-z0-9_], 3-32 chars, começa com letra.
func ValidateUsername(u string) error {
	if len(u) < 3 || len(u) > 32 {
		return ErrInvalidUsername
	}
	if !((u[0] >= 'a' && u[0] <= 'z') || (u[0] >= 'A' && u[0] <= 'Z')) {
		return ErrInvalidUsername
	}
	for _, r := range u {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-'
		if !ok {
			return ErrInvalidUsername
		}
	}
	return nil
}
