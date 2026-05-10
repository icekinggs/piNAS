<div align="center">

# 🗄️ PiNAS

**Private NAS for Raspberry Pi**

Sistema NAS self-hosted, privado, leve e moderno para Raspberry Pi 4+ com Ubuntu Server 24.04 LTS ARM64.

[![License](https://img.shields.io/badge/license-MIT-green)]()
[![Status](https://img.shields.io/badge/status-MVP-orange)]()
[![Platform](https://img.shields.io/badge/platform-arm64%20%7C%20amd64-blue)]()
[![Go](https://img.shields.io/badge/backend-Go%201.23-00ADD8)]()
[![Svelte](https://img.shields.io/badge/frontend-SvelteKit%202-FF3E00)]()

</div>

---

## ✨ Por que PiNAS?

Alternativa minimalista ao **TrueNAS / OpenMediaVault / CasaOS**:

- 🔒 **Privacidade total** — zero telemetria, zero serviços externos, zero nuvem.
- ⚡ **Leve** — backend Go (~25 MB RAM ocioso) + SvelteKit estático.
- 🌐 **Offline-first** — funciona 100% sem internet.
- 🤖 **Instalação inteligente** — bootstrap detecta conflitos (Pi-hole, UID), zero config manual.

### Funcionalidades

- 📊 Dashboard com CPU, RAM, temperatura, disco, rede em tempo real
- 📁 Explorador de arquivos com upload drag-and-drop
- 👥 Gerenciamento de usuários com roles, quotas e ACL
- 🔐 Auth com JWT + Argon2id + refresh tokens rotativos
- 🌐 HTTPS local automático via Caddy
- 🔌 Multi-protocolo opcional: SMB, SFTP

---

## 🚀 Instalação rápida

**Pré-requisitos:**
- Raspberry Pi 4 (4 GB+) ou Pi 5
- Ubuntu Server 24.04 LTS ARM64 instalado
- SSH funcionando

```bash
ssh seu_usuario@pinas.local
git clone https://github.com/icekinggs/piNAS.git
cd piNAS
sudo ./bootstrap.sh
```

Espera ~20 min. No final imprime URL e senha do admin. Pronto.

### O que o bootstrap detecta automaticamente

✅ **Conflito de portas** — Se 80/443 já estão em uso (Pi-hole, etc), usa 8080/8443
✅ **UID:GID do dono dos dados** — Container roda com o mesmo UID:GID, sem problema de permissão
✅ **Existência de Docker/Node** — Pula reinstalação se já presentes
✅ **Hostname mDNS** — Configura `https://<seu-hostname>.local`

---

## 🎯 Cenários de uso

Variáveis de ambiente que você pode passar pro bootstrap:

### A. Instalação padrão (mais simples)

```bash
sudo ./bootstrap.sh
```

Cria pasta de dados em `/srv/pinas/data/`, gerenciada exclusivamente pelo PiNAS.

### B. Integrar com Samba existente / pasta atual

Se você já tem Samba compartilhando uma pasta (ex: `/home/iceking`) e quer o painel mostrando os mesmos arquivos:

```bash
sudo DATA_DIR=/home/iceking ./bootstrap.sh
```

O painel web e o Samba vão enxergar os mesmos arquivos. Mexer num lado reflete no outro.

### C. Disco USB dedicado

```bash
sudo USB_DEVICE=/dev/sda1 ./bootstrap.sh
```

Monta o disco em `/etc/fstab` (UUID-based) e usa como pasta de dados.

### D. Com SMB configurado

```bash
sudo SAMBA_USER=meunome ./bootstrap.sh
```

Cria compartilhamento Samba pra Windows/Mac/Linux.

### E. Tudo combinado

```bash
sudo USB_DEVICE=/dev/sda1 SAMBA_USER=admin DATA_DIR=/srv/pinas/data ./bootstrap.sh
```

### Variáveis disponíveis

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `DATA_DIR` | `/srv/pinas/data` | Pasta dos arquivos do NAS |
| `USB_DEVICE` | _(vazio)_ | Disco USB pra automount (ex: `/dev/sda1`) |
| `SAMBA_USER` | _(vazio)_ | Usuário Samba a criar |
| `HTTP_PORT` | _auto_ | Forçar porta HTTP (caso queira override) |
| `HTTPS_PORT` | _auto_ | Forçar porta HTTPS |
| `INSTALL_DIR` | `/opt/pinas` | Onde instalar o código |
| `PINAS_FORCE_UPDATE` | `0` | Use `1` para atualizar sobrescrevendo alterações locais no `INSTALL_DIR` |

---

## 🔓 Como acessar pela primeira vez

Após `bootstrap.sh` terminar:

1. Pega a senha:
   ```bash
   sudo cat /root/.pinas/admin-password.txt
   ```

2. Abre no navegador (do seu PC, **mesma rede**):
   ```
   https://<hostname>.local        (porta padrão se livre)
   ou
   https://<hostname>.local:8443   (porta alternativa se Pi-hole rodava)
   ```

3. Aviso de certificado é normal — TLS local. Clica em **Avançado** → **Continuar**.

4. Login: `admin` + senha do passo 1.

---

## ⚠️ Aprendizado do mundo real

Esses são os **problemas reais** que aparecem em primeiras instalações e como o bootstrap resolve:

### "Permission denied" no banco SQLite
**Causa:** UID/GID do usuário Linux não bate com o do container.
**Como resolvido:** bootstrap detecta `stat -c %u %g $DATA_DIR` e injeta `user: "UID:GID"` no override.

### "address already in use" na porta 80/443
**Causa:** Pi-hole, nginx ou outro serviço já escutando.
**Como resolvido:** bootstrap detecta com `ss -tln` e migra pra 8080/8443 automaticamente.

### "ERR_SSL_PROTOCOL_ERROR" no navegador
**Causa:** Caddy não tinha cert pro hostname/IP requisitado.
**Como resolvido:** Caddyfile usa `https://` (any host) + `tls internal { on_demand }`.

### "404 Not Found" nas rotas /api
**Causa:** `handle_path` strippava o `/api` antes do proxy reverso.
**Como resolvido:** Caddyfile usa `handle` (preserva path completo).

### Build do Go falhando por `go.sum` ausente
**Causa:** Repo novo sem `go.sum` versionado, build não baixava deps.
**Como resolvido:** `go.sum` é versionado e o Dockerfile usa `go mod download`.

---

## 🏗️ Arquitetura

```
┌────────────────────────────────────────────────────────────────────┐
│   Browser (SvelteKit SPA)   SMB/CIFS   SFTP   WebDAV (futuro)      │
└──────────┬──────────────────────┬────────┬──────────┬──────────────┘
           ▼                      │        │          │
┌────────────────────────────┐    │        │          │
│   Caddy (Docker)           │    │        │          │
│   TLS local, HTTP/3, WS    │    │        │          │
└──────────┬─────────────────┘    │        │          │
           ▼                      │        │          │
┌────────────────────────────────────────────────────────────────────┐
│             PiNAS Backend Go (Docker, Modular Monolith)            │
│  ┌────────┬────────┬────────┬────────┐                             │
│  │  auth  │ files  │ users  │ system │                             │
│  └────────┴────────┴────────┴────────┘                             │
└──────────┬──────────────────────────────────────┬──────────────────┘
           ▼                                      ▼
┌──────────────────────────┐         ┌──────────────────────────────┐
│ SQLite (WAL)             │         │ /srv/pinas/data ou           │
│ /srv/pinas/db/pinas.db   │         │ /home/<user> (se mapeado)    │
└──────────────────────────┘         └──────────────────────────────┘

       ┌──────────────────────────────────────────────────┐
       │ Host (não containerizado)                        │
       │   smbd / nmbd      sshd (SFTP)     ufw + fail2ban│
       └──────────────────────────────────────────────────┘
```

### Stack

| Camada | Tecnologia |
|--------|------------|
| Backend | Go 1.23 (chi, sqlx, gopsutil) |
| Frontend | SvelteKit 2 + Svelte 5 (build estático) |
| Banco | SQLite 3 (WAL) |
| Reverse proxy | Caddy 2 |
| Auth | JWT HS256 + Argon2id |
| Container | Docker + Compose |

---

## 📡 Operação no dia-a-dia

```bash
# Status
cd /opt/pinas && sudo docker compose ps

# Logs
sudo docker compose logs -f
sudo docker compose logs -f pinas-api

# Reiniciar
sudo systemctl restart pinas

# Atualizar
cd /opt/pinas && sudo git pull && sudo docker compose up -d --build

# Senha admin
sudo cat /root/.pinas/admin-password.txt

# Backup automático (crontab)
0 3 * * * /opt/pinas/scripts/backup.sh /mnt/backup-externo
```

---

## 🆘 Troubleshooting

### Backend não fica "healthy"

```bash
sudo docker compose logs pinas-api
```

99% dos casos: permissão. O bootstrap detecta UID:GID, mas se você editou manualmente:

```bash
# Confere dono real da pasta
ls -ld /srv/pinas/data    # ou DATA_DIR que você usou

# Confere o que tá no override
cat /opt/pinas/docker-compose.override.yml | grep user

# Devem bater. Se não:
sudo ./bootstrap.sh    # regenera o override
```

### Build do frontend trava (OOM no Pi 4 com 2GB)

```bash
sudo dphys-swapfile swapoff
sudo sed -i 's/CONF_SWAPSIZE=.*/CONF_SWAPSIZE=2048/' /etc/dphys-swapfile
sudo dphys-swapfile setup
sudo dphys-swapfile swapon
```

### "ERR_SSL_PROTOCOL_ERROR"

Cache do navegador segurando handshake antigo. Hard reload (`Ctrl+Shift+R`) ou janela anônima.

### Não resolve `pinas.local` no Windows

Windows às vezes não tem mDNS. Soluções:
1. Instala [Bonjour Print Services](https://support.apple.com/downloads/bonjour-for-windows) (Apple, gratuito)
2. Ou usa o IP direto: `https://192.168.x.x:8443`

### Reset total

```bash
sudo systemctl disable --now pinas
cd /opt/pinas && sudo docker compose down -v
sudo rm -rf /opt/pinas /srv/pinas /root/.pinas /etc/systemd/system/pinas.service
sudo systemctl daemon-reload
```

---

## 🛣️ Roadmap

- ✅ **v0.1** — Auth, files CRUD, painel web, monitor, SMB, instalação automática
- 🚧 **v0.2** — WebSocket payload completo, search FTS5, thumbnails, audit UI
- 📋 **v0.3** — WebDAV, snapshots agendáveis, preview de vídeo (HLS)
- 🌟 **v1.0** — Multi-disco RAID, criptografia LUKS, plugins, app mobile

Veja [`docs/ROADMAP.md`](docs/ROADMAP.md) pra detalhes.

---

## 📚 Documentação

| Doc | Conteúdo |
|-----|----------|
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Decisões técnicas detalhadas |
| [`docs/API.md`](docs/API.md) | Referência da API REST |
| [`docs/DEPLOY.md`](docs/DEPLOY.md) | Deploy avançado, hardening |
| [`docs/ROADMAP.md`](docs/ROADMAP.md) | Funcionalidades futuras |

---

## 🤝 Contribuindo

Issues e PRs são bem-vindos. Mantenha o princípio **offline-first / zero-telemetria**.

## 📄 Licença

MIT — use, modifique, distribua livremente.

---

<div align="center">

**Sem nuvem. Sem telemetria. Sem assinaturas. Só seu Pi e seus arquivos.**

</div>
