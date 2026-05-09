# Como publicar no GitHub e instalar no Raspberry Pi

Guia passo-a-passo do zero. Tempo total: ~30-45 min (a maior parte é build no Pi).

---

## Parte 1 — Publicar no GitHub

### 1.1 Criar o repositório

1. Vá em https://github.com/new
2. Nome do repo: `pinas` (ou o que preferir)
3. Visibilidade: **Private** (recomendado — você não quer expor `.env` por acidente).
4. **NÃO marque** "Add a README" ou "Add .gitignore" (já temos os nossos).
5. Clique em **Create repository**.

GitHub vai te mostrar a URL — algo como `https://github.com/SEU_USUARIO/pinas.git`. Anote.

### 1.2 Subir o código (no seu computador, não no Pi)

Extraia o `pinas.tar.gz` que recebeu e abra um terminal nessa pasta:

```bash
tar -xzf pinas.tar.gz
cd pinas
```

Agora inicie o git e suba:

```bash
git init -b main
git add .
git commit -m "Initial commit: PiNAS v0.1"

# Substitua SEU_USUARIO pela sua conta do GitHub
git remote add origin https://github.com/SEU_USUARIO/pinas.git
git push -u origin main
```

Se for repo privado, o `push` vai pedir credenciais. Use um **Personal Access Token** (PAT) em vez da senha da conta:
- GitHub → Settings → Developer settings → Personal access tokens → **Tokens (classic)** → Generate new token (classic)
- Marque o escopo `repo` (todo)
- Copie o token e cole quando o `push` pedir senha

✅ Pronto. Código no GitHub.

### 1.3 Editar o `bootstrap.sh` com a URL do seu repo

No `bootstrap.sh`, linha ~32, troque:

```bash
REPO_URL="${REPO_URL:-https://github.com/SEU_USUARIO/pinas.git}"
```

…pelo seu repo real, **commite e push de novo**:

```bash
git add bootstrap.sh
git commit -m "set repo url"
git push
```

---

## Parte 2 — Preparar o Raspberry Pi

### 2.1 Hardware necessário

- Raspberry Pi 4 (4 GB ou mais) ou Pi 5
- Cartão microSD 32 GB+ classe 10 (para o sistema)
- Disco USB 3.0 (HDD ou SSD) para armazenamento — opcional mas **muito** recomendado
- Cabo de rede Ethernet (Wi-Fi funciona mas é mais lento)
- Fonte oficial 5V/3A
- Acesso SSH (ou monitor + teclado para a primeira config)

### 2.2 Instalar o Ubuntu Server 24.04 ARM64

1. Baixe o **Raspberry Pi Imager**: https://www.raspberrypi.com/software/
2. Abra, escolha:
   - **Device**: seu modelo de Pi
   - **OS**: Other general-purpose OS → Ubuntu → **Ubuntu Server 24.04 LTS (64-bit)**
   - **Storage**: o cartão SD
3. Clique no ícone de **engrenagem** (configurações). Configure:
   - ✅ Set hostname: `pinas`
   - ✅ Enable SSH → "Use password authentication"
   - ✅ Set username and password (anote)
   - ✅ Configure wireless LAN (se for usar Wi-Fi)
   - ✅ Set locale: timezone `America/Sao_Paulo`
4. Salvar → Write. Espera ~5 min.

### 2.3 Primeiro boot

1. Coloque o SD no Pi, conecte o disco USB e o cabo de rede.
2. Ligue. Espera ~2 min para o primeiro boot completar.
3. Descubra o IP do Pi: olhe no roteador, ou use:
   ```bash
   # do seu computador, na mesma rede
   ping pinas.local
   # ou:
   nmap -sn 192.168.1.0/24    # ajuste para sua faixa
   ```

### 2.4 Conecte via SSH

```bash
ssh seu_usuario@pinas.local
# ou
ssh seu_usuario@192.168.1.42
```

Atualize o sistema:

```bash
sudo apt update && sudo apt upgrade -y
sudo reboot     # se atualizou kernel
```

Reconecte depois do reboot.

---

## Parte 3 — Instalar o PiNAS

Aqui é onde a mágica acontece. **Uma única linha**.

### 3.1 Identificar seu disco USB (se houver)

```bash
sudo lsblk -o NAME,SIZE,FSTYPE,MOUNTPOINT
```

Procure o disco USB. Vai aparecer algo como `sda` com uma partição `sda1`. Anote: `/dev/sda1`.

> ⚠️ Se o disco não estiver formatado em ext4 ainda:
> ```bash
> sudo mkfs.ext4 -L pinas-data /dev/sda1   # APAGA TUDO!
> ```

### 3.2 Rodar o bootstrap

Se seu repo é **público**:

```bash
curl -fsSL https://raw.githubusercontent.com/SEU_USUARIO/pinas/main/bootstrap.sh | \
  sudo USB_DEVICE=/dev/sda1 SAMBA_USER=meunome bash
```

Se seu repo é **privado** (mais comum), você precisa autenticar. Mais simples: clone manualmente:

```bash
# Configure git com seu Personal Access Token:
git config --global credential.helper store
git clone https://github.com/SEU_USUARIO/pinas.git
# vai pedir usuário e senha (use o PAT como senha)

cd pinas
sudo USB_DEVICE=/dev/sda1 SAMBA_USER=meunome ./bootstrap.sh
```

> Variáveis disponíveis (todas opcionais):
> - `USB_DEVICE=/dev/sda1` — monta esse disco em `/srv/pinas/data`
> - `SAMBA_USER=meunome` — cria compartilhamento SMB
> - `INSTALL_DIR=/opt/pinas` — onde instalar (padrão `/opt/pinas`)
> - `REPO_BRANCH=main`

O script vai:
1. Atualizar apt e instalar deps (git, docker, node, ufw, fail2ban, etc.) — ~3-5 min
2. Clonar o repo em `/opt/pinas`
3. Gerar senha aleatória de admin e salvar no `.env`
4. Criar diretórios `/srv/pinas/`
5. Configurar firewall, fail2ban, mDNS
6. (Opcional) Montar o disco USB no `/etc/fstab`
7. (Opcional) Configurar SMB
8. Build do frontend SvelteKit — ~2-3 min no Pi 4
9. Build da imagem Docker do backend Go — ~3-5 min no Pi 4
10. Subir o stack
11. Habilitar auto-start no boot via systemd

**Tempo total no Pi 4:** ~15-25 min.

No final ele imprime:

```
╔═══════════════════════════════════════════════════════════════╗
║                  PiNAS instalado com sucesso!                 ║
╚═══════════════════════════════════════════════════════════════╝

  Acesso web:       https://pinas.local
                    https://192.168.1.42

  Login admin:      admin
  Senha:            xY9vKp2nQrM4tA8z
                    (também em /root/.pinas/admin-password.txt)
```

### 3.3 Acessar o painel

Abra no navegador: `https://pinas.local`

**Aviso de certificado:** o Caddy gera um TLS local autoassinado. Clique em "Avançado" → "Continuar". É seguro — é seu próprio Pi, não tem certificado público porque não tem domínio público.

Se quiser remover o aviso, importe o root CA do Caddy nos seus dispositivos (instruções em `docs/DEPLOY.md`).

---

## Parte 4 — Operação no dia-a-dia

### Ver logs

```bash
cd /opt/pinas
sudo docker compose logs -f                # tudo
sudo docker compose logs -f pinas-api      # só backend
sudo docker compose logs -f caddy          # só proxy
```

### Reiniciar

```bash
sudo systemctl restart pinas
# ou
cd /opt/pinas && sudo docker compose restart
```

### Atualizar para nova versão

```bash
cd /opt/pinas
sudo git pull
sudo docker compose up -d --build
```

Ou rode o `bootstrap.sh` de novo — é idempotente.

### Backup automático

Adicione ao crontab do root:

```bash
sudo crontab -e
```

Cole:

```
0 3 * * * /opt/pinas/scripts/backup.sh /mnt/backup-externo/pinas-backups >> /var/log/pinas-backup.log 2>&1
```

(ajuste o caminho do backup para um disco externo separado)

### Acesso SMB do Windows

`\\pinas.local\pinas` (ou `\\192.168.1.42\pinas`).
Login: o `SAMBA_USER` que você definiu, com a senha que digitou.

### Acesso SFTP

Qualquer cliente SFTP (FileZilla, WinSCP, Cyberduck):
- Host: `pinas.local`
- Porta: `22`
- Usuário/senha: o usuário Linux do Pi

---

## Solução de problemas

### "docker compose" diz "permission denied"
Sua sessão SSH ainda não pegou o grupo docker. Logout/login, ou:
```bash
sudo usermod -aG docker $USER
newgrp docker
```

### Build do frontend trava ou OOM no Pi 4
Pi 4 com 4GB às vezes não dá conta do build do Vite. Soluções:
1. Ative swap antes:
   ```bash
   sudo dphys-swapfile swapoff
   sudo sed -i 's/CONF_SWAPSIZE=.*/CONF_SWAPSIZE=2048/' /etc/dphys-swapfile
   sudo dphys-swapfile setup
   sudo dphys-swapfile swapon
   ```
2. Ou builde o frontend no seu computador e copie pro Pi:
   ```bash
   # no seu PC:
   cd pinas/frontend && npm install && npm run build
   tar -czf build.tar.gz build/
   scp build.tar.gz pi@pinas.local:/tmp/
   # no Pi:
   cd /opt/pinas/frontend && tar -xzf /tmp/build.tar.gz
   sudo docker compose restart caddy
   ```

### Backend não fica "healthy"
```bash
cd /opt/pinas
sudo docker compose logs pinas-api
```
Erros comuns:
- `PINAS_ADMIN_PASSWORD obrigatório em produção` → editar `/opt/pinas/.env`
- Permissão em `/srv/pinas/...` → `sudo chown -R 1000:1000 /srv/pinas`

### Esqueci a senha do admin
```bash
sudo cat /root/.pinas/admin-password.txt
```
Ou redefina via SQL:
```bash
# Acesso direto no SQLite (gera o hash com argon2 dentro do container)
sudo docker compose exec pinas-api sh
# (no shell do container — funciona se você tiver script auxiliar; ou)
sudo docker compose down
sudo rm /srv/pinas/db/pinas.db*
# ATENÇÃO: isso apaga todos os usuários e sessões. Os ARQUIVOS continuam intactos.
# Edite .env com nova senha e:
sudo docker compose up -d
```

### Quero remover tudo
```bash
sudo systemctl disable --now pinas
cd /opt/pinas && sudo docker compose down -v
sudo rm -rf /opt/pinas /srv/pinas /root/.pinas /etc/systemd/system/pinas.service
sudo systemctl daemon-reload
```
