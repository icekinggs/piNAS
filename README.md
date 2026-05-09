<div align="center">

# 🗄️ PiNAS

**Private NAS for Raspberry Pi**

Sistema NAS self-hosted, privado, leve e moderno para Raspberry Pi 4+ com Ubuntu Server 24.04 LTS ARM64.

[![License](https://img.shields.io/badge/license-MIT-green)]()
[![Status](https://img.shields.io/badge/status-MVP-orange)]()
[![Platform](https://img.shields.io/badge/platform-arm64%20%7C%20amd64-blue)]()
[![Go](https://img.shields.io/badge/backend-Go%201.23-00ADD8)]()
[![Svelte](https://img.shields.io/badge/frontend-SvelteKit%202-FF3E00)]()

[**Quickstart**](#-quickstart) · [**Cenários de uso**](#-qual-cenário-é-o-seu) · [**Como funciona**](#-como-funciona) · [**Documentação**](#-documentação)

</div>

---

## ✨ Por que PiNAS?

Alternativa minimalista ao **TrueNAS / OpenMediaVault / CasaOS**, focada em três princípios:

- 🔒 **Privacidade total** — zero telemetria, zero serviços externos, zero nuvem.
- ⚡ **Leve** — backend Go (binário único, ~25 MB RAM ocioso) + SvelteKit estático. Roda em 512 MB.
- 🌐 **Offline-first** — funciona 100% sem internet. DNS, certificados e auth são locais.

### Funcionalidades

- 📊 **Dashboard** com CPU, RAM, temperatura, disco, rede em tempo real
- 📁 **Explorador de arquivos** com upload drag-and-drop, rename, move, copy, delete
- 👥 **Gerenciamento de usuários** com roles (admin/user), quotas e ACL básica
- 🔐 **Auth segura** com JWT + Argon2id + refresh tokens rotativos
- 🌐 **HTTPS local** automático via Caddy (TLS interno)
- 🔌 **Multi-protocolo** opcional: SMB, SFTP, WebDAV (futuramente)
- 📈 **WebSocket** para métricas live e progresso de upload
- 🛡️ **Segurança em camadas**: path jail, rate limit, fail2ban, UFW

---

## 🎯 Qual cenário é o seu?

Escolha um e pule pra seção correspondente. **Todos usam o mesmo `bootstrap.sh`**, mudam só os parâmetros.

| Cenário | Pra quem | Tempo | Seção |
|---------|----------|-------|-------|
| 🟢 **A. Só painel web** | "Quero um lugar bonito pra gerenciar meus arquivos pelo navegador" | 20 min | [Cenário A](#-cenário-a--apenas-painel-web) |
| 🟢 **B. Painel + Samba já existente** | "Já uso Samba, só quero ver/gerenciar pelo navegador também" | 20 min | [Cenário B](#-cenário-b--integrar-com-samba-existente) |
| 🟡 **C. NAS completo do zero** | "Não tenho nada ainda, quero painel + SMB + tudo configurado" | 30 min | [Cenário C](#-cenário-c--nas-completo-do-zero) |
| 🟡 **D. Disco USB dedicado** | "Tenho um HDD/SSD externo só para os arquivos" | 30 min | [Cenário D](#-cenário-d--disco-usb-dedicado) |
| 🔵 **E. Multi-usuário corporativo** | "Quero famílias/equipes com áreas separadas" | 45 min | [Cenário E](#-cenário-e--multi-usuário-com-áreas-separadas) |

---

## 📋 Pré-requisitos (todos os cenários)

### Hardware

- Raspberry Pi 4 (4 GB+) ou Pi 5
- Cartão microSD 32 GB+ classe 10 (sistema)
- (Opcional) HDD/SSD USB 3.0 para dados
- Cabo Ethernet (Wi-Fi funciona mas é bem mais lento)
- Fonte oficial 5V/3A (ou 5V/5A no Pi 5)

### Software

- **Ubuntu Server 24.04 LTS ARM64** instalado
- Acesso SSH funcionando
- Conexão à internet (só durante a instalação)

### Como instalar Ubuntu Server no Pi (se ainda não tem)

1. Baixe **Raspberry Pi Imager**: https://www.raspberrypi.com/software/
2. Choose Device → seu Pi
3. Choose OS → Other general-purpose OS → Ubuntu → **Ubuntu Server 24.04 LTS (64-bit)**
4. Choose Storage → seu cartão SD
5. Clique em ⚙️ (engrenagem):
   - ✅ Set hostname: `pinas`
   - ✅ Enable SSH (password auth)
   - ✅ Set username and password
   - ✅ Configure wireless LAN (se usar Wi-Fi)
   - ✅ Locale: timezone `America/Sao_Paulo`
6. Save → Write
7. Coloca no Pi, liga, espera 2 min, conecta via SSH

```bash
ssh seu_usuario@pinas.local
sudo apt update && sudo apt upgrade -y
sudo reboot
```

---

## 🟢 Cenário A — Apenas painel web

> "Quero um lugar bonito pra gerenciar arquivos do meu Pi pelo navegador. Sem SMB, sem nada além do painel."

**O que você terá:**
- Painel web em `https://pinas.local`
- Pasta padrão `/srv/pinas/data/` gerenciada pelo painel
- Auth com usuário/senha
- Dashboard de monitoramento

**Não terá:** acesso por Samba, SFTP customizado, multi-disco.

### Instalação

```bash
ssh seu_usuario@pinas.local
git clone https://github.com/icekinggs/piNAS.git
cd piNAS
sudo ./bootstrap.sh
```

Espera ~20 min. No final:

```
╔═══════════════════════════════════════════════════════════════╗
║                  PiNAS instalado com sucesso!                 ║
╚═══════════════════════════════════════════════════════════════╝

  Acesso web:       https://pinas.local
  Login admin:      admin
  Senha:            xY9vKp2nQrM4tA8z
```

Acessa `https://pinas.local`, login com `admin` + senha gerada. Pronto.

> ℹ️ **Aviso de certificado no navegador:** é normal — o Caddy gera um TLS local autoassinado (não tem domínio público pra Let's Encrypt). Clique em "Avançado" → "Continuar mesmo assim". Pra remover o aviso, importe o root CA do Caddy: veja [`docs/DEPLOY.md`](docs/DEPLOY.md).

---

## 🟢 Cenário B — Integrar com Samba existente

> "Eu já tenho Samba instalado e compartilhando uma pasta na rede. Quero o painel web mostrando os MESMOS arquivos."

**Cenário típico:** você já fez `sudo apt install samba` e tem `/home/usuario/` ou `/mnt/disco/` compartilhado pelo Windows/Mac. Quer que o painel PiNAS também enxergue essa pasta.

### Passo B.1 — Descubra qual pasta o Samba compartilha

```bash
sudo grep -A 5 "^\[" /etc/samba/smb.conf | grep "path ="
```

Anote o `path =`. Exemplo: `/home/gustavo` ou `/mnt/storage`.

### Passo B.2 — Confira o UID do dono dos arquivos

```bash
ls -ld /home/gustavo    # ou o caminho que você anotou
# saída: drwxr-xr-x 12 gustavo gustavo 4096 ... /home/gustavo

id gustavo
# saída: uid=1000(gustavo) gid=1000(gustavo) ...
```

> ⚠️ **Se o UID NÃO for 1000**, anota — você vai precisar ajustar o `Dockerfile` ou as permissões. Veja [seção de troubleshooting](#-troubleshooting).

### Passo B.3 — Roda o bootstrap

```bash
git clone https://github.com/icekinggs/piNAS.git
cd piNAS
sudo ./bootstrap.sh
```

### Passo B.4 — Aponta o PiNAS para a pasta do Samba

```bash
cd /opt/pinas
sudo docker compose down
```

Edita o `docker-compose.yml`:

```bash
sudo nano docker-compose.yml
```

Procura a linha:
```yaml
      - /srv/pinas/data:/var/lib/pinas/data
```

E troca o lado esquerdo pelo path do Samba:
```yaml
      - /home/gustavo:/var/lib/pinas/data
```

(Salva: `Ctrl+O`, `Enter`, `Ctrl+X`)

### Passo B.5 — Sobe de novo

```bash
sudo docker compose up -d
```

**Pronto!** Acessa `https://pinas.local`, login com `admin`, e o explorador vai mostrar **exatamente os arquivos do compartilhamento Samba**.

> 💡 Login com **admin**: vê tudo na raiz `/`.
> Login com usuário comum: fica preso em `/users/<nome>/`.

---

## 🟡 Cenário C — NAS completo do zero

> "Não tenho nada ainda. Quero painel web + SMB + SFTP + firewall + backup, tudo configurado de uma vez."

**O que você terá:**
- Painel web em `https://pinas.local`
- Compartilhamento SMB acessível em `\\pinas.local\pinas`
- SFTP via porta 22 (já vem com Ubuntu)
- UFW + Fail2Ban configurados
- Backup script pronto (você define onde rodar)

### Instalação

```bash
ssh seu_usuario@pinas.local
git clone https://github.com/icekinggs/piNAS.git
cd piNAS

# Substitua "gustavo" pelo nome de usuário que você quer usar no SMB
sudo SAMBA_USER=gustavo ./bootstrap.sh
```

Durante a instalação, vai te pedir a **senha SMB** para esse usuário (pode ser diferente da senha do Linux). Anote.

### Pós-instalação

**Acesso web:** `https://pinas.local` (login com `admin` + senha gerada)

**Acesso SMB:**
- Windows: `\\pinas.local\pinas` ou `\\<ip>\pinas`
- macOS: Finder → Cmd+K → `smb://pinas.local/pinas`
- Linux: `smb://pinas.local/pinas` (Nautilus) ou `smbclient -U gustavo //pinas.local/pinas`

**Acesso SFTP:** qualquer cliente (FileZilla, WinSCP, Cyberduck) — host `pinas.local`, porta 22, usuário/senha do Linux.

**Tanto o painel web quanto o SMB enxergam a mesma pasta** (`/srv/pinas/data/`).

### Configurar backup automático

Adicione ao crontab do root:

```bash
sudo crontab -e
```

Cole (ajusta o destino):

```cron
0 3 * * * /opt/pinas/scripts/backup.sh /mnt/disco-backup/pinas-backups >> /var/log/pinas-backup.log 2>&1
```

Roda diariamente às 3h, snapshots incrementais via rsync + hardlinks (estilo Time Machine — só ocupa espaço de mudanças).

---

## 🟡 Cenário D — Disco USB dedicado

> "Tenho um HDD/SSD USB 3.0 que quero usar SÓ para os arquivos. Sistema fica no SD card."

**O que você terá:**
- Sistema no SD (rápido pra boot)
- Dados no USB 3.0 (mais espaço, mais durável)
- Falha no SD não compromete os dados (e vice-versa)

### Passo D.1 — Identifica o disco

```bash
sudo lsblk -o NAME,SIZE,FSTYPE,MOUNTPOINT
```

Procura o disco USB (geralmente `sda` ou `sdb`). Exemplo:
```
NAME        SIZE FSTYPE MOUNTPOINT
sda         931G
└─sda1      931G ext4
mmcblk0      30G
├─mmcblk0p1 256M vfat   /boot/firmware
└─mmcblk0p2  29G ext4   /
```

Aqui, o USB é `/dev/sda1`.

### Passo D.2 — Formata em ext4 (se necessário)

> ⚠️ **Isso APAGA tudo no disco.** Pula se o disco já tem ext4 e arquivos importantes.

```bash
sudo umount /dev/sda1 2>/dev/null || true
sudo mkfs.ext4 -L pinas-data /dev/sda1
```

### Passo D.3 — Instala com automount

```bash
git clone https://github.com/icekinggs/piNAS.git
cd piNAS
sudo USB_DEVICE=/dev/sda1 SAMBA_USER=gustavo ./bootstrap.sh
```

O bootstrap vai:
1. Adicionar entrada no `/etc/fstab` via UUID (sobrevive a reboot)
2. Montar em `/srv/pinas/data/`
3. Garantir permissões corretas
4. Configurar SMB apontando pro disco USB

### Verificação pós-instalação

```bash
df -h /srv/pinas/data
# /dev/sda1   931G   24K   884G   1%   /srv/pinas/data
```

Se aparecer assim, está montado corretamente.

---

## 🔵 Cenário E — Multi-usuário com áreas separadas

> "Família com 4 pessoas, cada um deve ter sua área privada + uma pasta compartilhada."

**O que você terá:**
- Cada usuário acessa só `/users/<nome>/` (privado)
- Pasta `/shared/` acessível por todos
- Admin (você) vê tudo

### Estrutura

```
/srv/pinas/data/
├── users/
│   ├── gustavo/      ← só Gustavo vê
│   ├── maria/        ← só Maria vê
│   ├── joao/         ← só João vê
│   └── lucia/        ← só Lúcia vê
└── shared/           ← todos veem
```

### Passo E.1 — Instala (cenário C ou D)

Rode o bootstrap normalmente, com ou sem disco dedicado:

```bash
sudo SAMBA_USER=gustavo ./bootstrap.sh
# ou com USB dedicado:
sudo USB_DEVICE=/dev/sda1 SAMBA_USER=gustavo ./bootstrap.sh
```

### Passo E.2 — Cria usuários no painel web

1. Acessa `https://pinas.local`
2. Login com `admin` (senha em `/root/.pinas/admin-password.txt`)
3. Vai em **Usuários** → **+ Novo usuário**
4. Pra cada pessoa:
   - Username: `maria`, `joao`, `lucia`
   - Senha: mínimo 8 chars
   - Role: `user` (não admin)
   - Quota: `0` para ilimitado, ou `53687091200` (50 GB) etc.

### Passo E.3 — Cria as pastas

Pelo painel (logado como admin), na aba **Arquivos**:
- Cria `/users/maria/` (e cada usuário)
- Cria `/shared/`

Cada usuário vai logar e ver só a própria pasta. Admin vê tudo na raiz.

### (Opcional) Espelhar no SMB

Por padrão, o SMB do passo E.1 expõe **toda a `/srv/pinas/data/`** com autenticação Samba. Se quiser cada usuário com seu próprio share Samba (mais granular), rode pra cada usuário:

```bash
sudo /opt/pinas/scripts/samba-setup.sh maria
sudo /opt/pinas/scripts/samba-setup.sh joao
# ...
```

> ℹ️ Há uma diferença importante: usuários do **PiNAS** (banco SQLite) e usuários do **Samba** (sistema Linux) são **independentes**. Para uma única identidade unificada, veja o roadmap fase 4.

---

## ⚙️ Variáveis de ambiente do bootstrap

Todas opcionais. Use `VAR=valor sudo ./bootstrap.sh`:

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `REPO_URL` | `https://github.com/icekinggs/piNAS.git` | URL do repo |
| `REPO_BRANCH` | `main` | Branch a clonar |
| `INSTALL_DIR` | `/opt/pinas` | Onde instalar |
| `USB_DEVICE` | (vazio) | `/dev/sda1` etc. para automount |
| `SAMBA_USER` | (vazio) | Usuário Samba a criar |

Exemplo combinado:
```bash
sudo INSTALL_DIR=/opt/nas USB_DEVICE=/dev/sda1 SAMBA_USER=admin ./bootstrap.sh
```

---

## 🏗️ Como funciona

### Arquitetura geral

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

### Stack

| Camada | Tecnologia | Por quê |
|--------|------------|---------|
| Backend | Go 1.23 (chi, sqlx) | Binário único, ~25 MB RAM ocioso |
| Frontend | SvelteKit 2 + Svelte 5 | Bundle 5-10× menor que React, build estático |
| Banco | SQLite 3 (WAL) | Zero RAM ociosa vs ~100 MB do Postgres |
| Reverse proxy | Caddy 2 | TLS local automático, HTTP/3 |
| Auth | JWT HS256 + Argon2id | Padrão de mercado, calibrado pro Pi |
| Realtime | gorilla/websocket | Métricas live, progresso de upload |
| Container | Docker + Compose | Isolamento, fácil rebuild |

### Layout do filesystem

```
/srv/pinas/
├── db/             SQLite + WAL
├── data/           Arquivos do usuário (jail aqui)
├── thumbs/         Cache de thumbnails (descartável)
├── logs/
└── secrets/        jwt.key (gerada na 1ª subida, perm 0600)
```

O backend é **chrootado logicamente** (jail) em `/srv/pinas/data/` — toda operação passa por `Resolve()` que impede `..` ou paths absolutos escaparem.

📖 Detalhes em [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

---

## 📡 Operação no dia-a-dia

### Comandos comuns

```bash
# ver status
sudo systemctl status pinas
cd /opt/pinas && sudo docker compose ps

# logs
sudo docker compose logs -f                # tudo
sudo docker compose logs -f pinas-api      # backend
sudo docker compose logs -f caddy          # proxy

# reiniciar
sudo systemctl restart pinas

# parar / iniciar
sudo systemctl stop pinas
sudo systemctl start pinas
```

### Atualizar para nova versão

```bash
cd /opt/pinas
sudo git pull
sudo docker compose up -d --build
```

Migrations idempotentes — sem passos manuais. Para máxima segurança, faça backup do DB antes:

```bash
sudo cp /srv/pinas/db/pinas.db /srv/pinas/db/pinas.db.bak.$(date +%F)
```

### Backup automático

```bash
sudo crontab -e
```

```cron
# Diário às 3h, snapshots incrementais (hardlinks)
0 3 * * * /opt/pinas/scripts/backup.sh /mnt/backup-externo/pinas-snapshots >> /var/log/pinas-backup.log 2>&1
```

Cada execução cria `snapshot-YYYYMMDD-HHMM/` reaproveitando blocos via `--link-dest`. Mantém os últimos 14 snapshots.

### Resetar senha do admin

```bash
sudo cat /root/.pinas/admin-password.txt
```

Esqueceu de salvar? Reset hard (apaga apenas usuários, **arquivos ficam intactos**):

```bash
cd /opt/pinas
sudo docker compose down
sudo rm /srv/pinas/db/pinas.db*
# edita /opt/pinas/.env, troca PINAS_ADMIN_PASSWORD
sudo docker compose up -d
```

### Remover completamente

```bash
sudo systemctl disable --now pinas
cd /opt/pinas && sudo docker compose down -v
sudo rm -rf /opt/pinas /srv/pinas /root/.pinas /etc/systemd/system/pinas.service
sudo systemctl daemon-reload
```

---

## 🆘 Troubleshooting

### "docker compose" diz "permission denied"

Sua sessão SSH ainda não pegou o grupo docker. Logout/login, ou:
```bash
sudo usermod -aG docker $USER
newgrp docker
```

### Build do frontend trava ou OOM no Pi 4

Pi 4 com 2GB ou 4GB às vezes não dá conta do build do Vite. **Solução 1** — ativa swap:

```bash
sudo dphys-swapfile swapoff
sudo sed -i 's/CONF_SWAPSIZE=.*/CONF_SWAPSIZE=2048/' /etc/dphys-swapfile
sudo dphys-swapfile setup
sudo dphys-swapfile swapon
```

**Solução 2** — builda no seu computador e copia:
```bash
# no seu PC:
git clone https://github.com/icekinggs/piNAS.git
cd piNAS/frontend
npm install && npm run build
tar -czf build.tar.gz build/
scp build.tar.gz pi@pinas.local:/tmp/

# no Pi:
cd /opt/pinas/frontend
tar -xzf /tmp/build.tar.gz
sudo docker compose restart caddy
```

### Backend não fica "healthy"

```bash
cd /opt/pinas
sudo docker compose logs pinas-api
```

Erros comuns:

| Erro | Causa | Solução |
|------|-------|---------|
| `PINAS_ADMIN_PASSWORD obrigatório` | `.env` sem senha | Edita `/opt/pinas/.env`, define `PINAS_ADMIN_PASSWORD=...` |
| `permission denied: /var/lib/pinas/...` | UID dos arquivos ≠ 1000 | `sudo chown -R 1000:1000 /srv/pinas` |
| `database is locked` | Outro processo segurando o DB | `sudo docker compose restart pinas-api` |

### UID do dono dos arquivos não é 1000

Se você está integrando com pasta existente (cenário B) e o dono tem UID diferente:

```bash
# verifica
stat -c '%u %g' /home/gustavo

# OPÇÃO 1: muda o UID dos arquivos para 1000
sudo chown -R 1000:1000 /home/gustavo

# OPÇÃO 2: roda o container com o UID certo
# editar docker-compose.yml e adicionar abaixo de "container_name: pinas-api":
#   user: "1001:1001"   # use seu UID real
sudo docker compose up -d
```

### Não consigo acessar `https://pinas.local`

```bash
# verifica se o avahi está rodando
systemctl status avahi-daemon

# do seu PC (Mac/Linux):
ping pinas.local

# Windows: pinga pelo IP direto
ping 192.168.x.x
```

Windows às vezes não resolve `.local`. Use o IP direto, ou instale **Bonjour Print Services** (Apple).

### Mudei de rede / IP do Pi mudou

Se você acessa por `pinas.local`, não muda nada — mDNS resolve o IP novo automaticamente.

Se acessa por IP fixo, atualize seu bookmark.

### "Aviso de certificado" no navegador

Esperado — TLS local não tem CA pública. Pra remover o aviso, exporte o root do Caddy:

```bash
sudo docker compose exec caddy cat /data/caddy/pki/authorities/local/root.crt | sudo tee /tmp/pinas-root.crt
scp pi@pinas.local:/tmp/pinas-root.crt ~/Downloads/
```

E importe no seu sistema/navegador como CA confiável. Detalhes por SO em [`docs/DEPLOY.md`](docs/DEPLOY.md).

---

## 🛣️ Roadmap

### ✅ Fase 1 — MVP (atual, v0.1)
- Auth, users, files CRUD, painel web, monitor, SMB via host

### 🚧 Fase 2 — Hardening (v0.2)
- WebSocket completo, search FTS5, thumbnails, audit UI, upload chunked

### 📋 Fase 3 — Funcionalidades (v0.3+)
- WebDAV nativo, snapshots agendáveis, preview de vídeo (HLS), plugins, sync entre PiNAS, app mobile

### 🌟 Fase 4 — Avançado (v1.0+)
- Multi-disco RAID 1, encriptação LUKS, integração Tailscale, PAM module unificando users PiNAS+SMB, apps Docker on-demand, IA local opcional

📖 Detalhes em [`docs/ROADMAP.md`](docs/ROADMAP.md).

---

## 📚 Documentação

| Doc | Conteúdo |
|-----|----------|
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Decisões técnicas, padrões, performance |
| [`docs/API.md`](docs/API.md) | Referência completa da API REST |
| [`docs/DEPLOY.md`](docs/DEPLOY.md) | Deploy detalhado, hardening, recovery |
| [`docs/ROADMAP.md`](docs/ROADMAP.md) | Funcionalidades planejadas |
| [`INSTALL.md`](INSTALL.md) | Guia passo-a-passo (publicar no GitHub + instalar no Pi) |

---

## 🤝 Contribuindo

PiNAS é um projeto pessoal aberto. Se quiser contribuir:

1. Abra uma issue descrevendo o que pretende fazer
2. Fork → branch → PR
3. Mantenha código simples, comentado, sem dependências pesadas
4. Mantenha o princípio **offline-first / zero-telemetria**

---

## 📄 Licença

MIT. Use, modifique, distribua livremente. Apenas mantenha a atribuição.

---

## 💬 Contato

- GitHub: [@icekinggs](https://github.com/icekinggs)
- Issues: https://github.com/icekinggs/piNAS/issues

---

<div align="center">

**Feito com ☕ e respeito à sua privacidade.**

Sem nuvem. Sem telemetria. Sem assinaturas. Só seu Pi e seus arquivos.

</div>
