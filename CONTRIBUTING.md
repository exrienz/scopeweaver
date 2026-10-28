# Contributing to ScopeWeaver

English · [한국어](CONTRIBUTING.ko.md) · [简体中文](CONTRIBUTING.zh-Hans.md) · [繁體中文](CONTRIBUTING.zh-Hant.md) · [日本語](CONTRIBUTING.ja.md) · [Español](CONTRIBUTING.es.md)

Contributions are welcome: bug fixes, upstream ARTEX backports, dependency upgrades, documentation,
and tests.

## Before you start

- Read the [usage restrictions](README.md#license-and-disclaimer). Contributions must not add
  features that target real online systems or weaken those restrictions.
- For larger changes, open an issue first so we can agree on the approach.
- Keep each pull request focused on one topic. Smaller PRs are reviewed faster.

## Requirements for a pull request

- **Complete shipped languages and corresponding documentation.** User-facing text and maintained user/contributor guides need English, Korean, Simplified Chinese (`zh-Hans`), Traditional Chinese (`zh-Hant`), Japanese (`ja`) and Spanish (`es`):
  - server messages: keep `locale.Register(key, en, ko)` compatible and include additional shipped languages through `locale.RegisterAll` and the language-keyed resources in `locale/`
  - web UI: all current English keys in the EN/KO tables and `messages/{zh-hans,zh-hant,ja,es}.ts`
  - documentation: `README.md`, `CONTRIBUTING.md` and maintained guides with corresponding `ko`, `zh-Hans`, `zh-Hant`, `ja` and `es` editions; preserve executable English skill entries
- **Tests for changed behavior.** Add or update tests that would fail without your change. Do not
  disable, skip, or loosen existing checks to make them pass.
- **Disposable databases only.** Tests that write to PostgreSQL must use a fresh database named
  `scopeweaver_test_*` (or `scopeweaver_adapter_*` for the agent adapter E2E test).
- **Credit upstream work.** For ARTEX backports, add `Upstream-commit: mhtsec/ARTEX@<sha>` and a
  `Co-authored-by:` trailer for the original author, and update `docs/PROVENANCE.md` and
  `docs/ko/PROVENANCE.md`.
- **License.** Contributions are accepted under the repository's [AGPL-3.0](LICENSE) license.

## Adding or changing a locale

Follow the [locale guide](web/src/i18n/README.md): update the shared `locale/definitions.json`
(canonical ID, native name, Intl mapping and aliases), matching vectors, complete frontend/backend
catalogs, date-fns mapping, focused tests and browser smoke. Bare `zh`, `zh-CN` and `zh-SG` map to
`zh-Hans`; `zh-TW`, `zh-HK` and `zh-MO` map to `zh-Hant`. An explicit script wins a conflicting region.
Keep English as the source for keys, meaning, placeholders and rich tags. Backend runtime English/key
fallback is a compatibility safeguard, not proof that a language is ready to ship. Do not translate
user content, drafts, raw evidence, identifiers or historical data. Review translations and record
concerns; do not claim native-speaker approval without evidence. Keep maintained feature guidance complete in all six languages, including the English/Korean pair. Keep the standalone adapter canonical-ID list and root shell/batch message helpers synchronized too. Run `node e2e/script-locales.mjs` and the native Windows fixture `node e2e/windows-launcher.mjs`/Windows CI; packaged adapters must not depend at runtime on repository-root JSON.

## Checks to run

These match the CI jobs in [.github/workflows/ci.yml](.github/workflows/ci.yml):

```bash
# patched Norma SDK (a separate Go module)
(cd third_party/norma && go test ./... -count=1)

# backend database-backed suites (db, server, agent): against a throwaway database
# that is created and dropped for you. They require ARTEX_TEST_DSN and skip (never
# fall back to the live DSN) without it — see docs/VERIFICATION.md.
make test-db

# backend: every Go package (set ARTEX_TEST_DSN to a disposable scopeweaver_test_* database)
ARTEX_TEST_DSN=postgres://sw_user_test@/scopeweaver_test_1?host=/tmp&sslmode=disable go test ./... -count=1

# frontend
cd web && npm ci && npm test && npm run build:static

# agent adapter (and its real-backend E2E test against a scopeweaver_adapter_* database)
cd adapters/agent && npm ci && npm test && npm run check
SCOPEWEAVER_ADAPTER_E2E=1 go test ./server -run '^TestAgentAdapterE2E$' -count=1
```

The `e2e` workflow also runs an essential browser smoke test against the real binary on every PR and
after merge. To run it locally against a running server and a fresh `scopeweaver_test_*` database
(requires `psql` and the embedded binary built from the same checkout):

```bash
cd e2e && npm ci && npx playwright install chromium
SCOPEWEAVER_URL=http://127.0.0.1:8787 ARTEX_PG_DSN=postgres://.../scopeweaver_test_e2e \
  SCOPEWEAVER_BINARY=/absolute/path/to/scopeweaver npm run smoke
```

After the existing smoke checks initialize the password, the dashboard regression starts its own
isolated copy of that binary on a free loopback port against the same test database. It checks
English/Korean Legacy/New token stacks and profile filters, then stops its server and removes its
fixtures. Dashboard evidence and the child server log are saved in `e2e/e2e-results/dashboard-token-chart/`.

The essential smoke covers all six languages for preference restore, setup/login/task/settings/chat drafts,
first API language, SSE reconnect/filter preservation and rendered number formatting.

Describe in the PR what you ran and anything you could not run.

## Review and merging

Create each `feat/*` or `fix/*` branch from the latest `dev`, and open its PR against `dev`.
After integration, wait for all checks and E2E on that exact `dev` commit, then open a promotion
PR from this repository's `dev` to `main`. Keep promotion history with a merge commit so `dev`
remains an ancestor of `main`. Verify the checks again after promotion; start subsequent work
from `dev`. The `branch-policy` check rejects other PR routes.

1. All required checks must pass on the current candidate: `branch-policy`, `frontend`,
   `agent-adapter`, `backend`, `windows`, `chat-browser`, and `smoke`. Both `dev` and `main` run
   CI and E2E after merge. For first-time contributors, the maintainer approves the CI run.
2. The PR is reviewed against its diff for correctness, security, complete shipped translations, and six-language documentation parity.
3. **The maintainer performs a final human check of every PR before merging.** Nothing is merged
   without it.

Releases are cut by the maintainer after a full end-to-end test before and after merging.
