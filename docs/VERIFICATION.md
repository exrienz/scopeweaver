# Verification and known limitations

English · [한국어](ko/VERIFICATION.md) · [简体中文](zh-Hans/VERIFICATION.md) · [繁體中文](zh-Hant/VERIFICATION.md) · [日本語](ja/VERIFICATION.md) · [Español](es/VERIFICATION.md)

## Current-dev PR #40 reconstruction — 2026-10-25

Only the four intended commits from `aebea3e..9b04754` were first rebuilt on `f354c920`; divergent branch history was not replayed. The candidate was then rebased onto `7af06f1`, preserving its six-language integration. Current integration requires exactly **1,987** production keys with raw English, Korean, Simplified Chinese, Traditional Chinese, Japanese and Spanish translations. Existing OpenCode environment-key isolation, session-header precedence and User-Agent defaults are retained.

Broader cancellation/race coverage exposed an existing unsynchronized planner result: the streaming session can return during pause while the real `prove_goal` handler is still setting `GoalMet` and its reason. Both terminal tool writers and the planner's result snapshot now share a private mutex. This preserves the exported fields, authentic tool/terminal receipts and current pause-versus-completion precedence; it adds no provider request or production schema change.

The terminal-proof gate also requires an **authentic successful SDK tool-result receipt** matching the marked tool-use ID and originating model response. The private capture observer runs before the next compaction or inference boundary; handler completion or a `PostToolUse` hook alone cannot qualify it. The adversarial regression used real `goal_met`, a **30 ms** run budget and a **120 ms** post-tool hook: the old gate swallowed settlement after an interrupted receipt, making **1 provider call instead of 2**, while the native reason remained `timeout` and the error remained nil. The corrected path keeps that interrupted receipt, permits the real settlement call and its compaction, and rejects late-hook qualification. A second synchronized regression holds the hook until capture observes the interruption, proving the real handler ran before the timeout receipt; normal successful proof, parent cancellation, reactive overflow and late-origin guards remain covered. See [terminal-proof regressions](../agent/terminal_proof_test.go).

The initial reconstruction still had an inherited shutdown gap: on untouched `f354c920`, `TestMainServeFailureRunsShutdown` failed **3 of 10** isolated runs, including scheduler writes after database close. The follow-up repair registers scheduler ownership before launch. `Manager.Close` cancels and joins schedulers outside the manager lock for at most **5 seconds** before stopping the log writer and closing the database. If scheduler SQL remains blocked, Close returns `context.DeadlineExceeded` and **keeps the database open**; after the work finishes, a retry can close it. The CLI already reports a close failure and exits 1. This does not claim that existing non-context SQL calls become cancelable. A pre-canceled scheduler no longer initializes watermarks or starts another tick. The [held-SQL regression](../server/scheduler_shutdown_test.go) checks concurrent close timeouts, an open database while work is active, and successful concurrent retries after release. A first parallel all-package run also failed the log-sink test's global row-count assertion while other package subprocesses used the same database; cross-package interference is a hypothesis, so final verification follows CI's separate disposable database per package. These historical receipts do not claim final exact-head hosted CI, native Windows or live-provider verification.

The integrated candidate uses web Next.js **16.4.3** and Zod **4.6.7**, and adapter Zod **4.6.7**, with compatible lockfile fixes for body-parser **2.4.0**, fast-uri **3.1.9**, ip-address **10.7.4** and qs **6.16.1**. The adapter SDK pin remains **1.32.0**. After the six-language rebase, fresh `npm ci` passed for both packages; web tests passed **75/75**, the cold static build exported **31 pages**, adapter tests passed **18/18** and its syntax check passed. The earlier 72/7-test receipts remain historical evidence. Both current `npm audit --audit-level=low` checks report **0 vulnerabilities**. These results supersede the initial Next.js 16.4.0 advisory/build-failure limitation; the original receipts remain historical evidence. Native paste/undo browser keys now use `ControlOrMeta+V` and `ControlOrMeta+Z`, preserving assertions and timeouts. Final canonical browser runs and exact-head hosted CI are separate gates and are not claimed here.

Compatibility checks after the six-language integration must cover prior-stock planner/worker prompts and `add_intent` schemas from every shipped source language and each requested output language. Additional zh-Hant/ja/es fixtures come from immutable pre-PR40 dev `7af06f1`. Exact recognition must preserve edited prompts, customized schema fields/defaults, stored rows and raw history. This extends the original three-language verification described below.

### Patched SDK and model-specific context windows

The nested Norma suite is required in the backend CI job. Run
`(cd third_party/norma && go test ./... -count=1)` separately from root packages;
see [source and modification notices](PROVENANCE.md#norma-execution-and-compaction-fixes--2026-10-25).

The regression set covers the original 30 ms / 120 ms interrupted-tool race,
small-window pacing and long-session bounds, worthwhile summary consolidation,
actual overflow retry views, monotonic learned ceilings and output errors that
must not trigger context recovery. Existing numeric bounds and original history
are retained. The old assertion that 40K cannot reach tier two was already false
on unchanged upstream; its replacement requires actual consolidation, token
reclamation and bounded summaries using the same fixture.

`TestNoaRetainsConfiguredModelContextWindow` exercises actual Noa sessions at
40K, 128K, 200K, 250K, 400K and 1M. Before the host fix, all non-200K cases became
200K. `enableNoa` now passes the positive resolved profile window before clearing
the replaced compactor. Separate tests cover missing-window fallback, disabled
Noa, damaged-state recovery and the existing local Ollama policy. `MaxTokens`
continues to limit output independently. The 200K fallback is for an unset
window; it is not a universal model limit. SDK default values at 200K and above
remain unchanged, while worthwhile-summary count eligibility intentionally
changes across windows.

## Trigger evidence in the first planner request (2026-10-18)

Local candidate on original head `ffe260435fed6576394d613991aaf6e8010d3f48`, with the shared CLI correction backported. Normally a finding or completed-worker event wakes the planner with trigger summaries and a compact `graph_overview`; `prove_goal` still requires the recorded evidence for every goal condition. The compact entries lacked full finding/fact fields, so reading `node_detail` before proof was legitimate.

Abridged actual change in [Planner.Plan](../agent/planner.go), with unchanged intervening task-context setup omitted:

```go
// Before
situational := renderTriggers(ts, triggers) + renderGraphOverview(tsx.graphOverviewData())
// After
overview := tsx.graphOverviewData()
situational := renderTriggers(ts, triggers) + renderGraphOverview(overview)
// After ensureRunDir and WithReviewContext:
situational += renderPlannerEvidence(ctx, overview, triggers, tools, ts.ID(), taskDir)
```

The first new line captures the existing bounded overview once; the second renders that same snapshot. The last line appends full detail for its current-exploration fact/finding candidates yielded by concrete `finding`/`done` trigger intents, after task/review context exists. It does not traverse `related_tasks` or expand global lineage. The [attachment implementation](../agent/planner_evidence.go) makes at most **4 read attempts** and admits whole nodes only within **32,768 UTF-8 bytes for the entire attachment**, including its localized header, JSON envelope and `omitted_count`; an oversized node is omitted rather than truncated. Repeated candidates are deduplicated.

Only the final resolved `node_detail` capability may read: its schema must accept the input, `IsReadOnly` must hold, and permissions must explicitly allow an unchanged input. Wrappers, defaults, metering, restricted results and errors remain effective; unsafe or incomplete results are omitted with no raw-helper fallback. The header is complete in EN/KO/zh-Hans and labels recorded content as untrusted observations. Existing explicit reads, including associated direct-source nodes, retain their original access rules.

Concrete row from the **CLI-owned synthetic loopback R22 lab**, not an original-production incident: goal `2` remained open after intent `3` completed one owned GET and persisted fact `4` plus finding `5`. The goal required the exact finding name `SYNTHETIC LAB ONLY public debug marker`, severity `low`, marker and observations. At **2026-10-17 22:42:31+09:00**, the planner legitimately read `node_detail` `5` and `4` for those missing full fields. Harness cleanup paused it at **22:42:54+09:00** before proof; this was not a human pause. R22 failed after **470.50 seconds overall** under the unchanged **420-second task budget**. The underlying model/provider latency cause is **unknown**. With the correction, eligible full details for those same snapshot candidates reach the first request of the triggered round; they do not establish proof by themselves.

| Local candidate check | Observed result |
| --- | --- |
| Original complete `agent` package, exact production source without overlay | 598 tests/subtests passed, 0 failures/skips; CLI equivalent 607 passed, 0 failures/skips |
| New regressions in three `planner_evidence*test.go` files | 31 test/subtest passes; actual `Planner.Plan` streaming/nonstreaming first-request equality, nine permission cases, count/byte bounds, deduplication/isolation, missing/deleted/late/hidden evidence, explicit recovery, no automatic proof and multiple goals |
| Original focused server proof/locale checks and complete `locale` package | 5 and 19 passed respectively, 0 failures/skips; strict completeness retained for **1,986 keys × 3 languages**, updating the expected count from 1,985 for the new header |
| Original `go build ./...`, `go vet ./...` | Both native exit 0 |
| Linux `-race`, current equivalent CLI source | Focused checks 49 passed and complete `agent` package 607 passed, 0 failures/skips; both actual Go exits 0, no `DATA RACE`. All 1,175 snapshot source files verified; this is not an original-module run |
| Real packaged CLI loopback acceptance, all five cases | Native exit 0 for core 19.14s, GLM LOW 82.63s, Flash LOW 114.84s, OAuth LOW 108.12s and Flash HIGH 317.83s, each at the unchanged 420-second task budget. These single lab runs are not a performance comparison |

The unchanged original baseline failed the streaming and nonstreaming missing-context assertions (two leaf failures plus their parent), without setup failure. Controlled-provider usage in regression fixtures is synthetic, not paid-token evidence. Real `prove_goal`, authentic usage/receipts and the existing terminal proof gate remain required; phase, worker, effort, deadline and retry behavior are unchanged. Counterexample: a disabled, restricted, missing, oversized, untriggered or inherited node receives no automatic full detail. When available, an explicit read can recover omitted/fresh evidence; reading data alone cannot complete a goal, and proving one of multiple goals cannot complete the task.

These are local working-tree results, not exact-new-head CI. The unchanged Windows `TestChatBranchCopyRejectsDestinationSymlink` privilege failure also reproduced on CLI baseline `650dc82910e5e36668161a5276455506f64713fe`; no check was weakened, and full native-server success is not claimed. All seven required hosted checks on the new candidate remain pending. PR #40 remains unmerged with final human review pending; this section claims no publication or release. No public API, schema or migration changes are introduced. Earlier dated sections retain their historical results.

## Chat context readiness (2026-10-17)

Local UI candidate on top of PR #40 head c9fb642. The inspector can open before its initial controls request finishes. Previously it identified an unresolved session as draft, then treated the first resolved session as a switch and closed the panel. The Chinese desktop task case failed with an unstable, then detached close button in [checks run 39031048389](https://github.com/cskwork/scopeweaver/actions/runs/39031048389). That hosted head remains failure evidence; these new results are local.

Abridged reset logic (binding is the requested controls key; session is its resolved identity):
~~~ts
// Before: every change from draft to resolved identity ran the reset.
epoch.current++;
setOpen(false);
// After: qualify the change before the existing reset.
const previous = hydrated.current;
hydrated.current = { binding, session };
if (previous.binding === binding && !previous.session) return;
epoch.current++;
setOpen(false);
~~~
The first added line reads the previous binding and session; the second remembers the current pair. The third preserves an opened panel only while the same binding has not yet resolved its first session. A different requested binding or a later resolved-session change still reaches the epoch bump and full view/form reset, including focus and retention count. The hook exposes its existing requested key; no server API or translated label changes.

Concrete controlled row: task 1, exploration 1, segment 0 was paused through the real task API. Its first controls response was held until /context had opened. Releasing that real response previously closed the trigger and made the normal click time out after 15 seconds (Go exit 1); the byte-identical browser oracle now retains the open trigger and closes normally (exit 0), with zero provider calls. Task pause is distinct from the chat gate's paused field.

| Local check on frozen fixed source | Observed result |
| --- | --- |
| Existing frontend tests | 65 passed, 0 failures/skips |
| Fresh static frontend build | TypeScript and 31 pages completed; verified native exit 0 |
| Unchanged independent delayed-first-response oracle | 1 Go test passed, 0 failures/skips; zero provider calls |
| Canonical context browser fixture | All 12 EN/KO/zh-Hans × 1440/390 × standalone/task cases plus hydration and requested-binding guards passed (14 browser steps; 1 Go test, 0 failures/skips) |

The added guard uses the real first backend response, then synthetic route state for a later resolved-session change and a held old context response. Every replacement response stays held until the stale-content assertion, preventing a fresh result from hiding a leak. Nondefault focus and retention 7 reset to empty focus and default 2. Counterexample: a genuine requested binding or later resolved session must still reset; late results must not restore old context. A second real task-query switch (task 7 to 8 in the full run) observed the new controls URL while its response was held, retained the identical connected inspector trigger and document, reset focus/retention and rejected the late old context before replacement data could mask it. This proves a mounted requested-binding change rather than a remount. Late error/compaction-success branches remain source-reviewed rather than separately exercised. The synthetic identity guards prove client handling, not server exploration replacement. Other context cases use the controlled loopback provider; no paid/external provider was called.

Initial shell-wrapper attempts stopped on environment/capture setup; final build/test exits are recorded separately with warnings retained and no source/assertion workaround. Prior planner/shutdown changes are preserved. Paired documentation records these local results; independent final review and all seven hosted checks on the next committed head remain required. PR readiness is not maintainer approval, merge or a public release.

## Log writer shutdown ownership (2026-10-17)

The server keeps logs in memory and persists them through an asynchronous database writer. Canceling the server context previously did not join that writer: an insert using a background context could remain active while `Manager.Close` closed its pool. The latest backend run on PR #40 head `14b5403` exposed this in the unchanged occupied-port shutdown test.

Abridged flow across the writer and manager:
```go
// Before: writer, then manager shutdown
pg.InsertLog(level, tag, text) // background context
return m.pg.Close()           // no writer join
// After: writer, then manager shutdown
pg.InsertLogContext(writeCtx, level, tag, text)
logSink.StopDB(m.pg)
return m.pg.Close()
```
The first new line makes the actual insert cancelable; the initial history restore uses the same context. The second detaches only this exact pool, cancels outstanding SQL and waits for writer completion outside the sink lock. Every same-owner stop shares that completion; another manager's pool cannot stop it. Only then does the third line close the pool. Existing context-free DB methods delegate to the same SQL; ring, stderr and SSE delivery remain available, and the enqueue channel is never closed.

One observed row: backend run [39028154916](https://github.com/cskwork/scopeweaver/actions/runs/39028154916), job `117383132313`, failed `TestMainServeFailureRunsShutdown` at **2026-10-17 11:18:45 UTC**. The expected occupied-port bind failure was followed by shutdown and `[logsink] db write: sql: database is closed`. Independently, a real PostgreSQL table lock reproduced `Manager.Close` returning with its writer still blocked on unchanged `14b5403` (Go exit 1). The exact same fixture now passes while the lock remains held, before pool close; no diagnostic is filtered.

| Original local R19 check | Observed result |
| --- | --- |
| Unchanged independent blocked-insert regression | 1 passed, 0 failures/skips; baseline behavioral RED retained |
| Genuine SQL-rejection guard | 1 passed, 0 failures/skips; SQLSTATE `23514` visible, rejected row absent and later valid row persisted |
| Frozen `TestStopDB*` fixtures | 6 passed, 0 failures/skips: no binding, blocked insert, foreign pool, canceled context, concurrent stops and stderr continuity |
| Unchanged occupied-port shutdown test | 1 run plus 20 repeated runs passed; expected child failure retained, parent Go exit 0 |
| `go build ./...`, `go vet ./...` | Both exit 0 |

Counterexample: a genuine live-context SQL rejection must still be reported; cancellation/join does not turn it into a successful insert. This remains best-effort logging, without a new flush guarantee for queued lines. The concurrent fixture checks aggregate completion; per-waiter shared-join behavior is also verified in source review, rather than claimed as a separate runtime oracle. All database runs use disposable test databases; independent fixtures now use owned schemas in the supplied disposable database and need no CREATE DATABASE privilege. Production privileges, HTTP/schema contracts and existing planner tests are unchanged. These are local results, not a full native-server or release claim: the inherited Windows symlink privilege limitation remains, and all seven hosted checks must pass on the next committed candidate.

## Terminal goal completion (2026-10-17)

These are local R17 candidate results; final independent source review and the exact-candidate hosted gates are separate. The R17 shared correction keeps a completed proof turn on the normal SDK path, then ends the planner locally before another upstream request. Normally `prove_goal` links a confirmed fact/finding to its goal and marks that goal met; only proof of every goal sets `GoalMet`. Previously `Plan` waited for `captureRun` to finish before returning that flag to the engine, so the engine could not complete while the session waited on another model response.

Abridged control flow (`...` is the unchanged activity callback):
```go
// Before
_, _, err = captureRun(ctx, opts, input, ...)
return tsx.GoalMet, tsx.Reason, err
// After
installTerminalProofGateOn(&opts, tsx)
_, _, err = captureRun(ctx, opts, input, ...)
return tsx.GoalMet, tsx.Reason, err
```
The first old line waits for the whole session; the second exposes its goal result only afterward. The added line installs the next-request boundary before capture starts. It lets the current response, actual tool result, late usage and transcript finish, then supplies a content-free local `end_turn` with zero upstream usage and skips unnecessary post-proof `Compactor.Pre`. It preserves the selected compactor, optional `ContextView`, normal pre-proof/recovery behavior and existing provider/tool wrappers. Original `ToolUseID` and successful/failed response qualification prevent a delayed handler from attributing an old failed proof to a later successful response. The newest proof origin is retained with compare-and-swap; an older or unknown origin cannot erase a newer valid proof. Failure bookkeeping runs before yielding an error, including when its consumer stops immediately.

Concrete lab row: in the **standalone CLI R11 candidate**, not an original-production incident, finding node `5` proved goal `2` at **2026-10-17 18:36:24+09:00**. Model output was still active when the unchanged 420-second harness paused at **18:37:48+09:00**, 84 seconds later. That run had exactly one GET, one intent and one finding. The receipt is `scopeweaver-cli/dist/pi-glm-flash-fix-20261017/r11-flash-high-digest.txt`; it demonstrates the completion wait, not an underlying provider-latency cause. Counterexample: partial/rejected proof must continue normally; a failed proof response retains its real error/recovery path and cannot qualify local completion.

Original-source proof uses controlled providers and disposable PostgreSQL, with no paid inference:

| Local candidate check | Observed result |
| --- | --- |
| Complete `agent` package, R17 with corrected owned-fact fixture | 567 tests/subtests passed, 0 failures, 0 skips; includes streaming/nonstreaming, partial/rejected proof, parent cancellation, error/recovery, compaction/view and delayed-origin cases |
| `TestWireTools`, stock-runtime S7 and actual-engine S18 | 28 passed, 0 failures, 0 skips; S18 persists done after one real provider call, authentic proof/terminal receipts and exact input/output/cache-read/cache-write usage **11/7/3/2**, also in the raw transcript |
| Real pause/provider-error guards in S18 | Honest error/cause receipts retain emitted usage **11/0/3/0** for user pause and **11/7/3/2** for provider failure; post-proof pause-versus-task-completion status precedence is observed without a new policy assertion |
| Complete `locale`, build and vet | 19 tests passed, 0 failures, 0 skips; `go build ./...` and `go vet ./...` exit 0 |

The saved original S18 baseline failed because it attempted provider call 2 and had no terminal result/completion; its cancellation/provider-error receipt guards passed. The independent immutable R14 regression also failed with calls=2 instead of 3 when an old failed handler borrowed a later response's identity; the candidate retains the three-call regression. The reviewer then reproduced the converse order on the original R16 candidate: an old failed response-1 handler finished after a valid response-2 proof and erased its origin, causing an unnecessary third request. The unchanged actual-session overlay failed with calls=3 before R17 and now passes with calls=2/origin=2 and one authentic new proof result. Both delayed-handler orderings pass in the [boundary regressions](../agent/terminal_boundary_test.go); the [actual engine regression](../server/engine_terminal_proof_test.go) separately preserves completion/receipt/usage checks. The first integrated agent run failed only because a new fixture expected `CreateExploration` to create an origin fact. R16 seeds its own confirmed fact; all assertions and production code are unchanged by that fixture correction. Earlier failing receipts remain retained.

S18 now reuses S7's owned-schema helper inside the supplied disposable database, closing stores before dropping its own schema; a database-owning NOCREATEDB test role is sufficient and production privileges are unchanged. The unchanged `TestChatBranchCopyRejectsDestinationSymlink` still fails locally at `server/chat_history_test.go:270` because Windows denies symlink creation. Full native-server success and the seven exact-candidate hosted checks are not claimed; CI/frontend/browser delivery remains pending. Root owns separate race/live acceptance; this section reports only these original local checks.

## Planner/tool contract and stored-stock compatibility (2026-10-17)

Local Windows results for `fix/glm-agent-contracts`, based on `dev` commit `d7365ec`. The planner's `parent_ids` guidance now matches the existing fact/finding-only handler, and batched `add_intent` metadata describes its fields and required `summary`. Planner/worker guidance preserves fresh validation and retests, requires evidence for every goal condition, and ends rounds only after successful dispatch or confirmed final-goal proof. EN/KO/zh-Hans resources are paired.

Exact previous stock planner/worker prompts and `add_intent` schemas resolve to corrected runtime defaults. User edits, bindings, enabled flags, stored rows and prompt history remain unchanged; there is no production database migration or handler-semantic change. See [contract regressions](../agent/planner_contract_test.go), [stock compatibility regressions](../agent/legacy_stock_compat_test.go), and [HTTP/runtime PostgreSQL proof](../server/tools_stock_schema_test.go).

Tests-first proof compiled and produced **8 top-level agent assertion failures** and **9 S7 stock-schema subcase failures** on the baseline. The temporary unchanged-tail extraction and pass-through resolver used only to compile the tests were removed before applying the fix. Candidate results:

| Local check | Observed result |
| --- | --- |
| Complete `agent` package with disposable PostgreSQL | 549 tests/subtests passed, 0 failures, 0 skips |
| Complete `locale` package | 19 passed, 0 failures, 0 skips |
| Focused server locale/tool/provenance checks | 11 passed, 0 failures, 0 skips; 1,985 production keys retain raw three-language completeness checks |
| `TestExplorationDirectSourceReadView` | Passed with disposable PostgreSQL |
| `TestStockMetadataRuntimePostgres` plus `TestWireTools`, `-count=2` | 48 tests/subtests passed, 0 failures, 0 skips; actual loopback HTTP handlers, runtime resolution and stored-row/history readback |
| Saved public-tool scenarios | S11: all 10 main-agent/planner callback cases passed; S12: historical lineage, source provenance and unrelated/indirect parent rejection passed; S3: invalid-kind/blank-summary rejection, partial errors and omitted/empty/null origin linkage passed |
| `go build ./...`, `go vet ./...`, diff whitespace | Passed |

`TestStockMetadataRuntimePostgres` and `TestEngineTerminalProofPersistsAuthenticReceipt` require an explicit disposable `ARTEX_TEST_DSN` selecting `scopeweaver_test_*`. The database-owning test role may have **NOCREATEDB**, matching `make test-db`. Each fixture creates its own `stock_*` schema and sets every pooled connection's `search_path` to only that schema, preserving the selected database and other connection options. Tests close stores before dropping only their owned schema with `CASCADE`; cleanup errors are reported, but schema deletion does not itself detect idle leaked connections. `TestStockFixtureIsolationAndCleanup` checks multiple pooled connections, parent-table invisibility/preservation and schema removal. Reproduce isolation with `go test ./server -run '^Test(WireTools|StockMetadataRuntimePostgres|StockFixtureIsolationAndCleanup|EngineTerminalProofPersistsAuthenticReceipt)$' -count=2`. Production database privileges and schema are unchanged. This fixture repair supersedes the earlier child-database/CREATE DATABASE requirement.

The first S3 harness attempt failed while seeding a hint with invalid state `open`; the fixture was corrected to the existing `active` state, with no product/assertion change, before the passing rerun. The final S7 setup also passed twice after other tests populated the parent database. These checks establish contract and compatibility behavior, not the cause of the earlier live Flash duplicate/timeout. Full `go test ./...`, frontend/browser checks and the seven required exact-candidate CI jobs remain pending for this branch; no merge or release is claimed.

## Shared runtime fixes from standalone CLI verification (2026-10-17)

Candidate branch `fix/cli-discovered-runtime-bugs`, based on `dev` commit `3f0ef25`. These are local results; the PR's exact-head hosted checks are separate. This change does not add the standalone CLI or subscription providers to the original application.

- HTTP custom tools now use the complete recording-proxy URL returned by `Manager.ProxyAddr`. Before: `"http://" + addr` turned the controlled fixture's `http://127.0.0.1:<port>` into `http://http://127.0.0.1:<port>`, so the request failed before reaching the proxy. After: `proxyStr = addr` preserves the URL. Real direct, explicitly configured and automatic proxy routes pass; the automatic route failed before the fix.
- Startup now checks that an existing transcript path is a directory before reading it. The Windows regular-file regression failed before the fix; it now rejects the path while preserving the file. Missing directories remain valid. The new diagnostic is available in English, Korean and Simplified Chinese.
- Shutdown cancels work, drains HTTP, then waits for admitted chats, side answers and checkpoint publication before closing stores. Previously HTTP drain alone could leave background finalizers writing after database close. Final snapshot/context write errors now propagate, retain pending entries and make process shutdown fail explicitly; deleted/conflicting records remain benign. This bounded drain does not claim to cover every scheduler or engine producer.
- The cancellation-usage test fixture now creates the existing optional usage table, matching production initialization. Its missing-table failure reproduced before the fixture fix; the corrected fresh-database test preserves the exact usage assertion.

Real PostgreSQL shutdown checks passed 11 tests/subtests, including close/reopen persistence and injected final-write failures. Main-process checks passed 4 tests, covering normal exit 0, restart 75, bounded failures and cleanup on a real occupied-port serve failure. Focused proxy/startup/locale regressions passed. The full Windows backend run had 1,975 passes, 2 failures and 15 skips: one failure was an unchanged symlink-creation privilege limitation; the other exposed the catalog's expected count, updated from 1,980 to 1,982 for the two added diagnostics without removing completeness checks. The corrected catalog is verified separately. Linux/hosted checks must establish the remaining platform results.

Frontend: 65 tests passed and the static export passed using Git Bash for the existing POSIX-style npm script. Agent adapter: 7 tests and syntax checks passed. Focused backend static analysis passed. No user database, real online target or notification delivery was used; each database-writing run created and dropped a separate `scopeweaver_test_*` database. No merge or release is claimed.

## Chat controls and context (2026-10-16)

The author of [PR #29](https://github.com/cskwork/scopeweaver/pull/29) reported the checks below against upstream `6366429`. These are the contributor's historical Linux results, before the combined PR #29/#30 review; they do not establish the later candidate's status or Windows attachment-copy behavior.

| Check | Result |
| --- | --- |
| Disposable PostgreSQL 17 race suites (`db`, `server`, `agent`) | 667 top-level tests passed; no race reports |
| Go packages and static analysis | `go test ./... -count=1` and `go vet ./...` passed |
| Frontend tests, translations, audit and static export | 65 tests passed; 3,263 EN/KO/zh-Hans keys passed parity checks; 0 audit findings; TypeScript and 31 pages passed |
| Backend translations | 1,978 production keys passed completeness, parity and printf checks in three languages |
| Chat browser gate | 60 scenarios passed across EN/KO/zh-Hans, 1440/390 px and standalone/task chat; no uncaught page errors |
| Embedded-app smoke and dashboard | Smoke E2E and 14 dashboard scenarios passed; no uncaught browser errors |
| Agent adapter | 7 unit tests, syntax checks and 3 MCP/Pi scenarios passed |
| Installer and packaging | 8 installer fixtures, 11 Compose cases, 5 packaging archives per language (EN/KO), and 6 release-notice tests passed |

The chat browser cases cover queue edits and ordering, restart acknowledgement, retry/edit/branch, context inspection and compaction, tool steering, and document paste. Backend regressions cover accepted-language persistence, preflight errors, interrupted tool exchanges, abandoned branch preparations and document-read attribution.

You can inspect the model's current context apart from billed conversation usage. Compression keeps the recent completed exchanges you select; you retain the full display history and usage from previous attempts.

The contributor reported one reviewer pass that found six integration defects, followed by repairs to generated-input admission and retest recovery, attachment paths after compaction, legacy display history, and steering delivery controls. The attachment test deletes the parent before the child's next prompt; another checks that rebasing preserves literal manifest text. The contributor reported a failing regression before each repair and a full local rerun. The dashboard reasoning-total assertion was scoped to the summary row because the hover tooltip uses the same label; the reported rerun passed 14 dashboard scenarios.

The reported inference tests used loopback models and disposable databases. Windows runtime, paid providers and remote CI require separate verification. The contributor used Python 3 for the notice fixture, matching CI's interpreter requirement.

## Native hardening candidate — 2026-10-15

Local and independent checks on `feat/ollama-zero-egress-hardening`, based on main `33b131e`. These are candidate results; exact-head remote CI and PR delivery remain pending. Historical sections below retain their original results.

| Check actually run | Result |
| --- | --- |
| Independent Go run, 20 packages with isolated disposable PostgreSQL databases | 1,366 tests/subtests passed, 0 failures; `enrich` has no test files; 2 opt-in skips (`TestAgentAdapterE2E`, `TestLiveContextReview`) |
| Agent adapter, checked separately | 7 unit tests and 3 real MCP/Pi scenarios passed; external live-provider review intentionally not run |
| Focused backend `-race` suites | Passed with no race reports: transport, wire/usage, context, origin selection/failure, search/update and admission |
| `go vet ./...` | Passed |
| Frontend unit/localization and static export | 45 passed; 3,168 EN/KO/zh-Hans keys preserve parity, placeholders and rich tags; TypeScript and all 31 static pages passed |
| Backend catalog | 1,943 production keys passed raw three-language completeness/parity and printf checks |
| Installer, Compose and notices | 8 installer fixtures, 11 Compose credential cases, notice checks in 5 packaging-fixture archives per language (EN/KO), and 6 release-notice tests under Python 3.12 passed |
| Static builds and dependency check | macOS embedded build and Windows amd64 cross-build passed; no `go.mod`/`go.sum` drift; native Windows runtime not run |
| Embedded-browser smoke | Rebuilt final UI: 60 passed (48 smoke + 12 dashboard), 0 failures. Settled desktop/mobile screenshots and horizontal bounds passed in EN/KO/zh-Hans: sheet [0,390] at 390px and [864,1440] at 1440px; all 20 critical contents fit. The earlier 420px minimum-width clipping was reproduced and corrected |

The [native HTTP priority regression](../server/ollama_priority_test.go) holds worker 1 inside `/api/chat`, queues workers 2/3, then invokes the real wired judge without a caller-supplied urgency marker. It passed under `-race` with HTTP order **worker 1 → judge → worker 2 → worker 3**, peak concurrency 1, a valid native judge schema/verdict, all calls completed and no leaked slot. Existing queue tests cover FIFO roles, cancellation and lowered limits.

The [native-origin failure regression](../server/provider_origin_failure_test.go) failed before the fix: a pending native tool followed by a stream error could inherit the global cloud `allow` default. After the fix, Stream/Complete and terminal routing errors retain the concrete native profile, 90-second/`ask` model policy and 300-second/`deny` human-approval expiry, without calling the reviewer. Actual cloud failover still clears native preference. The regression and affected race suites passed.

Real authenticated API checks toggled strict mode **false → true → false** against a local proxy that never forwards requests. All four search backends (`ddgs`, `brave-free`, `tavily`, `deepseek`) reached that sink only while strict mode was off; each produced zero additional sink requests while strict. Strict update checks and apply were blocked while local status remained available. [Update regressions](../server/update_policy_test.go) also passed for cached data, redirects, live body reads and cleanup/failure completion after an admitted asynchronous download is blocked by a later toggle. These checks do not claim whole-host egress control.

Host Ollama **0.32.15**, using existing `qwen3:4b` with a **4,096-token output budget**, passed native diagnostics, decision/comment schema, structured tool call/result replay, usage and requested 32K context (`/api/ps`: **32,768**). Earlier failures with 256/512-token budgets remain part of the evidence; they were not discarded as successful runs. The observed load/speed/offload values are one warm-run snapshot, not a benchmark or residency guarantee. No model was downloaded or host configuration changed, and no autonomous task completion or security-assessment quality is claimed.

Browser coverage includes three-language diagnostics/warnings, strict-update status, judge loading/failure protection and existing dashboard/usage flows. Visual QA reproduced a fixed 420px minimum-width editor extending 30px beyond the left edge of a 390px viewport. The minimum now fits the viewport; the rebuilt app passed all 60 checks, and all six desktop/mobile captures show readable labels with settled content inside the viewport. An independent embedded rebuild matched the browser binary byte for byte. Exact-head remote CI and PR readback are separate delivery gates. No merge, release, paid provider call, external target scan or real notification is claimed.

## Provider authentication follow-up — 2026-10-15

Local regression checks on `fix/provider-env-auth`, based on merged main `38fde23`, reproduce and fix two authenticated-local-gateway failures. Startup provider-key fallback and blank-key connection testing each returned HTTP 401 with the previous implementation for Anthropic, OpenAI Chat Completions and OpenAI Responses. The same fixtures pass with the fixes.

| Check | Result |
| --- | --- |
| [`FromEnv → NewProvider` local HTTP authentication](../agent/provider_env_test.go) | 21 scenario cases passed: 18 generic-format cases covering provider keys, file keys, modern/legacy overrides and explicit blank overrides; 3 native cases covering cloud-key isolation, an Ollama token and an explicit blank override |
| [Connection-test handler through local HTTP fixtures](../server/llm_gateway_test.go) | 16 scenario cases passed: 15 generic-format cases covering a matching normalized service, different endpoint/format/proxy and preserved non-local fallback; 1 native case rejecting active cloud-key inheritance |
| Both focused groups with `-race` | Passed, no race reports |

The **37 cases** above count leaf scenarios, excluding parent tests. Authentication failures were reproduced before each product fix and the same tests passed afterward. Public documentation addresses in the non-local handler cases use an explicit local fixture proxy; no public provider is contacted.

A blank-key connection test without a profile ID can reuse the active local service's key only when the generic format, normalized base URL and proxy all match, including for a new draft. Different local services and native Ollama remain isolated. Saving a new blank-key profile does not import environment credentials.

These checks establish the tested protocol and credential boundaries, not paid-account availability or every historical feature. Full-suite, browser, CI, merged-main and release-artifact results after these fixes are separate gates. No paid provider calls were made.

## Installed-model selection follow-up — 2026-10-15

Local verification of the new-profile Ollama default selection on `feat/ollama-native-flexible-providers`. This follow-up supersedes the earlier template behavior: selecting Ollama now performs read-only model discovery. The historical 37-check browser snapshot below predates this behavior; its “no implicit requests” assertion is historical, not the current Ollama contract.

| Check | Result |
| --- | --- |
| Frontend unit and localization tests (`npm --prefix web test`) | 45 passed, 0 failures |
| Production static export (`npm --prefix web run build:static`) | Passed, including TypeScript and all 31 pages |
| Real host installed-model discovery through the browser and authenticated backend | Passed on an unsaved new profile, same-machine Ollama, blank proxy and API key |
| Expanded full browser smoke on the embedded binary and fresh `scopeweaver_test_default_final` database | 43 passed, 0 failures (31 smoke checks + 12 dashboard scenarios). Covers English/Korean installed defaults for all three templates, direct format selection, cloud compatibility, metadata-only selection, failed/empty discovery, manual edits and late-response rejection, reopening drafts and existing profiles, plus existing setup, sign-in, localization, SSE and dashboard regressions |

The real host's `/api/tags` returned `qwen2.5-coder:7b`, `qwen3:4b`, and `embeddinggemma:latest`. The browser selected **`qwen2.5-coder:7b`**, the first returned installed model declaring `tools`. Its metadata reported `completion`, `tools`, `insert`, 7.6B parameters and Q4_K_M quantization. `qwen3:4b` also declared `tools`; `embeddinggemma:latest` declared only `embedding`.

The browser sent exactly one LLM operation, `POST /api/llm/models`, using the host endpoint with no key or proxy. The new draft was closed without save, activation, connection testing or inference. Model and machine configuration were unchanged. This proves installed-model selection and metadata discovery; it does not establish tool execution or autonomous task reliability.

## Historical native Ollama branch — 2026-10-15 (branch-local, not a release)

Local checks on `feat/ollama-native-flexible-providers`, recorded before the pull request was opened. Remote PR CI is tracked separately in GitHub Actions; nothing here is a release, merge or CI result.

| Check | Result |
| --- | --- |
| Go suites, all 20 packages run separately, each on a fresh disposable PostgreSQL 16 database | 1,197 tests/subtests passed; 2 opt-in skips (`TestAgentAdapterE2E`, `TestLiveContextReview`) |
| Agent adapter, run separately | 3 real MCP/Pi scenarios passed |
| `go vet ./...` | Passed |
| Frontend unit and localization tests | 43 passed; 3,159 unique keys in each of English, Korean and Simplified Chinese |
| Backend message catalog | 1,938 unique keys in each of the three languages; printf and parity tests passed |
| TypeScript and production static export | Passed, rebuilt after the final health UI changes |
| Frontend dependency audit (`npm audit`) | 0 vulnerabilities |
| Builds | Embedded `CGO_ENABLED=0` binary rebuilt after the final health UI changes; Windows amd64 `CGO_ENABLED=0` cross-build passed (native Windows runtime not run) |
| Browser smoke (`npm run smoke`) on the real embedded binary and a fresh `scopeweaver_test_ollama_final_health` database | 37 passed, 0 failures (25 smoke checks + 12 dashboard scenarios). Covers the three Ollama templates in English and Korean with no implicit requests, keyless save/activate, keyless local compatible health versus a public cloud profile with a missing key, cloud formats and editable GLM, per-model metadata and tool warnings, the exact missing-model pull command, tool/context/load/speed diagnostics and the default-off local-only policy, plus the existing English/Korean/Simplified Chinese setup, sign-in, language, draft, SSE and approval-usage regressions |
| Compose credentials | 11 cases passed in real `network_mode: none` containers, through container environment and `FromEnv` |
| Installer and packaging | 8 installer fixtures passed in total, including English and Korean selection cases; 5 archives passed in each of English and Korean |
| Release notice (Python 3.12) | 6 tests passed |
| Independent backend security/correctness review | Passed, including the DeepSeek local-only policy, live CIDR checks and 404 classification |

### Real Ollama run

Host Ollama 0.32.15 with `qwen3:4b`, through the native streaming API:

- In an SDK-level streaming check with a synthetic tool fixture, the model made a `record_fact` tool call; ScopeWeaver executed it locally, replayed the result, and the turn completed. This was not a platform database `record_fact`.
- 32K context was requested; `/api/ps` reported 32,768 allocated.
- Exact usage in that check: streaming turn 186 input / 750 output tokens; completing turn 826 input / 193 output.
- Keyless save/activate through the real API passed. Diagnostics reported `tool_call_verified` and `context_verified` true, context 32,768, load 2.225375 ms, 29.231 tokens/s and latency 18,958 ms. **This was one warm run, not a cold-start benchmark or a hardware generalization.**
- The exact Korean pull hint for a missing model, and the strict-policy toggle blocking a public IP, passed.
- **Platform task worker Bash proof:** a real task worker (task 1) running `qwen3:4b` (32K requested, 32,768 allocated; concurrency cap 1; keep-alive 30m; thinking disabled; max 4,096 tokens) made a native Bash tool call `call_219yv72x`. Activity 9 (call) and 10 (`tool_result`) share that ID with no error, and the output contained the marker `LOCAL_TASK_OLLAMA_OK`. The ledger recorded exactly 6,704 input / 4,096 output tokens for that native call, and the next native call replayed the same `tool_call_id`. The intent was seeded manually through the existing `db.AddIntent`, not created by the LLM. The output also contains a literal `\n` from fixture escaping; no newline-handling or model-bug conclusion is drawn. The run used a binary built from the uncommitted branch source before the final UI/DTO changes.

### Autonomous task reliability is not validated

A passing tool-call diagnostic proves the wire path, not that a model can complete a task.

- `qwen2.5-coder:7b` declares `tools`, yet wrote tool-shaped JSON in `assistant.content` instead of wire `tool_calls`, so no Bash or fact tool ran. ScopeWeaver correctly kept it as text.
- `qwen3:4b` as the main agent, given a long prompt, deliberated and wrote the expected marker as text. As a task worker it did run Bash (above), but the task was paused at 360 s without `record_fact` or autonomous completion; one successful worker tool call does not establish autonomous completion.

**No end-to-end autonomous task completion or security-assessment quality is claimed for any local model, and 7B-class models are not validated as a full agent.** The [hardware tiers](llm-providers.md#hardware-tiers) are estimates, not benchmarks.

### Not exercised

No paid cloud calls, external target scans, real notifications, publication, deployment, release or merge. Anthropic, OpenAI Chat/Responses and custom gateways were verified through wire-contract tests, regressions and editable browser flows, not live provider accounts. All test databases were disposable, and the host Ollama configuration was not changed. The optional `local_llm_only` policy is off by default; servers still need `OLLAMA_NO_CLOUD` set separately.

## Running database-backed tests

The `db`, `server` and `agent` suites need PostgreSQL. They resolve their
connection string **only** from the `ARTEX_TEST_DSN` environment variable and
will never fall back to the application's own DSN (`ARTEX_PG_DSN` /
`config.json`). This is deliberate: it keeps `go test` from mutating the
live/production database the running app uses.

- **`ARTEX_TEST_DSN` unset** → every database-backed test skips. No database is
  touched.
- **`ARTEX_TEST_DSN` set to a throwaway database** → the suites run against it.
- **`ARTEX_TEST_DSN` resolving to the same database as the app** (same host,
  port and database name) → the run fails fast with
  `refusing to run tests against the live database; point ARTEX_TEST_DSN at a
  throwaway DB`, before any test connects.

The simplest way to run them is:

```bash
make test-db
```

`make test-db` (via `scripts/test-db.sh`) creates a throwaway database named
`scopeweaver_test_<pid>` owned by the `sw_user_test` role, points
`ARTEX_TEST_DSN` at it, runs `go test ./db/ ./server/ ./agent/ -count=1`, and
drops the database on exit — even if the tests fail. Override `TEST_DB_ROLE`,
`TEST_DB_HOST` or `TEST_DB_NAME` to match your local PostgreSQL. The database
name must start with `scopeweaver_test_` (or `scopeweaver_adapter_` for the
agent-adapter E2E test).

To run a subset against a database you manage yourself, export `ARTEX_TEST_DSN`
and call `go test` directly:

```bash
export ARTEX_TEST_DSN='postgres://sw_user_test@/scopeweaver_test_1?host=/tmp&sslmode=disable'
go test ./server/ -run TestSomething -count=1
```

## v0.2.1 candidate — 2026-10-11

The Windows release candidate was built with the real static frontend and tested against disposable PostgreSQL 16.15 databases. These results supersede the historical dependency and Windows-runtime limitations below; the older record is retained for traceability.

| Check | Result |
| --- | --- |
| Frontend dependency audit | 0 vulnerabilities across all severities, including development dependencies |
| Frontend regression and localization tests | 26 passed; no skips |
| Production static export | TypeScript check passed; 31 pages generated |
| Native Windows Go suites, 18 packages | 1,016 tests/subtests passed; 0 failures; 13 existing platform/opt-in skips |
| Native Windows complete server suite | 311 tests/subtests passed; 3 existing opt-in/platform skips |
| Agent adapter | 7 unit tests and syntax checks passed; 3 MCP/Pi scenarios passed against the real authenticated backend and a fresh database |
| Local installer | All 4 isolated fixtures passed: fresh install, stale-file replacement, dependency failure and frontend-build failure |
| Backend translation catalog | 1,868 production keys passed English/Korean completeness and printf argument checks |
| Notification isolation | Deterministic stale-worker regression failed before repair and passed afterward; management/notification selection passed 5 consecutive runs |
| Windows evidence | Full PostgreSQL-backed suite passed, including concurrent writes, commit failures and long paths |
| Windows Python and skill files | Real Python execution and unusable-launcher fallback passed; nested Unicode paths passed through file listing and read handlers |
| Browser E2E on embedded Windows binary | 8 checks passed: setup outage protection in both languages, setup/login, same-origin SSE, usage API isolation, both localized usage cards, and no uncaught page errors |

Go counts include parent tests and subtests. The 13 native skips comprise ten Unix self-update fixtures, one unavailable symlink privilege, the opt-in adapter integration test, and the live-provider test. Adapter integration is checked separately, and Linux CI covers the Unix fixtures. No tests were disabled to produce these results.

Regressions reproduce the original Windows directory-sync failure, localization separator mismatch, unusable Python launcher, notification worker leakage, and profile-cleanup ordering. The E2E SQL helper now puts options before its connection URL; a direct outage/restore check confirmed that Windows psql executes the statements.

The release gate additionally requires the `frontend`, `windows`, `backend`, `agent-adapter`, and browser `smoke` jobs before and after merge. Publication results, platform archives, and checksums are available on the [v0.2.1 release](https://github.com/cskwork/scopeweaver/releases/tag/v0.2.1); this source record does not claim those later jobs or downloaded assets have already passed.

Existing configured Python interpreter paths are preserved. If a prior installation stored an unusable launcher alias, choose a runnable interpreter in Settings. Native Windows tests do not establish Windows self-update behavior or macOS runtime behavior. Live paid providers, external targets, and real notification recipients were not used.

## Historical verification — 2026-10-03

Validation recorded on **2026-10-03** for the English/Korean ScopeWeaver edition, based on upstream ARTEX commit `160fe13`. Tests use disposable PostgreSQL databases and local fixtures. This is source delivery; no binary release or container image has been published.

| Check | Result |
| --- | --- |
| Frontend regression and localization tests | 21 passed; TypeScript check passed |
| Production static export and embedded Go binary | Built successfully |
| Go regression suites, 18 packages | 986 tests/subtests passed; 1 inherited test failed; 1 external-provider test skipped |
| Complete server suite | 285 tests/subtests passed; only the opt-in live-provider test skipped |
| Backend translation catalog | 1,865 production keys passed English/Korean completeness and printf argument checks |
| Authored Chinese text audit | No untranslated authored source/docs or Go comments; deliberate compatibility/data exceptions retained |
| Desktop browser coverage | 24 routes in each language; no missing labels or page-level horizontal overflow |
| Mobile browser coverage | Dashboard, tasks, findings and settings in both languages; no page-level horizontal overflow at 390px |
| Real application screens | Login and notification forms checked in both languages; setup validation and initialization verified |
| Language switching | Navigation/status labels update; selection and unsent drafts are preserved |
| HTTP localization | Markdown/CSV export, language negotiation, saved preference and invalid-language rejection checked |
| GLM-5.3 provider contract | 21 focused tests/subtests passed with local HTTP/SSE fixtures; both endpoint paths, reasoning and tool-call replay covered |
| GLM template browser flow | Both templates checked in both languages; entered name/key/proxy preserved; no save, activation or provider request |
| Packaging and launch scripts | Shell syntax, bilingual packaging fixtures and 10 supervisor cases passed |
| Provenance and dependencies | Original license unchanged; dependency versions and Go module unchanged; documentation links and diff checks passed |

Counts include Go parent tests and their subtests. Package runs use separate databases initialized with the same supplementary metering tables as normal server startup. The original tests and checks remain enabled.

### Inherited issues at the time

`TestGraphOverviewExpandsAssociatedCompanyScope` failed with `task scope missing: <nil>` in both ScopeWeaver and untouched upstream source. The test expected detailed scope in the graph overview, which upstream commit `06a43f3` deliberately made compact. Commit `75a706d` replaced it with `TestGraphOverviewKeepsAssociatedCompanyScopeCompact`, which checks the compact overview and verifies the stored company scope directly. The complete backend CI has passed since then.

`TestTaskMetadataPatchReturnsRenameAndPin` previously produced an intermittent temporary-directory cleanup failure in upstream verification. It passed in the final ScopeWeaver server run; no unrelated lifecycle change was made to hide it.

The upstream mock preview has no notification metadata mock, so its Notifications page errors. The **real backend Notifications page and add-channel form pass** in both languages. No notification was sent during verification.

The v0.1.0 verification recorded **14 npm audit findings: 1 critical, 9 high and 4 moderate**. Framework/tooling updates and removal of the unused shadcn CLI dependency resolved the findings in the v0.2.1 candidate above. Run `cd web && npm audit` for the current registry report; audit results can change as new advisories are published.

### Historical scope and limits

- The optional `TestLiveContextReview` requires private external-provider configuration and was not run. No paid model calls, external target scans, or real notification deliveries were performed.
- GLM tests prove the local request/response contract, not account access or provider availability. Coding Plan is restricted to officially supported tools, and ScopeWeaver is not listed; see [provider setup](llm-providers.md).
- Docker image execution and Windows batch execution were not verified. Packaging fixtures do not claim a real Windows runtime test.
- Setup reached the authenticated route, but one browser wait for the window load event timed out; initialization and the new password were then confirmed through the real API.
- User-entered text, stored evidence, edited prompts, wire identifiers, legacy parsing markers, Unicode fixtures, and original upstream screenshots retain their original content. The English/Korean app does not translate arbitrary user data or third-party tool output.
- Historical validation documents under `sidequestion/` describe upstream work, not new ScopeWeaver runs.

Current remote checks are available in [GitHub Actions](https://github.com/cskwork/scopeweaver/actions/workflows/ci.yml).

## Chat document paste (2026-10-16)

PR #29 records the following paste checks against a disposable PostgreSQL 17 database and a local backend. These are contributor-reported results from 2026-10-16; current integration evidence is recorded separately:

- The frontend suite passed 52 tests; the production static export passed.
- Clipboard browser checks passed 12 combinations: standalone/task chat, EN/KO/zh-Hans and 1440/390 px.
- Cases covered rapid document pastes with one draft owner, accompanying text, native 3,000-character insertion and undo, Unicode preview, draft restoration, upload retry, navigation, mentions, IME and both send preferences. Pasting submitted no messages.
- The contributor disabled the React paste handler to confirm the browser check failed on the missing second attachment chip, then restored it and reran the check.
- The backend smoke suite passed the clipboard checks and Ollama loopback fixtures with no uncaught page errors.

The file cases supply browser File objects through paste events. The text and undo cases use the Chromium clipboard and keyboard paste.
