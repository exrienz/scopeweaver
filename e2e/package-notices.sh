#!/usr/bin/env bash
# Run the real release packaging path with tiny, deterministic fixture executables.
set -euo pipefail
repo="$(cd "$(dirname "$0")/.." && pwd)"
temp_root="$(cd "${TMPDIR:-/tmp}" && pwd -P)"
fixture="$(mktemp -d "$temp_root/scopeweaver-package.XXXXXX")"
cleanup() {
  case "$fixture" in
    "$temp_root"/scopeweaver-package.*) rm -rf -- "$fixture" ;;
    *) printf 'Refusing cleanup outside packaging fixture: %s\n' "$fixture" >&2 ;;
  esac
}
trap cleanup EXIT
export PACKAGE_FIXTURE_PYTHON="$(command -v python || command -v python3)"
cp "$repo/build.sh" "$repo/THIRD_PARTY_NOTICES.txt" "$fixture/"
mkdir -p "$fixture/bin" "$fixture/server/webui/dist" "$fixture/skills" "$fixture/docs" "$fixture/log" \
  "$fixture/adapters/agent/src" "$fixture/adapters/agent/test" \
  "$fixture/adapters/opencode-sdk/src" "$fixture/adapters/opencode-sdk/test"
for name in start.sh start.bat config.example.json README.md README.ko.md LICENSE CHANGELOG.md \
  adapters/agent/package.json adapters/agent/package-lock.json adapters/agent/README.md adapters/agent/pi-extension.js \
  adapters/opencode-sdk/package.json adapters/opencode-sdk/bun.lock \
  adapters/opencode-sdk/src/fixture.mjs adapters/opencode-sdk/test/fixture.mjs \
  log/changelog-v0.2.1.md skills/SKILL.md docs/guide.md adapters/agent/src/fixture.js adapters/agent/test/fixture.js; do
  printf 'fixture %s\n' "$name" > "$fixture/$name"
done
"$PACKAGE_FIXTURE_PYTHON" - "$repo" "$fixture" <<'PYDOCS'
import json
from pathlib import Path
import sys
repo, fixture = map(Path, sys.argv[1:])
languages = [locale["id"] for locale in json.loads((repo / "locale/definitions.json").read_text())["locales"]]
# Keep the fixture tied to maintained source guides, without shipping their source trees.
documents = {Path(name) for name in ("PRODUCT.md", "DESIGN.md", "adapters/agent/README.md")}
for directory in ("docs", "skills", "sidequestion"):
    documents.update(path.relative_to(repo) for path in (repo / directory).rglob("*.md"))
documents.update(path.relative_to(repo) for path in (repo / "web/src/i18n").glob("README*.md"))
for language in languages:
    suffix = "" if language == "en" else f".{language}"
    documents.update(Path(f"{name}{suffix}.md") for name in ("README", "CONTRIBUTING"))
    documents.add(Path("adapters/opencode-sdk") / f"README{suffix}.md")
    if language != "en":
        for guide in (repo / "docs/ko").rglob("*.md"):
            documents.add(Path("docs") / language / guide.relative_to(repo / "docs/ko"))
for document in sorted(documents):
    assert (repo / document).is_file(), f"Missing maintained document: {document}"
    target = fixture / document
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(f"fixture {document.as_posix()}\n")
(fixture / "expected-documents.txt").write_text("\n".join(path.as_posix() for path in sorted(documents)))
(fixture / "languages.txt").write_text(" ".join(languages))
for name in ("sidequestion/fixture.go", "web/src/i18n/fixture.ts", "web/node_modules/fixture.js",
             "adapters/opencode-sdk/node_modules/fixture.js", "adapters/opencode-sdk/.env",
             "adapters/opencode-sdk/.env.local", "adapters/opencode-sdk/provider-config.local.json",
             "adapters/opencode-sdk/.runtime/fixture.json", "adapters/opencode-sdk/.cache/fixture.json"):
    target = fixture / name
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text("source-only fixture\n")
PYDOCS
cat > "$fixture/bin/go" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
[[ "$1" == build && "$CGO_ENABLED" == 0 ]]
while [[ "$#" -gt 0 ]]; do
  if [[ "$1" == -o ]]; then printf 'fixture executable %s/%s\n' "$GOOS" "$GOARCH" > "$2"; exit 0; fi
  shift
done
exit 1
SH
chmod +x "$fixture/bin/go"
if ! command -v zip >/dev/null 2>&1; then
  # Git Bash may omit zip; only the fixture dependency is replaced, not build.sh.
  cat > "$fixture/bin/zip" <<'SH'
#!/usr/bin/env bash
"$PACKAGE_FIXTURE_PYTHON" - "$@" <<'PY'
from pathlib import Path
import sys
import zipfile
archive, root = sys.argv[-2:]
with zipfile.ZipFile(archive, "w", compression=zipfile.ZIP_DEFLATED) as output:
    for path in Path(root).rglob("*"):
        output.write(path, path.as_posix())
PY
SH
  chmod +x "$fixture/bin/zip"
fi
for language in $(cat "$fixture/languages.txt"); do
  (cd "$fixture" && PATH="$fixture/bin:$PATH" SCOPEWEAVER_LANGUAGE="$language" \
    ARTEX_BUILD_VERSION=v0.2.1 ARTEX_SKIP_FRONTEND=1 ARTEX_OUTPUT_DIR="dist-$language" bash ./build.sh --release)
  "$PACKAGE_FIXTURE_PYTHON" - "$fixture/dist-$language" "$repo/THIRD_PARTY_NOTICES.txt" "$fixture/expected-documents.txt" <<'PY'
import hashlib
from pathlib import Path
import sys
import zipfile
directory, notice, expected_documents = map(Path, sys.argv[1:])
archives = sorted(directory.glob("*.zip"))
assert len(archives) == 5, archives
for path in archives:
    with zipfile.ZipFile(path) as archive:
        root = path.stem
        assert archive.read(f"{root}/THIRD_PARTY_NOTICES.txt") == notice.read_bytes()
        for document in expected_documents.read_text().splitlines():
            member = f"{root}/{document}"
            assert member in archive.namelist(), f"Missing archived document: {document}"
            assert archive.read(member) == f"fixture {document}\n".encode()
        for name in ("adapters/opencode-sdk/package.json", "adapters/opencode-sdk/bun.lock",
                     "adapters/opencode-sdk/src/fixture.mjs", "adapters/opencode-sdk/test/fixture.mjs"):
            member = f"{root}/{name}"
            assert member in archive.namelist(), f"Missing SDK adapter payload: {name}"
            assert archive.read(member) == f"fixture {name}\n".encode()
        for name in ("sidequestion/fixture.go", "web/src/i18n/fixture.ts", "web/node_modules/fixture.js",
                     "adapters/opencode-sdk/node_modules/fixture.js", "adapters/opencode-sdk/.env",
                     "adapters/opencode-sdk/.env.local", "adapters/opencode-sdk/provider-config.local.json",
                     "adapters/opencode-sdk/.runtime/fixture.json", "adapters/opencode-sdk/.cache/fixture.json"):
            assert f"{root}/{name}" not in archive.namelist(), f"Unexpected archived source: {name}"
        target = root.removeprefix("scopeweaver-0.2.1-").replace("-", "/")
        binary = "scopeweaver.exe" if target.startswith("windows/") else "scopeweaver"
        assert archive.read(f"{root}/{binary}") == f"fixture executable {target}\n".encode()
for line in (directory / "SHA256SUMS").read_text().splitlines():
    digest, name = line.split()
    name = name.removeprefix("*")
    assert hashlib.sha256((directory / name).read_bytes()).hexdigest() == digest
print(f"PASS: {directory.name.removeprefix('dist-')}: all five archives contain exact notices, SDK adapter payload and unchanged fixture executables without local dependencies/configuration")
PY
done
