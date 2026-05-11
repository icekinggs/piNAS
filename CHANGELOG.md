# Changelog

Formato baseado em [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/),
versionamento [SemVer](https://semver.org/lang/pt-BR/).

## [Unreleased]

## [0.3.0] — 2026-05-11

Sprint 1: fechar o produto. Acabamento e qualidade percebida.

### Adicionado
- **Sistema de toasts** substitui `alert()`/`confirm()` do navegador em todo o frontend.
  Notificações por canto da tela com cor por tipo (success/error/warn/info) e auto-dismiss
  configurável. `toast.confirm()` retorna `Promise<boolean>` pra confirmações com botão
  destrutivo.
- **Comando CLI `pinas reset-admin`** dentro do container resolve "esqueci a senha"
  sem precisar reinstalar. Gera senha aleatória de 24 chars (default) ou aceita
  `--password=X`. Re-habilita usuário se estiver desabilitado.
- **GitHub Actions** de CI (lint + build do frontend, vet + test do Go, shellcheck)
  e Release (build do frontend num runner potente, empacota tarball, publica como
  asset da release com SHA256).
- **`bootstrap.sh`** detecta release oficial e baixa o tarball pré-buildado.
  Etapa de build do frontend cai de ~3 min com risco de OOM no Pi 4 pra ~5 segundos
  de download. Build local continua como fallback automático.

### Mudou
- Frontend `FileExplorer.svelte` agora usa toasts. Operações destrutivas
  (excluir arquivo, excluir múltiplos) usam confirm destrutivo (botão vermelho).

### Variáveis novas no bootstrap
- `PINAS_VERSION` — versão a baixar (default: `latest`).
- `PINAS_FRONTEND_TARBALL` — URL direta pra um tarball (override).
- `PINAS_SKIP_TARBALL=1` — força build local (pra dev).

## [0.2.0] — 2026-05-10

Primeira versão pública usável.

### Adicionado
- Backend Go (Clean Architecture, módulos: auth, files, users, system, samba, ws).
- Frontend SvelteKit com tema brutalist-utilitarian + sidebar + 6 rotas
  (dashboard, files, samba, users, system, settings).
- **Gerenciamento web do Samba** (lotes 1-4): web UI cria/edita/exclui shares
  e usuários SMB; backend escreve estado declarativo (`desired-state.json`);
  systemd `.path` + `.service` no host sincroniza com `/etc/samba/smb.conf`
  via script idempotente que valida com `testparm` antes de aplicar.
- **`bootstrap.sh`** detecta conflito de porta (Pi-hole etc.) e cai pra 8080/8443.
- **`bootstrap.sh`** detecta UID/GID do dono do `DATA_DIR` e ajusta override do compose.
- **`uninstall.sh`** com `--purge`, `--keep-samba`, `--dry-run`.
- Caddy com `tls internal { on_demand }` aceita qualquer hostname/IP local.
- Healthcheck do compose usa GET (não HEAD).
- Backend cria UID 1000 fixo pra match com volumes.
- Backend router resolve panic do chi com `r.Get("/auth/me", ...)` direto.
