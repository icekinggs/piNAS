#!/usr/bin/env bash
# PiNAS — samba-setup.sh
# Cria/atualiza compartilhamento SMB para /srv/pinas/data.
# Usa autenticação local via usuário do sistema (não está integrado ao banco do
# PiNAS — para multi-protocolo unificado, ver roadmap fase 4: PAM module ou LDAP local).
#
# Uso:
#   sudo ./scripts/samba-setup.sh [usuario_smb]

set -euo pipefail

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
	echo "Execute como root." >&2
	exit 1
fi

SMB_USER="${1:-pinas}"
SMB_DATA_DIR="/srv/pinas/data"

if ! id "$SMB_USER" >/dev/null 2>&1; then
	echo "==> Criando usuário do sistema '$SMB_USER' (sem shell)..."
	useradd --system --no-create-home --shell /usr/sbin/nologin "$SMB_USER"
fi

echo "==> Defina a senha SMB para '$SMB_USER':"
smbpasswd -a "$SMB_USER"
smbpasswd -e "$SMB_USER"

# Garante existência da pasta.
install -d -m 0775 "$SMB_DATA_DIR"
chown -R 1000:1000 "$SMB_DATA_DIR"
chmod -R g+rwX "$SMB_DATA_DIR"

# Backup do smb.conf na primeira execução.
[[ -f /etc/samba/smb.conf.pinas-backup ]] || cp /etc/samba/smb.conf /etc/samba/smb.conf.pinas-backup

# Remove bloco antigo idempotentemente.
sed -i '/# >>> PINAS SHARE BEGIN/,/# >>> PINAS SHARE END/d' /etc/samba/smb.conf

cat >>/etc/samba/smb.conf <<EOF
# >>> PINAS SHARE BEGIN
[pinas]
   comment = PiNAS shared storage
   path = $SMB_DATA_DIR
   browseable = yes
   read only = no
   guest ok = no
   valid users = $SMB_USER @sambashare
   create mask = 0664
   directory mask = 0775
   force user = $SMB_USER
# >>> PINAS SHARE END
EOF

testparm -s >/dev/null
systemctl restart smbd nmbd

echo
echo "==> SMB pronto."
echo "    Windows  :  \\\\<ip-do-pi>\\pinas"
echo "    macOS    :  smb://<ip-do-pi>/pinas"
echo "    Linux    :  smbclient -U $SMB_USER //<ip-do-pi>/pinas"
