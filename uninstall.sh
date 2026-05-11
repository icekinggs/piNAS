#!/usr/bin/env bash
# PiNAS — uninstall.sh
#
# Remove PiNAS de forma segura. Por padrão PRESERVA os dados do usuário
# (arquivos do NAS, banco de metadados, configs Samba) — você precisa
# passar --purge explicitamente pra apagar tudo.
#
# Uso:
#   sudo ./uninstall.sh                # remove serviços, mantém dados
#   sudo ./uninstall.sh --purge        # remove TUDO (sem volta)
#   sudo ./uninstall.sh --keep-samba   # remove PiNAS mas deixa Samba instalado
#   sudo ./uninstall.sh --dry-run      # mostra o que faria, sem executar
#
# Variáveis de ambiente respeitadas:
#   INSTALL_DIR  — diretório onde o PiNAS está (padrão: /opt/pinas)

set -euo pipefail

# ---------- helpers ----------
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
info()  { echo -e "${BLUE}==>${NC} $*"; }
ok()    { echo -e "${GREEN}[OK]${NC} $*"; }
warn()  { echo -e "${YELLOW}[!!]${NC} $*"; }
fail()  { echo -e "${RED}[ERRO]${NC} $*" >&2; exit 1; }

# ---------- args ----------
PURGE=false
KEEP_SAMBA=false
DRY_RUN=false
NON_INTERACTIVE=false

while [[ $# -gt 0 ]]; do
	case "$1" in
		--purge)        PURGE=true ;;
		--keep-samba)   KEEP_SAMBA=true ;;
		--dry-run)      DRY_RUN=true ;;
		--yes|-y)       NON_INTERACTIVE=true ;;
		-h|--help)
			sed -n '3,18p' "$0" | sed 's/^# //; s/^#//'
			exit 0
			;;
		*) fail "argumento desconhecido: $1" ;;
	esac
	shift
done

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
	fail "Execute como root: sudo $0"
fi

INSTALL_DIR="${INSTALL_DIR:-/opt/pinas}"

# Se este script está rodando de dentro do INSTALL_DIR, copia pra /tmp
# e re-executa de lá, pra não se auto-deletar no meio do trabalho.
SCRIPT_PATH="$(readlink -f "$0")"
if [[ "$SCRIPT_PATH" == "$INSTALL_DIR"/* ]]; then
	TMP_SCRIPT="/tmp/pinas-uninstall-$$.sh"
	cp "$SCRIPT_PATH" "$TMP_SCRIPT"
	chmod +x "$TMP_SCRIPT"
	# Re-executa de /tmp passando os mesmos args.
	exec "$TMP_SCRIPT" "$@"
fi

# Cleanup do tmp script ao sair (se aplicável).
trap 'rm -f /tmp/pinas-uninstall-$$.sh 2>/dev/null || true' EXIT

# Wrapper que executa OU só imprime (--dry-run).
run() {
	if [[ "$DRY_RUN" == true ]]; then
		echo "  [dry-run] $*"
	else
		eval "$@"
	fi
}

# ---------- summary ----------
echo
echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║              PiNAS — desinstalador                           ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo
echo "  Diretório de instalação:  $INSTALL_DIR"
echo "  Modo purge (apaga tudo):  $PURGE"
echo "  Manter Samba instalado:   $KEEP_SAMBA"
echo "  Dry-run (só simulação):   $DRY_RUN"
echo

if [[ "$PURGE" == true ]]; then
	echo -e "${RED}⚠  ATENÇÃO: --purge vai APAGAR PERMANENTEMENTE:${NC}"
	echo -e "${RED}    • /srv/pinas/db/        (banco de usuários, sessões, audit)${NC}"
	echo -e "${RED}    • /srv/pinas/data/      (arquivos do NAS — se for /srv/pinas/data)${NC}"
	echo -e "${RED}    • /srv/pinas/secrets/   (chave JWT)${NC}"
	echo -e "${RED}    • /srv/pinas/thumbs/    (cache de miniaturas)${NC}"
	echo -e "${RED}    • /srv/pinas/logs/      (logs)${NC}"
	echo -e "${RED}    • /srv/pinas/samba/     (estado declarativo dos shares)${NC}"
	echo -e "${RED}    • /etc/samba/smb.conf   (configuração — backup será preservado)${NC}"
	echo -e "${RED}    • /root/.pinas/         (senha de admin gerada)${NC}"
	echo
	echo -e "${YELLOW}  NÃO APAGA:${NC}"
	echo -e "${YELLOW}    • Pastas APONTADAS por shares (ex: /home/<user>) — só desmonta${NC}"
	echo -e "${YELLOW}    • Disco USB montado externo (em /etc/fstab) — só remove a entrada${NC}"
	echo
else
	echo -e "${GREEN}✓ Modo seguro: dados em /srv/pinas/ serão PRESERVADOS.${NC}"
	echo "  Pra apagar tudo, rode novamente com: sudo $0 --purge"
	echo
fi

if [[ "$NON_INTERACTIVE" != true && "$DRY_RUN" != true ]]; then
	read -p "  Continuar? (digite 'sim' pra confirmar): " confirm
	if [[ "$confirm" != "sim" ]]; then
		echo "Abortado."
		exit 0
	fi
fi

echo

# ---------- 1. parar containers ----------
info "Parando stack Docker..."
if [[ -d "$INSTALL_DIR" && -f "$INSTALL_DIR/docker-compose.yml" ]]; then
	(cd "$INSTALL_DIR" && run "docker compose down -v 2>/dev/null || true")
	# Remove imagem do backend (não é volumosa, mas limpa).
	run "docker image rm pinas/api:latest 2>/dev/null || true"
	ok "Stack parada"
else
	warn "$INSTALL_DIR não existe ou não tem docker-compose.yml — pulando"
fi

# ---------- 2. systemd units ----------
info "Removendo units systemd..."
for unit in pinas.service pinas-samba-sync.service pinas-samba-sync.path; do
	if systemctl list-unit-files "$unit" 2>/dev/null | grep -q "$unit"; then
		run "systemctl disable --now $unit 2>/dev/null || true"
		run "rm -f /etc/systemd/system/$unit"
		ok "$unit removido"
	fi
done
run "systemctl daemon-reload"

# Remove helper scripts do PATH.
if [[ -f /usr/local/bin/pinas-ports ]]; then
	run "rm -f /usr/local/bin/pinas-ports"
	ok "pinas-ports removido"
fi

# ---------- 3. limpa bloco PINAS-MANAGED do smb.conf ----------
if [[ -f /etc/samba/smb.conf ]]; then
	if grep -q "PINAS-MANAGED BEGIN" /etc/samba/smb.conf 2>/dev/null; then
		info "Removendo bloco PiNAS-managed do /etc/samba/smb.conf..."
		# Backup antes de qualquer coisa.
		run "cp /etc/samba/smb.conf /etc/samba/smb.conf.uninstall-backup-$(date +%Y%m%d-%H%M%S)"
		run "sed -i '/# >>> PINAS-MANAGED BEGIN/,/# >>> PINAS-MANAGED END/d' /etc/samba/smb.conf"
		run "systemctl reload smbd nmbd 2>/dev/null || true"
		ok "Bloco removido (backup salvo em smb.conf.uninstall-backup-*)"
	fi

	# Bloco antigo do samba-setup.sh (legado).
	if grep -q "PINAS SHARE BEGIN" /etc/samba/smb.conf 2>/dev/null; then
		run "sed -i '/# >>> PINAS SHARE BEGIN/,/# >>> PINAS SHARE END/d' /etc/samba/smb.conf"
		run "systemctl reload smbd nmbd 2>/dev/null || true"
	fi
fi

# ---------- 4. remove usuários SMB órfãos criados pelo PiNAS ----------
# Heurística: system users (UID < 1000) sem home dir = criados por nós.
if command -v pdbedit >/dev/null 2>&1; then
	info "Verificando usuários SMB criados pelo PiNAS..."
	while IFS= read -r u; do
		[[ -z "$u" ]] && continue
		uid=$(id -u "$u" 2>/dev/null || echo 99999)
		homedir=$(getent passwd "$u" | cut -d: -f6 || echo /)
		if [[ "$uid" -lt 1000 && ( "$homedir" == "/nonexistent" || ! -d "$homedir" ) ]]; then
			info "  removendo user SMB órfão: $u"
			run "smbpasswd -x $u 2>/dev/null || true"
			run "userdel $u 2>/dev/null || true"
		fi
	done < <(pdbedit -L 2>/dev/null | cut -d: -f1)
fi

# ---------- 5. UFW (mantém regras 80/443 mas limpa as 8080/8443 se foi PiNAS) ----------
info "Limpando regras UFW do PiNAS..."
if command -v ufw >/dev/null 2>&1; then
	for port in 8080 8443; do
		run "ufw delete allow $port/tcp 2>/dev/null || true"
		run "ufw delete allow $port/udp 2>/dev/null || true"
	done
fi

# ---------- 6. fstab — remove entrada se foi adicionada pelo automount-disk.sh ----------
if grep -q "/srv/pinas/data" /etc/fstab 2>/dev/null; then
	info "Removendo entrada /srv/pinas/data do /etc/fstab..."
	run "cp /etc/fstab /etc/fstab.uninstall-backup-$(date +%Y%m%d-%H%M%S)"
	run "sed -i.bak '\\|/srv/pinas/data|d' /etc/fstab"
	# Tenta desmontar (não falha se já desmontado).
	run "umount /srv/pinas/data 2>/dev/null || true"
	ok "fstab limpo (backup salvo em fstab.uninstall-backup-*)"
fi

# ---------- 7. fail2ban — remove jail customizada ----------
if [[ -f /etc/fail2ban/jail.d/pinas.local ]]; then
	info "Removendo jail Fail2Ban..."
	run "rm -f /etc/fail2ban/jail.d/pinas.local"
	run "systemctl reload fail2ban 2>/dev/null || true"
fi

# ---------- 8. código-fonte ----------
if [[ -d "$INSTALL_DIR" ]]; then
	info "Removendo $INSTALL_DIR..."
	run "rm -rf $INSTALL_DIR"
	ok "código removido"
fi

# ---------- 9. dados (só com --purge) ----------
if [[ "$PURGE" == true ]]; then
	info "Apagando dados (--purge)..."

	# /srv/pinas — TUDO incluindo o data se for /srv/pinas/data.
	# Se o data está em outro lugar (/home/user, /mnt/usb), NÃO apaga lá.
	if [[ -d /srv/pinas ]]; then
		run "rm -rf /srv/pinas"
		ok "/srv/pinas/ apagado"
	fi

	# Senha do admin gerada.
	if [[ -d /root/.pinas ]]; then
		run "rm -rf /root/.pinas"
	fi

	# Volumes nomeados do Docker (caddy_data, caddy_config).
	run "docker volume rm pinas_caddy_data pinas_caddy_config 2>/dev/null || true"
fi

# ---------- 10. Samba (só remove se --purge OU se não foi pedido pra manter) ----------
if [[ "$PURGE" == true && "$KEEP_SAMBA" != true ]]; then
	info "Removendo Samba do sistema..."
	run "systemctl stop smbd nmbd 2>/dev/null || true"
	run "DEBIAN_FRONTEND=noninteractive apt-get remove -y samba samba-common-bin 2>/dev/null || true"
	# Banco de senhas SMB.
	run "rm -rf /var/lib/samba/private/passdb.tdb 2>/dev/null || true"
	ok "Samba removido"

	# Restaura smb.conf original se houver backup.
	if [[ -f /etc/samba/smb.conf.pinas-original ]]; then
		run "cp /etc/samba/smb.conf.pinas-original /etc/samba/smb.conf"
		ok "smb.conf restaurado do backup"
	fi
fi

# ---------- summary final ----------
echo
echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║              Desinstalação concluída                         ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
echo

if [[ "$PURGE" == false ]]; then
	echo "  Foi REMOVIDO:"
	echo "    • Containers Docker e imagens"
	echo "    • Units systemd (pinas, pinas-samba-sync.*)"
	echo "    • Bloco PINAS-MANAGED do /etc/samba/smb.conf"
	echo "    • $INSTALL_DIR/"
	echo "    • Regras UFW de portas alternativas (8080/8443)"
	echo
	echo "  Foi PRESERVADO:"
	echo "    • /srv/pinas/db/        (banco de usuários do PiNAS)"
	echo "    • /srv/pinas/data/      (arquivos do NAS — se aplicável)"
	echo "    • /srv/pinas/secrets/   (jwt.key)"
	echo "    • /srv/pinas/samba/     (estado dos shares)"
	echo "    • Samba instalado (mas sem shares do PiNAS)"
	echo
	echo "  Se quiser apagar TUDO agora:"
	echo "    sudo $0 --purge"
else
	echo "  TUDO foi removido."
	if [[ "$KEEP_SAMBA" != true ]]; then
		echo "  Samba também foi desinstalado."
	fi
fi

echo
echo "  Reinstalar:"
echo "    git clone https://github.com/icekinggs/piNAS.git"
echo "    cd piNAS && sudo ./bootstrap.sh"
echo
