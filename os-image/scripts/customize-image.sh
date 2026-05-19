#!/usr/bin/env bash
set -euo pipefail

if [[ "${EUID}" -ne 0 ]]; then
  echo "Execute com sudo."
  exit 1
fi

if [[ $# -ne 3 ]]; then
  echo "Uso: $0 BASE_IMAGE.img.xz APP_BUNDLE.tar.gz OUTPUT_IMAGE.img"
  exit 1
fi

BASE_IMAGE="$1"
APP_BUNDLE="$2"
OUTPUT_IMAGE="$3"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WORK_DIR="$(mktemp -d)"
MOUNT_BOOT="${WORK_DIR}/boot"
MOUNT_ROOT="${WORK_DIR}/root"
QEMU_BIN="$(command -v qemu-aarch64-static || true)"

cleanup() {
  set +e
  mountpoint -q "${MOUNT_ROOT}/dev/pts" && umount "${MOUNT_ROOT}/dev/pts"
  mountpoint -q "${MOUNT_ROOT}/dev" && umount "${MOUNT_ROOT}/dev"
  mountpoint -q "${MOUNT_ROOT}/proc" && umount "${MOUNT_ROOT}/proc"
  mountpoint -q "${MOUNT_ROOT}/sys" && umount "${MOUNT_ROOT}/sys"
  mountpoint -q "${MOUNT_BOOT}" && umount "${MOUNT_BOOT}"
  mountpoint -q "${MOUNT_ROOT}" && umount "${MOUNT_ROOT}"
  [[ -n "${LOOP_DEVICE:-}" ]] && losetup -d "${LOOP_DEVICE}"
  rm -rf "${WORK_DIR}"
}
trap cleanup EXIT

mkdir -p "$(dirname "${OUTPUT_IMAGE}")" "${MOUNT_BOOT}" "${MOUNT_ROOT}"

if [[ -z "${QEMU_BIN}" ]]; then
  echo "qemu-aarch64-static não encontrado. Instale qemu-user-static."
  exit 1
fi

echo "Preparing image"
if [[ "${BASE_IMAGE}" == *.xz ]]; then
  xz -dc "${BASE_IMAGE}" > "${OUTPUT_IMAGE}"
else
  cp "${BASE_IMAGE}" "${OUTPUT_IMAGE}"
fi

LOOP_DEVICE="$(losetup --find --partscan --show "${OUTPUT_IMAGE}")"

BOOT_PART="${LOOP_DEVICE}p1"
ROOT_PART="${LOOP_DEVICE}p2"
if [[ ! -b "${BOOT_PART}" ]]; then
  BOOT_PART="${LOOP_DEVICE}"
fi

mount "${ROOT_PART}" "${MOUNT_ROOT}"
mount "${BOOT_PART}" "${MOUNT_BOOT}"

run_chroot() {
  chroot "${MOUNT_ROOT}" /usr/bin/env bash -lc "$*"
}

echo "Injecting cloud-init"
cp "${ROOT_DIR}/os-image/cloud-init/user-data" "${MOUNT_BOOT}/user-data"
cp "${ROOT_DIR}/os-image/cloud-init/meta-data" "${MOUNT_BOOT}/meta-data"
cp "${ROOT_DIR}/os-image/cloud-init/network-config" "${MOUNT_BOOT}/network-config"

echo "Installing app bundle"
mkdir -p "${MOUNT_ROOT}/opt"
tar -xzf "${APP_BUNDLE}" -C "${WORK_DIR}"
rm -rf "${MOUNT_ROOT}/opt/meu-nas"
mkdir -p "${MOUNT_ROOT}/opt/meu-nas"
rsync -a "${WORK_DIR}/meu-nas-app/backend/" "${MOUNT_ROOT}/opt/meu-nas/backend/"
rsync -a "${WORK_DIR}/meu-nas-app/frontend/" "${MOUNT_ROOT}/opt/meu-nas/frontend/"
mkdir -p "${MOUNT_ROOT}/opt/meu-nas/data"

echo "Preparing ARM64 chroot"
cp "${QEMU_BIN}" "${MOUNT_ROOT}/usr/bin/qemu-aarch64-static"
cp /etc/resolv.conf "${MOUNT_ROOT}/etc/resolv.conf"
mount --bind /dev "${MOUNT_ROOT}/dev"
mount --bind /dev/pts "${MOUNT_ROOT}/dev/pts"
mount -t proc proc "${MOUNT_ROOT}/proc"
mount -t sysfs sys "${MOUNT_ROOT}/sys"

echo "Installing apt packages inside image"
cp "${ROOT_DIR}/os-image/manifests/packages.apt" "${MOUNT_ROOT}/tmp/meu-nas-packages.apt"
run_chroot "apt-get update"
run_chroot "DEBIAN_FRONTEND=noninteractive xargs -a /tmp/meu-nas-packages.apt apt-get install -y --no-install-recommends"
run_chroot "rm -rf /var/lib/apt/lists/* /tmp/meu-nas-packages.apt"

echo "Creating app user and Python virtualenv"
run_chroot "getent group meunas >/dev/null || groupadd --system meunas"
run_chroot "id -u meunas >/dev/null 2>&1 || useradd --system --gid meunas --home /opt/meu-nas --shell /usr/sbin/nologin meunas"
run_chroot "python3 -m venv /opt/meu-nas/backend/.venv"
run_chroot "/opt/meu-nas/backend/.venv/bin/pip install --upgrade pip"
run_chroot "/opt/meu-nas/backend/.venv/bin/pip install /opt/meu-nas/backend"
run_chroot "chown -R meunas:meunas /opt/meu-nas"

echo "Installing services"
cp "${WORK_DIR}/meu-nas-app/scripts/meu-nas-api.service" \
  "${MOUNT_ROOT}/etc/systemd/system/meu-nas-api.service"
cp "${WORK_DIR}/meu-nas-app/scripts/nginx-meu-nas.conf" \
  "${MOUNT_ROOT}/etc/nginx/sites-available/meu-nas"
ln -sf /etc/nginx/sites-available/meu-nas "${MOUNT_ROOT}/etc/nginx/sites-enabled/meu-nas"
cp "${WORK_DIR}/meu-nas-app/first-boot/meu-nas-first-boot.service" \
  "${MOUNT_ROOT}/etc/systemd/system/meu-nas-first-boot.service"
cp "${WORK_DIR}/meu-nas-app/first-boot/meu-nas-first-boot.sh" \
  "${MOUNT_ROOT}/usr/local/sbin/meu-nas-first-boot.sh"
chmod 0755 "${MOUNT_ROOT}/usr/local/sbin/meu-nas-first-boot.sh"

echo "Enabling first boot service"
mkdir -p "${MOUNT_ROOT}/etc/systemd/system/multi-user.target.wants"
ln -sf /etc/systemd/system/meu-nas-first-boot.service \
  "${MOUNT_ROOT}/etc/systemd/system/multi-user.target.wants/meu-nas-first-boot.service"

sync
echo "Image ready: ${OUTPUT_IMAGE}"
