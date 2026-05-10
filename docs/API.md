# API REST — PiNAS

Base: `https://pinas.local/api/v1`

Autenticação: `Authorization: Bearer <access_token>` em todas as rotas exceto `/auth/login`, `/auth/refresh` e `/healthz`.

Refresh token vive em cookie HttpOnly chamado `pinas_refresh`, escopado em `/api/v1/auth`.

## Codes

- `200` OK
- `201` Created
- `204` No Content
- `400` Bad Request
- `401` Unauthorized (token ausente/inválido — tente refresh)
- `403` Forbidden (autenticado mas sem permissão)
- `404` Not Found
- `409` Conflict (ex: usuário já existe)
- `429` Too Many Requests
- `500` Internal Server Error

Erros sempre em JSON: `{ "error": "mensagem" }`.

---

## Healthz

```
GET /healthz   →  200  { "status": "ok" }
```

## Auth

### POST /auth/login
```json
Request:  { "username": "admin", "password": "..." }
Response: {
  "access_token": "eyJ...",
  "expires_at": "2026-05-09T13:00:00Z",
  "user": { "id": 1, "username": "admin", "role": "admin", "home_path": "/users/admin" }
}
Set-Cookie: pinas_refresh=<opaque>; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth
```

### POST /auth/refresh
Sem body, lê o cookie `pinas_refresh`. Mesma resposta do login. Rotaciona o refresh.

### POST /auth/logout
Revoga a sessão referenciada pelo cookie. Limpa o cookie. Resposta `{ "ok": true }`.

### GET /auth/me
```json
{ "id": 1, "username": "admin", "role": "admin", "home_path": "/users/admin" }
```

---

## Users (admin)

### GET /users
```json
{ "users": [ { "id": 1, "username": "admin", "role": "admin", "home_path": "/users/admin", "quota_bytes": 0, "disabled": false, "created_at": 1715200000 } ] }
```

### POST /users
```json
Request:  { "username": "maria", "password": "min8chars", "role": "user", "quota_bytes": 0 }
Response: 201 Created — view do user.
```

### GET /users/{id}    →  view
### PATCH /users/{id}  →  body: `{ "disabled": true }` (futuramente role, quota)
### DELETE /users/{id} →  `{ "deleted": true }`. Não permite auto-delete.
### POST /users/{id}/password →  body: `{ "new_password": "..." }`

---

## Files

> Usuários comuns são limitados ao próprio `home_path`. Admins acessam tudo.

### GET /files?path=/users/maria
```json
{
  "path": "/users/maria",
  "entries": [
    { "name": "fotos", "path": "/users/maria/fotos", "is_dir": true, "size": 4096, "mtime": 1715200000, "mode": "drwxr-xr-x" }
  ]
}
```

### GET /files/stat?path=/users/maria/foto.jpg
View do entry.

### GET /files/download?path=/users/maria/foto.jpg
Stream do arquivo. Suporta header `Range:` (downloads retomáveis e seek de mídia).

### POST /files/upload
`multipart/form-data`:
- `path`: diretório de destino
- `file`: pode ser repetido para múltiplos uploads

```json
{ "results": [
  { "name": "a.jpg", "ok": true, "path": "/users/maria/a.jpg", "size": 12345 },
  { "name": "b.zip", "ok": false, "error": "..." }
]}
```

### POST /files/folder
```json
{ "path": "/users/maria/nova" }
```

### PATCH /files/rename
```json
{ "path": "/users/maria/foto.jpg", "new_name": "perfil.jpg" }
```

### POST /files/move
```json
{ "from": "/users/maria/a.jpg", "to": "/shared/a.jpg" }
```

### POST /files/copy
Mesmo body do `move`.

### DELETE /files?path=/users/maria/a.jpg
`{ "deleted": "/users/maria/a.jpg" }`. Recursivo se for diretório.

### GET /files/usage
```json
{ "total": 1000204886016, "used": 12345678, "free": 1000192540338 }
```

---

## System

### GET /system/stats
```json
{
  "hostname": "pinas",
  "os": "Ubuntu 24.04",
  "kernel": "6.8.0-1011-raspi",
  "arch": "arm64",
  "uptime_sec": 123456,
  "cpu_percent": 12.5,
  "cpu_cores": 4,
  "load_avg": [0.31, 0.40, 0.42],
  "temp_celsius": 48.5,
  "memory": { "total": 4194304000, "used": 1073741824, "free": 1073741824, "available": 3120562176, "used_percent": 25.6 },
  "disk":   { "total": 1000204886016, "used": 12345678, "free": 1000192540338 },
  "network": [ { "name": "eth0", "bytes_in": 12345, "bytes_out": 6789 } ],
  "timestamp": 1715200000
}
```

---

## WebSocket

`GET /ws` (upgrade). Requer `Authorization` header.

Mensagens recebidas:
```json
{ "type": "metrics_tick", "payload": { "ts": 1715200000 }, "ts": 1715200000 }
```

No MVP é apenas um sinal para o frontend pollar `/system/stats`. Em v2: payload completo no próprio evento, eventos `upload_progress`, `audit`, etc.

---

## Rate limiting

`/auth/*` é limitado a **2 req/s burst 5** por IP. Outros endpoints sem rate-limit por enquanto (defesa via reverse proxy / fail2ban).
