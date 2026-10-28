# ScopeWeaver provenance and modifications

English · [한국어](ko/PROVENANCE.md) · [简体中文](zh-Hans/PROVENANCE.md) · [繁體中文](zh-Hant/PROVENANCE.md) · [日本語](ja/PROVENANCE.md) · [Español](es/PROVENANCE.md)

ScopeWeaver is a standalone derivative of [Autumn-27/ARTEX](https://github.com/Autumn-27/ARTEX), based on commit `160fe13`. It is maintained separately at [cskwork/scopeweaver](https://github.com/cskwork/scopeweaver), without a GitHub fork-network relationship.

The ScopeWeaver modifications dated **2026-10-02** cover the product name, English-default/Korean-selectable interface and messages, documentation and bundled agent guidance, setup/build/start/update/password-reset prompts, report and agent language handling, and distribution naming. The executable is `scopeweaver` (`scopeweaver.exe` on Windows); archive names follow `scopeweaver-<version>-<os>-<arch>.zip`. Docker Compose builds the checked-out source locally. The configured release workflow targets this repository and GHCR.

The original `LICENSE` is retained unchanged. Original authorship, contributor acknowledgments and upstream changelog history remain attributed to ARTEX. The Go module `github.com/Autumn-27/artex`, source entry point `./cmd/artex`, `ARTEX_*` configuration contracts, `artex` database defaults and `ARTEX` login username remain for compatibility.

This modification date is not a release tag. It does not assert that binary releases or container images have been published. Historical validation documents under `sidequestion/` describe upstream runs; they are not new ScopeWeaver verification receipts. Translated historical JSON preserves technical identifiers and measurements while labeling translated narrative.

Modifications on **2026-10-15** add an optional native Ollama profile format (`/api/chat` with `num_ctx`, `keep_alive` and `num_predict`), three new-form Ollama templates, keyless local profiles, per-profile concurrency, Ollama-aware pre-warm, compaction and approval-judge defaults, and an opt-in `local_llm_only` LLM network policy. `SCOPEWEAVER_LLM_*` and `SCOPEWEAVER_LOCAL_LLM_*` variables are added as aliases that take precedence over the retained `ARTEX_*` contracts. Existing Anthropic, OpenAI, OpenAI Responses, custom URL, proxy and GLM template behavior is preserved, and the policy is off by default.

The follow-up native hardening candidate on **2026-10-15** adds urgent next-slot admission without preemption, actual native-origin judge selection with schema-constrained output and 90-second/ask defaults, and 72% compaction with 16/32 KiB tool-output caps through 64K. It narrows opt-in strict destinations to loopback plus explicit inference CIDRs (including LAN/Docker migration), refuses redirects/proxies, and blocks all agent search and update network actions while retaining local status/rollback. Read-only GPU-memory estimates and shared-model warnings, safe judge-settings loading, and English/Korean/Simplified Chinese interface text accompany the change. These are application controls, not whole-host offline or model-quality guarantees; no dependency, migration, automatic model pull or release is introduced. See the [provider guide](llm-providers.md) and [candidate verification](VERIFICATION.md#native-hardening-candidate--2026-10-15).

Additional modifications on **2026-10-03** complete the server translation audit and add GLM-5.3 configuration templates plus its required always-enabled reasoning compatibility. Templates use the existing provider adapter and do not create or activate a profile automatically.

The subsequent upstream fixes from ARTEX commits `e2c8a23`, `86729b6`, and
`a951e4a` are backported here: same-origin production SSE, separate approval-model
token accounting, and fail-closed authentication with atomic initial password
creation and bounded database pooling. New messages and deployment guidance are
available in English and Korean. The original authorship is credited in the
backport commits; the other ScopeWeaver changes remain in place.

## Norma execution and compaction fixes — 2026-10-25

The repository-owned replacement in `third_party/norma` starts from
[`52f17dd52d95b087265e5d5b306fe85c1ea05599`](https://github.com/Autumn-27/Norma/commit/52f17dd52d95b087265e5d5b306fe85c1ea05599).
That commit adds only the Apache-2.0 license to v0.4.3
(`320ae32989da97dd91469bb89164cbfe87d83d5d`): 211 files / 1,293,448 bytes before
local changes. There is no upstream NOTICE file in this snapshot.

The local fixes retain an executor's originating timeout context through
settlement, repair the retry test's missing client, scale small-window default
compaction pacing, and prevent meaningful summary consolidation from starving.
They also learn only explicit smaller context limits, keep learned ceilings from
increasing, and distinguish output-length errors from context overflow. Raw
history and archives remain intact; irreducible views still fail explicitly.
The stale small-window limitation test becomes a positive consolidation test.

`third_party/norma/README.scopeweaver.md` lists all ten modified upstream files
and four additions. Each modified upstream file carries a prominent modification
notice. Other upstream files, the original license and module metadata are
unchanged. Exported signatures and serialized formats are unchanged; the listed
behavioral fixes are intentional.

The root module replacement and Docker dependency-copy order select this source.
The complete Apache license is also included in `THIRD_PARTY_NOTICES.txt` for
existing archive/container packaging. The backend CI job runs the full nested
SDK suite separately; root package traversal does not include it. No upstream
repository publication or database migration is included.

The host-owned `enableNoa` fix forwards the already configured context window,
including 250K and 1M, rather than silently replacing it with the SDK's 200K
fallback. Output-token limits, disabled Noa and native Ollama policies are
separate and retained. See the paired [verification guide](VERIFICATION.md).
