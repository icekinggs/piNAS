#!/usr/bin/env bash
# PiNAS — samba-sync.sh
#
# Lê /srv/pinas/samba/desired-state.json e aplica em:
#   - /etc/samba/smb.conf      (shares)
#   - smbpasswd                (usuários)
#
# É IDEMPOTENTE: reaplica do zero a cada execução.
# É SEGURO: faz backup do smb.conf antes; preserva blocos não-PiNAS.
#
# Disparado automaticamente quando o desired-state.json muda
# (via systemd .path unit pinas-samba-sync.path).
#
# Pode ser rodado manualmente também:
#   sudo /opt/pinas/scripts/samba-sync.sh
#
# Senhas dos usuários chegam em formato "PENDING:senha-em-claro" no JSON.
# Após processar, o script reescreve o JSON sem o PENDING.

set -euo pipefail

STATE_FILE="${PINAS_SAMBA_STATE:-/srv/pinas/samba/desired-state.json}"
SMB_CONF="/etc/samba/smb.conf"
PINAS_BLOCK_BEGIN="# >>> PINAS-MANAGED BEGIN <<<"
PINAS_BLOCK_END="# >>> PINAS-MANAGED END <<<"
LOG_TAG="pinas-samba-sync"

log() { logger -t "$LOG_TAG" "$*"; echo "[$(date -Iseconds)] $*"; }
err() { log "ERROR: $*" >&2; }

# ---------- pré-checagens ----------

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
	err "precisa rodar como root"
	exit 1
fi

if [[ ! -f "$STATE_FILE" ]]; then
	log "desired-state.json não existe ainda em $STATE_FILE — nada a fazer"
	exit 0
fi

# Garante deps.
for cmd in jq smbpasswd testparm pdbedit useradd usermod; do
	if ! command -v "$cmd" >/dev/null 2>&1; then
		err "comando ausente: $cmd. Instale samba e jq."
		exit 1
	fi
done

# ---------- 1. parsing do desired-state ----------

log "lendo $STATE_FILE"

# Valida JSON.
if ! jq empty "$STATE_FILE" 2>/dev/null; then
	err "JSON inválido em $STATE_FILE"
	exit 2
fi

# ---------- 2. processa usuários ----------

# Lista usuários atuais no smbpasswd.
existing_smb_users=$(pdbedit -L 2>/dev/null | cut -d: -f1 | sort -u || true)

# Lista usuários no desejado.
desired_users=$(jq -r '.users[]?.username' "$STATE_FILE" | sort -u)

log "usuários desejados: $(echo "$desired_users" | tr '\n' ' ')"

# Para cada usuário desejado, garante que existe (Linux + Samba) e atualiza senha se PENDING.
state_changed=false
TMP_STATE=$(mktemp)
cp "$STATE_FILE" "$TMP_STATE"

while IFS= read -r username; do
	[[ -z "$username" ]] && continue

	# Garante user Linux.
	if ! id "$username" >/dev/null 2>&1; then
		log "criando usuário Linux: $username (system, sem shell)"
		useradd --system --no-create-home --shell /usr/sbin/nologin "$username" || {
			err "falha ao criar user Linux $username"
			continue
		}
	fi

	# Pega o NTHash do estado.
	nt_hash=$(jq -r --arg u "$username" '.users[] | select(.username==$u) | .nt_hash // ""' "$STATE_FILE")
	disabled=$(jq -r --arg u "$username" '.users[] | select(.username==$u) | .disabled // false' "$STATE_FILE")

	# Se hash começa com "PENDING:", é uma senha em texto plano nova/atualização.
	if [[ "$nt_hash" == PENDING:* ]]; then
		password="${nt_hash#PENDING:}"
		log "atualizando senha SMB para: $username"

		# smbpasswd -s lê senha do stdin.
		# -a adiciona, -e habilita.
		if echo "$existing_smb_users" | grep -qx "$username"; then
			# Já existe, troca senha.
			(echo "$password"; echo "$password") | smbpasswd -s "$username" >/dev/null
		else
			# Adiciona novo.
			(echo "$password"; echo "$password") | smbpasswd -s -a "$username" >/dev/null
		fi
		smbpasswd -e "$username" >/dev/null

		# Marca o estado pra reescrita: limpa o PENDING.
		# Não guardamos o NT hash real no JSON (não é necessário — Samba mantém ele em /var/lib/samba/private/).
		# Apenas remove o campo nt_hash.
		jq --arg u "$username" '
			.users |= map(
				if .username == $u then .nt_hash = "" else . end
			)
		' "$TMP_STATE" > "$TMP_STATE.new" && mv "$TMP_STATE.new" "$TMP_STATE"
		state_changed=true
	fi

	# Aplica disabled/enabled.
	if [[ "$disabled" == "true" ]]; then
		smbpasswd -d "$username" >/dev/null 2>&1 || true
	else
		smbpasswd -e "$username" >/dev/null 2>&1 || true
	fi
done <<< "$desired_users"

# Remove usuários que existiam no smbpasswd mas não estão mais no desired.
# IMPORTANTE: só remove os que foram criados pelo PiNAS (system users sem home).
# Não toca em usuários "humanos" que existiam antes.
while IFS= read -r username; do
	[[ -z "$username" ]] && continue
	if ! echo "$desired_users" | grep -qx "$username"; then
		# Existe no SMB mas não no desired. Confere se foi criado pelo PiNAS.
		# Heurística: system user (UID < 1000) E sem home dir = criado por nós.
		uid=$(id -u "$username" 2>/dev/null || echo 99999)
		homedir=$(getent passwd "$username" | cut -d: -f6)

		if [[ "$uid" -lt 1000 && ( "$homedir" == "/nonexistent" || "$homedir" == "/" || ! -d "$homedir" ) ]]; then
			log "removendo usuário SMB órfão: $username"
			smbpasswd -x "$username" >/dev/null 2>&1 || true
			# Remove user Linux também (porque foi criado por nós).
			userdel "$username" 2>/dev/null || true
		else
			log "deixando usuário SMB intacto (não foi criado pelo PiNAS): $username"
		fi
	fi
done <<< "$existing_smb_users"

# ---------- 3. processa shares -> smb.conf ----------

log "regenerando bloco PiNAS no smb.conf"

# Backup smb.conf antes da primeira execução.
if [[ ! -f "${SMB_CONF}.pinas-original" ]]; then
	cp "$SMB_CONF" "${SMB_CONF}.pinas-original"
	log "backup criado: ${SMB_CONF}.pinas-original"
fi

# Remove bloco PiNAS antigo (se existir).
if grep -q "$PINAS_BLOCK_BEGIN" "$SMB_CONF"; then
	sed -i "/$PINAS_BLOCK_BEGIN/,/$PINAS_BLOCK_END/d" "$SMB_CONF"
fi

# Constrói o bloco novo.
TMP_BLOCK=$(mktemp)
{
	echo ""
	echo "$PINAS_BLOCK_BEGIN"
	echo "# Gerenciado automaticamente pelo PiNAS. NÃO EDITE MANUALMENTE."
	echo "# Para alterar, use o painel web em https://<seu-pi>:8443/samba"
	echo "# Estado: $STATE_FILE"
	echo "# Última sync: $(date -Iseconds)"
	echo ""

	# Itera os shares do JSON.
	jq -c '.shares[]?' "$STATE_FILE" | while IFS= read -r share; do
		name=$(echo "$share" | jq -r '.name')
		path=$(echo "$share" | jq -r '.path')
		comment=$(echo "$share" | jq -r '.comment // ""')
		browseable=$(echo "$share" | jq -r '.browseable // true')
		readonly=$(echo "$share" | jq -r '.read_only // false')
		guestok=$(echo "$share" | jq -r '.guest_ok // false')
		valid_users=$(echo "$share" | jq -r '.valid_users // [] | join(" ")')
		create_mask=$(echo "$share" | jq -r '.create_mask // "0664"')
		dir_mask=$(echo "$share" | jq -r '.directory_mask // "0775"')
		force_user=$(echo "$share" | jq -r '.force_user // ""')

		# Garante que o path existe e é diretório.
		if [[ ! -d "$path" ]]; then
			log "WARN: path do share '$name' não existe: $path (criando)"
			mkdir -p "$path" 2>/dev/null || {
				err "não conseguiu criar $path — pulando share $name"
				continue
			}
		fi

		echo "[$name]"
		[[ -n "$comment" ]] && echo "   comment = $comment"
		echo "   path = $path"
		echo "   browseable = $([[ "$browseable" == "true" ]] && echo yes || echo no)"
		echo "   read only = $([[ "$readonly" == "true" ]] && echo yes || echo no)"
		echo "   guest ok = $([[ "$guestok" == "true" ]] && echo yes || echo no)"
		[[ -n "$valid_users" ]] && echo "   valid users = $valid_users"
		echo "   create mask = $create_mask"
		echo "   directory mask = $dir_mask"
		[[ -n "$force_user" ]] && echo "   force user = $force_user"
		echo ""
	done

	echo "$PINAS_BLOCK_END"
} > "$TMP_BLOCK"

# Anexa ao smb.conf.
cat "$TMP_BLOCK" >> "$SMB_CONF"
rm -f "$TMP_BLOCK"

# ---------- 4. valida com testparm ----------

log "validando smb.conf com testparm..."
if ! testparm -s "$SMB_CONF" >/dev/null 2>&1; then
	err "testparm reportou erro no smb.conf!"
	err "veja: testparm -s"
	# Restaura backup pra não deixar Samba quebrado.
	if [[ -f "${SMB_CONF}.pinas-original" ]]; then
		err "restaurando ${SMB_CONF}.pinas-original"
		cp "${SMB_CONF}.pinas-original" "$SMB_CONF"
		systemctl reload smbd nmbd 2>/dev/null || true
	fi
	exit 3
fi

# ---------- 5. recarrega Samba ----------

log "recarregando smbd + nmbd..."
systemctl reload smbd nmbd 2>/dev/null || systemctl restart smbd nmbd

# ---------- 6. reescreve o estado limpo ----------

if [[ "$state_changed" == "true" ]]; then
	# Mantém ownership/perms do arquivo original.
	chown --reference="$STATE_FILE" "$TMP_STATE"
	chmod --reference="$STATE_FILE" "$TMP_STATE"
	mv "$TMP_STATE" "$STATE_FILE"
	log "estado limpo (PENDING removido)"
else
	rm -f "$TMP_STATE"
fi

# ---------- 7. gera arquivo de status ----------

# Útil pra UI mostrar quando foi a última sync e se deu certo.
STATUS_FILE="$(dirname "$STATE_FILE")/sync-status.json"
cat > "$STATUS_FILE" <<EOF
{
  "last_sync": "$(date -Iseconds)",
  "ok": true,
  "shares_count": $(jq '.shares | length' "$STATE_FILE"),
  "users_count": $(jq '.users | length' "$STATE_FILE")
}
EOF
chown --reference="$STATE_FILE" "$STATUS_FILE" 2>/dev/null || true
chmod 0644 "$STATUS_FILE"

log "sync OK"
exit 0
