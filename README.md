# PiNAS — Private NAS for Raspberry Pi

> Sistema NAS self-hosted, privado, leve e moderno para Raspberry Pi 4+ rodando Ubuntu Server 24.04 LTS (ARM64).

PiNAS é uma alternativa minimalista ao TrueNAS / OpenMediaVault / CasaOS, focada em:

- **Privacidade total** — zero telemetria, zero serviços externos, zero nuvem.
- **Leveza** — feito em Go (binário único) + SvelteKit estático. Roda confortavelmente em 512 MB de RAM.
- **Offline-first** — funciona 100% sem internet. DNS, certificados e auth são locais.
- **Modularidade** — Clean Architecture, módulos isolados, fácil de evoluir para plugins.

---

## 1. Arquitetura geral

```
┌────────────────────────────────────────────────────────────────────┐
│                            CLIENTES                                │
│   Browser (SvelteKit SPA)   SMB/CIFS   SFTP   WebDAV (opcional)    │
└──────────┬──────────────────────┬────────┬──────────┬──────────────┘
           │                      │        │          │
           ▼                      │        │          │
┌────────────────────────────┐    │        │          │
│   Caddy (Docker)           │    │        │          │
│   TLS local, HTTP/3, WS    │    │        │          │
└──────────┬─────────────────┘    │        │          │
           │                      │        │          │
           ▼                      │        │          │
┌────────────────────────────────────────────────────────────────────┐
│             PiNAS Backend Go (Docker, modular monolith)            │
│  ┌────────┬────────┬────────┬────────┐                             │
│  │  auth  │ files  │ users  │ system │                             │
│  ├────────┴────────┴────────┴────────┤                             │
│  │    middleware    │   ws hub       │                             │
│  └──────────────────┴────────────────┘                             │
└──────────┬──────────────────────────────────────┬──────────────────┘
           │                                      │
           ▼                                      ▼
┌──────────────────────────┐         ┌──────────────────────────────┐
│ SQLite (WAL)             │         │ /srv/pinas/data (ext4, USB)  │
│ /srv/pinas/db/pinas.db   │         │ arquivos do usuário          │
└──────────────────────────┘         └──────────────────────────────┘

       ┌──────────────────────────────────────────────────┐
       │ Serviços nativos do host (NÃO containerizados)   │
       │   smbd / nmbd      sshd (SFTP)     ufw + fail2ban│
       └──────────────────────────────────────────────────┘
```

### Por que esta divisão?

- **Caddy + Backend em Docker**: isolamento, rebuild fácil, restart policies.
- **smbd / sshd / ufw no host**: SMB e SFTP precisam de acesso ao kernel (módulos, PAM, sockets de baixo nível) e a usuários reais do sistema. Containerizar SMB no Pi é caro e frágil. UFW e Fail2Ban também são serviços de host.
- **SQLite**: zero-config, ACID, suficiente até dezenas de milhares de arquivos. WAL mode para concorrência.
- **Disco USB montado em `/srv/pinas/data`**: o backend nunca acessa fora desse jail.

---

## 2. Stack

| Camada       | Tecnologia                                  |
|--------------|---------------------------------------------|
| Backend      | Go 1.23 (chi router, sqlx, fsnotify)        |
| Frontend     | SvelteKit 2 (build estático)                |
| Banco        | SQLite 3 (WAL mode)                         |
| Reverse proxy| Caddy 2 (TLS local com CA interna)          |
| Auth         | JWT (HS256) + Argon2id                      |
| Realtime     | gorilla/websocket                           |
| Container    | Docker + Docker Compose                     |
| Host services| Samba, OpenSSH (SFTP), UFW, Fail2Ban        |
| OS alvo      | Ubuntu Server 24.04 LTS ARM64               |

### Por que Go no backend?
Binário único, sem runtime, baixíssimo footprint de RAM (~20 MB ocioso), goroutines tornam websocket + uploads concorrentes triviais, cross-compile ARM64 nativo.

### Por que SvelteKit?
Bundle 5–10× menor que React/Next, build estático servido pelo Caddy (zero Node em produção no Pi), reatividade sem virtual DOM.

### Por que SQLite e não Postgres?
No Pi, Postgres consome 80–150 MB ociosos. SQLite consome ~0. Para metadados de NAS doméstico, SQLite com WAL aguenta tranquilamente milhões de linhas. Migração futura para Postgres é trivial via repository pattern.

---

## 3. Estrutura de diretórios

```
pinas/
├── README.md
├── docker-compose.yml
├── .env.example
├── Makefile
│
├── backend/                          # Go modular monolith
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   ├── cmd/pinas/main.go             # entrypoint
│   ├── migrations/                   # SQL migrations
│   │   └── 0001_init.sql
│   ├── pkg/                          # libs reutilizáveis
│   │   ├── argon2id/                 # hashing
│   │   ├── jwt/                      # token issuance/verify
│   │   └── utils/
│   └── internal/
│       ├── config/                   # carregamento .env
│       ├── database/                 # conexão SQLite + migrations
│       ├── logger/                   # log estruturado
│       ├── api/                      # router + rotas
│       ├── auth/                     # módulo: login, refresh, sessions
│       │   ├── handler.go
│       │   ├── service.go
│       │   └── repository.go
│       ├── users/                    # módulo: usuários, grupos, ACL
│       ├── files/                    # módulo: upload, download, indexer
│       ├── system/                   # módulo: monitor (cpu/ram/temp/smart)
│       ├── storage/                  # camada de I/O com jail de path
│       ├── middleware/               # JWT, rate-limit, CORS, recovery, audit
│       └── websocket/                # hub realtime
│
├── frontend/                         # SvelteKit
│   ├── Dockerfile                    # build apenas
│   ├── package.json
│   ├── svelte.config.js
│   ├── vite.config.js
│   ├── static/
│   └── src/
│       ├── app.html
│       ├── app.css
│       ├── lib/
│       │   ├── api/                  # client fetch
│       │   ├── stores/               # auth, ws, theme
│       │   ├── components/           # UI
│       │   └── utils/
│       └── routes/
│           ├── +layout.svelte
│           ├── +page.svelte          # redirect login/dashboard
│           ├── login/
│           ├── dashboard/
│           ├── files/
│           ├── users/
│           ├── system/
│           └── settings/
│
├── caddy/
│   └── Caddyfile
│
├── deploy/
│   └── systemd/
│       └── pinas-host.service        # opcional
│
├── scripts/
│   ├── install.sh                    # instala host deps + automount
│   ├── automount-disk.sh             # /etc/fstab para o disco USB
│   ├── samba-setup.sh
│   └── backup.sh
│
└── docs/
    ├── ARCHITECTURE.md
    ├── API.md
    ├── DEPLOY.md
    └── ROADMAP.md
```

---

## 4. Layout do filesystem no host

```
/srv/pinas/
├── db/
│   └── pinas.db              # SQLite + WAL
├── data/                     # disco USB montado aqui (ext4)
│   ├── users/
│   │   └── <username>/       # home de cada usuário
│   └── shared/               # pasta compartilhada
├── thumbs/                   # cache de thumbnails
├── logs/
│   ├── audit.log
│   └── pinas.log
└── secrets/
    └── jwt.key               # chave HS256, 32 bytes random, 0600
```

O backend é **chrootado logicamente** (jail) em `/srv/pinas/data` — nunca aceita paths que escapem via `..`.

---

## 5. Fluxo de autenticação

```
1. POST /api/v1/auth/login  { username, password }
   ─► service.Login
       ─► repo.GetUserByUsername
       ─► argon2id.Verify(hash, password)
       ─► gera access_token (15min, JWT HS256)
       ─► gera refresh_token (7 dias, opaco, salvo em sessions)
       ─► retorna { access_token, expires_at }
       ─► seta refresh_token em cookie HttpOnly + Secure + SameSite=Strict

2. Requests autenticadas: Authorization: Bearer <access_token>
   ─► middleware.JWT verifica assinatura + exp
   ─► injeta user_id e role no context

3. POST /api/v1/auth/refresh   (cookie refresh_token)
   ─► valida sessão no banco (não expirada, não revogada)
   ─► rotaciona refresh (revoga o antigo, gera novo)
   ─► retorna novo access_token

4. POST /api/v1/auth/logout
   ─► revoga sessão atual
```

**Hashing de senha**: Argon2id com `time=3, memory=64MB, threads=2, keyLen=32, saltLen=16` — calibrado para o Pi 4 (~250ms/hash).

---

## 6. Schema SQLite

Veja `backend/migrations/0001_init.sql` — resumo:

- `users` (id, username, password_hash, role, disabled, created_at)
- `groups` (id, name)
- `user_groups` (user_id, group_id)
- `sessions` (id, user_id, refresh_token_hash, expires_at, revoked, ua, ip)
- `acls` (path, principal_type, principal_id, perm) — perm = bitmask read/write/admin
- `audit_log` (id, ts, user_id, action, target, ip, success, details)
- `file_index` (path, size, mtime, mime, hash, indexed_at) — popular via `fsnotify`

Todas as tabelas com índices apropriados. WAL ativado em runtime via `PRAGMA journal_mode=WAL`.

---

## 7. API REST (resumo)

Prefix: `/api/v1`

```
auth/
  POST   /auth/login
  POST   /auth/refresh
  POST   /auth/logout
  GET    /auth/me

users/
  GET    /users
  POST   /users                 (admin)
  GET    /users/:id
  PATCH  /users/:id
  DELETE /users/:id             (admin)
  POST   /users/:id/password

files/
  GET    /files?path=/foo       lista
  GET    /files/download?path=  stream
  POST   /files/upload          multipart, chunked
  POST   /files/folder          cria pasta
  PATCH  /files/rename
  POST   /files/move
  POST   /files/copy
  DELETE /files?path=
  GET    /files/search?q=
  GET    /files/thumb?path=

system/
  GET    /system/stats          cpu/ram/temp/disk/uptime/net
  GET    /system/processes
  GET    /system/smart          requer smartmontools
  GET    /system/logs

ws/
  GET    /ws                    progresso upload, métricas live
```

Detalhes em `docs/API.md`.

---

## 8. Segurança — princípios aplicados

- HTTPS local com CA interna do Caddy (`tls internal`).
- JWT HS256, chave de 256 bits gerada na 1ª execução em `/srv/pinas/secrets/jwt.key` (perm 0600).
- Argon2id para senhas (nunca bcrypt/sha).
- Refresh token rotativo, hash do token em banco (nunca o token cru).
- Path traversal: `filepath.Clean` + verificação de prefixo após `filepath.Abs`.
- Upload: limite por user, validação MIME, nome saneado, escrita atômica (`tmp` + rename).
- Rate-limit por IP em `/auth/*` (token bucket em memória).
- CORS restrito à origem do Caddy.
- Audit log de toda ação sensível.
- UFW: só portas 80, 443, 22, 445/139, 3478 (mDNS).
- Fail2Ban com jails para sshd e PiNAS API.

---

## 9. Roadmap MVP → Produção

### Fase 1 — MVP (este projeto)
- Auth, users, files (CRUD + upload/download), painel web, monitor básico, SMB.

### Fase 2 — Hardening
- WebSocket completo, search com FTS5, thumbnails de imagem, audit UI.

### Fase 3 — Funcionalidades
- WebDAV, snapshots (rsync + hardlinks), preview de vídeo (ffmpeg), busca full-text em conteúdo.

### Fase 4 — Avançado
- Multi-disco, RAID 1 via mdadm, criptografia LUKS, plugins, sync entre PiNAS, app mobile (Capacitor reaproveitando o front).

Detalhes em `docs/ROADMAP.md`.

---

## 10. Quickstart

> Para guia completo passo-a-passo (do zero, com Pi novo), veja [`INSTALL.md`](INSTALL.md).

**TL;DR** — num Pi com Ubuntu Server 24.04 já SSH-ável:

```bash
# clone (use seu PAT se for repo privado)
git clone https://github.com/SEU_USUARIO/pinas.git
cd pinas

# instala TUDO: deps, docker, build, sobe stack
sudo USB_DEVICE=/dev/sda1 SAMBA_USER=meunome ./bootstrap.sh
```

No final ele imprime a URL e a senha aleatória do admin.

Acesse `https://pinas.local` (ou `https://<ip-do-pi>`).
