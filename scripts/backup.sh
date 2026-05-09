#!/usr/bin/env bash
# PiNAS — backup.sh
# Snapshot incremental via rsync com hard-links (estratégia "Apple Time Machine").
# Cada execução cria /srv/pinas/backups/snapshot-YYYYMMDD-HHMM/, reaproveitando
# blocos inalterados via --link-dest, ocupando espaço só de DIFFs.
#
# Uso:
#   sudo ./scripts/backup.sh /caminho/para/disco/externo
#
# Recomendado: rodar via cron diário/semanal.

set -euo pipefail

SRC="/srv/pinas/data"
DEST_BASE="${1:-/srv/pinas/backups}"

if [[ ! -d "$SRC" ]]; then
	echo "Origem não existe: $SRC" >&2
	exit 1
fi

install -d -m 0750 "$DEST_BASE"
NEW="$DEST_BASE/snapshot-$(date +%Y%m%d-%H%M)"
LATEST_LINK="$DEST_BASE/latest"

LINK_OPT=()
if [[ -L "$LATEST_LINK" && -d "$(readlink -f "$LATEST_LINK")" ]]; then
	LINK_OPT=(--link-dest="$(readlink -f "$LATEST_LINK")")
fi

echo "==> Backup: $SRC -> $NEW"
rsync -aH --delete --numeric-ids \
	--exclude='.cache/' \
	--exclude='*.tmp' \
	"${LINK_OPT[@]}" \
	"$SRC/" "$NEW/"

ln -sfn "$NEW" "$LATEST_LINK"

# Retenção: mantém os últimos 14 snapshots.
KEEP=14
mapfile -t old < <(ls -1d "$DEST_BASE"/snapshot-* 2>/dev/null | sort | head -n -$KEEP)
for d in "${old[@]:-}"; do
	[[ -n "$d" ]] || continue
	echo "==> Removendo snapshot antigo: $d"
	rm -rf "$d"
done

echo "==> OK. Latest -> $NEW"
du -sh "$NEW" "$DEST_BASE" 2>/dev/null || true
