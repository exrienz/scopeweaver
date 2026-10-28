#!/usr/bin/env bash
# Reset the administrator password stored as bcrypt in settings/auth.password_hash.
# Local connection precedence: CLI fields, --dsn/ARTEX_PG_DSN, config.json.
# Docker mode runs psql inside the PostgreSQL container (no published port needed).
# The new password is passed through the environment and psql \getenv, not argv.
# psql :'newpw' quotes the value safely; pgcrypto generates a compatible bcrypt hash.
set -euo pipefail
# Interface language only; keep command tokens and operator input unchanged.
normalize_language() (
  set -f
  tag=$(printf '%s' "$1" | tr '[:upper:]_' '[:lower:]-' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
  case "$tag" in
    ko|ko-*) printf ko; return ;;
    en|en-*) printf en; return ;;
    ''|*-|*--*|*' '*|*';'*) printf en; return ;;
    ja|ja-*) printf ja; return ;;
    es|es-*) printf es; return ;;
    zh|zh-cn|zh-sg) printf zh-Hans; return ;;
    zh-tw|zh-hk|zh-mo) printf zh-Hant; return ;;
    zh-*)
      script=''
      IFS=-
      set -- $tag
      shift
      for part do
        case "$part" in
          ????) [ -z "$script" ] || [ "$script" = "$part" ] || { printf en; return; }; script="$part" ;;
        esac
      done
      case "$script" in hans) printf zh-Hans ;; hant) printf zh-Hant ;; *) printf en ;; esac
      ;;
    *) printf en ;;
  esac
)
LANG_SEL=$(normalize_language "${SCOPEWEAVER_LANGUAGE:-${ARTEX_LANGUAGE:-en}}")
# msg EN KO HANS HANT JA ES; no runtime catalog or extra packaging dependency.
msg() {
  case "$LANG_SEL" in
    ko) printf '%s' "$2" ;;
    zh-Hans) printf '%s' "$3" ;;
    zh-Hant) printf '%s' "$4" ;;
    ja) printf '%s' "$5" ;;
    es) printf '%s' "$6" ;;
    *) printf '%s' "$1" ;;
  esac
}


PASS_KEY="auth.password_hash"
BCRYPT_COST=10

MODE=""            # local | docker (empty = automatic)
DSN=""
HOST="" PORT="" USER="" DBPASS="" DBNAME="" SSLMODE=""
CONFIG=""
CONTAINER=""       # PostgreSQL service/container name (default postgres)
EXEC_KIND=""       # compose | docker (empty = automatic)
NEWPASS=""
ASSUME_YES=0

die() { echo "$(msg "Error: $*" "오류: $*" "错误：$*" "錯誤：$*" "エラー: $*" "Error: $*")" >&2; exit 1; }
info() { echo "· $*" >&2; }

usage() {
  echo "$(msg 'Reset ScopeWeaver administrator password (legacy username ARTEX).' 'ScopeWeaver 관리자 비밀번호 재설정 (기존 사용자 이름 ARTEX).' "重置 ScopeWeaver 管理员密码（旧用户名为 ARTEX）。" "重設 ScopeWeaver 管理員密碼（原使用者名稱為 ARTEX）。" "ScopeWeaver の管理者パスワードをリセットします（従来のユーザー名は ARTEX）。" "Restablece la contraseña de administrador de ScopeWeaver (nombre de usuario heredado: ARTEX).")"
  echo "$(msg 'Usage: ./reset-password.sh [options]; prompts for a password when omitted.' '사용법: ./reset-password.sh [옵션]; 비밀번호 생략 시 입력을 요청합니다.' "用法：./reset-password.sh [options]；省略密码时将提示输入。" "用法：./reset-password.sh [options]；未提供密碼時會提示輸入。" "使用法: ./reset-password.sh [options]。パスワードを省略すると入力を求めます。" "Uso: ./reset-password.sh [options]; se pedirá la contraseña si se omite.")"
  echo '-m/--mode local|docker; --dsn DSN; --config PATH; --sslmode MODE'
  echo '-H/--host HOST; -P/--port PORT; -U/--user USER; -W/--db-password PASSWORD'
  echo '-d/--dbname DATABASE; -c/--container NAME; --exec compose|docker'
  echo '-p/--new-password PASSWORD; -y/--yes; -h/--help'
  exit 0
}

# Parse arguments.
while [[ $# -gt 0 ]]; do
  case "$1" in
    -m|--mode)        MODE="${2:-}"; shift 2 ;;
    --dsn)            DSN="${2:-}"; shift 2 ;;
    -H|--host)        HOST="${2:-}"; shift 2 ;;
    -P|--port)        PORT="${2:-}"; shift 2 ;;
    -U|--user)        USER="${2:-}"; shift 2 ;;
    -W|--db-password) DBPASS="${2:-}"; shift 2 ;;
    -d|--dbname)      DBNAME="${2:-}"; shift 2 ;;
    --sslmode)        SSLMODE="${2:-}"; shift 2 ;;
    --config)         CONFIG="${2:-}"; shift 2 ;;
    -c|--container)   CONTAINER="${2:-}"; shift 2 ;;
    --exec)           EXEC_KIND="${2:-}"; shift 2 ;;
    -p|--new-password) NEWPASS="${2:-}"; shift 2 ;;
    -y|--yes)         ASSUME_YES=1; shift ;;
    -h|--help)        usage ;;
    *) die "$(msg "Unknown argument: $1 (see -h)" "알 수 없는 인수: $1 (-h 참고)" "未知参数：$1（参见 -h）" "未知的引數：$1（請參閱 -h）" "不明な引数: $1（-h を参照）" "Argumento desconocido: $1 (consulta -h)")" ;;
  esac
done

# Read database fields from config.json in local mode when not explicitly supplied.
# Prefer Python JSON parsing; the fallback supports simple string/number fields.
read_config_json() {
  local path="$1"
  [[ -f "$path" ]] || return 1
  if command -v python3 >/dev/null 2>&1; then
    python3 - "$path" <<'PY'
import json, sys
try:
    d = json.load(open(sys.argv[1])).get("database", {})
except Exception:
    sys.exit(1)
# Accept either a DSN or individual connection fields.
if d.get("dsn"):
    print("DSN\t" + d["dsn"]); sys.exit(0)
for k in ("host","port","user","password","dbname","sslmode"):
    if d.get(k) is not None:
        print(k.upper() + "\t" + str(d[k]))
PY
  else
    # Minimal fallback: extract each string or numeric field.
    local k
    for k in host port user password dbname sslmode; do
      local v
      v=$(grep -oE "\"$k\"[[:space:]]*:[[:space:]]*(\"[^\"]*\"|[0-9]+)" "$path" 2>/dev/null \
            | head -1 | sed -E "s/.*:[[:space:]]*//; s/^\"//; s/\"$//") || true
      [[ -n "$v" ]] && echo -e "${k^^}\t$v"
    done
  fi
}

apply_config_fields() {
  local line key val
  while IFS=$'\t' read -r key val; do
    [[ -z "$key" ]] && continue
    case "$key" in
      DSN)      [[ -z "$DSN" ]] && DSN="$val" ;;
      HOST)     [[ -z "$HOST" ]] && HOST="$val" ;;
      PORT)     [[ -z "$PORT" ]] && PORT="$val" ;;
      USER)     [[ -z "$USER" ]] && USER="$val" ;;
      PASSWORD) [[ -z "$DBPASS" ]] && DBPASS="$val" ;;
      DBNAME)   [[ -z "$DBNAME" ]] && DBNAME="$val" ;;
      SSLMODE)  [[ -z "$SSLMODE" ]] && SSLMODE="$val" ;;
    esac
  done
}

# Detect deployment mode.
if [[ -z "$MODE" ]]; then
  if [[ -n "$DSN$HOST$USER$DBNAME" || -n "${ARTEX_PG_DSN:-}" || -f "${CONFIG:-config.json}" ]]; then
    MODE="local"
  elif command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yml ]]; then
    MODE="docker"
  else
    MODE="local"
  fi
fi
info "$(msg "Deployment mode: $MODE" "배포 방식: $MODE" "部署方式：${MODE}" "部署方式：${MODE}" "デプロイ方式: ${MODE}" "Modo de despliegue: ${MODE}")"

# Collect and confirm the new password.
if [[ -z "$NEWPASS" ]]; then
  read -r -s -p "$(msg "New password (legacy username ARTEX): " "새 비밀번호 (기존 사용자 이름 ARTEX): " "新密码（旧用户名为 ARTEX）： " "新密碼（原使用者名稱為 ARTEX）： " "新しいパスワード（従来のユーザー名は ARTEX）: " "Nueva contraseña (nombre de usuario heredado: ARTEX): ")" NEWPASS; echo >&2
  [[ -n "$NEWPASS" ]] || die "$(msg "Password cannot be empty" "비밀번호는 비어 있을 수 없습니다" "密码不能为空" "密碼不可為空" "パスワードは空にできません" "La contraseña no puede estar vacía")"
  read -r -s -p "$(msg "Confirm password: " "비밀번호 확인: " "确认密码： " "確認密碼： " "パスワードの確認: " "Confirma la contraseña: ")" NEWPASS2; echo >&2
  [[ "$NEWPASS" == "$NEWPASS2" ]] || die "$(msg "Passwords do not match" "비밀번호가 일치하지 않습니다" "两次输入的密码不一致" "兩次輸入的密碼不一致" "パスワードが一致しません" "Las contraseñas no coinciden")"
fi
[[ -n "$NEWPASS" ]] || die "$(msg "Password cannot be empty" "비밀번호는 비어 있을 수 없습니다" "密码不能为空" "密碼不可為空" "パスワードは空にできません" "La contraseña no puede estar vacía")"

# Pass the password through the environment for psql \getenv.
export ARTEX_RESET_NEWPASS="$NEWPASS"

# Generate bcrypt and upsert in the database; psql quotes the password safely.
# CREATE EXTENSION is idempotent but requires an authorized database role.
SQL=$(cat <<SQL
\\set ON_ERROR_STOP on
\\getenv newpw ARTEX_RESET_NEWPASS
CREATE EXTENSION IF NOT EXISTS pgcrypto;
INSERT INTO settings(key, value)
VALUES ('$PASS_KEY', crypt(:'newpw', gen_salt('bf', $BCRYPT_COST)))
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now();
SQL
)

# Execute the reset.
if [[ "$MODE" == "local" ]]; then
  # Connection precedence: CLI, --dsn/ARTEX_PG_DSN, config.json.
  if [[ -z "$DSN" && -z "$HOST$USER$DBNAME" ]]; then
    [[ -n "${ARTEX_PG_DSN:-}" ]] && DSN="$ARTEX_PG_DSN"
  fi
  if [[ -z "$DSN" && -z "$HOST$USER$DBNAME" ]]; then
    cfg="${CONFIG:-config.json}"
    if [[ -f "$cfg" ]]; then
      info "$(msg "Reading database configuration from $cfg" "${cfg}에서 데이터베이스 설정을 읽습니다" "正在从 ${cfg} 读取数据库配置" "正在從 ${cfg} 讀取資料庫設定" "${cfg} からデータベース設定を読み込んでいます" "Leyendo la configuración de la base de datos de ${cfg}")"
      apply_config_fields < <(read_config_json "$cfg")
    fi
  fi

  command -v psql >/dev/null 2>&1 || die "$(msg "psql not found; install postgresql-client or use -m docker" "psql이 없습니다. postgresql-client를 설치하거나 -m docker를 사용하세요" "未找到 psql；请安装 postgresql-client 或使用 -m docker" "找不到 psql；請安裝 postgresql-client 或使用 -m docker" "psql が見つかりません。postgresql-client をインストールするか、-m docker を使用してください" "No se ha encontrado psql; instala postgresql-client o usa -m docker")"

  declare -a PSQL_ARGS=()
  if [[ -n "$DSN" ]]; then
    PSQL_ARGS=("$DSN")
    target="$DSN"
  else
    [[ -n "$USER"   ]] || die "$(msg "Missing database user (-U) or valid config.json/DSN" "데이터베이스 사용자(-U) 또는 유효한 config.json/DSN이 필요합니다" "缺少数据库用户 (-U) 或有效的 config.json/DSN" "缺少資料庫使用者 (-U) 或有效的 config.json/DSN" "データベースのユーザー (-U)、または有効な config.json/DSN が必要です" "Falta el usuario de la base de datos (-U) o un config.json/DSN válido")"
    [[ -n "$DBNAME" ]] || die "$(msg "Missing database name (-d) or valid config.json/DSN" "데이터베이스 이름(-d) 또는 유효한 config.json/DSN이 필요합니다" "缺少数据库名称 (-d) 或有效的 config.json/DSN" "缺少資料庫名稱 (-d) 或有效的 config.json/DSN" "データベース名 (-d)、または有効な config.json/DSN が必要です" "Falta el nombre de la base de datos (-d) o un config.json/DSN válido")"
    HOST="${HOST:-127.0.0.1}"; PORT="${PORT:-5432}"; SSLMODE="${SSLMODE:-disable}"
    PSQL_ARGS=(-h "$HOST" -p "$PORT" -U "$USER" -d "$DBNAME")
    [[ -n "$SSLMODE" ]] && export PGSSLMODE="$SSLMODE"
    [[ -n "$DBPASS" ]] && export PGPASSWORD="$DBPASS"
    target="$USER@$HOST:$PORT/$DBNAME"
  fi

  info "$(msg "Target database: $target" "대상 데이터베이스: $target" "目标数据库：${target}" "目標資料庫：${target}" "対象のデータベース: ${target}" "Base de datos de destino: ${target}")"
  if [[ "$ASSUME_YES" -ne 1 ]]; then
    read -r -p "$(msg "Reset the administrator password in this database? [y/N] " "이 데이터베이스에서 관리자 비밀번호를 재설정할까요? [y/N] " "重置此数据库中的管理员密码？[y/N] " "重設此資料庫中的管理員密碼？[y/N] " "このデータベースの管理者パスワードをリセットしますか？[y/N] " "¿Restablecer la contraseña de administrador en esta base de datos? [y/N] ")" ans
    [[ "$ans" == "y" || "$ans" == "Y" ]] || die "$(msg "Cancelled" "취소됨" "已取消" "已取消" "キャンセルしました" "Cancelado")"
  fi

  if ! printf '%s\n' "$SQL" | psql "${PSQL_ARGS[@]}" -v ON_ERROR_STOP=1 -q >/dev/null; then
    die "$(msg "Write failed. For pgcrypto permission/missing errors, use a role allowed to create extensions or run CREATE EXTENSION pgcrypto first." "쓰기 실패. pgcrypto 권한/누락 오류라면 확장 생성 권한이 있는 계정을 사용하거나 CREATE EXTENSION pgcrypto를 먼저 실행하세요." "写入失败。若出现 pgcrypto 权限或缺失错误，请使用有权创建扩展的角色，或先运行 CREATE EXTENSION pgcrypto。" "寫入失敗。若出現 pgcrypto 權限或缺少擴充功能的錯誤，請使用獲准建立擴充功能的角色，或先執行 CREATE EXTENSION pgcrypto。" "書き込みに失敗しました。pgcrypto の権限不足や未導入のエラーの場合は、拡張機能を作成できるロールを使用するか、先に CREATE EXTENSION pgcrypto を実行してください。" "Error de escritura. Si pgcrypto no existe o faltan permisos, usa un rol autorizado para crear extensiones o ejecuta primero CREATE EXTENSION pgcrypto.")"
  fi

else
  # ---- docker ----
  command -v docker >/dev/null 2>&1 || die "$(msg "docker not found" "docker를 찾을 수 없습니다" "未找到 docker" "找不到 docker" "docker が見つかりません" "No se ha encontrado docker")"
  CONTAINER="${CONTAINER:-postgres}"

  # Prefer docker compose exec by service name; otherwise docker exec by container.
  if [[ -z "$EXEC_KIND" ]]; then
    if docker compose version >/dev/null 2>&1 && [[ -f docker-compose.yml ]]; then
      EXEC_KIND="compose"
    else
      EXEC_KIND="docker"
    fi
  fi

  # Credentials: CLI, POSTGRES_* from .env, then compose defaults (artex).
  if [[ -f .env ]]; then
    # shellcheck disable=SC1091
    set -a; . ./.env; set +a
  fi
  DUSER="${USER:-${POSTGRES_USER:-artex}}"
  DNAME="${DBNAME:-${POSTGRES_DB:-artex}}"
  [[ -n "$DBPASS" ]] && export PGPASSWORD="$DBPASS"
  [[ -z "${PGPASSWORD:-}" && -n "${POSTGRES_PASSWORD:-}" ]] && export PGPASSWORD="$POSTGRES_PASSWORD"

  info "$(msg "Target: container $CONTAINER, psql -U $DUSER -d $DNAME (exec=$EXEC_KIND)" "대상: 컨테이너 $CONTAINER, psql -U $DUSER -d $DNAME (exec=$EXEC_KIND)" "目标：容器 ${CONTAINER}，psql -U ${DUSER} -d ${DNAME} (exec=${EXEC_KIND})" "目標：容器 ${CONTAINER}，psql -U ${DUSER} -d ${DNAME} (exec=${EXEC_KIND})" "対象: コンテナー ${CONTAINER}、psql -U ${DUSER} -d ${DNAME} (exec=${EXEC_KIND})" "Destino: contenedor ${CONTAINER}, psql -U ${DUSER} -d ${DNAME} (exec=${EXEC_KIND})")"
  if [[ "$ASSUME_YES" -ne 1 ]]; then
    read -r -p "$(msg "Reset the administrator password in this container database? [y/N] " "이 컨테이너 데이터베이스에서 관리자 비밀번호를 재설정할까요? [y/N] " "重置此容器数据库中的管理员密码？[y/N] " "重設此容器資料庫中的管理員密碼？[y/N] " "このコンテナーのデータベースの管理者パスワードをリセットしますか？[y/N] " "¿Restablecer la contraseña de administrador en la base de datos de este contenedor? [y/N] ")" ans
    [[ "$ans" == "y" || "$ans" == "Y" ]] || die "$(msg "Cancelled" "취소됨" "已取消" "已取消" "キャンセルしました" "Cancelado")"
  fi

  # Pass only variable names to -e so secrets stay out of docker argv.
  declare -a EXEC_CMD
  if [[ "$EXEC_KIND" == "compose" ]]; then
    EXEC_CMD=(docker compose exec -T -e ARTEX_RESET_NEWPASS -e PGPASSWORD "$CONTAINER"
              psql -U "$DUSER" -d "$DNAME" -v ON_ERROR_STOP=1 -q)
  else
    EXEC_CMD=(docker exec -i -e ARTEX_RESET_NEWPASS -e PGPASSWORD "$CONTAINER"
              psql -U "$DUSER" -d "$DNAME" -v ON_ERROR_STOP=1 -q)
  fi

  if ! printf '%s\n' "$SQL" | "${EXEC_CMD[@]}" >/dev/null; then
    die "$(msg "Write failed. Check container (-c), database credentials (POSTGRES_* in .env), and pgcrypto permissions." "쓰기 실패. 컨테이너(-c), 데이터베이스 계정(.env의 POSTGRES_*), pgcrypto 권한을 확인하세요." "写入失败。请检查容器 (-c)、数据库凭据（.env 中的 POSTGRES_*）以及 pgcrypto 权限。" "寫入失敗。請檢查容器 (-c)、資料庫認證資訊（.env 中的 POSTGRES_*）及 pgcrypto 權限。" "書き込みに失敗しました。コンテナー (-c)、データベースの認証情報（.env の POSTGRES_*）、pgcrypto の権限を確認してください。" "Error de escritura. Comprueba el contenedor (-c), las credenciales de la base de datos (POSTGRES_* en .env) y los permisos de pgcrypto.")"
  fi
fi

unset ARTEX_RESET_NEWPASS
echo "$(msg "Administrator password reset. Sign in as ARTEX with the new password; no restart needed." "관리자 비밀번호를 재설정했습니다. ARTEX와 새 비밀번호로 로그인하세요. 재시작은 필요하지 않습니다." "管理员密码已重置。请使用 ARTEX 和新密码登录；无需重启。" "管理員密碼已重設。請使用 ARTEX 和新密碼登入；無須重新啟動。" "管理者パスワードをリセットしました。ARTEX と新しいパスワードでログインしてください。再起動は不要です。" "Contraseña de administrador restablecida. Inicia sesión como ARTEX con la nueva contraseña; no es necesario reiniciar.")"
