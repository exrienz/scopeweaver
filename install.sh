#!/usr/bin/env bash
# ScopeWeaver install script: ① all-in-Docker  ② build and run locally
#
# Messages default to English; set SCOPEWEAVER_LANGUAGE to en, ko, zh-Hans,
# zh-Hant, ja or es. ARTEX_LANGUAGE remains supported.
set -euo pipefail
cd "$(cd "$(dirname "$0")" && pwd)"

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

info(){ printf '\033[36m[*]\033[0m %s\n' "$*"; }
ok(){   printf '\033[32m[+]\033[0m %s\n' "$*"; }
warn(){ printf '\033[33m[!]\033[0m %s\n' "$*"; }
die(){  printf '\033[31m[x]\033[0m %s\n' "$*" >&2; exit 1; }
ask(){  local p="$1" d="${2:-}" a; read -rp "$p${d:+ [$d]}: " a; echo "${a:-$d}"; }
rand(){ head -c 18 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 24; }

# Detect only; configuring ScopeWeaver never changes the host Ollama daemon.
choose_host_ollama(){
  OLLAMA_MODEL=''
  if command -v curl >/dev/null 2>&1 && curl -fsS --connect-timeout 1 --max-time 2 http://127.0.0.1:11434/api/tags >/dev/null 2>&1; then
    info "$(msg 'Detected Ollama on the host' '호스트에서 Ollama를 감지했습니다' "检测到主机上的 Ollama" "偵測到主機上的 Ollama" "ホスト上の Ollama を検出しました" "Se ha detectado Ollama en el equipo")" >&2
    if [ "$(ask "$(msg 'Configure ScopeWeaver to use host Ollama? (y/n)' 'ScopeWeaver에서 호스트 Ollama를 사용할까요? (y/n)' "配置 ScopeWeaver 使用主机上的 Ollama？(y/n)" "設定 ScopeWeaver 使用主機上的 Ollama？(y/n)" "ホスト上の Ollama を ScopeWeaver で使用しますか？(y/n)" "¿Configurar ScopeWeaver para usar Ollama del equipo? (y/n)")" n)" = y ]; then
      OLLAMA_MODEL="$(ask "$(msg 'Downloaded model name (no download will be started)' '이미 다운로드한 모델 이름 (다운로드하지 않습니다)' "已下载的模型名称（不会开始下载）" "已下載的模型名稱（不會開始下載）" "ダウンロード済みのモデル名（ダウンロードは開始しません）" "Nombre del modelo descargado (no se iniciará ninguna descarga)")" 'qwen2.5-coder:7b')"
      case "$OLLAMA_MODEL" in *[!A-Za-z0-9_./:@-]*|'') die "$(msg 'Invalid model name' '잘못된 모델 이름' "模型名称无效" "模型名稱無效" "モデル名が無効です" "Nombre de modelo no válido")" ;; esac
    fi
  fi
}

# ── docker environment detection / auto-install ─────────────
ensure_docker(){
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    ok "$(msg 'Found docker and docker compose' 'docker와 docker compose를 감지했습니다' "已找到 docker 和 docker compose" "已找到 docker 和 docker compose" "docker と docker compose が見つかりました" "Se han encontrado docker y docker compose")"; return
  fi
  warn "$(msg 'docker / docker compose not found' 'docker / docker compose를 찾지 못했습니다' "未找到 docker / docker compose" "找不到 docker / docker compose" "docker / docker compose が見つかりません" "No se ha encontrado docker / docker compose")"
  case "$(uname -s)" in
    Linux)
      if [ "$(ask "$(msg 'Install Docker automatically? (y/n)' 'Docker를 자동으로 설치할까요? (y/n)' "自动安装 Docker？(y/n)" "自動安裝 Docker？(y/n)" "Docker を自動インストールしますか？(y/n)" "¿Instalar Docker automáticamente? (y/n)")" y)" = y ]; then
        curl -fsSL https://get.docker.com | sh
        sudo usermod -aG docker "$USER" || true
        ok "$(msg 'Docker installed (group change needs re-login to use without sudo)' 'Docker 설치 완료 (그룹 변경은 재로그인 후 sudo 없이 적용)' "Docker 已安装（更改用户组后需重新登录才能免 sudo 使用）" "Docker 已安裝（變更使用者群組後須重新登入，才能不使用 sudo 執行）" "Docker をインストールしました（グループ変更後、sudo なしで使用するには再ログインが必要です）" "Docker instalado (hay que volver a iniciar sesión para usarlo sin sudo tras el cambio de grupo)")"
      else
        die "$(msg 'Please install docker yourself and retry' 'docker를 직접 설치한 뒤 다시 시도하세요' "请自行安装 docker 后重试" "請自行安裝 docker 後再試" "docker を手動でインストールしてから再試行してください" "Instala docker manualmente y vuelve a intentarlo")"
      fi ;;
    Darwin) die "$(msg 'On macOS install Docker Desktop: https://www.docker.com/products/docker-desktop/' 'macOS에서는 Docker Desktop을 설치하세요: https://www.docker.com/products/docker-desktop/' "在 macOS 上请安装 Docker Desktop：https://www.docker.com/products/docker-desktop/" "在 macOS 上請安裝 Docker Desktop：https://www.docker.com/products/docker-desktop/" "macOS では Docker Desktop をインストールしてください: https://www.docker.com/products/docker-desktop/" "En macOS, instala Docker Desktop: https://www.docker.com/products/docker-desktop/")" ;;
    *)      die "$(msg 'Please install docker yourself and retry' 'docker를 직접 설치한 뒤 다시 시도하세요' "请自行安装 docker 后重试" "請自行安裝 docker 後再試" "docker を手動でインストールしてから再試行してください" "Instala docker manualmente y vuelve a intentarlo")" ;;
  esac
}

# ── ① all-in-Docker ─────────────────────────────────────────
install_docker(){
  ensure_docker
  if [ ! -f .env ]; then
    cp .env.example .env 2>/dev/null || true
    local pw key
    pw="$(ask "$(msg 'Postgres password (Enter for a random one)' 'Postgres 비밀번호 (Enter 시 무작위 생성)' "Postgres 密码（按 Enter 随机生成）" "Postgres 密碼（按 Enter 隨機產生）" "Postgres のパスワード（Enter でランダム生成）" "Contraseña de Postgres (Enter para generar una aleatoria)")" "$(rand)")"
    key="$(ask "$(msg 'ANTHROPIC_API_KEY (optional, can set later in the UI)' 'ANTHROPIC_API_KEY (선택, 나중에 UI에서 설정 가능)' "ANTHROPIC_API_KEY（可选，稍后可在界面中设置）" "ANTHROPIC_API_KEY（選填，之後可在介面中設定）" "ANTHROPIC_API_KEY（任意。後から画面で設定できます）" "ANTHROPIC_API_KEY (opcional; se puede configurar después en la interfaz)")" '')"
    sed -i.bak "s|^POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=${pw}|" .env
    sed -i.bak "s|^ANTHROPIC_API_KEY=.*|ANTHROPIC_API_KEY=${key}|" .env
    rm -f .env.bak
    choose_host_ollama
    if [ -n "$OLLAMA_MODEL" ]; then
      sed -i.bak 's|^SCOPEWEAVER_LLM_PROVIDER=.*|SCOPEWEAVER_LLM_PROVIDER=ollama|' .env
      sed -i.bak "s|^SCOPEWEAVER_LLM_MODEL=.*|SCOPEWEAVER_LLM_MODEL=${OLLAMA_MODEL}|" .env
      sed -i.bak 's|^SCOPEWEAVER_LLM_BASE_URL=.*|SCOPEWEAVER_LLM_BASE_URL=http://host.docker.internal:11434|' .env
      rm -f .env.bak
    fi
    ok "$(msg '.env generated (POSTGRES_PASSWORD set)' '.env 생성 완료 (POSTGRES_PASSWORD 설정됨)' ".env 已生成（POSTGRES_PASSWORD 已设置）" ".env 已產生（POSTGRES_PASSWORD 已設定）" ".env を生成しました（POSTGRES_PASSWORD 設定済み）" ".env generado (POSTGRES_PASSWORD configurado)")"
  else
    info "$(msg 'Reusing existing .env' '기존 .env를 그대로 사용합니다' "复用现有 .env" "沿用現有的 .env" "既存の .env を再利用します" "Se reutiliza el archivo .env existente")"
  fi
  info "$(msg 'Building ScopeWeaver from this checkout…' '현재 소스로 ScopeWeaver를 빌드합니다…' "正在从当前工作副本构建 ScopeWeaver…" "正在從目前的工作副本建置 ScopeWeaver…" "このチェックアウトから ScopeWeaver をビルドしています…" "Compilando ScopeWeaver desde esta copia de trabajo…")"
  docker compose build scopeweaver
  docker compose up -d
  ok "$(msg 'Started → http://localhost:8787' '시작 완료 → http://localhost:8787' "已启动 → http://localhost:8787" "已啟動 → http://localhost:8787" "起動しました → http://localhost:8787" "Iniciado → http://localhost:8787")"
  info "$(msg 'View logs: docker compose logs -f scopeweaver' '로그 보기: docker compose logs -f scopeweaver' "查看日志：docker compose logs -f scopeweaver" "檢視記錄：docker compose logs -f scopeweaver" "ログを表示: docker compose logs -f scopeweaver" "Ver registros: docker compose logs -f scopeweaver")"
}

# ── ② build and run locally ─────────────────────────────────
install_local(){
  echo "$(msg 'Database setup:' '데이터베이스 설치 방식:' "数据库设置：" "資料庫設定：" "データベースの設定:" "Configuración de la base de datos:")"
  echo "  1) $(msg 'Connect to an existing PostgreSQL' '기존 PostgreSQL에 연결' "连接现有 PostgreSQL" "連線至現有的 PostgreSQL" "既存の PostgreSQL に接続" "Conectar a un PostgreSQL existente")"
  echo "  2) $(msg 'Start a PostgreSQL with Docker (needs docker)' 'Docker로 PostgreSQL 하나 띄우기 (docker 필요)' "使用 Docker 启动 PostgreSQL（需要 docker）" "使用 Docker 啟動 PostgreSQL（需要 docker）" "Docker で PostgreSQL を起動（docker が必要）" "Iniciar PostgreSQL con Docker (requiere docker)")"
  case "$(ask "$(msg 'Choose' '선택' "请选择" "請選擇" "選択" "Elige")" 1)" in
    2)
      ensure_docker
      local pw; pw="$(ask "$(msg 'Postgres password (Enter for a random one)' 'Postgres 비밀번호 (Enter 시 무작위 생성)' "Postgres 密码（按 Enter 随机生成）" "Postgres 密碼（按 Enter 隨機產生）" "Postgres のパスワード（Enter でランダム生成）" "Contraseña de Postgres (Enter para generar una aleatoria)")" "$(rand)")"
      docker run -d --name scopeweaver-pg -p 5432:5432 \
        -e POSTGRES_USER=artex -e POSTGRES_PASSWORD="$pw" -e POSTGRES_DB=artex \
        -v scopeweaver-pg:/var/lib/postgresql/data postgres:16-alpine
      DB_HOST=127.0.0.1 DB_PORT=5432 DB_USER=artex DB_PASS="$pw" DB_NAME=artex DB_SSL=disable ;;
    *)
      DB_HOST="$(ask "$(msg 'Database host' '데이터베이스 주소' "数据库主机" "資料庫主機" "データベースのホスト" "Servidor de la base de datos")" 127.0.0.1)"
      DB_PORT="$(ask "$(msg 'Port' '포트' "端口" "連接埠" "ポート" "Puerto")" 5432)"
      DB_USER="$(ask "$(msg 'User' '계정' "用户" "使用者" "ユーザー" "Usuario")" artex)"
      DB_PASS="$(ask "$(msg 'Password' '비밀번호' "密码" "密碼" "パスワード" "Contraseña")" '')"
      DB_NAME="$(ask "$(msg 'Database name' '데이터베이스 이름' "数据库名称" "資料庫名稱" "データベース名" "Nombre de la base de datos")" artex)"
      DB_SSL="$(ask "$(msg 'sslmode (disable/require)' 'sslmode (disable/require)' "sslmode（SSL 模式，disable/require）" "sslmode（SSL 模式，disable/require）" "sslmode（SSL モード、disable/require）" "Modo sslmode (disable/require)")" disable)" ;;
  esac

  # Existing configuration belongs to the operator; never overwrite it.
  if [ -f config.json ]; then
    info "$(msg 'Reusing existing config.json' '기존 config.json을 그대로 사용합니다' "复用现有 config.json" "沿用現有的 config.json" "既存の config.json を再利用します" "Se reutiliza el archivo config.json existente")"
  else
    choose_host_ollama
    local llm_json=''
    if [ -n "$OLLAMA_MODEL" ]; then
      llm_json=", \"llm\": {\"provider\": \"ollama\", \"base_url\": \"http://127.0.0.1:11434\", \"model\": \"${OLLAMA_MODEL}\", \"context_window_k\": 32, \"keep_alive\": \"30m\", \"llm_max_concurrent\": 1}"
    fi
  cat > config.json <<JSON
{
  "database": {
    "host": "${DB_HOST}",
    "port": ${DB_PORT},
    "user": "${DB_USER}",
    "password": "${DB_PASS}",
    "dbname": "${DB_NAME}",
    "sslmode": "${DB_SSL}"
  }${llm_json}
}
JSON
  ok "$(msg 'config.json generated' 'config.json 생성 완료' "config.json 已生成" "config.json 已產生" "config.json を生成しました" "config.json generado")"
  fi

  # go environment check
  command -v go >/dev/null 2>&1 || die "$(msg 'Go not found, install Go (>=1.26): https://go.dev/dl/' 'Go를 찾지 못했습니다. Go(>=1.26)를 먼저 설치하세요: https://go.dev/dl/' "未找到 Go，请先安装 Go (>=1.26)：https://go.dev/dl/" "找不到 Go，請先安裝 Go (>=1.26)：https://go.dev/dl/" "Go が見つかりません。Go (>=1.26) をインストールしてください: https://go.dev/dl/" "No se ha encontrado Go; instala Go (>=1.26): https://go.dev/dl/")"
  ok "Go: $(go version)"

  # Embedding the frontend needs node to produce the static output
  if command -v npm >/dev/null 2>&1; then
    info "$(msg 'Building the frontend static output…' '프런트엔드 정적 산출물을 빌드합니다…' "正在构建前端静态文件…" "正在建置前端靜態檔案…" "フロントエンドの静的ファイルをビルドしています…" "Generando los archivos estáticos del frontend…")"
    ( cd web && npm ci && npm run build:static )
    mkdir -p server/webui && rm -rf server/webui/dist && cp -r web/out server/webui/dist
    info "$(msg 'Compiling the single binary with the frontend embedded…' '프런트엔드가 내장된 단일 바이너리를 컴파일합니다…' "正在编译内嵌前端的单一二进制文件…" "正在編譯內嵌前端的單一執行檔…" "フロントエンドを埋め込んだ単一バイナリをコンパイルしています…" "Compilando un único binario con el frontend integrado…")"
    CGO_ENABLED=0 go build -tags embedui -trimpath -o scopeweaver ./cmd/artex
  else
    warn "$(msg 'npm not found: building the backend WITHOUT the embedded frontend (run npm run dev for the frontend)' 'npm을 찾지 못함: 프런트엔드 미내장 백엔드를 빌드합니다 (프런트엔드는 따로 npm run dev)' "未找到 npm：仅构建不含前端的后端（请运行 npm run dev 启动前端）" "找不到 npm：僅建置未內嵌前端的後端（請執行 npm run dev 啟動前端）" "npm が見つかりません。フロントエンドを含めずバックエンドのみビルドします（フロントエンドは npm run dev で起動してください）" "No se ha encontrado npm: se compila el backend SIN el frontend integrado (ejecuta npm run dev para el frontend)")"
    CGO_ENABLED=0 go build -o scopeweaver ./cmd/artex
  fi
  ok "$(msg 'Build complete → ./scopeweaver' '컴파일 완료 → ./scopeweaver' "构建完成 → ./scopeweaver" "建置完成 → ./scopeweaver" "ビルド完了 → ./scopeweaver" "Compilación terminada → ./scopeweaver")"

  info "$(msg 'Starting… (Ctrl-C to quit)' '시작합니다… (Ctrl-C로 종료)' "正在启动…（按 Ctrl-C 退出）" "正在啟動…（按 Ctrl-C 結束）" "起動しています…（Ctrl-C で終了）" "Iniciando… (Ctrl-C para salir)")"
  ./start.sh
}

echo "=============================="
echo "  $(msg 'ScopeWeaver install' 'ScopeWeaver 설치' "安装 ScopeWeaver" "安裝 ScopeWeaver" "ScopeWeaver のインストール" "Instalación de ScopeWeaver")"
echo "  1) $(msg 'All-in-Docker install' '전부 Docker로 설치' "全部使用 Docker 安装" "全部使用 Docker 安裝" "すべて Docker でインストール" "Instalación completa con Docker")"
echo "  2) $(msg 'Local run (go build)' '로컬 실행 (go 컴파일)' "本地运行 (go build)" "本機執行 (go build)" "ローカルで実行 (go build)" "Ejecución local (go build)")"
echo "=============================="
case "$(ask "$(msg 'Choose' '선택' "请选择" "請選擇" "選択" "Elige")" 1)" in
  1) install_docker ;;
  2) install_local ;;
  *) die "$(msg 'Invalid choice' '잘못된 선택' "选项无效" "選項無效" "選択が無効です" "Opción no válida")" ;;
esac
