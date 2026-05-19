#!/usr/bin/env bash
set -euo pipefail

APP_DIR="/opt/meu-nas"
APP_USER="meunas"

if [[ "${EUID}" -ne 0 ]]; then
  echo "Execute como root: sudo bash scripts/install_phase1_raspberrypi.sh"
  exit 1
fi

apt-get update
apt-get install -y python3.11 python3.11-venv python3-pip nginx nodejs npm

id -u "${APP_USER}" >/dev/null 2>&1 || useradd --system --home "${APP_DIR}" --shell /usr/sbin/nologin "${APP_USER}"

mkdir -p "${APP_DIR}/backend" "${APP_DIR}/frontend" "${APP_DIR}/data"
chown -R "${APP_USER}:${APP_USER}" "${APP_DIR}"

echo "Copie backend/, frontend/ e scripts/ para ${APP_DIR} antes de continuar."
echo "Depois execute:"
echo "  cd ${APP_DIR}/backend && python3.11 -m venv .venv && . .venv/bin/activate && pip install -e ."
echo "  cd ${APP_DIR}/frontend && npm install && npm run build"
echo "  sudo cp ${APP_DIR}/scripts/meu-nas-api.service /etc/systemd/system/"
echo "  sudo cp ${APP_DIR}/scripts/nginx-meu-nas.conf /etc/nginx/sites-available/meu-nas"
echo "  sudo ln -sf /etc/nginx/sites-available/meu-nas /etc/nginx/sites-enabled/meu-nas"
echo "  sudo systemctl daemon-reload && sudo systemctl enable --now meu-nas-api && sudo systemctl reload nginx"
