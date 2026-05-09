# Roadmap — PiNAS

## Fase 1 — MVP (atual, v0.1)

✅ Auth com JWT + Argon2id
✅ Refresh tokens rotativos
✅ Users CRUD (admin)
✅ Files: list/upload/download/rename/move/copy/delete/mkdir
✅ Path jail
✅ Painel web (dashboard, files, users, system, settings)
✅ Monitor: CPU, RAM, temp, disk, net
✅ Caddy + TLS local
✅ SMB via host
✅ SQLite com WAL

## Fase 2 — Hardening + UX (v0.2)

- [ ] WebSocket payload completo (eventos de upload, métricas inline, audit live)
- [ ] Search com FTS5 (índice em SQLite)
- [ ] Thumbnails de imagens (govips ou imagemagick CLI)
- [ ] Audit log UI
- [ ] Upload chunked (Content-Range) para arquivos > 1 GB
- [ ] Quotas funcionais (verificação no upload)
- [ ] Two-factor auth (TOTP)
- [ ] Rate limit em `/files/upload` por usuário
- [ ] Compressão de respostas em endpoints listagem grandes (gzip já no Caddy)

## Fase 3 — Funcionalidades (v0.3 - v0.5)

- [ ] WebDAV nativo no Go (golang.org/x/net/webdav)
- [ ] Snapshots agendáveis (rsync + hardlinks via cron-like interno)
- [ ] Preview de vídeo (ffmpeg streaming, HLS)
- [ ] Preview/edit de texto (.md, .txt)
- [ ] Plugins simples (manifest YAML + sandboxing nsjail)
- [ ] Sync entre PiNAS (rclone-like via API própria)
- [ ] Mobile app (Capacitor reaproveitando o frontend SvelteKit)

## Fase 4 — Avançado (v1.0+)

- [ ] Multi-disco com RAID 1 via mdadm (UI)
- [ ] Encriptação LUKS (UI de unlock no boot)
- [ ] Integração com Tailscale para acesso remoto opt-in
- [ ] PAM module para integrar SMB/SFTP com usuários do PiNAS
- [ ] Apps Docker on-demand (CasaOS-style: portainer integrado)
- [ ] IA local opcional (whisper.cpp, llama.cpp para busca semântica)
- [ ] Streaming de mídia compatível com Jellyfin/Plex (manifest)
- [ ] Federação opt-in entre PiNAS via WireGuard mesh

## Long term ideas

- **Cluster Pi**: 2+ Pis com gluster/syncthing.
- **HSM-backed JWT**: chave HMAC saindo de TPM 2.0 (TPM2-Tools).
- **Verificação de integridade contínua**: re-hash periódico de arquivos
  e comparação com `file_index` para detectar bit-rot.
- **CRDT collaborative editing**: Yjs sync sobre WebSocket para .md.
