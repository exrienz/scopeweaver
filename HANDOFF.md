# ScopeWeaver 전체 이슈·PR 인계 — 2026-10-26

이 문서는 다른 환경에서 작업을 이어가기 위한 **시점 고정 인계 기록**이다. 제품 사용 설명서나 완료 선언이 아니다. 사용자 요청에 따라 이 문서만 `dev`에 전달한다. 미완성 구현, 기존 작업 트리, 진단 제한과 릴리스 보류는 유지한다.

## 1. 먼저 알아야 할 상태

- 이번 원격 복구 보완 전 `dev`: `c9d9a9fc28c8168ba53988c749e85633ddc69f11` (#58 인계 문서 병합).
- 원격 `main`: `a8145ad0440ad5864e1bcb6b5cc94195465a7c81`.
- 최신 공개 릴리스: `v0.7.0`. 다음 minor 후보는 `v0.8.0`이며 아직 생성하지 않았다. 실제 릴리스 전에 최신 태그를 다시 확인한다.
- 이 인계 문서의 배포 PR을 제외하고 **열린 PR 4개, 열린 이슈 3개**다. 전체 51개 PR 및 5개 이슈를 GitHub API로 조회했다. 아래 표는 배포 전 스냅샷이며, 인계 문서 반영으로 `dev`와 PR #53의 head는 바뀐다.
- #52, #54, #55, #56, #58은 `dev`에 병합됐다. #40, #46, #57은 미병합이다. #53은 `dev → main` 승격용 draft이며 보류 상태다.
- #57의 최신 CI가 통과했어도 SDK 의존성 감사 실패와 과거 CI 실패 원인 미확정은 해소되지 않았다.
- #40의 수리본은 기존 원격 PR #40과 다르지만, 이제 미커밋 7개 파일까지 별도 원격 checkpoint에 커밋됐다. #46 수리본과 재개 자료도 원격에 있다. **§3의 Git-only 복구 절차를 사용하면 기존 호스트의 로컬 bundle/patch 없이 이어갈 수 있다.**

## 2. 권한·중단 조건

사용자의 기존 지시는 “adversarial review if not issue push merge fix any failures”, “SDK 실패 7개까지 수정”이다. #40/#46은 수정·독립 리뷰·필수 검사 후 `dev`로 전달하는 범위다.

후속 릴리스 지시는 모든 에이전트 담당 이슈·PR을 완료하고 `dev`에 통합한 뒤, **통합된 정확한 dev 커밋에서 전체 E2E**, 문제가 없으면 `main` 승격, 검증 후 minor release를 수행하는 조건이다. 최종 조율 담당은 기존 세션 `9441381`이다. 이 문서 작성·푸시 요청은 미완료 구현 병합, 감사 위험 수용, 진단 재실행, 태그·릴리스 승인으로 확대하지 않는다.

- `CONTRIBUTING.md`와 `AGENTS.md`를 먼저 읽는다. 일반 기능/수정 경로는 최신 `dev`에서 분기한 `feat/*` 또는 `fix/*` → `dev`, 승격은 `dev → main`이다.
- 필수 hosted checks: `branch-policy`, `frontend`, `agent-adapter`, `backend`, `windows`, `chat-browser`, `smoke`. 정확한 후보 SHA와 병합 후 SHA에 대한 결과를 구분한다.
- `CONTRIBUTING.md`의 병합 절차는 각 PR에 대한 **maintainer의 최종 human check**도 요구한다. CI 통과나 에이전트 독립 리뷰 PASS를 그 확인의 대체물로 기록하지 않는다.
- `dev`는 PR 및 위 7개 검사, 관리자 적용까지 보호된다. 보호 설정이나 검사 조건을 변경해서 인계 문서를 전달하지 않는다.
- 이번 `HANDOFF.md`는 날짜가 붙은 작업 기록이다. 사용자 요청은 이 Markdown 인계 파일이며, 기존 6개 언어 제품·기여 가이드 요건을 축소하지 않는다.
- 거부된 planner/worker 번역 생성은 재시도하거나 다른 모델·provider로 우회하지 않는다. 검사 완화, 미완성 카탈로그 병합, 증거 없는 PASS도 허용되지 않는다.
- 다른 작업 트리의 reset/clean/stash, 브랜치 삭제, 캐시 정리, 실행 중인 프로세스 종료를 하지 않는다. 이전 세션/프로세스 번호는 기록이지 재실행 지시가 아니다.
- 새 실제 provider 호출이나 #57 진단 실행은 아래의 별도 사용 횟수·승인 제한을 따른다. 이 인계 요청으로 실행 예산이 초기화되지 않는다.

## 3. 원격만으로 복구 — 로컬 전송 묶음 불필요

사용자의 추가 요청에 따라 원래 미커밋/로컬 전용 상태를 다음 원격 checkpoint에 보존했다. 미완성 구현은 dev에 병합하지 않았다. 기존 PR #40/#46 브랜치도 덮어쓰지 않았다.

| 복구 대상 | 원격 브랜치 | 고정 커밋 |
| --- | --- | --- |
| #40 수리본 + 원래 미커밋 7개 파일 | `checkpoint/pr40-resume-20261026` | `3509fb14e24ddffbd58537c0f51ccc8b75c26570` |
| #46 CI 수리본 | `checkpoint/pr46-resume-20261026` | `42183237572c1def559bff9ccd834498c9fb2369` |
| 계획·승인·검증·진단·재현 자료 | `checkpoint/handoff-evidence-20261026` | `fbc03000c8836a29f1c17fff03a9cbcb65adda50` |
| #49 / #57 현재 코드 | `feat/gh-49-opencode-sdk` | `b9781b2c4b92d9cef4a048f1c897616ce9ed8ff9` |

새 환경에서 실행할 Git/파일 복구 절차다. 같은 이름의 브랜치/디렉터리가 이미 있으면 새 이름을 고른다. 기존 작업을 reset하지 않는다.

```bash
git clone https://github.com/cskwork/scopeweaver.git
cd scopeweaver
git fetch origin
git switch dev
git worktree add -b fix/pr40-resume ../scopeweaver-pr40-resume \
  origin/checkpoint/pr40-resume-20261026
git worktree add -b fix/pr46-resume ../scopeweaver-pr46-resume \
  origin/checkpoint/pr46-resume-20261026
git worktree add --detach ../scopeweaver-handoff-evidence \
  origin/checkpoint/handoff-evidence-20261026
python3 ../scopeweaver-handoff-evidence/handoff/2026-10-26/verify.py \
  --extract ../scopeweaver-resume-evidence
```

shallow/single-branch clone을 사용했다면 checkpoint refs도 별도로 fetch한다. #49를 재개할 때는 기존 원격 feature 브랜치에서 별도 작업 트리를 만든다. 이 절차는 DB·앱·브라우저·provider나 과거 진단을 실행하지 않는다.

### 무엇이 저장됐는가

- #40의 다음 7개 파일은 이제 `3509fb1`에 커밋되어 있다. **과거 `pr40-uncommitted.patch`를 다시 적용하면 안 된다.** 원래 바이트와 checkpoint/새 clone 파일 해시를 대조한다.

```text
agent/legacy_stock_compat_test.go
agent/testdata/legacy_add_intent_schemas.json
agent/testdata/legacy_stock_prompts.json
agent/tool_stock_schema.go
docs/VERIFICATION.md
docs/ko/VERIFICATION.md
locale/zh-Hant.json
```

- 증거 브랜치의 `handoff/2026-10-26/`에는 `RECOVERY.json`, `MANIFEST.json`, `PACKAGE-RESULT.json`, `EVIDENCE.tar.gz`, `verify.py`, `README.md`, `REMOTE-RESTORE.json`이 있다. 계획·리뷰·승인 제한·재현 스크립트·로그·attribution ledger 등 **4,250개 파일**의 체크섬을 검증한다.
- 압축 파일: **12,343,226 bytes**, SHA256 `5d46e936a714254c357bbe41649bfb09fbbeb1757854e88101077aa404abc57b`.
- 새 HTTPS clone에서 #40/#46 checkpoint와 #49 원격 head를 확인했고, 원래 미커밋 7개 파일 및 증거 4,250개 파일의 해시 검증·추출이 통과했다. 증거 payload 전체는 `4b85388`에서 검증했으며, 최종 `fbc0300`은 `REMOTE-RESTORE.json`만 추가한 커밋임을 별도 fetch/diff로 확인했다. 압축 파일 해시는 동일하다. 로컬 bundle/patch나 원래 저장소 object를 사용하지 않았다. 이는 복구 증명이며 프로그램 동작의 정확성/최종 QA PASS는 아니다.
- 증거 경로는 `pr40/`, `pr46/`, `gh49-plan/`, `gh49-evidence/`, `gh15/`, `gh50/`, `gh53-verification/`, `gh53-retry/`, `release/`로 구분된다. 원래 호스트 경로와의 대응은 manifest `roots`에 있다. 과거 절대 경로는 새 환경에 맞게 조정한다.
- **273개 파일**은 인증값 제거 또는 인코딩 정규화가 적용됐다. `source_sha256`은 로컬 원본, `published_sha256`은 공개용 사본의 해시다. 바이트가 다른 사본을 원본과 동일한 실행 증거라고 주장하지 않는다. 원본 로컬 자료는 변경하지 않았다.
- 제외된 **1,179개 항목**도 파일별 이유가 manifest에 있다: DB/WAL·캐시·의존성 995, 미검토 시각 자료 138, 컨테이너 31, binary/미확인 인코딩 12, 명시적 compiled/database binary 2, 인증 설정 1. 원격 소스 브랜치의 실제 앱 코드·추적 assets는 원래대로 보존된다. 이 제외 목록은 앱 소스 누락 목록이 아니다.
- 실행에 필요한 dependency/build 산출물은 프로젝트의 설치·빌드 절차로 재생성한다. 과거 실행 스크립트의 제거된 임시 인증값은 새 disposable 값으로 설정한다. 비밀값·실제 DB 데이터·현재 로그인 세션을 복원하려 하지 않는다.

기존 로컬 `scopeweaver-handoff-transfer-20261026.tar.gz`는 이전 시점의 보조 백업이다. 새 원격 복구에는 필요하지 않다. 이미 사라진 임시 파일을 이 패키지로 복구했다고 주장하지 않는다. 추가 원본 이미지/바이너리가 검증에 꼭 필요하면 원래 GitHub artifact ID·해시·로컬 원본의 존재 여부를 확인하고 증거 공백을 명시한다.

## 4. 담당 세션·작업 트리

아래 경로의 공통 접두사는 원래 호스트의 `/Users/danny/Documents/Git/scopeweaver-worktrees/`다. 새 환경에서는 실제 경로로 바꾼다. 인계 조회 시 관련 타 세션은 idle이었으며 깨우거나 테스트를 재시작하지 않았다.

| 담당 | 작업 | 로컬 경로 / 브랜치 | 인계 시 HEAD |
| --- | --- | --- | --- |
| `c2f2c817-4789-4194-aaa8-ebaefa08138f` | #40 | `pr40-review-repair`, `fix/pr40-review-repair` | `3509fb14e24ddffbd58537c0f51ccc8b75c26570`, clean, 원격 checkpoint로 보존 |
| 동일 | #46 | `pr46-ci-cancellation`, `fix/pr46-ci-cancellation` | `42183237572c1def559bff9ccd834498c9fb2369`, clean |
| `37a3a0f9-7c4c-42f8-ab7a-84998b2e45ae` | #49 / #57 | `gh49-opencode-sdk`, `feat/gh-49-opencode-sdk` | `b9781b2c4b92d9cef4a048f1c897616ce9ed8ff9`, clean, 원격과 같음 |
| `9441381b-e0ce-42c3-ad25-ea152c781ace` | #15 / #53 / 최종 통합·릴리스 | `gh53-verification-gates`, `fix/gh-53-retry-dialog` | `80b135df7e39b19111c799cc0b9c1cfd2eb7f867`, clean, #56으로 병합됨 |
| `c829cc27-de09-4f9a-9588-f4fb474e5602` | #50 / #54 | 원래 checkout `/Users/danny/Documents/Git/scopeweaver`, `fix/gh-50-scope-enforcement` | `d2e84f2753e00fd93fa6bc7bee49e927cc3d8f9d`, #54로 병합됨 |

`/private/tmp/gh53-retry-diagnosis-20261025` detached worktree도 남아 있다. 이 인계 작업은 이를 변경하지 않았다. 전역 규칙 파일의 별도 미커밋 수정은 프로젝트 인계 구현에 포함하지 않는다.

## 5. PR #40 — 로컬 수리는 진행됐지만 번역·통합·최종 QA 미완료

원격 PR: [#40](https://github.com/cskwork/scopeweaver/pull/40), `fix/glm-agent-contracts → dev`, 원격 head `9b047546edf15d782978cdb78287f15055a6eace`. 원격 PR은 최신 로컬 수리본이 아니다.

### 완료한 부분과 증거 한계

- 의도한 네 커밋을 `f354c920` 위에 재구성한 뒤 6개 언어 dev `7af06f1`로 rebase했다. 이후 최신 dev의 #54/#55/#56 통합은 아직 안 했다.
- 실제 SDK tool-result receipt와 originating context/response를 보존하는 terminal gate 수정, 기존 **30 ms / 120 ms** 회귀 사례 유지.
- 로컬 `third_party/norma`는 upstream `52f17dd52d95b087265e5d5b306fe85c1ea05599` 기준이다. SDK의 기존 실패 7개까지 수정했고, 저장된 전체 nested SDK test/build/vet receipt는 각각 exit 0이다. 원본 RED와 focused race 근거도 보존했다. 원본 211개 파일 중 201개 그대로, 10개 수정, 4개 추가라는 출처 기록과 Apache 고지를 유지한다.
- 호스트가 설정된 context window를 Noa에 전달한다. 실제 session 검사에는 40K/128K/200K/250K/400K/1M, fallback, disabled Noa, 손상 상태 복구, output limit 독립성과 Ollama 예외가 포함된다. 200K는 미설정 시 fallback이지 모든 모델의 한도가 아니다.
- 과거 번체/일본어/스페인어 기본 스키마 해시 3개를 정확히 등록하는 5줄 변경은 복구 checkpoint `3509fb1`에 포함됐다. 실패 재현 후 기존 schema/customization/decorated 검사 **219 test/subtest pass events, 실패 0, skip 0** 및 이 변경만의 독립 리뷰 PASS. 번역 품질이나 실제 PostgreSQL 저장 보존의 최종 증명은 아니다.
- 비운영 문서 12개(4개 언어의 CONTRIBUTING/PROVENANCE/VERIFICATION)는 커밋 `3fb3bca`에 완료했고, 현재 원격 checkpoint `3509fb1`의 이력에 포함돼 있다. Sol high/fast 작성·독립 의미 리뷰 PASS, 로컬 링크/앵커 **159개** 통과. 요약 크기/개수 오역과 문맥 크기 적용 범위 표현을 수정했다. 원어민 승인이나 최종 후보 준비 완료를 뜻하지 않는다.
- 이전 frontend 준비 결과 web **75**, adapter **18**, static **31 pages**, audit **0**은 해당 과거 소스의 결과다. 최신 dev/SDK 통합 후 최종 검증으로 재사용하지 않는다.

### 남은 번역 입력 — 생성 거부 상태 유지

`agent/prompt_zh-Hant.json`, `agent/prompt_ja.json`, `agent/prompt_es.json` 각각의 `plannerDefaultTmpl`, `workerDefaultTmpl`: 총 **6개 값**의 검토된 번역 입력이 없다. 이전 Claude 자동 검토가 `[cyber]`로 거부했다고 대화에 기록돼 있다. 알려진 원본 결과/stderr 디렉터리 `/private/tmp/scopeweaver-six-locales-20261025/`는 없어졌으며, 이유와 문장별 거부 사유는 알 수 없다.

- 재시도·다른 모델/provider로 우회하지 않는다. 외부 검토된 UTF-8 key-level 값/patch, 기준 원문 revision과 검토 범위를 받아 기존 검증으로 확인한다. 입력 제공 자체가 검사 면제나 안전 검토 우회 승인은 아니다.
- 기준 영어 원문은 로컬 `aad0d924`의 `agent/planner.go:305`, `agent/worker.go:238`. 원격 증거를 추출한 `pr40/refs/translation-input-manifest.json`에 원문 파일 해시·기존 값 해시·정확한 키가 있다.
- `locale/ja.json`, `locale/es.json`에는 각각 5개 키의 값이 미완료다: `add_intent` item의 summary/asset_ids/parent_ids 설명 3개, 기록된 evidence header, 확장된 Assess-goals round guidance. 정확한 키·개행은 manifest에 있다. 번체의 5개 추가/1개 obsolete 제거는 checkpoint `3509fb1`에 커밋됐으나 의미 검토가 남았다.
- obsolete Assess-goals 키만 교체한다. zh-Hans JSON에 없다고 다른 **52개 정상 등록 키**를 삭제하지 않는다. 현 기준 strict count는 **1,987 = 1,983 + 4**이며, 새로운 실제 dev 변경이 있으면 근거를 다시 산출한다. 검사를 맞추기 위해 기대값을 임의로 올리지 않는다.
- #54가 바꾼 `goalsScopeTail`을 오래된 전체 JSON 파일로 덮어쓰지 않는다. 12개 문서는 완료됐으므로 새로 번역하지 않는다. 최종 소스 통합 시 EN/KO를 포함한 현재 사실만 별도로 정합화한다.
- 번역 확정 후 실제 historical prompt recognition RED → exact hash 등록 → 여섯 source/output 언어의 render/customization GREEN을 실행한다. 이전 번역이 현재 stock과 같을 때의 통과를 upgrade 증거로 쓰지 않는다. 실제 DB 저장 row/history 보존도 확인한다.

### 최신 dev 정합화

`aad0d924`와 dev `935915b`의 **읽기 전용 committed-tree 예측**에서는 11개 파일이 겹치고 6개에 textual conflict가 있었다. dirty work는 예측에 포함하지 않았다. 최신 `71a487f`와 실제 합칠 때 다시 확인한다.

| 파일 | 보존해야 할 양쪽 동작 |
| --- | --- |
| `agent/capture.go` | #40 optional event observer/terminal receipt와 #54 `newOwnedSession(ctx, opts)` session ownership |
| `agent/prompt_translations.go`, `agent/prompt_zh-Hans.json` | #40 contract 수정과 #54 scope clarification을 key/constant 단위로 병합 |
| `server/logsink.go`, `server/manager.go`, `server/server.go` | #54 server-wide drain/backgroundWG/enrich.Close, #40의 직접 Manager.Close 보호·명시적 SQL cancellation/join을 함께 검토 |

한쪽 파일 전체를 선택하면 다른 쪽 보장을 잃는다. scheduler를 두 번 시작하거나 writer를 drain에서 빠뜨리지 않는다. `server/scheduler.go`의 context-aware init과 취소 확인도 보존한다. #54 drain은 `server/chat_shutdown.go`, `server/background.go`에 있다.

종료 검사는 중복이 아니다: #54는 SQL이 막히면 drain이 완료되지 않고 DB를 유지함을, #40 `TestStopDBCompletesWhileInsertBlocked`는 명시적 cancel로 SQL lock 해제 전에도 writer join이 완료됨을 검사한다. direct Manager.Close와 server DrainChats의 timeout/retry 경로도 구분한다. 보장 정합화 전 검사를 삭제하지 않는다.

`e2e/chat-context.mjs`에는 #55/#57의 current-summary attribution/diagnostic 기록과 #40의 readiness/requested-binding 회귀가 겹친다. #56의 `e2e/chat-controls.mjs` Edit-cancel hidden wait를 보존한다. 당시 dev package manifests는 `7af06f1`과 같았으나 새 통합 시 재확인한다.

마지막 순서: 변경을 보존해 최신 dev에 정합화 → 전체 새 독립 리뷰 → 정확한 commit의 로컬 backend/SDK/browser QA → 기존 원격 head를 확인한 guarded push → 필수 CI 7개 → 조건부 dev merge → 병합된 dev 검사. 미완성 카탈로그 상태에서는 진행 완료로 표시하지 않는다.

## 6. PR #46 — #40 이후 CI 변경 전달

원격 [#46](https://github.com/cskwork/scopeweaver/pull/46)의 head는 `fbbe340fd15282d6cf24a6b03dcf687e28937866`, 브랜치는 `fix/ci-churn`이다. 제목의 “push checks on main only”와 달리, 검토한 로컬 재구성본은 **dev/main 검증을 둘 다 유지**한다.

`42183237`의 변경은 `.github/workflows/ci.yml` 4줄뿐이다: `edited` 이벤트 이유 주석 및 workflow/ref별 concurrency cancellation. 기존 리뷰/로컬 검증 기록이 있으며 dev `935915b` 위 inspection-only 적용은 충돌 없고 `actionlint` exit 0이었다. 보관용 원격 checkpoint는 푸시했지만, 기존 PR #46 브랜치의 실제 rebase·갱신은 하지 않았다.

**#40 다음 #46** 순서를 유지한다. 최종 dev 기준 rebase, 정확한 4줄 delta/최신 workflow 보존 리뷰, 필수 CI 7개, dev merge와 병합 후 검증이 남았다. 옛 원격 변경으로 dev push 검증을 없애지 않는다.

## 7. 이슈 #49 / PR #57 — CI 녹색과 별개인 두 보류

[PR #57](https://github.com/cskwork/scopeweaver/pull/57), `feat/gh-49-opencode-sdk → dev`, 로컬/원격 head `b9781b2c4b92d9cef4a048f1c897616ce9ed8ff9`. 인계 조회 시 필수 **7/7 SUCCESS**다. 이는 병합/위험 수용 승인이 아니다.

### 기능 및 현재 진단 근거

- OpenCode SDK/free-model 경로, host ownership/permission 경계, connection cap 전달과 여섯 언어 문서 작업을 담는다. 이전 기능 후보 `29b6addb`의 코드·문서 독립 리뷰 PASS와 제한된 실제 free inference 결과가 있다. 그때의 broad Go **2,343 pass / opt-in skip 3**, cap 수정 후 affected **44 + 117**, SDK **62 pass / 324 assertions**, web **78 / static 31 pages**는 각 시점·범위의 기록이며 합산하거나 최신 전체 증명으로 바꾸지 않는다.
- 최신 진단 커밋 `b9781b2`는 부모 `4144f146606fec1eb465ef82f56b00167f1024f3`에 딱 3개 파일(`server/chat_controls_e2e_test.go`, `e2e/chat-controls-runner.mjs`, `e2e/chat-context.mjs`)의 검토된 관찰 패치를 1회 커밋·push한 것이다.
- [checks run 40212928847](https://github.com/cskwork/scopeweaver/actions/runs/40212928847), [e2e run 40212928938](https://github.com/cskwork/scopeweaver/actions/runs/40212928938), 각각 attempt 1 성공. chat-browser draft/standard는 **125/125 steps** 통과했다.
- `chat-browser-results` artifact ID `12606184732`, ZIP **792,347 bytes**, SHA256 `391f3763f23da52b60e4afdae1cea240f25b941e4a69af4bf1244af28148e7fe`. 한 번 내려받아 GitHub digest와 대조했다. 원래 호스트에 보존돼 있으므로 중복 다운로드를 자동 실행하지 않는다.
- 관측된 **144 interval pairs** 중 no-call **96개**의 delta는 0이었다. queue 24쌍은 각각 +2, manual compaction 24쌍은 각각 +1이었다. **156 planner requests**는 측정된 positive intervals 밖에 있었다. queue 입력 48개 중 1개는 반환 run/turn 기록이 없어 완전한 attribution이 아니다.

### 남은 보류 A — 과거 실패 원인

이 실행은 이전 `/context`·capacity global-counter 실패를 재현하지 못했다. 과거 CI attribution은 **INSUFFICIENT**, 원인은 **unknown**이다. 녹색 실행만으로 과거 실패가 수정됐거나 모든 producer/run 원인이 밝혀졌다고 하지 않는다. per-process/descendant cleanup도 검증 완료로 주장하지 않는다.

횟수 제한: 원래 시도 **3회**, 기존 독립 진단 FAIL **2회**, 별도 hosted 진단 **1/1 사용 완료**. Phase B preflight **0/1 INACTIVE**, Phase B browser **0/1 INACTIVE**. 해당 담당자는 결과 수집 후 **STOP**했다. 추가 local/hosted 진단, repair, rerun, 구현·의존성 변경이나 병합은 그 진단 승인의 범위 밖이다. 다음 담당자는 원래 제한을 읽고 새로 필요한 범위만 명시적으로 정해야 한다.

### 남은 보류 B — SDK 의존성 감사

별도 SDK adapter `bun audit --json`은 **exit 1**이다. frontend audit 0과 혼동하지 않는다.

| 의존성 | 보존된 감사 결과 | 미해결 이유 |
| --- | --- | --- |
| `@ai-sdk/provider-utils` 4.0.21 / 4.0.23 | `GHSA-866g-f22w-33x8`, low, fixed >=4.0.33 | core/provider의 명시적 pin |
| `toml` 4.3.0 | `GHSA-4qx8-w7c3-pj3p`, `GHSA-9j65-hcg3-gc64`, `GHSA-9qmh-m7c6-f92p`, high, fixed 5.0.1 | major override의 호환성 미증명 |
| `zod` 4.1.8 / 3.25.76 | `GHSA-mrhx-c7xc-m3w5`, high, fixed >=4.6.6 | plugin/core pin 및 `gitlab-ai-provider`의 3.x 계약; 조사 당시 patched stable 3.x를 찾지 못함 |

감사 억제, 무검증 major override, 취약 경로가 도달 불가능하다는 주장은 적용하지 않았다. 공식 CLI 번들 내부 의존성의 안전성까지 이 package audit가 보증하지 않는다. **호환성이 증명된 수정 또는 명시적인 release-owner risk disposition**이 필요하다. 인계는 위험 수용이 아니다.

원래 생성된 SDK lock의 정확한 bytes는 복구하지 못했고, 현재 재생성된 lock은 다르다. 직접 runtime pin과 native CLI binary 해시의 일치는 확인했지만, 과거 전체 transitive graph의 동일성이나 감사 실패가 lock drift 때문에 생겼다는 인과는 증명하지 않았다.

원래 호스트 evidence: `gh49-recovery-evidence/ci-context-diagnosis/hosted-execution/{REPORT.md,result.json,SHA256SUMS}`, `gh49-opencode-sdk/.sdlc/work/gh-49-opencode-support/refs/recovered-dependency-audit.md`. 원격 증거의 `gh49-plan/`, `gh49-evidence/`에 핵심 보고서와 텍스트 자료를 보존했다. 기존 임시 raw 디렉터리가 사라진 과거 기록과 새로 실행한 결과를 구분한다.

## 8. 이슈 #15 / #50, 병합 완료 PR, 승격 #53

- **#15**: #52가 여섯 언어 애플리케이션·문서를 dev에 통합했다. 이슈는 GitHub에서 여전히 OPEN이다. 이후 #40/#57이 바꾸는 내용도 여섯 언어 요건을 만족해야 한다.
- **#50**: #54가 scope 문구, domain-label boundary matching, gated asset writes와 shutdown ownership을 dev에 통합했다. scope는 prompt 지침/asset 기록 필터이며 네트워크 경계라는 보장을 새로 부여하지 않는다. 이슈는 여전히 OPEN이다. #51은 CLOSED다. 상태 변경은 수용 조건 확인 후 수행한다.
- **#55**: 현재 manual-compaction 요청의 focus를 검증하도록 attribution assertion 보강. merged dev `935915b`.
- **#56**: Edit-cancel dialog가 hidden 상태가 된 뒤 retry하는 한 줄 수정. PR head `80b135d`; merged dev `71a487f`. 기존 담당자의 local **120/120**, 해당 후보 7/7 결과와 post-dev 결과를 새 통합 결과로 혼동하지 않는다.
- **#53**: [draft promotion PR](https://github.com/cskwork/scopeweaver/pull/53). snapshot `71a487f`에서 PR/push 검사가 녹색이어도 #40/#46/#57 통합과 최종 전체 E2E 이전에는 승격하지 않는다. 인계 문서의 dev 반영으로 head/검사가 다시 바뀐다.

## 9. 재개 순서와 완료 기준

1. 최신 GitHub 이슈·PR·dev/main/tag 상태와 실행 중인 프로세스를 읽기 전용 확인한다. 이 문서의 고정 SHA가 현재값이라고 가정하지 않는다.
2. §3의 원격 refs와 증거 브랜치로 #40/#46 상태를 별도 작업 트리에 복구한다. 이미 있는 작업을 다시 만들거나 완료된 worker를 재시작하지 않는다.
3. #40의 차단된 번역 입력, 카탈로그 의미·형식·저장 호환성 검사를 해결한다. 독립적으로 가능한 reconciliation은 기존 승인 범위에서 진행하되 미완료 상태로 병합하지 않는다.
4. #57의 진단 중단 조건·SDK 감사 위험에 대한 다음 결정을 먼저 명확히 한다. 기존 1회 실행 승인을 재사용하지 않는다. #40와 겹치는 capture/worker/context/문서·의존성 변경은 최신 dev 기준으로 통합 리뷰한다.
5. #40 → #46의 종속 순서를 지키고, #57도 담당자와 합의된 최신 dev 위에서 완료한다. 각 정확한 후보의 필수 7개 검사와 병합 후 결과를 확인한다.
6. 모든 담당 작업이 dev에 들어간 **하나의 최종 SHA**에서 전체 backend/SDK/frontend/adapter/browser/E2E와 실패 경로를 검증한다. 날짜·SHA·환경·원래 실패·skip을 유지하며 서로 다른 후보의 결과를 합쳐 PASS로 만들지 않는다.
7. 최종 검증과 남은 위험 처리가 완료되면 #53의 제목/본문을 최종 범위로 갱신하고 dev→main을 merge commit으로 승격한다. main 검사 후 최신 태그를 재조회하고 minor release를 수행한다. 이슈 종료·릴리스 여부도 실제 원격 상태로 검증한다.

테스트 DB는 `scopeweaver_test_*` 또는 `scopeweaver_adapter_*`의 disposable DB만 사용한다. package별 분리, UTF-8, `LC_ALL=C LANG=C`, 독립 socket을 유지한다. 기본적으로 Docker 대신 native PostgreSQL을 쓴다. 환경 설치/라이브 호출을 기존 증거 복구와 혼동하지 않는다.

## 10. 근거의 위치와 해석

- GitHub 상태·heads·checks·release는 이 인계 작성 중 API로 직접 조회했다. 아래 전체 목록이 그 스냅샷이다.
- `.sdlc`와 recovery evidence의 과거 PASS/FAIL은 실제 저장된 기록을 읽었으며, 새로 재실행한 결과라고 쓰지 않았다.
- PR40 세부 기록: 원래 worktree의 `.sdlc/work/261025-pr40-review-repair/refs/resume-checkpoint.md`, `refs/release-coordination-handoff-20261025.md`, `refs/translation-input-manifest.json`, `docs-only-sol/documentation-evidence.json` 및 `review.out.md`.
- 릴리스 승인·한도 기록: `gh53-verification-gates/.sdlc/work/integrated-release-v0-8/refs/owner-authorization.md`, `pr57-hosted-execution-authorization.md`. 원격 증거의 `release/`에도 있다.
- 역사적 보고서의 “pending CI” 등은 해당 문서 시점의 표현이다. 최신 #57 head/7개 검사 상태는 §7이 우선하지만, 별도 audit/원인 규명 제한을 덮어쓰지 않는다.
- 이 문서의 배포는 인계 작업만 완료한다. 진행 중인 기능·리뷰·릴리스의 남은 승인과 검사를 면제하지 않는다.

## 11. 전체 이슈·PR 목록

아래 목록은 본 인계 문서를 위한 PR 생성 **전**의 GitHub 전체 조회 결과다. CLOSED와 MERGED를 구분하며, 현재 열리지 않은 항목을 재실행 대상으로 해석하지 않는다.

<!-- GitHub inventory appended from the captured API response. -->

목록 생성 시각: `2026-10-26T11:35:35+09:00`. PR 51개, 이슈 5개.

### 이슈 전체

| 번호 | 상태 | 제목 |
| --- | --- | --- |
| [#15](https://github.com/cskwork/scopeweaver/issues/15) | OPEN | Proposal: extensible i18n and additional locales (Chinese, Japanese, Spanish) |
| [#24](https://github.com/cskwork/scopeweaver/issues/24) | CLOSED | High Token Consumption – Recommended LLM Settings and Optimization Strategies |
| [#49](https://github.com/cskwork/scopeweaver/issues/49) | OPEN | [Feature] Add support for OpenCode |
| [#50](https://github.com/cskwork/scopeweaver/issues/50) | OPEN | Company/task scope reads as an authorization boundary but is not technically enforced for tool execution |
| [#51](https://github.com/cskwork/scopeweaver/issues/51) | CLOSED | Scope enforcement is prompt-level only: planner self-expands task_scope, chat asset inserts are ungated, task allowlist is opt-in and substring-escapable |

### PR 전체

| 번호 | 상태 | 경로 | 제목 |
| --- | --- | --- | --- |
| [#1](https://github.com/cskwork/scopeweaver/pull/1) | MERGED | `feat/agent-control-adapter` → `main` | Add ScopeWeaver control adapter for Claude Code, Codex and Pi |
| [#2](https://github.com/cskwork/scopeweaver/pull/2) | MERGED | `feat/landing-page` → `main` | Publish bilingual ScopeWeaver landing page |
| [#4](https://github.com/cskwork/scopeweaver/pull/4) | MERGED | `xKeNcHii/backport-artex-updates` → `main` | feat: backport ARTEX SSE, auth, and approval usage updates |
| [#5](https://github.com/cskwork/scopeweaver/pull/5) | MERGED | `chore/contrib-docs-usage-ci` → `main` | chore: contribution guide, E2E smoke test, and v0.2.0 prep |
| [#6](https://github.com/cskwork/scopeweaver/pull/6) | MERGED | `fix/release-notes-links` → `main` | ci(release): fix 404 links in release notes |
| [#7](https://github.com/cskwork/scopeweaver/pull/7) | MERGED | `xKeNcHii/upgrade-web-tooling` → `main` | fix(deps): update frontend tooling dependencies |
| [#8](https://github.com/cskwork/scopeweaver/pull/8) | MERGED | `xKeNcHii/upgrade-next-react` → `main` | fix(deps): upgrade Next.js and React |
| [#9](https://github.com/cskwork/scopeweaver/pull/9) | MERGED | `fix/install-mkdir-webui-dist` → `main` | fix(install): create server/webui before copying web/out into it |
| [#10](https://github.com/cskwork/scopeweaver/pull/10) | MERGED | `fix/release-0.2.1-quality` → `main` | fix: harden Windows and prepare v0.2.1 |
| [#11](https://github.com/cskwork/scopeweaver/pull/11) | MERGED | `fix/release-license-notices` → `main` | fix: preserve notices in release packages |
| [#12](https://github.com/cskwork/scopeweaver/pull/12) | MERGED | `fix/notice-manifest-read` → `main` | fix: read raw image manifests for notice verification |
| [#13](https://github.com/cskwork/scopeweaver/pull/13) | CLOSED | `feat/openrouter-provider-routing` → `main` | feat: add OpenRouter provider format with routing pin and fallback control |
| [#14](https://github.com/cskwork/scopeweaver/pull/14) | MERGED | `feat/llm-max-concurrent` → `main` | feat(llm): add per-provider concurrency cap (llm_max_concurrent) |
| [#16](https://github.com/cskwork/scopeweaver/pull/16) | MERGED | `fix/dashboard-daily-token-chart` → `main` | fix(dashboard): daily token chart double-counts cache reads |
| [#17](https://github.com/cskwork/scopeweaver/pull/17) | MERGED | `feat/asset-export` → `main` | feat(assets): CSV/JSON export endpoint with UI buttons |
| [#18](https://github.com/cskwork/scopeweaver/pull/18) | MERGED | `release/v0.3.0` → `main` | docs(release): prepare v0.3.0 |
| [#19](https://github.com/cskwork/scopeweaver/pull/19) | MERGED | `feat/gh-15-extensible-i18n` → `main` | feat(i18n): add extensible locales and Simplified Chinese |
| [#20](https://github.com/cskwork/scopeweaver/pull/20) | MERGED | `release/v0.3.1` → `main` | fix(release): include shared locales and prepare v0.3.1 |
| [#21](https://github.com/cskwork/scopeweaver/pull/21) | CLOSED | `feat/japanese-localization` → `main` | feat(i18n): add Japanese localization (UI, backend, agent output) |
| [#22](https://github.com/cskwork/scopeweaver/pull/22) | MERGED | `feat/ollama-native-flexible-providers` → `main` | feat(llm): support native Ollama alongside flexible cloud providers |
| [#23](https://github.com/cskwork/scopeweaver/pull/23) | MERGED | `fix/provider-env-auth` → `main` | fix(llm): preserve authenticated gateway credentials |
| [#25](https://github.com/cskwork/scopeweaver/pull/25) | MERGED | `feat/ollama-zero-egress-hardening` → `main` | Harden native Ollama scheduling, judge policy and strict local inference |
| [#26](https://github.com/cskwork/scopeweaver/pull/26) | MERGED | `test/db-test-isolation` → `main` | test(db): require ARTEX_TEST_DSN and refuse to run against the live database |
| [#27](https://github.com/cskwork/scopeweaver/pull/27) | MERGED | `fix/gh-24-token-consumption` → `main` | fix(llm): cut cloud token use and fix provider usage accounting (#24) |
| [#28](https://github.com/cskwork/scopeweaver/pull/28) | CLOSED | `feat/chat-controls-and-context` → `main` | feat(chat): add chat controls and document paste |
| [#29](https://github.com/cskwork/scopeweaver/pull/29) | MERGED | `feat/chat-controls-clean` → `dev` | feat(chat): add chat controls and document paste |
| [#30](https://github.com/cskwork/scopeweaver/pull/30) | MERGED | `fix/chat-findings-and-agent-prompts` → `dev` | fix(chat): persist standalone findings and preserve prompt roles |
| [#31](https://github.com/cskwork/scopeweaver/pull/31) | CLOSED | `docs/japanese-readme` → `main` | docs: add Japanese README translation |
| [#32](https://github.com/cskwork/scopeweaver/pull/32) | MERGED | `fix/chat-pr-review` → `dev` | fix(chat): preserve compatibility and consolidate chat state |
| [#33](https://github.com/cskwork/scopeweaver/pull/33) | MERGED | `dev` → `main` | chore: promote verified chat controls and findings |
| [#34](https://github.com/cskwork/scopeweaver/pull/34) | MERGED | `fix/release-v0.6.0` → `dev` | docs(release): prepare v0.6.0 notes |
| [#35](https://github.com/cskwork/scopeweaver/pull/35) | MERGED | `dev` → `main` | docs(release): promote v0.6.0 release notes |
| [#36](https://github.com/cskwork/scopeweaver/pull/36) | MERGED | `fix/release-runtime-patch` → `dev` | fix(chat): keep text-only queued edits readable |
| [#37](https://github.com/cskwork/scopeweaver/pull/37) | MERGED | `dev` → `main` | fix: promote v0.6.1 queued-edit patch |
| [#38](https://github.com/cskwork/scopeweaver/pull/38) | MERGED | `fix/cli-discovered-runtime-bugs` → `dev` | fix: preserve chat shutdown writes and repair shared runtime bugs |
| [#39](https://github.com/cskwork/scopeweaver/pull/39) | MERGED | `dev` → `main` | Merge dev into main: shutdown persistence and runtime fixes |
| [#40](https://github.com/cskwork/scopeweaver/pull/40) | OPEN | `fix/glm-agent-contracts` → `dev` | fix: preserve planner evidence and lifecycle contracts |
| [#41](https://github.com/cskwork/scopeweaver/pull/41) | CLOSED | `company-scope-host` → `main` | Company scope: `domain` rules cover subdomains; add a `host` kind |
| [#42](https://github.com/cskwork/scopeweaver/pull/42) | MERGED | `fix/company-scope-host` → `dev` | Company scope: `domain` rules cover subdomains; add a `host` kind |
| [#44](https://github.com/cskwork/scopeweaver/pull/44) | MERGED | `dev` → `main` | Merge dev into main: company scope host kind and subdomain matching |
| [#45](https://github.com/cskwork/scopeweaver/pull/45) | CLOSED | `chore/ci-churn` → `dev` | ci: run push checks on main only and cancel superseded runs |
| [#46](https://github.com/cskwork/scopeweaver/pull/46) | OPEN | `fix/ci-churn` → `dev` | ci: run push checks on main only and cancel superseded runs |
| [#47](https://github.com/cskwork/scopeweaver/pull/47) | MERGED | `feat/opencode-go-llm` → `dev` | feat(llm): accept opencode-go provider config and gateway template |
| [#48](https://github.com/cskwork/scopeweaver/pull/48) | MERGED | `dev` → `main` | release: promote dev to main for v0.7.0 |
| [#52](https://github.com/cskwork/scopeweaver/pull/52) | MERGED | `feat/gh-15-complete-locales` → `dev` | feat(i18n): complete six-language application and documentation |
| [#53](https://github.com/cskwork/scopeweaver/pull/53) | OPEN / DRAFT | `dev` → `main` | Promote dev to main: complete six-language localization |
| [#54](https://github.com/cskwork/scopeweaver/pull/54) | MERGED | `fix/gh-50-scope-enforcement` → `dev` | fix(scope): honest scope wording, label-boundary domain rules, gated agent scope writes, and race-free shutdown (#50) |
| [#55](https://github.com/cskwork/scopeweaver/pull/55) | MERGED | `fix/gh-53-verification-gates` → `dev` | test(chat): verify focus on the current compaction request |
| [#56](https://github.com/cskwork/scopeweaver/pull/56) | MERGED | `fix/gh-53-retry-dialog` → `dev` | test(chat): settle cancelled edit dialog before retry |
| [#57](https://github.com/cskwork/scopeweaver/pull/57) | OPEN | `feat/gh-49-opencode-sdk` → `dev` | feat(llm): add guarded OpenCode SDK and free-model support |
| [#58](https://github.com/cskwork/scopeweaver/pull/58) | MERGED | `fix/handoff-20261026` → `dev` | docs: record all-issue and PR continuation handoff |
