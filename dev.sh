#!/usr/bin/env bash
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

cd "$(dirname "$0")"

cleanup() { kill 0 2>/dev/null || true; }
trap cleanup EXIT INT TERM

go run ./cmd/artex -addr :8787 -proxy 127.0.0.1:8788 &

( cd web && npm run dev ) &

echo "[dev] $(msg "Backend :8787 / proxy :8788 / frontend http://localhost:5173 (Ctrl-C to stop)" "백엔드 :8787 / 프록시 :8788 / 프런트엔드 http://localhost:5173 (Ctrl-C로 종료)" "后端 :8787 / 代理 :8788 / 前端 http://localhost:5173（按 Ctrl-C 停止）" "後端 :8787 / Proxy :8788 / 前端 http://localhost:5173（按 Ctrl-C 停止）" "バックエンド :8787 / プロキシ :8788 / フロントエンド http://localhost:5173（Ctrl-C で停止）" "Backend :8787 / proxy :8788 / frontend http://localhost:5173 (Ctrl-C para detener)")"
wait
