#!/usr/bin/env bash
# PiNAS — automount-disk.sh
# Monta um disco USB (idealmente formatado em ext4) em /srv/pinas/data,
# adicionando uma entrada idempotente no /etc/fstab via UUID.
#
# Uso:
#   sudo ./scripts/automount-disk.sh /dev/sda1

set -euo pipefail

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
	echo "Execute como root (sudo)." >&2
	exit 1
fi

DEV="${1:-}"
if [[ -z "$DEV" ]]; then
	echo "Uso: $0 /dev/sdXY" >&2
	echo
	echo "Discos disponíveis:" >&2
	lsblk -o NAME,SIZE,FSTYPE,UUID,MOUNTPOINT >&2
	exit 2
fi

if [[ ! -b "$DEV" ]]; then
	echo "Dispositivo não encontrado: $DEV" >&2
	exit 2
fi

UUID=$(blkid -s UUID -o value "$DEV" || true)
FSTYPE=$(blkid -s TYPE -o value "$DEV" || true)

if [[ -z "$UUID" ]]; then
	echo "Não consegui obter UUID. O disco está formatado?" >&2
	echo "Para formatar como ext4: mkfs.ext4 -L pinas-data $DEV" >&2
	exit 3
fi

if [[ "$FSTYPE" != "ext4" ]]; then
	echo "AVISO: filesystem é '$FSTYPE'. Recomendado ext4 para o disco do NAS." >&2
	echo "Continue? (s/N)"
	read -r ans
	[[ "$ans" =~ ^[sS]$ ]] || exit 0
fi

MNT="/srv/pinas/data"
install -d -m 0750 "$MNT"

LINE="UUID=$UUID  $MNT  $FSTYPE  defaults,nofail,noatime,x-systemd.device-timeout=20s  0  2"

# Remove entrada antiga referente ao mesmo mountpoint, se existir.
sed -i.bak "\|[[:space:]]$MNT[[:space:]]|d" /etc/fstab

echo "$LINE" >> /etc/fstab

echo "==> /etc/fstab atualizado:"
echo "    $LINE"

echo "==> Tentando montar..."
mount -a

if mountpoint -q "$MNT"; then
	df -h "$MNT"
	# Cria estrutura mínima.
	install -d -m 0750 -o 1000 -g 1000 "$MNT/users"
	install -d -m 0775 -o 1000 -g 1000 "$MNT/shared"
	echo "==> OK, disco montado em $MNT."
else
	echo "Falha ao montar. Verifique 'dmesg' e 'systemctl status'." >&2
	exit 4
fi
