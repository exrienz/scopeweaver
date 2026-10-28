#!/usr/bin/env bash
# ScopeWeaver cross-platform release builder.
#
# The default mode builds one target and embeds the already-exported frontend.
# `./build.sh --release` builds and packages all supported desktop/server targets.
#
# Environment variables:
#   ARTEX_TARGET_OS=linux             One target OS in single-target mode.
#   ARTEX_TARGET_ARCH=amd64           One target arch in single-target mode.
#   ARTEX_TARGETS=linux/amd64,...     Comma-separated targets for multi-target mode.
#   ARTEX_BUILD_VERSION=v0.3.3        Version embedded in the binary and archive name.
#   ARTEX_OUTPUT=/path/to/scopeweaver       Explicit binary path in single-target mode.
#   ARTEX_OUTPUT_DIR=dist             Directory for default binary paths.
#   ARTEX_PACKAGE=1                   Create a zip archive for each target.
#   ARTEX_PACKAGE_DIR=dist            Directory for release archives.
#   ARTEX_COMPRESS=off                UPX mode: off, auto, or required.
#   ARTEX_UPX_ARGS="--best --lzma"    Arguments passed to UPX.
#   ARTEX_SKIP_FRONTEND=1             Reuse server/webui/dist (for CI artifact builds).
#   ARTEX_SKIP_NPM_CI=1               Skip npm ci while rebuilding the frontend.
#   ARTEX_GOSUMDB=sum.golang.org      Go checksum database.
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


cd "$(cd "$(dirname "$0")" && pwd)"

info() { printf '\033[36m[*]\033[0m %s\n' "$*"; }
ok() { printf '\033[32m[+]\033[0m %s\n' "$*"; }
warn() { printf '\033[33m[!]\033[0m %s\n' "$*" >&2; }
die() { printf '\033[31m[x]\033[0m %s\n' "$*" >&2; exit 1; }

usage() {
  case "$LANG_SEL" in
    ko) cat <<'EOF'
사용법: ./build.sh [--target OS/ARCH | --release] [--upx | --no-compress]
기본값: 현재 시스템/아키텍처를 빌드합니다.
--release: Linux 및 macOS amd64/arm64, Windows amd64를 빌드하고 zip으로 묶습니다.
--target OS/ARCH: 대상 지정 (예: windows/amd64).
--upx: UPX 압축 필수 (일부 Linux 환경과 호환되지 않을 수 있음).
--no-compress: UPX 없이 Go linker 최적화 및 zip 압축만 사용합니다.
--help: 도움말 표시.
ARTEX_TARGETS=linux/amd64,windows/amd64 ./build.sh --release로 대상 목록을 지정합니다.
EOF
      ;;
    zh-Hans) cat <<'EOF'
用法：./build.sh [--target OS/ARCH | --release] [--upx | --no-compress]
默认：构建当前系统和架构。
--release：构建并打包 Linux/macOS amd64/arm64 和 Windows amd64 的 zip。
--target OS/ARCH：选择目标，例如 windows/amd64。
--upx：要求 UPX 压缩（可能降低在某些 Linux 系统上的兼容性）。
--no-compress：仅使用 Go 链接器剥离和 zip 压缩，不使用 UPX。
--help：显示帮助。
使用 ARTEX_TARGETS=linux/amd64,windows/amd64 ./build.sh --release 覆盖目标列表。
EOF
      ;;
    zh-Hant) cat <<'EOF'
用法：./build.sh [--target OS/ARCH | --release] [--upx | --no-compress]
預設：建置目前的系統與架構。
--release：建置並封裝 Linux/macOS amd64/arm64 和 Windows amd64 的 zip。
--target OS/ARCH：選擇目標，例如 windows/amd64。
--upx：要求 UPX 壓縮（可能降低在某些 Linux 系統上的相容性）。
--no-compress：僅使用 Go 連結器移除符號與 zip 壓縮，不使用 UPX。
--help：顯示說明。
使用 ARTEX_TARGETS=linux/amd64,windows/amd64 ./build.sh --release 覆寫目標清單。
EOF
      ;;
    ja) cat <<'EOF'
使用法: ./build.sh [--target OS/ARCH | --release] [--upx | --no-compress]
既定: 現在のシステムとアーキテクチャ向けにビルドします。
--release: Linux/macOS amd64/arm64 と Windows amd64 向けにビルドし、zip にまとめます。
--target OS/ARCH: ターゲットを指定します（例: windows/amd64）。
--upx: UPX 圧縮を必須にします（一部の Linux システムで互換性が低下する場合があります）。
--no-compress: UPX を使わず、Go リンカーによるシンボル削除と zip 圧縮を使用します。
--help: ヘルプを表示します。
ARTEX_TARGETS=linux/amd64,windows/amd64 ./build.sh --release でターゲットを上書きします。
EOF
      ;;
    es) cat <<'EOF'
Uso: ./build.sh [--target OS/ARCH | --release] [--upx | --no-compress]
Por defecto: compila para el sistema y la arquitectura actuales.
--release: compila y empaqueta en zip para Linux/macOS amd64/arm64 y Windows amd64.
--target OS/ARCH: selecciona un destino, por ejemplo windows/amd64.
--upx: exige compresión UPX (puede reducir la compatibilidad en algunos sistemas Linux).
--no-compress: usa la eliminación de símbolos del enlazador de Go y compresión zip sin UPX.
--help: muestra la ayuda.
Cambia los destinos con ARTEX_TARGETS=linux/amd64,windows/amd64 ./build.sh --release.
EOF
      ;;
    *) cat <<'EOF'
Usage: ./build.sh [--target OS/ARCH | --release] [--upx | --no-compress]
Default: build the current system and architecture.
--release: build and zip Linux/macOS amd64/arm64 and Windows amd64.
--target OS/ARCH: select a target, for example windows/amd64.
--upx: require UPX compression (may reduce compatibility on some Linux systems).
--no-compress: use Go linker stripping and zip compression without UPX.
--help: show help.
Override targets with ARTEX_TARGETS=linux/amd64,windows/amd64 ./build.sh --release.
EOF
      ;;
  esac
}

RELEASE_TARGETS_DEFAULT="linux/amd64,linux/arm64,darwin/amd64,darwin/arm64,windows/amd64"
ARTEX_RELEASE="${ARTEX_RELEASE:-0}"
ARTEX_COMPRESS="${ARTEX_COMPRESS:-off}"
ARTEX_PACKAGE="${ARTEX_PACKAGE:-0}"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --release)
      ARTEX_RELEASE=1
      ARTEX_PACKAGE=1
      shift
      ;;
    --target)
      [ "$#" -ge 2 ] || die "$(msg "--target requires OS/ARCH" "--target에 OS/ARCH가 필요합니다" "--target 需要 OS/ARCH" "--target 需要 OS/ARCH" "--target には OS/ARCH が必要です" "--target requiere OS/ARCH")"
      target_arg="$2"
      case "$target_arg" in
        */*)
          ARTEX_TARGET_OS="${target_arg%%/*}"
          ARTEX_TARGET_ARCH="${target_arg##*/}"
          ARTEX_TARGETS="$target_arg"
          ;;
        *) die "$(msg "Target must be OS/ARCH, e.g. linux/amd64" "대상은 OS/ARCH 형식이어야 합니다 (예: linux/amd64)" "目标必须为 OS/ARCH，例如 linux/amd64" "目標必須為 OS/ARCH，例如 linux/amd64" "ターゲットは OS/ARCH 形式で指定してください（例: linux/amd64）" "El destino debe tener el formato OS/ARCH, por ejemplo linux/amd64")" ;;
      esac
      shift 2
      ;;
    --no-compress)
      ARTEX_COMPRESS=0
      shift
      ;;
    --upx)
      ARTEX_COMPRESS=required
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *) die "$(msg "Unknown argument: $1 (see --help)" "알 수 없는 인수: $1 (--help 참고)" "未知参数：$1（参见 --help）" "未知的引數：$1（請參閱 --help）" "不明な引数: $1（--help を参照）" "Argumento desconocido: $1 (consulta --help)")" ;;
  esac
done

command -v go >/dev/null 2>&1 || die "$(msg "Go not found (Go 1.26 or later required)" "Go를 찾을 수 없습니다 (Go 1.26 이상 필요)" "未找到 Go（需要 Go 1.26 或更高版本）" "找不到 Go（需要 Go 1.26 或更新版本）" "Go が見つかりません（Go 1.26 以降が必要です）" "No se ha encontrado Go (se requiere Go 1.26 o posterior)")"

ARTEX_GOSUMDB="${ARTEX_GOSUMDB:-sum.golang.org}"
if [ -z "${ARTEX_BUILD_VERSION:-}" ]; then
  if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    ARTEX_BUILD_VERSION="$(git describe --tags --always --dirty)"
  else
    ARTEX_BUILD_VERSION="dev"
  fi
fi
# Release tags are commonly passed as v0.3.3; keep the binary version consistent.
ARTEX_BUILD_VERSION="${ARTEX_BUILD_VERSION#v}"
ARTEX_OUTPUT_DIR="${ARTEX_OUTPUT_DIR:-dist}"
ARTEX_PACKAGE_DIR="${ARTEX_PACKAGE_DIR:-$ARTEX_OUTPUT_DIR}"
ARTEX_UPX_ARGS="${ARTEX_UPX_ARGS:---best --lzma}"

if [ "${ARTEX_RELEASE}" = "1" ]; then
  ARTEX_TARGETS="${ARTEX_TARGETS:-$RELEASE_TARGETS_DEFAULT}"
else
  ARTEX_TARGET_OS="${ARTEX_TARGET_OS:-$(GOSUMDB="$ARTEX_GOSUMDB" go env GOOS)}"
  ARTEX_TARGET_ARCH="${ARTEX_TARGET_ARCH:-$(GOSUMDB="$ARTEX_GOSUMDB" go env GOARCH)}"
  ARTEX_TARGETS="${ARTEX_TARGETS:-${ARTEX_TARGET_OS}/${ARTEX_TARGET_ARCH}}"
fi

if [ "${ARTEX_SKIP_FRONTEND:-0}" = "1" ]; then
  [ -d server/webui/dist ] || die "$(msg "ARTEX_SKIP_FRONTEND=1 but server/webui/dist is missing" "ARTEX_SKIP_FRONTEND=1이지만 server/webui/dist가 없습니다" "ARTEX_SKIP_FRONTEND=1，但缺少 server/webui/dist" "ARTEX_SKIP_FRONTEND=1，但缺少 server/webui/dist" "ARTEX_SKIP_FRONTEND=1 ですが server/webui/dist がありません" "ARTEX_SKIP_FRONTEND=1, pero falta server/webui/dist")"
else
  command -v npm >/dev/null 2>&1 || die "$(msg "npm not found (Node.js/npm required for frontend export)" "npm을 찾을 수 없습니다 (프런트엔드 빌드에 Node.js/npm 필요)" "未找到 npm（导出前端需要 Node.js/npm）" "找不到 npm（匯出前端需要 Node.js/npm）" "npm が見つかりません（フロントエンドのエクスポートには Node.js/npm が必要です）" "No se ha encontrado npm (se requiere Node.js/npm para exportar el frontend)")"
  command -v rsync >/dev/null 2>&1 || die "$(msg "rsync not found" "rsync를 찾을 수 없습니다" "未找到 rsync" "找不到 rsync" "rsync が見つかりません" "No se ha encontrado rsync")"
  info "$(msg "Building frontend static assets" "프런트엔드 정적 파일 빌드 중" "正在构建前端静态资源" "正在建置前端靜態資源" "フロントエンドの静的アセットをビルドしています" "Generando los recursos estáticos del frontend")"
  if [ "${ARTEX_SKIP_NPM_CI:-0}" != "1" ]; then
    (cd web && npm ci)
  fi
  (cd web && npm run build:static)
  info "$(msg "Syncing frontend to server/webui/dist" "프런트엔드를 server/webui/dist에 동기화 중" "正在将前端同步到 server/webui/dist" "正在將前端同步至 server/webui/dist" "フロントエンドを server/webui/dist に同期しています" "Sincronizando el frontend con server/webui/dist")"
  mkdir -p server/webui/dist
  rsync -a --delete web/out/ server/webui/dist/
fi

compress_binary() {
  binary="$1"
  goos="$2"
  case "$ARTEX_COMPRESS" in
    0|off|false|none)
      info "$(msg "Skipping UPX: $binary" "UPX 건너뛰기: $binary" "跳过 UPX：${binary}" "略過 UPX：${binary}" "UPX をスキップ: ${binary}" "Se omite UPX: ${binary}")"
      return 0
      ;;
    auto|required|true|1) ;;
    *) die "$(msg "ARTEX_COMPRESS must be off, auto, or required" "ARTEX_COMPRESS는 off, auto 또는 required여야 합니다" "ARTEX_COMPRESS 必须为 off、auto 或 required" "ARTEX_COMPRESS 必須為 off、auto 或 required" "ARTEX_COMPRESS は off、auto、required のいずれかにしてください" "ARTEX_COMPRESS debe ser off, auto o required")" ;;
  esac

  if ! command -v upx >/dev/null 2>&1; then
    if [ "$ARTEX_COMPRESS" = "required" ]; then
      die "$(msg "ARTEX_COMPRESS=required but upx was not found" "ARTEX_COMPRESS=required이지만 upx를 찾을 수 없습니다" "ARTEX_COMPRESS=required，但未找到 upx" "ARTEX_COMPRESS=required，但找不到 upx" "ARTEX_COMPRESS=required ですが upx が見つかりません" "ARTEX_COMPRESS=required, pero no se ha encontrado upx")"
    fi
    warn "$(msg "upx not found; retaining linker output: $binary" "upx를 찾을 수 없어 linker 결과 유지: $binary" "未找到 upx；保留链接器输出：${binary}" "找不到 upx；保留連結器輸出：${binary}" "upx が見つからないため、リンカーの出力を保持します: ${binary}" "No se ha encontrado upx; se conserva la salida del enlazador: ${binary}")"
    return 0
  fi

  before=$(wc -c < "$binary" | tr -d ' ')
  upx_args="$ARTEX_UPX_ARGS"
  [ "$goos" = "darwin" ] && upx_args="$upx_args --force-macos"
  # shellcheck disable=SC2086
  if ! upx $upx_args -- "$binary"; then
    if [ "$ARTEX_COMPRESS" = "required" ]; then
      die "$(msg "UPX compression failed: $binary" "UPX 압축 실패: $binary" "UPX 压缩失败：${binary}" "UPX 壓縮失敗：${binary}" "UPX 圧縮に失敗しました: ${binary}" "Error de compresión con UPX: ${binary}")"
    fi
    warn "$(msg "UPX does not support this format; retaining binary: $binary" "UPX가 이 형식을 지원하지 않아 바이너리 유지: $binary" "UPX 不支持此格式；保留二进制文件：${binary}" "UPX 不支援此格式；保留執行檔：${binary}" "UPX はこの形式に対応していないため、バイナリを保持します: ${binary}" "UPX no admite este formato; se conserva el binario: ${binary}")"
    return 0
  fi
  after=$(wc -c < "$binary" | tr -d ' ')
  ok "$(msg "UPX complete: $binary (${before} -> ${after} bytes)" "UPX 완료: $binary (${before} -> ${after} 바이트)" "UPX 压缩完成：${binary}（${before} -> ${after} 字节）" "UPX 壓縮完成：${binary}（${before} -> ${after} 位元組）" "UPX 圧縮完了: ${binary}（${before} -> ${after} バイト）" "UPX terminado: ${binary} (${before} -> ${after} bytes)")"
}

package_binary() {
  binary="$1"
  goos="$2"
  goarch="$3"
  package_name="scopeweaver-${ARTEX_BUILD_VERSION}-${goos}-${goarch}"
  package_root="${ARTEX_PACKAGE_DIR}/${package_name}"
  archive="${ARTEX_PACKAGE_DIR}/${package_name}.zip"

  command -v zip >/dev/null 2>&1 || die "$(msg "Packaging requires zip" "패키징에 zip이 필요합니다" "打包需要 zip" "封裝需要 zip" "パッケージの作成には zip が必要です" "Se requiere zip para crear el paquete")"
  rm -rf "$package_root" "$archive"
  mkdir -p "$package_root"
  cp "$binary" "$package_root/"
  # Package the platform supervisor so updates can restart the process.
  if [ "$goos" = "windows" ]; then
    cp start.bat "$package_root/"
  else
    cp start.sh "$package_root/"
    chmod +x "$package_root/start.sh"
  fi
  cp -R skills "$package_root/"
  cp config.example.json "$package_root/"
  cp README*.md CONTRIBUTING*.md PRODUCT.md DESIGN.md LICENSE THIRD_PARTY_NOTICES.txt CHANGELOG.md "$package_root/"
  if [ -d docs ]; then cp -R docs "$package_root/"; fi
  mkdir -p "$package_root/sidequestion" "$package_root/web/src/i18n"
  cp sidequestion/*.md "$package_root/sidequestion/"
  cp web/src/i18n/README*.md "$package_root/web/src/i18n/"
  # Ship the adapter sources without local dependencies, credentials or runtime state.
  mkdir -p "$package_root/adapters/agent"
  cp adapters/agent/package.json adapters/agent/package-lock.json adapters/agent/README.md adapters/agent/pi-extension.js "$package_root/adapters/agent/"
  cp -R adapters/agent/src adapters/agent/test "$package_root/adapters/agent/"
  mkdir -p "$package_root/adapters/opencode-sdk"
  cp adapters/opencode-sdk/package.json adapters/opencode-sdk/bun.lock adapters/opencode-sdk/README*.md "$package_root/adapters/opencode-sdk/"
  cp -R adapters/opencode-sdk/src adapters/opencode-sdk/test "$package_root/adapters/opencode-sdk/"
  mkdir -p "$package_root/log"
  cp log/changelog-*.md "$package_root/log/"
  (cd "$ARTEX_PACKAGE_DIR" && zip -q -r -9 "$(basename "$archive")" "$(basename "$package_root")")
  rm -rf "$package_root"
  ok "$(msg "Release archive: $archive" "릴리스 압축 파일: $archive" "发布压缩包：${archive}" "發行壓縮檔：${archive}" "リリースアーカイブ: ${archive}" "Archivo de distribución: ${archive}")"
}

build_target() {
  target="$1"
  case "$target" in
    */*) ;;
    *) die "$(msg "Invalid target: $target (expected OS/ARCH)" "잘못된 대상: $target (OS/ARCH 필요)" "目标无效：${target}（应为 OS/ARCH）" "目標無效：${target}（應為 OS/ARCH）" "無効なターゲット: ${target}（OS/ARCH 形式で指定）" "Destino no válido: ${target} (se esperaba OS/ARCH)")" ;;
  esac
  goos="${target%%/*}"
  goarch="${target##*/}"
  case "$goos" in
    linux|darwin|windows) ;;
    *) die "$(msg "Unsupported OS: $goos (use linux, darwin, windows)" "지원하지 않는 OS: $goos (linux, darwin, windows 지원)" "不支持的操作系统：${goos}（请使用 linux、darwin、windows）" "不支援的作業系統：${goos}（請使用 linux、darwin、windows）" "未対応の OS: ${goos}（linux、darwin、windows を使用してください）" "Sistema operativo no compatible: ${goos} (usa linux, darwin o windows)")" ;;
  esac

  binary_name="scopeweaver"
  [ "$goos" = "windows" ] && binary_name="scopeweaver.exe"
  if [ -n "${ARTEX_OUTPUT:-}" ] && [ "$ARTEX_RELEASE" != "1" ]; then
    output="$ARTEX_OUTPUT"
  else
    output="${ARTEX_OUTPUT_DIR}/scopeweaver-${goos}-${goarch}/${binary_name}"
  fi
  mkdir -p "$(dirname "$output")"

  info "$(msg "Building ${goos}/${goarch}, version ${ARTEX_BUILD_VERSION}" "${goos}/${goarch} 빌드 중, 버전 ${ARTEX_BUILD_VERSION}" "正在构建 ${goos}/${goarch}，版本 ${ARTEX_BUILD_VERSION}" "正在建置 ${goos}/${goarch}，版本 ${ARTEX_BUILD_VERSION}" "${goos}/${goarch} をビルドしています。バージョン ${ARTEX_BUILD_VERSION}" "Compilando ${goos}/${goarch}, versión ${ARTEX_BUILD_VERSION}")"
  GOSUMDB="$ARTEX_GOSUMDB" \
  CGO_ENABLED=0 \
  GOOS="$goos" \
  GOARCH="$goarch" \
  go build \
    -tags embedui \
    -trimpath \
    -ldflags "-s -w -buildid= -X main.version=${ARTEX_BUILD_VERSION}" \
    -o "$output" \
    ./cmd/artex

  compress_binary "$output" "$goos"
  if command -v file >/dev/null 2>&1; then file "$output"; fi
  if [ "$ARTEX_PACKAGE" = "1" ]; then package_binary "$output" "$goos" "$goarch"; fi
  ok "$(msg "Build complete: $output" "빌드 완료: $output" "构建完成：${output}" "建置完成：${output}" "ビルド完了: ${output}" "Compilación terminada: ${output}")"
}

write_checksums() {
  [ "$ARTEX_PACKAGE" = "1" ] || return 0
  checksum_file="$ARTEX_PACKAGE_DIR/SHA256SUMS"
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$ARTEX_PACKAGE_DIR" && for archive in *.zip; do sha256sum "$archive"; done > "$(basename "$checksum_file")")
  elif command -v shasum >/dev/null 2>&1; then
    (cd "$ARTEX_PACKAGE_DIR" && for archive in *.zip; do shasum -a 256 "$archive"; done > "$(basename "$checksum_file")")
  else
    warn "$(msg "sha256sum and shasum unavailable; skipping SHA256SUMS" "sha256sum과 shasum이 없어 SHA256SUMS 건너뛰기" "sha256sum 和 shasum 均不可用；跳过 SHA256SUMS" "sha256sum 和 shasum 均無法使用；略過 SHA256SUMS" "sha256sum と shasum が利用できないため、SHA256SUMS をスキップします" "sha256sum y shasum no están disponibles; se omite SHA256SUMS")"
    return 0
  fi
  ok "$(msg "Checksum file: $checksum_file" "체크섬 파일: $checksum_file" "校验和文件：${checksum_file}" "總和檢查碼檔案：${checksum_file}" "チェックサムファイル: ${checksum_file}" "Archivo de sumas de verificación: ${checksum_file}")"
}

mkdir -p "$ARTEX_OUTPUT_DIR"
if [ "$ARTEX_PACKAGE" = "1" ]; then mkdir -p "$ARTEX_PACKAGE_DIR"; fi

old_ifs="$IFS"
IFS=','
read -r -a targets <<< "$ARTEX_TARGETS"
IFS="$old_ifs"
[ "${#targets[@]}" -gt 0 ] || die "$(msg "ARTEX_TARGETS cannot be empty" "ARTEX_TARGETS는 비어 있을 수 없습니다" "ARTEX_TARGETS 不能为空" "ARTEX_TARGETS 不可為空" "ARTEX_TARGETS は空にできません" "ARTEX_TARGETS no puede estar vacío")"
for target in "${targets[@]}"; do
  target="${target//[[:space:]]/}"
  [ -n "$target" ] || continue
  build_target "$target"
done

if [ "$ARTEX_PACKAGE" = "1" ]; then
  write_checksums
  info "$(msg "Release archives generated in: $ARTEX_PACKAGE_DIR" "릴리스 압축 파일 생성 위치: $ARTEX_PACKAGE_DIR" "发布压缩包生成位置：${ARTEX_PACKAGE_DIR}" "發行壓縮檔產生位置：${ARTEX_PACKAGE_DIR}" "リリースアーカイブの生成先: ${ARTEX_PACKAGE_DIR}" "Archivos de distribución generados en: ${ARTEX_PACKAGE_DIR}")"
fi
