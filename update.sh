#!/usr/bin/env bash
# ScopeWeaver update: rebuild the local Docker image or native binary.
# Startup applies the idempotent database schema. Back up data before upgrading.
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
info(){ printf '%s\n' "$*"; }
die(){ printf '%s\n' "$*" >&2; exit 1; }
ask(){ local p="$1" d="${2:-}" a; read -rp "$p${d:+ [$d]}: " a; echo "${a:-$d}"; }
sync_repo(){
  [ -d .git ] && command -v git >/dev/null 2>&1 || { info "$(msg 'Not a Git checkout; using current files' 'Git 작업 사본이 아니므로 현재 파일을 사용합니다' "当前目录不是 Git 工作副本；使用现有文件" "目前的目錄不是 Git 工作副本；使用現有檔案" "Git のチェックアウトではないため、現在のファイルを使用します" "No es una copia de trabajo de Git; se usarán los archivos actuales")"; return; }
  [ "$(ask "$(msg 'Fetch current code with git pull --ff-only? (y/n)' 'git pull --ff-only로 최신 코드를 가져올까요? (y/n)' "使用 git pull --ff-only 获取最新代码？(y/n)" "使用 git pull --ff-only 取得最新程式碼？(y/n)" "git pull --ff-only で最新のコードを取得しますか？(y/n)" "¿Obtener el código actual con git pull --ff-only? (y/n)")" y)" = y ] || return
  git pull --ff-only || die "$(msg 'Git cannot fast-forward. Resolve local changes or divergent history, then retry.' 'Git fast-forward 실패. 로컬 변경 또는 분기된 기록을 정리한 뒤 다시 시도하세요.' "Git 无法快进。请处理本地更改或分叉历史后重试。" "Git 無法快轉。請先處理本機變更或分岔的歷史記錄，再重試。" "Git を fast-forward できません。ローカルの変更や分岐した履歴を解消してから再試行してください。" "Git no puede avanzar mediante fast-forward. Resuelve los cambios locales o el historial divergente y vuelve a intentarlo.")"
}
update_docker(){
  command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1 || die "$(msg 'Install Docker and Docker Compose first' 'Docker와 Docker Compose를 먼저 설치하세요' "请先安装 Docker 和 Docker Compose" "請先安裝 Docker 和 Docker Compose" "先に Docker と Docker Compose をインストールしてください" "Instala primero Docker y Docker Compose")"
  [ -f .env ] || die "$(msg 'Missing .env; run ./install.sh first' '.env가 없습니다. ./install.sh를 먼저 실행하세요' "缺少 .env；请先运行 ./install.sh" "缺少 .env；請先執行 ./install.sh" ".env がありません。先に ./install.sh を実行してください" "Falta .env; ejecuta primero ./install.sh")"
  info "$(msg 'Building ScopeWeaver from this checkout…' '현재 소스로 ScopeWeaver를 빌드합니다…' "正在从当前工作副本构建 ScopeWeaver…" "正在從目前的工作副本建置 ScopeWeaver…" "このチェックアウトから ScopeWeaver をビルドしています…" "Compilando ScopeWeaver desde esta copia de trabajo…")"
  docker compose build scopeweaver
  docker compose up -d scopeweaver
  info "$(msg 'Updated: http://localhost:8787. Logs: docker compose logs -f scopeweaver' '업데이트 완료: http://localhost:8787. 로그: docker compose logs -f scopeweaver' "更新完成：http://localhost:8787。日志：docker compose logs -f scopeweaver" "更新完成：http://localhost:8787。記錄：docker compose logs -f scopeweaver" "更新しました: http://localhost:8787。ログ: docker compose logs -f scopeweaver" "Actualizado: http://localhost:8787. Registros: docker compose logs -f scopeweaver")"
}
update_local(){
  command -v go >/dev/null 2>&1 || die "$(msg 'Go 1.26 or later required: https://go.dev/dl/' 'Go 1.26 이상 필요: https://go.dev/dl/' "需要 Go 1.26 或更高版本：https://go.dev/dl/" "需要 Go 1.26 或更新版本：https://go.dev/dl/" "Go 1.26 以降が必要です: https://go.dev/dl/" "Se requiere Go 1.26 o posterior: https://go.dev/dl/")"
  if command -v npm >/dev/null 2>&1; then
    (cd web && npm ci && npm run build:static)
    mkdir -p server/webui/dist
    cp -R web/out/. server/webui/dist/
    CGO_ENABLED=0 go build -tags embedui -trimpath -o scopeweaver ./cmd/artex
  else
    info "$(msg 'npm unavailable: building backend only; run the frontend separately' 'npm이 없어 백엔드만 빌드합니다. 프런트엔드는 별도로 실행하세요' "npm 不可用：仅构建后端；请单独运行前端" "npm 無法使用：僅建置後端；請另行執行前端" "npm が利用できないため、バックエンドのみビルドします。フロントエンドは別途実行してください" "npm no está disponible: se compila solo el backend; ejecuta el frontend por separado")"
    CGO_ENABLED=0 go build -o scopeweaver ./cmd/artex
  fi
  info "$(msg 'Built ./scopeweaver. Restart the running process to apply the update.' './scopeweaver 빌드 완료. 실행 중인 프로세스를 재시작하여 적용하세요.' "已构建 ./scopeweaver。请重启运行中的进程以应用更新。" "已建置 ./scopeweaver。請重新啟動執行中的程序以套用更新。" "./scopeweaver をビルドしました。実行中のプロセスを再起動して更新を適用してください。" "Se ha compilado ./scopeweaver. Reinicia el proceso en ejecución para aplicar la actualización.")"
}
echo "ScopeWeaver"
echo "1) $(msg 'Rebuild Docker deployment' 'Docker 배포 다시 빌드' "重新构建 Docker 部署" "重新建置 Docker 部署" "Docker デプロイを再ビルド" "Volver a compilar el despliegue con Docker")"
echo "2) $(msg 'Rebuild local binary' '로컬 바이너리 다시 빌드' "重新构建本地二进制文件" "重新建置本機執行檔" "ローカルバイナリを再ビルド" "Volver a compilar el binario local")"
case "$(ask "$(msg 'Choose' '선택' "请选择" "請選擇" "選択" "Elige")" 1)" in
  1) sync_repo; update_docker ;;
  2) sync_repo; update_local ;;
  *) die "$(msg 'Invalid choice' '잘못된 선택' "选项无效" "選項無效" "選択が無効です" "Opción no válida")" ;;
esac
