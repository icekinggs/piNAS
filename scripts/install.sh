#!/usr/bin/env bash
# PiNAS — install.sh
# Instala dependências do host: samba, sshd (já vem em Ubuntu Server), ufw, fail2ban,
# smartmontools. Cria diretórios e ajusta permissões.
#
# Uso:
#   sudo ./scripts/install.sh

set -euo pipefail

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
	echo "Execute como root (sudo)." >&2
	exit 1
fi

if ! grep -q "Ubuntu" /etc/os-release; then
	echo "Aviso: este script foi testado em Ubuntu Server 24.04 LTS." >&2
fi

echo "==> Atualizando apt..."
apt-get update -y
apt-get upgrade -y

echo "==> Instalando pacotes do host..."
apt-get install -y \
	samba samba-common-bin \
	openssh-server \
	ufw fail2ban \
	smartmontools \
	avahi-daemon \
	curl wget jq \
	rsync \
	docker.io docker-compose-v2

echo "==> Habilitando serviços..."
systemctl enable --now smbd nmbd
systemctl enable --now ssh
systemctl enable --now fail2ban
systemctl enable --now avahi-daemon
systemctl enable --now docker

echo "==> Criando diretórios em /srv/pinas..."
install -d -m 0750 -o root -g root /srv/pinas
install -d -m 0750 /srv/pinas/db
install -d -m 0750 /srv/pinas/data
install -d -m 0750 /srv/pinas/data/users
install -d -m 0775 /srv/pinas/data/shared
install -d -m 0750 /srv/pinas/thumbs
install -d -m 0750 /srv/pinas/logs
install -d -m 0700 /srv/pinas/secrets

# UID/GID 1000 — uid padrão do container do PiNAS.
chown -R 1000:1000 /srv/pinas

echo "==> Configurando UFW..."
ufw --force reset
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp comment 'ssh/sftp'
ufw allow 80/tcp comment 'http (redirect)'
ufw allow 443/tcp comment 'pinas web'
ufw allow 443/udp comment 'http3'
ufw allow 137,138/udp comment 'samba netbios'
ufw allow 139,445/tcp comment 'samba'
ufw allow 5353/udp comment 'mdns/avahi'
ufw --force enable

echo "==> Fail2Ban: habilitando jails padrão..."
cat >/etc/fail2ban/jail.d/pinas.local <<EOF
[sshd]
enabled = true
maxretry = 5
bantime = 1h

[pinas-api]
enabled = false
# placeholder — configurar quando o filter PiNAS estiver pronto.
# logpath = /srv/pinas/logs/audit.log
EOF
systemctl restart fail2ban

echo
echo "==> OK. Próximos passos:"
echo "    1. ./scripts/automount-disk.sh /dev/sdXY  (se você tem disco USB)"
echo "    2. ./scripts/samba-setup.sh               (compartilhamento SMB)"
echo "    3. cp .env.example .env  &&  edite .env"
echo "    4. docker compose up -d --build"
echo
echo "Acesse depois: https://pinas.local  (ou https://<ip-do-pi>)"
