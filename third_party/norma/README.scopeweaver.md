# ScopeWeaver's Norma fixes

This directory starts from upstream commit
[`52f17dd52d95b087265e5d5b306fe85c1ea05599`](https://github.com/Autumn-27/Norma/commit/52f17dd52d95b087265e5d5b306fe85c1ea05599).
It contains exactly v0.4.3 (`320ae32989da97dd91469bb89164cbfe87d83d5d`)
plus the upstream Apache-2.0 `LICENSE`: 211 files / 1,293,448 bytes before local
changes. Original source, tests, module metadata and notices are retained.

ScopeWeaver modified these upstream files on 2026-10-25:

- `harness/streaming.go`, `harness/tools.go`, `harness/rawinput_test.go`: capture
  each executor's originating tool context, pass it to the private tool caller,
  and retain it during cancellation. Delayed original work cannot borrow the
  live settlement context. Permissions and hooks still use the parent context.
- `llm/retry_config_test.go`: supply the test HTTP client when directly calling
  the private retry helper; retain the HTTP 503 and two-attempt assertions.
- `noa/config.go`: scale the two default growth floors for positive windows below
  200K. Defaults at 200K and above, and explicit custom growth values, stay intact.
- `noa/nudge.go`: allow count-based consolidation below the usual usage floor
  when accumulated summaries meet the existing minimum-benefit threshold.
  Cadence, growth arithmetic and tier-selection order remain intact.
- `noaadapter/compactor.go`, `noaadapter/overflow.go`: distinguish output-length
  errors from context overflow and learn explicit smaller context windows without
  later raising the learned ceiling. Learning is in memory, not persisted state.
- `noaadapter/overflow_test.go`: verify actual rejected/retried views, honest
  refusal, unchanged raw history, output-error handling and monotonic ceilings.
- `noaadapter/tiering_test.go`: replace a disproven limitation assertion with a
  positive 40K consolidation regression, preserving the fixture and token bounds.

Added files are `harness/streaming_context_test.go`,
`noa/smallwindow_defaults_test.go`, `noa/nudge_count_benefit_test.go`, and this
README. Every modified upstream file carries an Apache section 4(b) modification
notice. Other upstream files and module metadata remain byte-identical.
Exported signatures and serialized formats are unchanged; the behavioral fixes
above are intentional.

The root module selects this source with a local `replace`. Docker copies it
before downloading dependencies. `THIRD_PARTY_NOTICES.txt` reproduces the full
upstream Apache license for archive and container packaging. The snapshot has no
upstream `NOTICE` file. The existing backend CI job runs this nested module's
suite separately because root `go test ./...` does not traverse nested modules.

```sh
cd third_party/norma
go test ./... -count=1
go test -race ./harness -run '^(TestStreamExecRetainsOriginatingToolContext|TestSettlementAfterSlowPostToolUseKeepsToolContextRaceFree)$' -count=20
go build ./...
go vet ./...
```

ScopeWeaver also forwards the profile's resolved context window when enabling
Noa. The host's real-session tests cover 40K, 128K, 200K, 250K, 400K and 1M;
output-token limits are separate. An absent window retains the existing 200K
fallback. This is a host integration fix, not a claim that all models share a
context limit.

Remove the replacement only after an upstream version passes these execution,
compaction and host regressions. Preserve the required source/license notices.
