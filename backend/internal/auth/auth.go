// Package auth — login, refresh, logout, sessões.
package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/pinas/pinas/internal/users"
	"github.com/pinas/pinas/pkg/argon2id"
	pjwt "github.com/pinas/pinas/pkg/jwt"
	"github.com/pinas/pinas/pkg/utils"
)

var (
	ErrInvalidCredentials = errors.New("auth: credenciais inválidas")
	ErrUserDisabled       = errors.New("auth: usuário desativado")
	ErrInvalidRefresh     = errors.New("auth: refresh token inválido")
	ErrSessionRevoked     = errors.New("auth: sessão revogada")
	ErrSessionExpired     = errors.New("auth: sessão expirada")
)

type Session struct {
	ID               string `db:"id"`
	UserID           int64  `db:"user_id"`
	RefreshTokenHash string `db:"refresh_token_hash"`
	UserAgent        sql.NullString `db:"user_agent"`
	IP               sql.NullString `db:"ip"`
	CreatedAt        int64  `db:"created_at"`
	ExpiresAt        int64  `db:"expires_at"`
	Revoked          int    `db:"revoked"`
}

type SessionRepository interface {
	Create(ctx context.Context, s *Session) error
	GetByHash(ctx context.Context, hash string) (*Session, error)
	Revoke(ctx context.Context, id string) error
	RevokeAllForUser(ctx context.Context, userID int64) error
	Cleanup(ctx context.Context) error // remove expired/revoked
}

type sessionRepo struct{ db *sqlx.DB }

func NewSessionRepository(db *sqlx.DB) SessionRepository {
	return &sessionRepo{db: db}
}

func (r *sessionRepo) Create(ctx context.Context, s *Session) error {
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO sessions(id, user_id, refresh_token_hash, user_agent, ip, created_at, expires_at, revoked)
        VALUES (?, ?, ?, ?, ?, ?, ?, 0)`,
		s.ID, s.UserID, s.RefreshTokenHash, s.UserAgent, s.IP, s.CreatedAt, s.ExpiresAt,
	)
	return err
}

func (r *sessionRepo) GetByHash(ctx context.Context, hash string) (*Session, error) {
	var s Session
	if err := r.db.GetContext(ctx, &s, `SELECT * FROM sessions WHERE refresh_token_hash = ?`, hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidRefresh
		}
		return nil, err
	}
	return &s, nil
}

func (r *sessionRepo) Revoke(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE sessions SET revoked=1 WHERE id=?`, id)
	return err
}

func (r *sessionRepo) RevokeAllForUser(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE sessions SET revoked=1 WHERE user_id=?`, userID)
	return err
}

func (r *sessionRepo) Cleanup(ctx context.Context) error {
	now := time.Now().Unix()
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ? OR revoked = 1`, now)
	return err
}

// ----- Service -----

type LoginResult struct {
	AccessToken  string
	ExpiresAt    time.Time
	RefreshToken string // crú — só é devolvido aqui, depois só hash em banco
}

type Service struct {
	users     users.Repository
	sessions  SessionRepository
	jwt       *pjwt.Issuer
	refreshTTL time.Duration
}

func NewService(u users.Repository, s SessionRepository, j *pjwt.Issuer, refreshTTL time.Duration) *Service {
	return &Service{users: u, sessions: s, jwt: j, refreshTTL: refreshTTL}
}

// Login valida usuário/senha e gera tokens.
func (s *Service) Login(ctx context.Context, username, password, ip, ua string) (*LoginResult, *users.User, error) {
	u, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, users.ErrNotFound) {
			// Mensagem genérica — não vazamos se o user existe.
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}
	if u.Disabled {
		return nil, nil, ErrUserDisabled
	}
	ok, err := argon2id.Verify(password, u.PasswordHash)
	if err != nil || !ok {
		return nil, nil, ErrInvalidCredentials
	}
	return s.issueTokens(ctx, u, ip, ua)
}

// Refresh rotaciona o refresh token: revoga o antigo, gera novo.
func (s *Service) Refresh(ctx context.Context, refreshToken, ip, ua string) (*LoginResult, *users.User, error) {
	hash := utils.SHA256Hex(refreshToken)
	sess, err := s.sessions.GetByHash(ctx, hash)
	if err != nil {
		return nil, nil, err
	}
	if sess.Revoked == 1 {
		return nil, nil, ErrSessionRevoked
	}
	if sess.ExpiresAt < time.Now().Unix() {
		return nil, nil, ErrSessionExpired
	}
	u, err := s.users.GetByID(ctx, sess.UserID)
	if err != nil {
		return nil, nil, err
	}
	if u.Disabled {
		_ = s.sessions.Revoke(ctx, sess.ID)
		return nil, nil, ErrUserDisabled
	}
	// Rotação: revoga a sessão antiga e cria nova.
	_ = s.sessions.Revoke(ctx, sess.ID)
	return s.issueTokens(ctx, u, ip, ua)
}

// Logout revoga uma sessão pelo refresh token.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	hash := utils.SHA256Hex(refreshToken)
	sess, err := s.sessions.GetByHash(ctx, hash)
	if err != nil {
		return nil // idempotente
	}
	return s.sessions.Revoke(ctx, sess.ID)
}

func (s *Service) issueTokens(ctx context.Context, u *users.User, ip, ua string) (*LoginResult, *users.User, error) {
	access, exp, err := s.jwt.Issue(u.ID, u.Username, u.Role)
	if err != nil {
		return nil, nil, err
	}
	refresh, err := utils.RandomToken(48)
	if err != nil {
		return nil, nil, err
	}
	hash := utils.SHA256Hex(refresh)
	sess := &Session{
		ID:               uuid.New().String(),
		UserID:           u.ID,
		RefreshTokenHash: hash,
		UserAgent:        sql.NullString{String: ua, Valid: ua != ""},
		IP:               sql.NullString{String: ip, Valid: ip != ""},
		CreatedAt:        time.Now().Unix(),
		ExpiresAt:        time.Now().Add(s.refreshTTL).Unix(),
	}
	if err := s.sessions.Create(ctx, sess); err != nil {
		return nil, nil, err
	}
	return &LoginResult{
		AccessToken:  access,
		ExpiresAt:    exp,
		RefreshToken: refresh,
	}, u, nil
}
