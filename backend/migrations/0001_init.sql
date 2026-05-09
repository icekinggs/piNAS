-- PiNAS — schema inicial
-- WAL é ativado em runtime via PRAGMA. Aqui só schema.

PRAGMA foreign_keys = ON;

-- Usuários locais.
CREATE TABLE IF NOT EXISTS users (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    username        TEXT    NOT NULL UNIQUE COLLATE NOCASE,
    password_hash   TEXT    NOT NULL,             -- argon2id encoded string
    role            TEXT    NOT NULL DEFAULT 'user' CHECK (role IN ('admin','user')),
    home_path       TEXT    NOT NULL,             -- ex: /users/joao
    quota_bytes     INTEGER NOT NULL DEFAULT 0,   -- 0 = sem limite
    disabled        INTEGER NOT NULL DEFAULT 0,
    created_at      INTEGER NOT NULL,             -- unix seconds
    updated_at      INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Grupos (para ACL futura mais rica).
CREATE TABLE IF NOT EXISTS groups (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL UNIQUE COLLATE NOCASE,
    created_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS user_groups (
    user_id  INTEGER NOT NULL,
    group_id INTEGER NOT NULL,
    PRIMARY KEY (user_id, group_id),
    FOREIGN KEY (user_id)  REFERENCES users(id)  ON DELETE CASCADE,
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
);

-- Sessões (refresh tokens).
-- Guardamos só o HASH SHA-256 do token cru — se o banco vazar, ninguém revive sessão.
CREATE TABLE IF NOT EXISTS sessions (
    id                 TEXT    PRIMARY KEY,        -- uuid
    user_id            INTEGER NOT NULL,
    refresh_token_hash TEXT    NOT NULL UNIQUE,
    user_agent         TEXT,
    ip                 TEXT,
    created_at         INTEGER NOT NULL,
    expires_at         INTEGER NOT NULL,
    revoked            INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sessions_user    ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

-- ACLs por path.
-- principal_type: 'user' | 'group'
-- perm: bitmask  1=read  2=write  4=admin
CREATE TABLE IF NOT EXISTS acls (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    path            TEXT    NOT NULL,                -- normalizado, sem trailing slash
    principal_type  TEXT    NOT NULL CHECK (principal_type IN ('user','group')),
    principal_id    INTEGER NOT NULL,
    perm            INTEGER NOT NULL,
    created_at      INTEGER NOT NULL,
    UNIQUE (path, principal_type, principal_id)
);

CREATE INDEX IF NOT EXISTS idx_acls_path ON acls(path);

-- Auditoria.
CREATE TABLE IF NOT EXISTS audit_log (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    ts        INTEGER NOT NULL,
    user_id   INTEGER,
    username  TEXT,                -- snapshot, sobrevive a delete do user
    action    TEXT    NOT NULL,    -- 'login', 'login_failed', 'file_upload', etc
    target    TEXT,                -- ex: '/users/joao/foto.jpg'
    ip        TEXT,
    success   INTEGER NOT NULL DEFAULT 1,
    details   TEXT                 -- JSON livre
);

CREATE INDEX IF NOT EXISTS idx_audit_ts     ON audit_log(ts DESC);
CREATE INDEX IF NOT EXISTS idx_audit_user   ON audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_log(action);

-- Índice de arquivos (populado pelo indexer com fsnotify).
CREATE TABLE IF NOT EXISTS file_index (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    path        TEXT    NOT NULL UNIQUE,
    parent_dir  TEXT    NOT NULL,
    name        TEXT    NOT NULL,
    size        INTEGER NOT NULL,
    is_dir      INTEGER NOT NULL DEFAULT 0,
    mtime       INTEGER NOT NULL,
    mime        TEXT,
    sha256      TEXT,
    indexed_at  INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_index_parent ON file_index(parent_dir);
CREATE INDEX IF NOT EXISTS idx_index_name   ON file_index(name COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_index_mime   ON file_index(mime);

-- Configurações chave/valor.
CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
