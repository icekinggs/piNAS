# Deploy — PiNAS

## Pré-requisitos

- Raspberry Pi 4 (4 GB+) ou superior
- Ubuntu Server 24.04 LTS ARM64 instalado
- Acesso SSH ao Pi como usuário com sudo
- Disco USB 3.0 (recomendado SSD) formatado em ext4 — opcional mas altamente recomendado

## Passo a passo

### 1. Clonar o projeto

```bash
ssh user@<ip-do-pi>
git clone <repo-url> pinas
cd pinas
```

### 2. Configurar variáveis

```bash
cp .env.example .env
$EDITOR .env       # defina PINAS_ADMIN_PASSWORD (mínimo 8 chars)
```

### 3. Instalar dependências do host

```bash
sudo ./scripts/install.sh
```

Isso instala: samba, openssh-server, ufw, fail2ban, smartmontools, avahi-daemon, docker.

### 4. Montar o disco USB (opcional)

```bash
sudo lsblk    # identifique o /dev/sdXY
sudo ./scripts/automount-disk.sh /dev/sda1
```

Sem disco USB, o PiNAS usará o cartão SD. **Não recomendado** para dados importantes.

### 5. Configurar SMB (opcional)

```bash
sudo ./scripts/samba-setup.sh maria
```

### 6. Build do frontend

Em uma máquina com Node.js (pode ser o próprio Pi, mas é mais rápido em x86):

```bash
cd frontend
npm install
npm run build
# resultado em frontend/build/
```

Ou via Docker:
```bash
docker build -t pinas-frontend ./frontend
docker create --name fe pinas-frontend
docker cp fe:/srv/frontend ./frontend/build
docker rm fe
```

### 7. Subir o stack

```bash
cd ..       # raiz do projeto
docker compose up -d --build
```

Verifique:
```bash
docker compose ps
docker compose logs -f pinas-api
```

### 8. Acessar

- mDNS: `https://pinas.local`
- IP: `https://<ip-do-pi>`

Login inicial: `admin` + a senha que você colocou em `PINAS_ADMIN_PASSWORD`.

> O navegador vai mostrar aviso de certificado na 1ª vez. Para remover:
> exporte o root do Caddy de `caddy_data/caddy/pki/authorities/local/root.crt`
> e instale como CA confiável nos clientes.

## Atualização

```bash
git pull
docker compose pull
docker compose up -d --build
```

Migrations idempotentes — não há passo manual de DB.

Para snapshot do banco antes de update grande:
```bash
sudo cp /srv/pinas/db/pinas.db /srv/pinas/db/pinas.db.bak.$(date +%F)
```

## Recovery

Tudo persistido em `/srv/pinas/`:

```
/srv/pinas/
├── db/         (SQLite + WAL — pode dar cp seguro com sqlite3 .backup)
├── data/       (arquivos do usuário — ext4 USB)
├── secrets/    (jwt.key — perder = invalidar todos os tokens)
├── thumbs/     (cache, descartável)
└── logs/
```

Backup recomendado:

```bash
sudo ./scripts/backup.sh /mnt/usb-externo/pinas-backups
# crontab: 0 3 * * *  /home/usuario/pinas/scripts/backup.sh /mnt/usb-externo/pinas-backups
```

Para restaurar em outro Pi:

1. Restaure `/srv/pinas/` inteiro.
2. Suba o stack (`docker compose up -d`).
3. mDNS e Caddy se reconfiguram automaticamente.

## Hardening pós-deploy

1. SSH com chave pública apenas:
   ```bash
   sudo sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config
   sudo systemctl restart ssh
   ```
2. Mudar porta SSH se exposta à internet (não recomendado expor o NAS).
3. Configurar fail2ban watch para `audit.log`.
4. Habilitar SMART checks:
   ```bash
   sudo smartctl -a /dev/sda
   sudo smartctl -t short /dev/sda
   ```
