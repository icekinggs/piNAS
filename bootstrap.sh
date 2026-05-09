#!/usr/bin/env bash
# PiNAS — bootstrap.sh
#
# Script ÚNICO de instalação a partir do zero num Raspberry Pi com
# Ubuntu Server 24.04 LTS ARM64 recém-instalado.
#
# Uso (via curl):
#   curl -fsSL https://raw.githubusercontent.com/<seu-usuario>/pinas/main/bootstrap.sh | sudo bash
#
# Uso (clonando manualmente):
#   git clone https://github.com/<seu-usuario>/pinas.git
#   cd pinas
#   sudo ./bootstrap.sh
#
# O script é idempotente: pode rodar de novo se algo der errado.

set -euo pipefail

# ---------- helpers ----------
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
info()  { echo -e "${BLUE}==>${NC} $*"; }
ok()    { echo -e "${GREEN}[OK]${NC} $*"; }
warn()  { echo -e "${YELLOW}[!!]${NC} $*"; }
fail()  { echo -e "${RED}[ERRO]${NC} $*" >&2; exit 1; }

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
	fail "Execute como root: sudo $0"
fi

# ---------- configuração ----------
# Você pode sobrescrever via variáveis de ambiente:
#   REPO_URL=https://github.com/fulano/pinas.git INSTALL_DIR=/opt/pinas ./bootstrap.sh
REPO_URL="${REPO_URL:-https://github.com/icekinggs/piNAS.git}"
REPO_BRANCH="${REPO_BRANCH:-main}"
INSTALL_DIR="${INSTALL_DIR:-/opt/pinas}"
USB_DEVICE="${USB_DEVICE:-}"          # ex: /dev/sda1 — vazio = pula automount
SAMBA_USER="${SAMBA_USER:-}"          # vazio = pula samba

# ---------- sanity checks ----------
info "Verificando ambiente..."

if ! grep -q "Ubuntu" /etc/os-release; then
	warn "Este script foi testado em Ubuntu Server 24.04. Pode não funcionar em outras distros."
fi

ARCH=$(dpkg --print-architecture)
if [[ "$ARCH" != "arm64" && "$ARCH" != "amd64" ]]; then
	warn "Arquitetura $ARCH não é arm64/amd64. Pode haver problemas com Docker images."
fi

ok "Ambiente OK ($(lsb_release -ds 2>/dev/null || echo desconhecido), $ARCH)"

# ---------- 1. Pacotes do sistema ----------
info "Atualizando apt e instalando dependências base..."
apt-get update -y
DEBIAN_FRONTEND=noninteractive apt-get install -y \
	git curl wget jq ca-certificates gnupg lsb-release \
	openssh-server \
	ufw fail2ban \
	smartmontools \
	avahi-daemon \
	rsync

ok "Pacotes base instalados"

# ---------- 2. Docker ----------
if ! command -v docker >/dev/null 2>&1; then
	info "Instalando Docker (engine oficial)..."
	# Repositório oficial do Docker é mais atualizado que o do Ubuntu.
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
	ok "Docker instalado: $(docker --version)"
else
	ok "Docker já presente: $(docker --version)"
fi

# ---------- 3. Node.js (para build do frontend) ----------
if ! command -v node >/dev/null 2>&1 || [[ "$(node --version | cut -dv -f2 | cut -d. -f1)" -lt 20 ]]; then
	info "Instalando Node.js 22 LTS..."
	curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
	DEBIAN_FRONTEND=noninteractive apt-get install -y nodejs
	ok "Node.js instalado: $(node --version)"
else
	ok "Node.js já presente: $(node --version)"
fi

# ---------- 4. Clone do repositório ----------
info "Clonando PiNAS em $INSTALL_DIR..."
if [[ -d "$INSTALL_DIR/.git" ]]; then
	info "Repo já existe, fazendo pull..."
	git -C "$INSTALL_DIR" fetch origin
	git -C "$INSTALL_DIR" reset --hard "origin/$REPO_BRANCH"
else
	rm -rf "$INSTALL_DIR"
	git clone --branch "$REPO_BRANCH" --depth 1 "$REPO_URL" "$INSTALL_DIR"
fi
cd "$INSTALL_DIR"
ok "Código em $INSTALL_DIR ($(git rev-parse --short HEAD))"

# ---------- 5. .env ----------
if [[ ! -f "$INSTALL_DIR/.env" ]]; then
	info "Criando .env (gerando senha de admin aleatória)..."
	ADMIN_PASS=$(openssl rand -base64 18 | tr -d '/+=' | head -c 20)
	IP_LOCAL=$(hostname -I | awk '{print $1}')
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

	# Salva a senha numa pasta protegida pra você poder consultar depois.
	install -d -m 0700 /root/.pinas
	echo "admin: $ADMIN_PASS" > /root/.pinas/admin-password.txt
	chmod 600 /root/.pinas/admin-password.txt

	ok ".env criado. Senha do admin: ${YELLOW}$ADMIN_PASS${NC}"
	ok "Senha também salva em /root/.pinas/admin-password.txt"
else
	ok ".env já existe, mantendo"
fi

# ---------- 6. Diretórios persistentes ----------
info "Criando /srv/pinas/ ..."
install -d -m 0750 /srv/pinas
install -d -m 0750 /srv/pinas/db
install -d -m 0750 /srv/pinas/data
install -d -m 0750 /srv/pinas/data/users
install -d -m 0775 /srv/pinas/data/shared
install -d -m 0750 /srv/pinas/thumbs
install -d -m 0750 /srv/pinas/logs
install -d -m 0700 /srv/pinas/secrets
chown -R 1000:1000 /srv/pinas
ok "Diretórios em /srv/pinas/ prontos"

# ---------- 7. UFW ----------
info "Configurando firewall (UFW)..."
ufw --force reset >/dev/null
ufw default deny incoming >/dev/null
ufw default allow outgoing >/dev/null
ufw allow 22/tcp comment 'ssh/sftp' >/dev/null
ufw allow 80/tcp comment 'http (redirect)' >/dev/null
ufw allow 443/tcp comment 'pinas web' >/dev/null
ufw allow 443/udp comment 'http3' >/dev/null
ufw allow 137,138/udp comment 'samba netbios' >/dev/null
ufw allow 139,445/tcp comment 'samba' >/dev/null
ufw allow 5353/udp comment 'mdns' >/dev/null
ufw --force enable >/dev/null
ok "UFW ativo"

# ---------- 8. Avahi/mDNS ----------
systemctl enable --now avahi-daemon
ok "Avahi (mDNS .local) ativo"

# ---------- 9. Fail2Ban ----------
info "Configurando Fail2Ban..."
cat > /etc/fail2ban/jail.d/pinas.local <<EOF
[sshd]
enabled = true
maxretry = 5
bantime = 1h
EOF
systemctl enable --now fail2ban
systemctl restart fail2ban
ok "Fail2Ban ativo"

# ---------- 10. Disco USB (opcional) ----------
if [[ -n "$USB_DEVICE" ]]; then
	info "Montando disco USB $USB_DEVICE em /srv/pinas/data..."
	if [[ ! -b "$USB_DEVICE" ]]; then
		warn "$USB_DEVICE não encontrado. Pulando."
	else
		bash "$INSTALL_DIR/scripts/automount-disk.sh" "$USB_DEVICE" || warn "automount falhou"
	fi
else
	warn "USB_DEVICE não definido. Pulando automount. Para montar depois:"
	echo "    sudo $INSTALL_DIR/scripts/automount-disk.sh /dev/sdXY"
fi

# ---------- 11. Samba (opcional) ----------
if [[ -n "$SAMBA_USER" ]]; then
	info "Configurando Samba para usuário '$SAMBA_USER'..."
	apt-get install -y samba samba-common-bin
	bash "$INSTALL_DIR/scripts/samba-setup.sh" "$SAMBA_USER" || warn "samba-setup falhou"
else
	warn "SAMBA_USER não definido. Para configurar SMB depois:"
	echo "    sudo apt-get install -y samba"
	echo "    sudo $INSTALL_DIR/scripts/samba-setup.sh <usuario>"
fi

# ---------- 12. Build do frontend ----------
info "Fazendo build do frontend (pode demorar 1-3 min no Pi 4)..."
cd "$INSTALL_DIR/frontend"
if [[ ! -d node_modules ]]; then
	npm ci 2>/dev/null || npm install
fi
npm run build
ok "Frontend buildado em $INSTALL_DIR/frontend/build/"

# ---------- 13. Docker compose ----------
cd "$INSTALL_DIR"
info "Subindo o stack Docker (build da imagem do backend pode demorar 3-5 min no Pi)..."
docker compose pull caddy 2>/dev/null || true
docker compose up -d --build

# Espera o healthcheck.
info "Aguardando o backend ficar pronto..."
for i in $(seq 1 60); do
	if docker compose ps pinas-api 2>/dev/null | grep -q "healthy"; then
		ok "Backend healthy"
		break
	fi
	sleep 2
	if [[ $i -eq 60 ]]; then
		warn "Backend não ficou healthy em 2 min. Verifique: docker compose logs pinas-api"
	fi
done

# ---------- 14. systemd unit (auto-start no boot) ----------
info "Instalando unit systemd para auto-start..."
sed "s|/opt/pinas|$INSTALL_DIR|g" "$INSTALL_DIR/deploy/systemd/pinas.service" \
	> /etc/systemd/system/pinas.service
systemctl daemon-reload
systemctl enable pinas.service >/dev/null
ok "Unit pinas.service registrado e habilitado no boot"

# ---------- 15. Resumo ----------
IP_LOCAL=$(hostname -I | awk '{print $1}')
HOSTNAME_LOCAL=$(hostname).local
ADMIN_PASS=$(grep '^PINAS_ADMIN_PASSWORD=' "$INSTALL_DIR/.env" | cut -d= -f2-)

echo
echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                  PiNAS instalado com sucesso!                 ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo
echo -e "  ${BLUE}Acesso web:${NC}       https://$HOSTNAME_LOCAL"
echo -e "                     https://$IP_LOCAL"
echo
echo -e "  ${BLUE}Login admin:${NC}      admin"
echo -e "  ${BLUE}Senha:${NC}            $ADMIN_PASS"
echo -e "                     (também em /root/.pinas/admin-password.txt)"
echo
echo -e "  ${YELLOW}Aviso:${NC} o navegador vai mostrar erro de certificado na 1ª vez."
echo -e "         Clique em 'Avançado' > 'Continuar mesmo assim' (é só TLS local)."
echo
echo -e "  ${BLUE}Comandos úteis:${NC}"
echo -e "    docker compose -f $INSTALL_DIR/docker-compose.yml logs -f"
echo -e "    docker compose -f $INSTALL_DIR/docker-compose.yml ps"
echo -e "    sudo systemctl restart pinas"
echo
echo -e "  ${BLUE}Próximos passos opcionais:${NC}"
[[ -z "$USB_DEVICE" ]] && echo -e "    • Montar disco USB:  sudo $INSTALL_DIR/scripts/automount-disk.sh /dev/sdXY"
[[ -z "$SAMBA_USER" ]] && echo -e "    • Configurar SMB:    sudo $INSTALL_DIR/scripts/samba-setup.sh <usuario>"
echo -e "    • Backup automatizado:  crontab -e  →  0 3 * * * $INSTALL_DIR/scripts/backup.sh"
echo
