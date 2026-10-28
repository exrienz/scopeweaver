#!/usr/bin/env bash
# Exercise the real local installer without installing dependencies or starting services.
set -euo pipefail

repo="$(cd "$(dirname "$0")/.." && pwd)"
installer="${1:-$repo/install.sh}"
temp_root="$(cd "${TMPDIR:-/tmp}" && pwd -P)"
fixture="$(mktemp -d "$temp_root/scopeweaver-install.XXXXXX")"
cleanup() {
  case "$fixture" in
    "$temp_root"/scopeweaver-install.*) rm -rf -- "$fixture" ;;
    *) printf 'Refusing cleanup outside the installer fixture: %s\n' "$fixture" >&2 ;;
  esac
}
trap cleanup EXIT
cp "$installer" "$fixture/install.sh"
cp "$repo/.env.example" "$fixture/.env.example"
mkdir -p "$fixture/bin" "$fixture/web"

cat > "$fixture/bin/npm" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
printf 'npm %s\n' "$*" >> ../calls.log
case "$*" in
  ci) [[ "${INSTALL_FIXTURE_FAIL:-}" != ci ]] ;;
  'run build:static')
    [[ "${INSTALL_FIXTURE_FAIL:-}" != build ]] || exit 1
    mkdir -p out
    printf '%s\n' "$INSTALL_FIXTURE_CONTENT" > out/index.html
    ;;
  *) exit 1 ;;
esac
SH
cat > "$fixture/bin/go" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
printf 'go %s\n' "$*" >> calls.log
case "$*" in
  version) printf 'go version fixture\n' ;;
  'build -tags embedui -trimpath -o scopeweaver ./cmd/artex')
    [[ "${CGO_ENABLED:-}" == 0 ]]
    [[ -f server/webui/dist/index.html ]]
    ;;
  *) exit 1 ;;
esac
SH
cat > "$fixture/bin/curl" <<'SH'
#!/usr/bin/env bash
# Deterministic detection fixture; never contacts the host or network.
[[ "$*" == *'http://127.0.0.1:11434/api/tags'* ]] || exit 1
[[ "${INSTALL_FIXTURE_OLLAMA:-0}" == 1 ]]
SH
cat > "$fixture/bin/docker" <<'SH'
#!/usr/bin/env bash
printf 'docker %s\n' "$*" >> calls.log
SH
cat > "$fixture/start.sh" <<'SH'
#!/usr/bin/env bash
printf 'start\n' >> calls.log
SH
chmod +x "$fixture/bin/"* "$fixture/start.sh"

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  cat "$fixture/output.log" >&2
  exit 1
}
run_installer() {
  (
    cd "$fixture"
    export PATH="$fixture/bin:$PATH" SCOPEWEAVER_LANGUAGE="${INSTALL_FIXTURE_LANGUAGE:-en}"
    # Local install, existing PostgreSQL, default connection values, disposable DB name.
    printf '%b' "${INSTALL_FIXTURE_ANSWERS:-2\n1\n\n\n\n\nscopeweaver_test_install\n\n}" | bash ./install.sh
  ) > "$fixture/output.log" 2>&1
}
assert_output() {
  [[ "$(cat "$fixture/server/webui/dist/index.html")" == "$INSTALL_FIXTURE_CONTENT" ]] || fail 'incorrect frontend output'
  [[ ! -e "$fixture/server/webui/dist/out" ]] || fail 'frontend output was nested under dist/out'
  grep -Fxq 'go build -tags embedui -trimpath -o scopeweaver ./cmd/artex' "$fixture/calls.log" || fail 'embedded build did not run'
  grep -Fxq start "$fixture/calls.log" || fail 'startup did not run'
}

export INSTALL_FIXTURE_CONTENT='first frontend' INSTALL_FIXTURE_FAIL=''
run_installer || fail 'fresh local install failed'
assert_output
printf 'PASS: fresh checkout creates embedded frontend directory\n'

mkdir -p "$fixture/server/webui/dist/out"
printf 'stale\n' > "$fixture/server/webui/dist/stale.html"
export INSTALL_FIXTURE_CONTENT='replacement frontend'
: > "$fixture/calls.log"
run_installer || fail 'repeated local install failed'
assert_output
[[ ! -e "$fixture/server/webui/dist/stale.html" ]] || fail 'stale frontend output survived reinstall'
printf 'PASS: reinstall replaces stale output without nesting\n'

for stage in ci build; do
  export INSTALL_FIXTURE_FAIL="$stage"
  : > "$fixture/calls.log"
  if run_installer; then fail "npm $stage failure was ignored"; fi
  [[ "$(cat "$fixture/server/webui/dist/index.html")" == "$INSTALL_FIXTURE_CONTENT" ]] || fail "npm $stage failure replaced existing frontend"
  if grep -Eq '^go build|^start$' "$fixture/calls.log"; then fail "npm $stage failure reached build or startup"; fi
  printf 'PASS: npm %s failure preserves frontend and prevents build/start\n' "$stage"
done

export INSTALL_FIXTURE_FAIL=''
for language in en ko zh-Hans zh-Hant ja es; do
  export INSTALL_FIXTURE_LANGUAGE="$language" INSTALL_FIXTURE_OLLAMA=1
  rm -f "$fixture/config.json"
  export INSTALL_FIXTURE_ANSWERS='2\n1\n\n\n\n\nscopeweaver_test_install\n\ny\nqwen2.5-coder:7b\n'
  run_installer || fail "$language Ollama selection failed"
  python3 - "$fixture/config.json" <<'PYTEST'
import json,sys
c=json.load(open(sys.argv[1]))
assert c['llm']['provider']=='ollama'
assert c['llm']['base_url']=='http://127.0.0.1:11434'
assert c['llm']['model']=='qwen2.5-coder:7b'
assert c['llm']['context_window_k']==32
PYTEST
  cp "$fixture/config.json" "$fixture/config.saved"
  run_installer || fail "$language existing config reinstall failed"
  cmp "$fixture/config.saved" "$fixture/config.json" || fail 'existing configuration overwritten'
  printf 'PASS: %s explicit host Ollama selection and existing config preservation\n' "$language"
done

rm -f "$fixture/config.json"
export INSTALL_FIXTURE_ANSWERS='2\n1\n\n\n\n\nscopeweaver_test_install\n\nn\n'
run_installer || fail 'declining host Ollama failed'
if grep -q '"llm"' "$fixture/config.json"; then fail 'declined Ollama configured'; fi
printf 'PASS: detected host Ollama remains opt-in\n'

rm -f "$fixture/.env"
export INSTALL_FIXTURE_ANSWERS='1\nfixture-password\n\ny\nqwen2.5-coder:7b\n'
run_installer || fail 'Docker Ollama selection failed'
grep -Fxq 'SCOPEWEAVER_LLM_PROVIDER=ollama' "$fixture/.env" || fail 'Docker provider missing'
grep -Fxq 'SCOPEWEAVER_LLM_BASE_URL=http://host.docker.internal:11434' "$fixture/.env" || fail 'Docker host endpoint incorrect'
cp "$fixture/.env" "$fixture/env.saved"
export INSTALL_FIXTURE_ANSWERS='1\n'
run_installer || fail 'Docker reinstall failed'
cmp "$fixture/env.saved" "$fixture/.env" || fail 'existing .env overwritten'
printf 'PASS: Docker host endpoint and existing .env preservation\n'
