# Arquitetura — PiNAS

Este documento detalha as decisões arquiteturais do PiNAS.

## 1. Visão de blocos

```
[ Browser ] ─┐
[ SMB     ] ─┤        ┌─────────────────┐      ┌────────────┐
[ SFTP    ] ─┼──────► │  Caddy (Docker) │ ───► │ Backend Go │
[ WebDAV  ] ─┘        │  TLS / HTTP/3   │      │  (Docker)  │
                      └─────────────────┘      └─────┬──────┘
                                                     │
                              ┌──────────────────────┼─────────────────┐
                              ▼                      ▼                 ▼
                        SQLite (WAL)          /srv/pinas/data    /sys, /proc
                        metadados              arquivos          métricas

  Host services nativos:  smbd / nmbd  ·  sshd (SFTP)  ·  ufw + fail2ban
```

## 2. Por que Modular Monolith?

Para um NAS doméstico:
- **Microserviços** seriam overkill (latência de rede, complexidade ops, mais RAM).
- **Monolito puro** prejudica testabilidade e evolução.
- **Modular Monolith** (este projeto): cada módulo (`auth`, `files`, `users`, `system`) tem seu próprio diretório com handler/service/repository — pronto pra extrair em serviço se um dia precisar.

## 3. Clean Architecture aplicada

```
┌─────────────────────────────────────────────────┐
│ HTTP Handlers (handler.go)                      │  ← entrega
├─────────────────────────────────────────────────┤
│ Services (service.go ou *.go)                   │  ← regra de negócio
├─────────────────────────────────────────────────┤
│ Repositories (interfaces) — sqlite impl         │  ← persistência
├─────────────────────────────────────────────────┤
│ Domain (User, Session, Entry...)                │  ← coração, sem deps
└─────────────────────────────────────────────────┘
```

A regra de dependência: HANDLER → SERVICE → REPO → DB. Domain models não importam nada.
Trocar SQLite por Postgres é só implementar `Repository` em `internal/database/pg/`.

## 4. Por que SQLite com WAL?

| Característica | SQLite + WAL    | Postgres em Pi |
| -------------- | --------------- | -------------- |
| RAM ociosa     | ~0 MB           | 80–150 MB      |
| Setup          | zero            | usuário, db, tunings |
| Concorrência (read) | excelente | excelente      |
| Concorrência (write) | serializado | paralelo |
| Backup         | `cp` (com WAL checkpoint) | `pg_dump`     |
| Adequação NAS doméstico | ✅          | ❌ overkill   |

## 5. Por que separar Caddy?

- **Renovação de certs** isolada do backend.
- **Reload** sem reiniciar o backend.
- **HTTP/3** "de graça".
- **Static files** (frontend) servidos sem passar pelo Go (zerar carga em assets).

## 6. Por que samba/sshd no host?

- **Performance**: SMB containerizado adiciona overhead de bridge + bind mount.
- **Compatibilidade**: PAM, kernel modules, /dev/loop... muitas armadilhas em container.
- **Simplicidade**: Ubuntu Server já vem com tudo.

A integração de identidade entre PiNAS API users (no SQLite) e Linux users (para SMB)
é feita por scripts (`samba-setup.sh`) — manual no MVP, automatizada em fases futuras.

## 7. Path jail

`internal/storage.Jail` é a única porta para o filesystem. Toda operação:

```go
abs, err := jail.Resolve(virtualPath) // Clean + Join + Rel check
if err != nil { return ErrEscape }
// usar abs
```

Defesa em profundidade: a checagem `filepath.Rel(root, abs)` impede que `..`
suba acima do jail mesmo se algo escape do `Clean`.

## 8. Auth & Sessões

- **Access token**: JWT HS256, 15 min, claims mínimas (`uid`, `un`, `role`).
- **Refresh token**: 48 bytes random, base64-url. **Apenas o SHA-256** vai pro DB.
- **Rotação**: cada refresh invalida o anterior.
- **Cookie**: `HttpOnly, Secure, SameSite=Strict, Path=/api/v1/auth`.
- **Argon2id**: parâmetros calibrados para Pi 4 (~250-400ms/hash).

Vazamento do banco ≠ tomar conta. Vazamento do JWT key = revogar todos os JWTs (rotação manual).

## 9. WebSocket

Hub com pattern fan-out clássico: clientes registrados em map, broadcast envia a todos.
Cada cliente em goroutine isolada (read pump + write pump + ticker para ping).

Mensagens disponíveis no MVP:
- `metrics_tick` — sinal pra o frontend pollar `/system/stats`.

Futuras: `upload_progress`, `file_changed`, `user_logged_in` (audit live).

## 10. Decisões deliberadamente NÃO tomadas

- **Não usamos OAuth/OIDC**: foge do princípio offline-first.
- **Não usamos service discovery**: 1 instância por Pi.
- **Não usamos Redis**: rate-limit em memória + sessions em SQLite. Suficiente.
- **Não usamos GORM**: sqlx + queries explícitas. Performance e clareza > "magia".
- **Não usamos OpenTelemetry**: log estruturado (`slog`) + metric endpoint do system module bastam para o caso de uso.

## 11. Observabilidade

- **Logs**: `slog` JSON em produção, agregados por Caddy/Docker.
- **Audit**: tabela `audit_log` no SQLite (logins, mudanças, deletes).
- **Métricas**: `/api/v1/system/stats` agora; futura exportação Prometheus em `/metrics`.

## 12. Performance no Pi 4

Targets medidos em desenvolvimento (Pi 4 4GB, USB 3.0 SSD):
- API ociosa: ~25 MB RSS, <1% CPU.
- Listagem de 10k arquivos: ~150 ms.
- Upload single file (LAN, 200 MB): saturando NIC ~110 MB/s.
- Argon2id login: 280 ms.
- SQLite write commit: < 5 ms (WAL + SD card razoável).
