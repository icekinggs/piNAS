#!/usr/bin/env bash
# PiNAS — bootstrap.sh
#
# Script ÚNICO de instalação e atualização do PiNAS em Raspberry Pi/Ubuntu Server.
# Pode ser usado tanto em instalação limpa quanto para atualizar uma instalação existente.
#
# Uso (clonando manualmente):
#   git clone https://github.com/icekinggs/piNAS.git
#   cd piNAS
#   sudo ./bootstrap.sh
#
# Atualização de instalação existente:
#   cd /opt/pinas
#   sudo ./bootstrap.sh
#
# Variáveis opcionais (use com `sudo VAR=valor ./bootstrap.sh`):
#   DATA_DIR        Onde ficam os arquivos do NAS. Padrão: /srv/pinas/data
#   HTTP_PORT       Porta HTTP. Padrão: auto-detecta (80 ou 8080 se ocupada)
#   HTTPS_PORT      Porta HTTPS. Padrão: auto-detecta (443 ou 8443 se ocupada)
#   USB_DEVICE      Disco USB pra automount. Ex: /dev/sda1. Padrão: vazio
#   SAMBA_USER      Usuário Samba a criar. Padrão: vazio
#   INSTALL_DIR     Onde instalar o código. Padrão: /opt/pinas
#   PINAS_MODE      auto, install ou update. Padrão: auto
#   SKIP_BACKUP     1 para não criar backup pré-update. Padrão: 0
#   BACKUP_DIR      Onde guardar backups. Padrão: /srv/pinas/backups
#
# O script é idempotente: pode rodar de novo para instalar, reparar ou atualizar.

set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
info()  { echo -e "${BLUE}==>${NC} $*"; }
ok()    { echo -e "${GREEN}[OK]${NC} $*"; }
warn()  { echo -e "${YELLOW}[!!]${NC} $*"; }
fail()  { echo -e "${RED}[ERRO]${NC} $*" >&2; exit 1; }

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
	fail "Execute como root: sudo $0"
fi

# ---------- configuração ----------
REPO_URL="${REPO_URL:-https://github.com/icekinggs/piNAS.git}"
REPO_BRANCH="${REPO_BRANCH:-main}"
INSTALL_DIR="${INSTALL_DIR:-/opt/pinas}"
DATA_DIR="${DATA_DIR:-/srv/pinas/data}"
USB_DEVICE="${USB_DEVICE:-}"
SAMBA_USER="${SAMBA_USER:-}"
HTTP_PORT="${HTTP_PORT:-}"
HTTPS_PORT="${HTTPS_PORT:-}"
PINAS_MODE="${PINAS_MODE:-auto}"
SKIP_BACKUP="${SKIP_BACKUP:-0}"
BACKUP_DIR="${BACKUP_DIR:-/srv/pinas/backups}"

# ---------- lifecycle install/update ----------
detect_mode() {
	if [[ "$PINAS_MODE" != "auto" ]]; then
		echo "$PINAS_MODE"
		return
	fi

	if [[ -d "$INSTALL_DIR/.git" || -f "$INSTALL_DIR/.env" || -f /etc/systemd/system/pinas.service ]]; then
		echo "update"
	else
		echo "install"
	fi
}

MODE="$(detect_mode)"
case "$MODE" in
	install|update) ;;
	*) fail "PINAS_MODE inválido: $MODE. Use auto, install ou update." ;;
esac
ok "Modo: $MODE"

backup_existing_install() {
	[[ "$MODE" == "update" ]] || return 0
	[[ "$SKIP_BACKUP" == "1" ]] && { warn "Backup pré-update desativado por SKIP_BACKUP=1"; return 0; }

	local stamp dest
	stamp="$(date +%Y%m%d-%H%M%S)"
	dest="$BACKUP_DIR/pre-update-$stamp"

	info "Criando backup pré-update em $dest..."
	install -d -m 0750 "$dest"

	[[ -f "$INSTALL_DIR/.env" ]] && cp -a "$INSTALL_DIR/.env" "$dest/.env"
	[[ -f "$INSTALL_DIR/docker-compose.override.yml" ]] && cp -a "$INSTALL_DIR/docker-compose.override.yml" "$dest/docker-compose.override.yml"
	[[ -f "$INSTALL_DIR/docker-compose.yml" ]] && cp -a "$INSTALL_DIR/docker-compose.yml" "$dest/docker-compose.yml"
	[[ -d /srv/pinas/db ]] && rsync -a --delete /srv/pinas/db/ "$dest/db/" || true
	[[ -d /srv/pinas/configs ]] && rsync -a --delete /srv/pinas/configs/ "$dest/configs/" || true
	[[ -d /srv/pinas/apps ]] && rsync -a --delete /srv/pinas/apps/ "$dest/apps/" || true

	cat > "$dest/restore-note.txt" <<EOF
Backup criado automaticamente antes de atualizar o PiNAS.
Data: $(date -Iseconds)
Install dir: $INSTALL_DIR
Branch: $REPO_BRANCH

Este backup preserva configuração, banco, manifests e override local.
Ele NÃO copia a pasta de dados do NAS para evitar duplicar arquivos grandes.
EOF
	ok "Backup pré-update criado: $dest"
}

# ---------- sanity checks ----------
info "Verificando ambiente..."
if ! grep -q "Ubuntu" /etc/os-release; then
	warn "Testado em Ubuntu Server 24.04. Outras distros podem ter problemas."
fi
ARCH=$(dpkg --print-architecture)
ok "Ambiente OK ($(lsb_release -ds 2>/dev/null || echo desconhecido), $ARCH)"

# ---------- detecção de portas ----------
info "Verificando portas disponíveis..."
port_in_use() {
	ss -tln | awk '{print $4}' | grep -qE ":${1}\$"
}

if [[ -z "$HTTP_PORT" ]]; then
	if port_in_use 80; then
		HTTP_PORT=8080
		WHO=$(ss -tlnp 2>/dev/null | awk '$4 ~ /:80$/ {print $NF}' | head -1 | sed 's/users://;s/[",()]//g')
		warn "Porta 80 em uso por: ${WHO:-?} → usando 8080"
	else
		HTTP_PORT=80
	fi
fi
if [[ -z "$HTTPS_PORT" ]]; then
	if port_in_use 443; then
		HTTPS_PORT=8443
		WHO=$(ss -tlnp 2>/dev/null | awk '$4 ~ /:443$/ {print $NF}' | head -1 | sed 's/users://;s/[",()]//g')
		warn "Porta 443 em uso por: ${WHO:-?} → usando 8443"
	else
		HTTPS_PORT=443
	fi
fi
ok "Portas: HTTP=$HTTP_PORT, HTTPS=$HTTPS_PORT"

# ---------- 1. Pacotes ----------
info "Instalando dependências..."
apt-get update -y
DEBIAN_FRONTEND=noninteractive apt-get install -y \
	git curl wget jq ca-certificates gnupg lsb-release \
	openssh-server ufw fail2ban smartmontools avahi-daemon rsync
ok "Pacotes base instalados"

# ---------- 2. Docker ----------
if ! command -v docker >/dev/null 2>&1; then
	info "Instalando Docker..."
	install -m 0755 -d /etc/apt/keyrings
	curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
		gpg --dearmor -o /etc/apt/keyrings/docker.gpg
	chmod a+r /etc/apt/keyrings/docker.gpg
	UBUNTU_CODENAME=$(. /etc/os-release && echo "$VERSION_CODENAME")
	echo "deb [arch=$ARCH signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $UBUNTU_CODENAME stable" \
		> /etc/apt/sources.list.d/docker.list
	apt-get update -y
	DEBIAN_FRONTEND=noninteractive apt-get install -y \
		docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
	systemctl enable --now docker
fi
ok "Docker: $(docker --version)"

# ---------- 3. Node.js ----------
if ! command -v node >/dev/null 2>&1 || [[ "$(node --version 2>/dev/null | cut -dv -f2 | cut -d. -f1)" -lt 20 ]]; then
	info "Instalando Node.js 22..."
	curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
	DEBIAN_FRONTEND=noninteractive apt-get install -y nodejs
fi
ok "Node.js: $(node --version)"

# ---------- 4. Repositório ----------
backup_existing_install
if [[ "$MODE" == "update" ]]; then
	info "Atualizando PiNAS em $INSTALL_DIR..."
else
	info "Clonando PiNAS em $INSTALL_DIR..."
fi

if [[ -d "$INSTALL_DIR/.git" ]]; then
	cd "$INSTALL_DIR"
	git fetch origin "$REPO_BRANCH"
	git reset --hard "origin/$REPO_BRANCH"
	git clean -fd -e .env -e docker-compose.override.yml -e docker-compose.yml.bak
else
	rm -rf "$INSTALL_DIR"
	git clone --branch "$REPO_BRANCH" --depth 1 "$REPO_URL" "$INSTALL_DIR"
fi
cd "$INSTALL_DIR"
ok "Código em $INSTALL_DIR ($(git rev-parse --short HEAD))"

# ---------- 5. .env ----------
if [[ ! -f "$INSTALL_DIR/.env" ]]; then
	ADMIN_PASS=$(openssl rand -base64 18 | tr -d '/+=' | head -c 20)
	HOSTNAME_LOCAL=$(hostname).local
	cat > "$INSTALL_DIR/.env" <<EOF
# Gerado pelo bootstrap.sh em $(date -Iseconds)
PINAS_HOST=$HOSTNAME_LOCAL
PINAS_ALLOWED_ORIGIN=https://$HOSTNAME_LOCAL
PINAS_ADMIN_USERNAME=admin
PINAS_ADMIN_PASSWORD=$ADMIN_PASS
PINAS_ENV=production
PINAS_LOG_LEVEL=info
TZ=America/Sao_Paulo
EOF
	chmod 600 "$INSTALL_DIR/.env"
	install -d -m 0700 /root/.pinas
	echo "admin: $ADMIN_PASS" > /root/.pinas/admin-password.txt
	chmod 600 /root/.pinas/admin-password.txt
	ok ".env criado, senha em /root/.pinas/admin-password.txt"
else
	ok ".env já existe"
fi

# ---------- 6. Disco USB ----------
if [[ -n "$USB_DEVICE" ]]; then
	if [[ -b "$USB_DEVICE" ]]; then
		bash "$INSTALL_DIR/scripts/automount-disk.sh" "$USB_DEVICE" || warn "automount falhou"
		DATA_DIR="/srv/pinas/data"
	else
		warn "$USB_DEVICE não é um device. Pulando."
	fi
fi

# ---------- 7. Detecção de UID:GID do dono dos dados ----------
info "Detectando dono de $DATA_DIR..."

if [[ ! -d "$DATA_DIR" ]]; then
	REAL_USER="${SUDO_USER:-$(getent passwd 1000 | cut -d: -f1)}"
	if [[ -z "$REAL_USER" || "$REAL_USER" == "root" ]]; then
		REAL_USER=$(getent passwd 1000 | cut -d: -f1)
	fi
	install -d -m 0750 -o "$REAL_USER" -g "$REAL_USER" "$DATA_DIR" 2>/dev/null || \
		install -d -m 0750 "$DATA_DIR"
	ok "Criado $DATA_DIR (dono: ${REAL_USER:-root})"
fi

DATA_UID=$(stat -c '%u' "$DATA_DIR")
DATA_GID=$(stat -c '%g' "$DATA_DIR")
DATA_OWNER=$(stat -c '%U:%G' "$DATA_DIR")
ok "Dados: $DATA_DIR (dono: $DATA_OWNER, UID:GID = $DATA_UID:$DATA_GID)"

# ---------- 8. Diretórios persistentes ----------
info "Criando /srv/pinas/ ..."
install -d -m 0750 /srv/pinas
install -d -m 0750 -o "$DATA_UID" -g "$DATA_GID" /srv/pinas/db
install -d -m 0750 -o "$DATA_UID" -g "$DATA_GID" /srv/pinas/thumbs
install -d -m 0750 -o "$DATA_UID" -g "$DATA_GID" /srv/pinas/logs
install -d -m 0750 -o "$DATA_UID" -g "$DATA_GID" /srv/pinas/configs
install -d -m 0750 -o "$DATA_UID" -g "$DATA_GID" /srv/pinas/apps
install -d -m 0750 -o "$DATA_UID" -g "$DATA_GID" /srv/pinas/docker
install -d -m 0750 -o "$DATA_UID" -g "$DATA_GID" "$BACKUP_DIR"
install -d -m 0700 -o "$DATA_UID" -g "$DATA_GID" /srv/pinas/secrets
install -d -m 0750 -o "$DATA_UID" -g "$DATA_GID" /srv/pinas/samba
ok "Diretórios persistentes prontos"

# ---------- 9. docker-compose.override.yml ----------
info "Gerando docker-compose.override.yml..."
cat > "$INSTALL_DIR/docker-compose.override.yml" <<EOF
# Auto-gerado pelo bootstrap.sh — específico desta instalação.
# NÃO comite este arquivo (já está no .gitignore).
# Para regerar, rode: sudo ./bootstrap.sh

services:
  pinas-api:
    user: "${DATA_UID}:${DATA_GID}"
    volumes:
      - ${DATA_DIR}:/var/lib/pinas/data
EOF
ok "Override: user=$DATA_UID:$DATA_GID, data=$DATA_DIR"

# ---------- 9b. portas no docker-compose.yml ----------
if [[ "$HTTP_PORT" != "80" || "$HTTPS_PORT" != "443" ]]; then
	info "Ajustando portas no docker-compose.yml (HTTP=$HTTP_PORT, HTTPS=$HTTPS_PORT)..."
	cp "$INSTALL_DIR/docker-compose.yml" "$INSTALL_DIR/docker-compose.yml.bak"
	sed -i \
		-e "s|- \"80:80\"|- \"${HTTP_PORT}:80\"|" \
		-e "s|- \"443:443\"|- \"${HTTPS_PORT}:443\"|" \
		-e "s|- \"443:443/udp\"|- \"${HTTPS_PORT}:443/udp\"|" \
		"$INSTALL_DIR/docker-compose.yml"
	ok "Portas no compose: $HTTP_PORT, $HTTPS_PORT (backup em docker-compose.yml.bak)"
fi

# ---------- 10. UFW ----------
info "Configurando UFW..."
ufw --force reset >/dev/null
ufw default deny incoming >/dev/null
ufw default allow outgoing >/dev/null
ufw allow 22/tcp >/dev/null
ufw allow ${HTTP_PORT}/tcp >/dev/null
ufw allow ${HTTPS_PORT}/tcp >/dev/null
ufw allow ${HTTPS_PORT}/udp >/dev/null
[[ "$HTTP_PORT" != "80" ]]   && ufw allow 80/tcp >/dev/null
[[ "$HTTPS_PORT" != "443" ]] && ufw allow 443/tcp >/dev/null
ufw allow 137,138/udp >/dev/null
ufw allow 139,445/tcp >/dev/null
ufw allow 5353/udp >/dev/null
ufw allow 53/tcp >/dev/null
ufw allow 53/udp >/dev/null
ufw --force enable >/dev/null
ok "UFW ativo"

# ---------- 11. Avahi & Fail2Ban ----------
systemctl enable --now avahi-daemon
cat > /etc/fail2ban/jail.d/pinas.local <<'EOF'
[sshd]
enabled = true
maxretry = 5
bantime = 1h
EOF
systemctl enable --now fail2ban
systemctl restart fail2ban
ok "Avahi + Fail2Ban ativos"

# ---------- 12. Samba ----------
info "Instalando Samba (gerenciado via painel web em /samba)..."
DEBIAN_FRONTEND=noninteractive apt-get install -y samba samba-common-bin
ok "Samba instalado"

info "Instalando watcher de sync Samba..."
cp "$INSTALL_DIR/deploy/systemd/pinas-samba-sync.service" /etc/systemd/system/
cp "$INSTALL_DIR/deploy/systemd/pinas-samba-sync.path"    /etc/systemd/system/
sed -i "s|/opt/pinas|$INSTALL_DIR|g" /etc/systemd/system/pinas-samba-sync.service
chmod +x "$INSTALL_DIR/scripts/samba-sync.sh"
systemctl daemon-reload
systemctl enable --now pinas-samba-sync.path
ok "Watcher pinas-samba-sync.path ativo"

if [[ -n "$SAMBA_USER" ]]; then
	info "Configurando usuário Samba inicial '$SAMBA_USER' (modo legado)..."
	bash "$INSTALL_DIR/scripts/samba-setup.sh" "$SAMBA_USER" || warn "samba-setup falhou"
fi

# ---------- 13. Build do frontend ----------
info "Build do frontend (1-3 min no Pi 4)..."
cd "$INSTALL_DIR/frontend"
if [[ -f package-lock.json ]]; then
	npm ci
else
	npm install
fi
npm run build
ok "Frontend OK"

# ---------- 14. Stack Docker ----------
cd "$INSTALL_DIR"
info "Subindo stack (build do backend leva 5-10 min no Pi 4)..."
docker compose pull caddy 2>/dev/null || true
docker compose up -d --build

info "Aguardando backend ficar healthy..."
for i in $(seq 1 60); do
	if docker compose ps pinas-api 2>/dev/null | grep -q "healthy"; then
		ok "Backend healthy"
		break
	fi
	sleep 2
	[[ $i -eq 60 ]] && warn "Backend não ficou healthy em 2 min."
done

# ---------- 15. systemd ----------
info "Registrando systemd unit..."
sed "s|/opt/pinas|$INSTALL_DIR|g" "$INSTALL_DIR/deploy/systemd/pinas.service" \
	> /etc/systemd/system/pinas.service
systemctl daemon-reload
systemctl enable pinas.service >/dev/null
ok "pinas.service habilitado no boot"

# ---------- 16. Resumo ----------
IP_LOCAL=$(hostname -I | awk '{print $1}')
HOSTNAME_LOCAL=$(hostname).local
ADMIN_PASS=$(grep '^PINAS_ADMIN_PASSWORD=' "$INSTALL_DIR/.env" | cut -d= -f2-)
URL_SUFFIX=""
[[ "$HTTPS_PORT" != "443" ]] && URL_SUFFIX=":$HTTPS_PORT"

echo
echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                  PiNAS pronto com sucesso!                   ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo
echo -e "  ${BLUE}Modo:${NC}             $MODE"
echo -e "  ${BLUE}Acesso web:${NC}       https://${HOSTNAME_LOCAL}${URL_SUFFIX}"
echo -e "                     https://${IP_LOCAL}${URL_SUFFIX}"
echo
echo -e "  ${BLUE}Login admin:${NC}      admin"
echo -e "  ${BLUE}Senha:${NC}            $ADMIN_PASS"
echo -e "                     (também em /root/.pinas/admin-password.txt)"
echo
echo -e "  ${BLUE}Pasta de dados:${NC}   $DATA_DIR  (UID:GID $DATA_UID:$DATA_GID)"
echo -e "  ${BLUE}Backups:${NC}          $BACKUP_DIR"
echo
echo -e "  ${YELLOW}Aviso:${NC} navegador mostra 'conexão não privada' na 1ª vez."
echo -e "         É TLS local. Clique em 'Avançado' > 'Continuar'."
echo
echo -e "  ${BLUE}Comandos úteis:${NC}"
echo -e "    cd $INSTALL_DIR && sudo docker compose ps"
echo -e "    cd $INSTALL_DIR && sudo docker compose logs -f"
echo -e "    sudo systemctl restart pinas"
echo -e "    cd $INSTALL_DIR && sudo ./bootstrap.sh   # instalar/reparar/atualizar"
echo
[[ -z "$USB_DEVICE" ]] && echo -e "  ${BLUE}Próximos passos opcionais:${NC}"
[[ -z "$USB_DEVICE" ]] && echo -e "    • Disco USB:  sudo $INSTALL_DIR/scripts/automount-disk.sh /dev/sdXY"
[[ -z "$SAMBA_USER" ]] && echo -e "    • SMB:        sudo $INSTALL_DIR/scripts/samba-setup.sh <usuario>"
echo
