#!/bin/sh
# ScopeWeaver supervisor: exit 0 stops, 75 restarts immediately, other codes back off.
# Forward termination signals and wait for the child to finish graceful shutdown.
# Download verification and atomic replacement are handled by selfupdate.Bootstrap.
set -u
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


cd "$(dirname "$0")" || exit 1

BIN=./scopeweaver
[ -x "$BIN" ] || { echo "[ScopeWeaver] $(msg "Executable not found: $BIN" "실행 파일을 찾을 수 없습니다: $BIN" "未找到可执行文件：${BIN}" "找不到執行檔：${BIN}" "実行ファイルが見つかりません: ${BIN}" "No se ha encontrado el ejecutable: ${BIN}")" >&2; exit 1; }

RESTART_CODE=75
MAX_DELAY=60

child=0
stopping=0

forward() {
	stopping=1
	if [ "$child" -ne 0 ]; then
		kill -TERM "$child" 2>/dev/null || true
	fi
}
trap forward INT TERM

delay=1
while :; do
	"$BIN" "$@" &
	child=$!

	wait "$child"
	code=$?
	if [ "$code" -gt 128 ]; then
		wait "$child"
		code=$?
	fi
	child=0

	if [ "$stopping" -eq 1 ]; then
		echo "[ScopeWeaver] $(msg "Stopped" "중지됨" "已停止" "已停止" "停止しました" "Detenido")"
		exit 0
	fi

	case "$code" in
		0)
			echo "[ScopeWeaver] $(msg "Exited normally" "정상 종료" "已正常退出" "已正常結束" "正常終了しました" "Finalizó normalmente")"
			exit 0
			;;
		"$RESTART_CODE")
			echo "[ScopeWeaver] $(msg "Restart requested (applying update)…" "다시 시작 요청 (업데이트 적용 중)…" "收到重启请求（正在应用更新）…" "收到重新啟動要求（正在套用更新）…" "再起動が要求されました（更新を適用中）…" "Se ha solicitado reiniciar (aplicando la actualización)…")"
			delay=1
			;;
		*)
			echo "[ScopeWeaver] $(msg "Exited with code $code; restarting in ${delay}s" "종료 코드 $code; ${delay}초 후 다시 시작" "退出码为 ${code}；${delay} 秒后重启" "結束代碼為 ${code}；${delay} 秒後重新啟動" "終了コード ${code}。${delay} 秒後に再起動します" "Finalizó con el código ${code}; se reiniciará en ${delay} s")" >&2
			sleep "$delay"
			delay=$((delay * 2))
			[ "$delay" -gt "$MAX_DELAY" ] && delay=$MAX_DELAY
			;;
	esac
done
