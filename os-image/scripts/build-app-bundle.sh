#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
STAGING_DIR="${DIST_DIR}/meu-nas-app"

rm -rf "${STAGING_DIR}"
mkdir -p "${STAGING_DIR}/backend" "${STAGING_DIR}/frontend" "${STAGING_DIR}/scripts"

rsync -a \
  --exclude '.venv' \
  --exclude '__pycache__' \
  --exclude '.pytest_cache' \
  "${ROOT_DIR}/backend/" "${STAGING_DIR}/backend/"

pushd "${ROOT_DIR}/frontend" >/dev/null
npm install
npm run build
popd >/dev/null

rsync -a "${ROOT_DIR}/frontend/dist/" "${STAGING_DIR}/frontend/dist/"
rsync -a "${ROOT_DIR}/scripts/meu-nas-api.service" "${STAGING_DIR}/scripts/"
rsync -a "${ROOT_DIR}/scripts/nginx-meu-nas.conf" "${STAGING_DIR}/scripts/"
rsync -a "${ROOT_DIR}/os-image/first-boot/" "${STAGING_DIR}/first-boot/"

tar -C "${DIST_DIR}" -czf "${DIST_DIR}/meu-nas-app.tar.gz" "meu-nas-app"
echo "${DIST_DIR}/meu-nas-app.tar.gz"
