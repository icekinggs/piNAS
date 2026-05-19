#!/usr/bin/env bash
set -euo pipefail

APP_DIR="/opt/meu-nas"
APP_USER="meunas"
APP_GROUP="meunas"
ENV_FILE="${APP_DIR}/backend/.env"

log() {
  printf '[meu-nas-first-boot] %s\n' "$*"
}

if [[ -f "${APP_DIR}/data/.first-boot-complete" ]]; then
  log "first boot already completed"
  exit 0
fi

log "creating system user"
if ! getent group "${APP_GROUP}" >/dev/null 2>&1; then
  groupadd --system "${APP_GROUP}"
fi
if ! id -u "${APP_USER}" >/dev/null 2>&1; then
  useradd --system --gid "${APP_GROUP}" --home "${APP_DIR}" --shell /usr/sbin/nologin "${APP_USER}"
fi

mkdir -p "${APP_DIR}/data" "${APP_DIR}/logs"
chown -R "${APP_USER}:${APP_GROUP}" "${APP_DIR}/data" "${APP_DIR}/logs"

log "generating environment"
if [[ ! -f "${ENV_FILE}" ]]; then
  SECRET_KEY="$(openssl rand -hex 32)"
  ADMIN_PASSWORD="$(openssl rand -base64 18)"
  cat > "${ENV_FILE}" <<EOF
MEUNAS_ENV=production
MEUNAS_DB_URL=sqlite+aiosqlite:////opt/meu-nas/data/meu-nas.db
MEUNAS_SECRET_KEY=${SECRET_KEY}
MEUNAS_ACCESS_TOKEN_MINUTES=15
MEUNAS_REFRESH_TOKEN_DAYS=7
MEUNAS_ADMIN_USERNAME=admin
MEUNAS_ADMIN_PASSWORD=${ADMIN_PASSWORD}
MEUNAS_CORS_ORIGINS=http://meu-nas.local,http://127.0.0.1
EOF
  chmod 0640 "${ENV_FILE}"
  chown "${APP_USER}:${APP_GROUP}" "${ENV_FILE}"
  cat > /etc/meu-nas-initial-password <<EOF
Usuario: admin
Senha inicial: ${ADMIN_PASSWORD}
Troque esta senha no primeiro acesso.
EOF
  chmod 0600 /etc/meu-nas-initial-password
fi

log "checking backend virtualenv"
if [[ ! -x "${APP_DIR}/backend/.venv/bin/uvicorn" ]]; then
  log "backend virtualenv missing"
  exit 1
fi

log "enabling services"
systemctl daemon-reload
systemctl enable --now meu-nas-api.service
systemctl enable --now nginx
systemctl enable --now avahi-daemon || true

touch "${APP_DIR}/data/.first-boot-complete"
chown "${APP_USER}:${APP_GROUP}" "${APP_DIR}/data/.first-boot-complete"
log "done"
