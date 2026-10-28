# `/btw` 검증 기록

[English](../../../sidequestion/VALIDATION.md) · 한국어 · [简体中文](../../zh-Hans/sidequestion/VALIDATION.md) · [繁體中文](../../zh-Hant/sidequestion/VALIDATION.md) · [日本語](../../ja/sidequestion/VALIDATION.md) · [Español](../../es/sidequestion/VALIDATION.md)

원본의 과거 기록입니다. 날짜: 2026-09-10. 브랜치: `codex/btw-side-question`. 기준: `8dae851b9b622f2ff2631f332fde9719d0b16fba`. 새 ScopeWeaver 결과가 아닙니다.

독립 PostgreSQL 테스트 DB와 데이터 디렉터리를 사용했습니다. 실제 모델 자격 증명은 해당 환경에만 주입했으며 코드나 기록에 저장하지 않았습니다. 제품 기본 모델도 바꾸지 않았습니다. Go 1.26.3, norma v0.3.6, Next.js 16.2.9.

모델 대화, 반환 객체, 검증 단언과 Qwen 검토는 [validation-2026-09-10.json](validation-2026-09-10.json)에 있으며 API 자격 증명은 없습니다. 서술 문자열은 번역했고 식별자와 측정값은 유지했습니다.

## 기술 검사

| 범위 | 결과 | 증거 |
| --- | --- | --- |
| 구조화 메시지와 도구 인수 깊은 복사 | 통과 | `TestCheckpointDeepCopyAndBoundaries` |
| 요약/압축 덮어쓰기 금지, 전체 응답/종료 공개, 부분 응답 제외 | 통과 | `TestCheckpointDeepCopyAndBoundaries`、`TestSnapshotExcludesPartialStreamAndSelectsPoolMember` |
| 실제 모델 풀 구성원 식별 | 통과 | `TestSnapshotExcludesPartialStreamAndSelectsPoolMember` |
| 도구 짝, 20쌍 재생, 예산 축소와 초과 오류 | 통과 | `TestBuildRequestCompactionToolPairingAndBudget` |
| 주/곁가지 병행과 양방향 취소 격리 | 통과 | 대기형 Provider，`TestMainSideConcurrencyAndIndependentCancellation` |
| 도구 실행 없음, 스트리밍/비스트리밍, 실패 시 확보 사용량 | 통과 | `TestServiceNoToolsAndUsageOnFailure` |
| 실제 norma ChatAgent + 로컬 Read, 주 transcript/활동 격리 | 통과 | `TestSideActualChatCheckpointToolResultAndTranscriptIsolation`, 스트리밍/비스트리밍 하위 검사 |
| 저장, 페이지 처리, 멱등성, 재시작 후 부분 답변 | 통과 | `TestSideHistoryIdempotencyPagingAndRecovery` |
| 비우기/늦은 쓰기 경합, 부모 삭제, 버전 비교 | 통과 | `TestSideClearLateWritersAndDeletedParent` |
| MainAgent/Worker 보관과 복원, v1/v2/v3 | 통과 | `TestSideTaskArchiveVersions` |
| 부모 API 세 종류, 인증, 소속, Worker 논리 삭제 | 통과 | `TestSideHTTPGlobalLimitTaskWorkerAndDeletion`、`TestSideCheckpointPersistsBeforeAdmissionAndRestart` |
| 주 세션 실행 중 질문, 독립 SSE 재연결/해제, 취소와 비우기 | 통과 | `TestSideHTTPBusyIsolationClearAndReconnect` |
| 부모당 1개 / 전체 4개 동시 실행 | 통과 | `TestSideHTTP…` 검사 두 개 |
| 제출 전 스냅샷 저장, 재시작 후 후속 질문, 구 세션 스냅샷 조작 금지 | 통과 | `TestSideCheckpointPersistsBeforeAdmissionAndRestart` |
| 캐시 프로필 삭제/모델 변경 시 거부 | 통과 | `TestSideRejectsDeletedOrChangedCachedProfile` |
| 보관 전 취소 및 최종 답변/사용량 저장 대기 | 통과 | `TestSideTaskDrainPersistsBeforeArchive` |
| 스트리밍 소비자 조기 취소 시 사용량/곁가지 귀속 한 번 기록 | 통과 | `TestSideUsageRecordedOnceOnConsumerCancellation` |
| 재시작 복구 Worker/deadline 실행 맥락이 새 스냅샷 공개 | 통과 | `TestSideRestoredWorkerRuntimePublishesNewCheckpoint` |
| 관련 패키지 race 검사 | 통과 | 아래 명령 |
| TypeScript와 운영 빌드 | 통과 | `npx tsc --noEmit`、`npm run build` |
| 새 프런트엔드 모듈 Biome | 통과 | `biome check`, 새 모듈 세 개 |

`ARTEX_PG_DSN`을 운영 DB가 아닌 별도 폐기 가능한 DB로 지정하여 자동 검사를 재현합니다:

```sh
go test -race ./agent ./db ./server ./sidequestion ./llmrec ./llmpool \
  -run 'Test(Side|Checkpoint|Snapshot|BuildRequest|Service|MainSide|CaptureRun|TaskArchive|CompleteForwards|StopIntent|CancelIntent)' -count=1
cd web
npx tsc --noEmit
npx biome check src/lib/side-questions.ts src/hooks/use-side-questions.ts src/components/side-question-workspace.tsx
npm run build
```

전체 Go 회귀 검사는 모두 통과하지 않았습니다. 기존 `server` 검사 두 개가 임시 디렉터리 정리 중 `TempDir RemoveAll … directory not empty`로 실패했습니다:

- `TestInheritedActivityDetailAndRelationDeletion`
- `TestTaskMetadataPatchReturnsRenameAndPin`

변경 없는 기준 소스를 내보내 같은 독립 환경에서 `server`를 다시 실행해 두 정리 실패를 재현했습니다. 기준 실행에서는 `TestCoreTaskLifecyclePG` 대상 노드 수 단언도 실패했지만 최종 수정본에서는 발생하지 않았습니다. 다른 패키지와 곁가지/race 검사는 통과했습니다. 기준 문제를 통과로 계산하거나 숨기기 위해 단언을 바꾸지 않았습니다.

Next.js는 기존 여러 lockfile/작업 루트 추론 경고를 출력했지만 빌드와 모든 페이지 생성은 완료했습니다.

## 브라우저 검사

Codex 내장 브라우저를 독립 로컬 Go 및 Next.js 개발 서버에 연결했습니다. 데스크톱과 390 × 844 화면에서 브라우저 자동 조작, 스크린샷/로그 검토를 수행했습니다:

- 일반 대화 실행 중 `/btw`로 주/곁가지 내용이 데스크톱 사이드바에 함께 표시됐습니다.
- 후속 질문이 동작했고 중지 시 일부 곁가지 답변을 유지하며 주 실행은 계속됐습니다.
- 패널을 닫아도 요청은 계속됐고 다시 열면 완료 답변이 복원됐습니다. 새로고침 뒤 빈 `/btw`로 기록을 복원했습니다.
- 좁은 화면 Drawer의 입력, 버튼, 기록, 닫기가 동작했고 가로 넘침이 없었습니다.
- 비우기는 확인창을 거쳐 기록을 삭제하고 주 transcript/스냅샷을 보존했습니다.
- MainAgent와 Worker 두 개 사이를 전환해도 이름표와 기록이 섞이지 않았습니다.
- 대기형 로컬 모델 테스트 구성으로 Worker를 실행 상태로 유지했습니다. 기본 입력에서 `/btw` 제출 후 곁가지만 중지해도 Worker 실행 상태와 자체 일시중지 버튼은 유지됐고 일부 곁가지 답변이 저장됐습니다.
- 브라우저 오류/경고 로그는 비어 있었습니다.

통제된 테스트 구성으로 모델 속도에 의존하지 않고 동시 실행 타이밍을 검증했습니다. 초기 Worker 검사 두 번은 작업/답변이 이미 끝나 유효한 병행 구간이 없었습니다. 구성을 고쳐 재검사한 결과 통과했으며 초기 시도는 통과로 세지 않았습니다.

## 실제 모델 대화

우선 `grok-4.6`을 OpenAI 호환 `http://127.0.0.1:12580/tingly/openai`로 확인해 2.82초 만에 HTTP 200, 모델 `grok-4.6`, `READY`를 받았습니다. 우선 모델이 동작하여 Tingly `glm`이나 Zhipu `glm-5.3` 대체 경로는 사용하거나 검증하지 않았습니다.

| 시나리오 | 실제 결과 |
| --- | --- |
| 주 실행 중 자산, 목표, 표식 질문 | `redhaze.top`, 홈페이지 읽기/요약 목표, `BTW-REAL-0910` 반환; 16.97초 완료 |
| 홈페이지 읽기 후 도구 증거 질문 | WebFetch 200, curl 301 → 302 → 200, 제목 정확히 인용; 7.24초 |
| 곁가지에 Bash 파일 생성 요청 | 거부; 대상 파일 생성 안 됨; 7.74초 |
| 완료된 곁가지가 주 맥락 유지 | 주 transcript SHA-256과 활동 동일; 곁가지 도구 실행 0회 |
| Go 실제 중지/재시작 후 후속 질문 | 이전 곁가지 기록 3개 유지; 주 에이전트 재실행 없이 저장 스냅샷으로 자산, 표식, 제목 답변 |
| 새 대화에서 비스트리밍 Grok | 자산과 `ATOMIC-0910` 정확; 사용량 반환/저장: input 11734, output 138, cache_read 11520 |

주 대화는 WebFetch와 Bash/curl로 공개 홈페이지를 읽었고 도착 주소는 `https://id.redhaze.top/home`, 제목은 “RedHaze Technology RedHaze Group · 글로벌 종합 그룹 포털”(번역)이었습니다. Bash는 로컬 임시 응답 파일을 저장했으며 원격 쓰기는 없었습니다. 이는 곁가지 도구 실행 없음과 별도로 확인했습니다.

주 transcript 체크섬: `e7e61f135a4a120954b539f357e8c4205d7d5cd7460dcaf3dc0fd066463e1d00`。

사용량 제한: Tingly Grok 스트리밍은 usage를 반환하지 않았습니다. 직접 `stream_options.include_usage=true`로 확인해 HTTP 200, 데이터 프레임 12개, usage 프레임 0개를 받았습니다. 따라서 0은 사용량 미제공이며 무료라는 뜻이 아닙니다. 비스트리밍과 테스트 실패/취소 사용량은 정상 저장됐습니다.

## Qwen 검토

검토 모델 `qwen-flash`, OpenAI 호환 `https://dashscope.aliyuncs.com/compatible-mode/v1`은 HTTP 200을 반환했습니다. 최초 실제 곁가지 대화 세 개, 주 도구 증거, 검증 단언을 제공했고 `verdict: accept`, `concerns: []`를 받았습니다. 자산, 표식, 페이지 증거와 답변이 일치하고 도구 거부가 제약에 맞는다고 평가했습니다. 사용량: prompt 6625, completion 312, total 6937.

이 검토는 이후 추가한 재시작/비스트리밍 검사를 포함하지 않았습니다. Qwen의 “쓰기 없음”은 과도한 일반화였습니다. 위 기록대로 주 세션 curl은 로컬 응답 파일을 만들었습니다. 병행, 곁가지 도구 실행 0회, transcript 격리는 기술 단언으로 판단하며 모델 검토는 답변 품질 평가를 보조합니다.
