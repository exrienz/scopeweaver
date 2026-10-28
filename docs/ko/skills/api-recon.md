# API Recon (프런트엔드 인터페이스 탐색)

[English](../../../skills/api-recon/SKILL.md) · 한국어 · [简体中文](../../zh-Hans/skills/api-recon.md) · [繁體中文](../../zh-Hant/skills/api-recon.md) · [日本語](../../ja/skills/api-recon.md) · [Español](../../es/skills/api-recon.md)

승인된 범위에서 백엔드 API(경로, 메서드, 매개변수, 응답 본문), 프런트엔드 경로, UI 동작(탭, 대화상자, 표 조작)을 가능한 한 완전하게 수집합니다.

---

## 범위와 금지 사항 (필독)

이 스킬은 API와 매개변수만 조사하며 취약점 탐색이나 악용을 수행하지 않습니다.

### 작업 범위

| 범위 | 허용 | 금지 | |
|---|---|---|
| 목표 | 경로, 메서드, 매개변수, 라우트, UI 동작 나열 | SQLi/XSS, 권한 공격, 무차별 대입, 취약점 퍼징, 요청 변조 공격, 파괴적 작업 | |
| 인증 | Hook과 stub/mock으로 클라이언트 로그인 가드 처리 | 자격 증명 요청/추측, 실제 로그인 폼 제출 | |
| 실행 시 | 자격 증명 없이 Hook 및 mock으로 SPA 로그인 후 틀 렌더링 | 실제 백엔드 세션이 필요한 흐름 | |

### 자격 증명 없는 동적 분석 (Phase 3 기본)

1. `preload.js` / `runtime_harvest.js`로 로그인, 권한, 메뉴 등 bootstrap API를 가로채 stub으로 처리합니다.
2. 조회 API에는 올바른 구조와 성공 코드의 mock 본문을 반환하며 데이터는 비어 있어도 됩니다.
3. 백엔드가 없거나 401이어도 로그인 후 화면을 렌더링하여 XHR/fetch/WebSocket 요청을 더 발생시킵니다.
4. 빈 데이터, 빈 표, 임시 UI는 예상된 상태입니다. 실제 로그인이나 취약점 검사로 전환하지 마세요.

mock으로 프런트엔드 경로/컴포넌트를 열고 나가는 요청만 기록합니다. 목적은 실제 데이터를 얻는 것이 아니라 프런트엔드가 추가로 보내는 API를 찾는 것입니다.

### 절차상 금지 사항

| 금지 | 대안 | |
|---|---|
| Phase 1 완료 전 주 `index-*.js`를 grep/curl/Read로 읽어 API 추출 | `OUTDIR/harvest_static.py` 실행 |
| `extract_apis.py` 등 harvest 대체 스크립트 작성 | `OUTDIR/harvest_static.py` 수정 후 재실행 |
| 같은 grep/명령이 두 번 이상 실패했는데 반복 | tool_logs 확인, harvest 수정, reference 참조로 전략 변경 |
| A/B 확인 없이 원본 `scripts/` 실행 | OUTDIR에 복사하여 대상에 맞게 수정 |
| 실제 계정/비밀번호, OTP, OAuth 인증 | 위의 stub/mock 사용 |
| 실제 데이터를 얻으려고 stub을 생략하고 권한/주입 검사 | 조사 범위 내 나가는 요청만 기록 |
| 삭제, 민감 데이터 내보내기, 일괄 쓰기 등 되돌릴 수 없는 작업 | coverage 클릭에도 동일 적용 |
| 실행 시/동적 열거 없이 모든 페이지/API 수집 주장 | 완료 정의 충족 또는 한계 명시 |
| 동작 매트릭스와 diff 없이 모든 매개변수 파악 주장 | Phase 3b 매트릭스 + Phase 5 diff |
| 실행 표본 하나로 필수/선택 판단 | 여러 표본 비교 또는 검증 규칙/오류 분석 |

---

## 두 계층과 실행 모드

| 계층 | 산출물 | 한계 | |
|---|---|---|
| 정적 (JS bundle) | 엔드포인트 경로, 라우트 초안, 요청 조립 지점의 필드 후보 | HTTP 메서드 없음, 매개변수는 Phase 1b 필요, 실행 중 조합 URL 누락 |
| 실행 시 (활성 세션) | 메서드, 본문, 응답, 동적 URL, WS/SSE; 표본 diff로 매개변수 보완 | 페이지가 렌더링되어야 요청 발생; 표본 하나로 필수/선택 확정 불가 |

| 모드 | 엔진 | 용도 | |
|---|---|---|
| **depth** | `runtime_harvest.js` (Puppeteer) | API 목록, 메서드/매개변수/응답, WS/SSE, 재현 가능한 일괄 실행 |
| **coverage** | browser + `preload.js` | 탭/대화상자/표 클릭으로 기능 범위 확대 |
| **both** | depth 후 coverage | 가장 완전하며 가장 오래 걸림 |

매개변수 방법론(범용 스크립트 없음): 경로는 harvest/정규식, 매개변수는 기준점 주변 확장, UI 연결 추적, 여러 표본 diff, 오류 역추론을 사용합니다. grep 예제는 [reference](api-recon-reference.md) J절입니다.

---

## 완료 정의

다음 조건을 모두 충족해야 조사가 완료됐다고 할 수 있습니다:

- [ ] 정적: Phase 1에서 `api_static.txt`, `routes.txt`, `js/` 생성
- [ ] 실행 시: 최소 depth 또는 coverage; coverage/both는 Hook 동작과 동적 열거 루프 필요
- [ ] 화면 틀: 업무 경로가 `/login`으로 돌아가지 않음 (hash 라우팅 확인)
- [ ] 매개변수: coverage/both 동작 매트릭스와 `param_samples.json` 완료; Phase 5에서 `params_merged.json` 병합
- [ ] 모듈이 비면 Phase 4 권한 트리를 복원하고 locale/bootstrap 외 모듈 API가 나올 때까지 재실행
- [ ] 전달: Phase 5 산출물 전체; `insert_assets`로 서비스/엔드포인트 자산 저장

---

## 스크립트와 확인 단계

`scripts/`는 참고 템플릿입니다. 수정 없는 원본 실행을 최종 결과로 삼지 마세요.

읽고 대상에 맞게 수정하여 `OUTDIR`(예: `recon/`)에 쓰고 `CHANGES.md`에 기록합니다. 맞지 않으면 방법론에 따라 다시 작성하고 구조만 참고합니다.

| 확인 | 시점 | 템플릿 → OUTDIR 사본 | 일반 수정 항목 | |
|---|---|---|---|
| A (정적) | Phase 0 후, 첫 harvest/spider 실행 전 | `harvest_static.py` / `spider_mpa.py` | 대부분 기본 정규식 사용 가능; manifest/문법 불일치 시 endpoint 정규식, webpack/Vite `publicPath`, MPA exclude/cookie 수정 |
| B (실행 시) | Phase 2 후, depth/coverage 전 | `runtime_harvest.js` / `preload.js` + `config.json` | Cookie/localStorage 키, neutralize 성공 값, stubs, login 정규식, API 접두사, hash/history |

SPA 필수 순서: 스크립트 전 탐색보다 단계 번호를 우선하며 순서를 바꾸지 않습니다:

| 단계 | 필수 | 금지 | |
|---|---|---|
| Phase 0 완료 후 | 다음 Bash 명령 = `python3 OUTDIR/harvest_static.py <URL> OUTDIR` | 주 `index-*.js` curl/grep/Read (보통 >500KB) |
| 확인 A | 스크립트 복사, 필요한 작은 수정, 즉시 실행 | API 수동 추출 후 harvest 여부 결정 |
| Phase 1 완료 전 | `wc -l`로 출력 확인; 404 시 harvest 수정 후 재시도 | 추출 스크립트 작성; 내려받지 않은 URL 반복 grep |
| Phase 1b부터 | `OUTDIR/js/*.js`만 grep | harvest 대신 주 bundle 사용 |

- 올바른 순서: `harvest_static.py` 복사, 필요 시 정규식 수정, 즉시 실행
- 잘못된 순서: 주 bundle curl, 반복 grep, 임시 추출기, 마지막에 harvest
- MPA: Phase 0 후 다음 Bash 명령 = `python3 OUTDIR/spider_mpa.py ...`

---

## 도구와 출력 제한

| 제한 | 규칙 | |
|---|---|
| 큰 파일 | 100KB 초과 `index-*.js`를 Read/grep으로 맥락에 넣지 말고 OUTDIR 스크립트로 처리 |
| grep 출력 | `\| head -20` 또는 `-m 5` 필수; bundle 조각 대신 경로 요약 유지 |
| 검증 | `wc -l`, `ls \| wc -l` 사용; 전체 디렉터리 Read 금지 |
| 정규식 예비 확인 | 선택, 최대 1회, 50KB 이하 chunk/HTML만; 정적 결과는 harvest 기준 |
| reference | 예제/템플릿/문제 해결은 [reference](api-recon-reference.md); 전체 내용을 중복 삽입하지 않음 |

---

## 실행 순서

```
Phase 0 분류 + OUTDIR
  → 확인 A → Phase 1 harvest (즉시 실행)
  → Phase 1b 매개변수 역추적
  → Phase 2 인증 확인 세 단계 → config.json
  → 확인 B → Phase 3 실행 시 분석 + 매개변수 매트릭스
  → Phase 4 필요 시 권한 트리 → Phase 3 재실행
  → Phase 5 보고서 병합 + 발견한 모든 서비스/엔드포인트를 insert_assets로 저장; 누락 금지
```

순서대로 확인하며 이전 단계를 마치기 전에 다음 단계로 가지 않습니다.

1. [ ] **Phase 0**: SPA/MPA 분류; `OUTDIR` 생성 → [Phase 0](#phase-0-분류)
2. [ ] **확인 A + Phase 1**: 스크립트 복사, 즉시 harvest, `wc -l` 검증 → [Phase 1](#phase-1-정적-탐색)
3. [ ] **Phase 1b**: 기준점 주변 확장 + 연결 계층 → `param_candidates.json` → [Phase 1b](#phase-1b-매개변수-역추적)
4. [ ] **Phase 2**: 인증 확인 세 단계 → `config.json` → [Phase 2](#phase-2-인증-확인-세-단계)
5. [ ] **확인 B**: 실행 시 스크립트 수정 → [Phase 3](#phase-3-실행-시-분석)
6. [ ] **Phase 3**: depth / coverage / both；화면 틀 진입 확인; 매개변수 동작 매트릭스 → `param_samples.json`
7. [ ] **Phase 4**필요 시: 권한 트리 → stub 수정 → Phase 3 재실행 → [Phase 4](#phase-4-권한-트리-복원)
8. [ ] **Phase 5**: 산출물 병합 + 보고서 + `insert_assets` → [Phase 5](#phase-5-병합과-보고)

---

## Phase 0: 분류

진입 HTML을 가져오고 `OUTDIR`을 만듭니다. 스킬의 `scripts/`는 수정하지 않습니다:

- **SPA**: 빈 틀 + `<div id=app>` + chunk → Phase 1–5
- **MPA**: SSR + `<form>`, endpoint bundle 없음; 확인 A 이후:

```bash
python3 recon/spider_mpa.py <BASE_URL> <OUTDIR> [--cookie "session=..."] [--max 300] [--depth 5] [--exclude "logout|delete|destroy"]
```

산출물: `forms.txt`, `links.txt`, `api_inline.txt`. SPA에서 form이 거의 없으면 Phase 1로 전환합니다.

---

## Phase 1: 정적 탐색

[스크립트와 확인 단계](#스크립트와-확인-단계), [도구와 출력 제한](#도구와-출력-제한)을 따릅니다.

```bash
python3 recon/harvest_static.py <BASE_URL> <OUTDIR>
```

harvest는 HTML script와 webpack/Vite manifest를 분석하고 모든 lazy chunk를 내려받아 `js/`, `api_static.txt`, `routes.txt`, `chunkmap.txt`를 만듭니다.

```bash
wc -l OUTDIR/api_static.txt OUTDIR/routes.txt
ls OUTDIR/js | wc -l
```

- chunk 수와 manifest 비교; 404 시 개별 curl 대신 harvest 수정 후 재시도
- `api_static.txt`가 너무 적으면 OUTDIR endpoint 정규식을 넓혀 재실행 (reference 참고)

### Phase 1b: 매개변수 역추적

경로는 Phase 1에서 얻고 매개변수 필드는 별도 조사합니다. [출력 제한](#도구와-출력-제한)을 따릅니다.

완료 기준: 주요 API의 필드 이름, 전송 위치, 추정 형식, 필수 여부, 표본 값, 신뢰도를 설명할 수 있어야 합니다.

#### 1b.0: 전송 형식

| 형식 | 매개변수 위치 | 정적 증거 | |
|---|---|---|
| REST JSON | body + query | 경로 기준점 주변 `(params\|data\|body)\s*:\s*\{` |
| GraphQL | `variables` | gql 템플릿, `$page: Int` |
| 일반 form | urlencoded | `<form>`, `FormData` |
| 파일 업로드 | multipart | `FormData.append` |
| 경로 매개변수 | `/user/:id` | 라우트 표 + `useParams` / `$route.params` |
| 암호화/서명 | `sign`/`data`에 포장 | 암호화 함수 입력 Hook (reference D) |

산출물: API별 다음 표시 `transport: query|json|form|graphql|encrypted`.

#### 1b.1: 기준점 주변 확장

알려진 경로 주변을 넓혀 요청 조립 객체를 찾습니다:

```bash
grep -n '"/api/user/list"' OUTDIR/js/*.js | head -20
grep -rhoaE '.{0,120}("/api[^"]+").{0,200}' OUTDIR/js/*.js | head -20
grep -rhoaE '(params|data|body|payload)\s*:\s*\{' OUTDIR/js/*.js | head -20
```

| 래퍼 | 매개변수 단서 | |
|---|---|
| axios 인스턴스 | `data` / `params` |
| 공통 request 래퍼 | 인터셉터가 넣는 전역 필드 |
| OpenAPI 클라이언트 | 생성된 메서드 시그니처 |
| React Query / SWR | hook 두 번째 인수 |
| Vue composable | composable 인수 |

형식 단서: `yup`/`zod`/rules, `Form.Item name=`, 내장 Swagger.

→ `param_candidates.json`: `{ path, fields[], source: "static-callsite", confidence }`

#### 1b.2: 연결 계층

```
Form field → onFinish/handleSubmit → transform → API payload
```

| 연결 원본 | 방법 | |
|---|---|
| 폼 submit | submit → transform → API 추적 |
| 표 검색 | `getFieldsValue()` → `params` |
| 라우트 | `:id` / `?tab=` |
| 인터셉터 | 전역 `tenantId`, 페이지, sign |
| 열거 select | `options` → API 열거 값 |

DevTools 호출 스택에서 `fetch`/`XHR.send`부터 요청 조립 함수를 역추적합니다.

#### 1b.3: 요청 조립 질문 세 가지 (Phase 2 인증과 구분)

| 질문 | 답할 내용 | |
|---|---|
| 조립 | payload 생성 위치와 변환 흔적 |
| 검증 | required, pattern, enum |
| 전송 | path / query / body / multipart / 헤더 |

Phase 2 인터셉터 확인에서 전역 삽입 필드(Authorization, `X-Tenant-Id`, sign)도 읽습니다.

#### 1b.4: Phase 3 연결

후보는 정적/연결 분석에서 얻습니다. 필수, 선택, 조건 의존성은 Phase 3 매트릭스/diff와 Phase 5 오류 역추론이 필요합니다.

---

## Phase 2: 인증 확인 세 단계

`OUTDIR/js/`를 `head`와 함께 grep하고 `config.json`에 기록합니다 (reference 예제):

| 확인 | 질문 | 키워드 | |
|---|---|---|
| 렌더링 | 로그인 상태를 어떻게 판단하는가? | `isLogin`, `getToken`, Cookie/localStorage |
| 인터셉터 | 무엇이 `/login` 이동을 유발하는가? | `response_code`, `errno`, axios interceptor |
| 내용 | 메뉴/권한은 어디에서 오는가? | `menu`, `permission`, `role`, `acl`, `routes` |

localStorage 키 이름을 자격 증명으로 간주하지 말고 chunk/요청 흐름으로 확인하세요.

출구 = 확인 B: 결론을 `config.json`에 쓰고 `OUTDIR/runtime_harvest.js` / `preload.js`를 수정합니다.

### Phase 2b: API 관찰 (선택)

OUTDIR의 `preload.js`로 세션 키, Authorization, 중첩 API URL을 확인합니다:

| 설정 | 산출물 | |
|---|---|
| `recordDetail: true` | `__API_RECON_DETAIL__` |
| `observe.xhrHeaders: true` | 헤더 관찰 |
| `extractUrlsFromResponse: true` | 응답 내 하위 API |
| `observe.storageReads/cookieReads: true` | config 반영 |
| `neutralizeVueRouter: true` | `__API_RECON_ROUTES__` |

coverage 매 회차 내보내기:`__API_RECON_LOG__`, `__API_RECON_DETAIL__`, `__API_RECON_ROUTES__`, `__API_RECON_OBSERVE__`.

---

## Phase 3: 실행 시 분석

확인 B를 마친 뒤 [범위](#범위와-금지-사항-필독)와 자격 증명 없는 mock 전략을 따릅니다.

`config.json`에 설정 `"runtimeMode": "depth" | "coverage" | "both"` (템플릿은 reference).

### Hook과 stub (depth/coverage 공통)

| 계층 | 범위 | 목적 | |
|---|---|---|
| L1 정확 | auth/권한/bootstrap stub | 첫 화면 인증 처리 |
| L2 실패 응답 보정 | 모든 JSON 응답 | 미로그인 코드 → 성공 |
| L3 대체 | L1에 맞지 않은 `/api` 등 | 빈 성공 본문으로 UI 렌더링 |

- **depth**: 가짜 인증 + `forward` 업무 코드 보정 + `stubs`; `routes` 순회(hash/history); `runtime_api.json` 생성
- **coverage**: 문서 시작 시 `preload.js` 주입 (CDP `addScriptToEvaluateOnNewDocument` 또는 userscript)

`window.__API_RECON_PRELOAD__`가 있고 업무 경로가 `/login`으로 돌아가지 않는지 확인합니다.

```bash
cd recon && npm install
node runtime_harvest.js config.json
```

### 3b: coverage 동적 열거 (필수)

1. 주 메뉴/사이드바: 각 항목 클릭 후 네트워크 1~3초 대기
2. 탭: `role=tab`, `.ant-tabs-tab`
3. 표: 첫 행 보기/편집/상세
4. 도구 모음: 내보내기, 필터, 새 항목 (되돌릴 수 없는 삭제 금지; 모든 금지 규칙 유지)
5. 각 모듈 진입: API/라우트 병합
6. SPA: 미방문 `routes.txt` 경로를 통제된 `pushState`로 열기 (MPA 금지)

필수 매개변수 동작 매트릭스: 모듈별 각 조작 유형을 기록하고 여러 표본을 비교합니다:

| 조작 | 일반 추가 매개변수 | |
|---|---|
| 목록 첫 화면 | 페이지 + 기본 필터 |
| 검색 | keyword, filter |
| 고급 필터 | 추가 선택 필드 |
| 생성/편집 | 전체 entity |
| 일괄/내보내기/정렬 | `ids[]`, `exportType`, `sortField` |

stub을 사용해도 나가는 본문/헤더는 실제 프런트엔드 출력이므로 요청을 근거로 삼습니다. `scan_raw.json`, `param_samples.json`, `api_detail.json`에 기록합니다.

- **Vue**: `neutralizeVueRouter: true` + document-start preload
- **React**: `routes.txt` + 사이드바 클릭 + `pushState`
- **both**: 3a depth 후 3b coverage

---

## Phase 4: 권한 트리 복원

조건: 모듈이 비거나 모든 경로에 bootstrap(locale 등)만 있으면 내용 확인을 통과하지 못한 상태입니다.

| 증상 | 의미 | |
|---|---|
| 화면 틀 진입 | 렌더링/인터셉터 확인 통과 |
| 사이드바 누락/클릭 후 빈 화면 | stub 구조 또는 권한 코드 부족 |
| 모든 경로에 같은 소수 API | `v-if permission` 실패 |
| `routes.txt`가 bundle 경로보다 훨씬 적음 | auth 모듈에서 보완 |

```bash
grep -rhoaE '"/api[^"]*(permission|perm|role|menu|acl)[^"]*"' OUTDIR/js/*.js | sort -u | head -30
grep -rhoaE 'userRouteAuth|getResultTree|routeMap|routeLink|menuList|authList' OUTDIR/js/*.js | head -20
```

일반 흐름:`role_permissions`（flat codes）+ `permissions/all`（tree）→ `getResultTree` → `userRouteAuth[CODE].url`.

```bash
python3 recon/extract_route_map.py recon/js recon/
python3 recon/build_perm_tree.py recon/js recon/ --config recon/config.json
```

중간 산출물:`route_map.json`, `userRouteAuth.json`, `permissions_tree.json`, `*_stub.json`, `perm_codes_all.txt`.

stub 확인: 외부 `response_code`가 인터셉터 조건과 일치하고, flat code와 tree가 맞으며, `routes`가 모든 `route_map` link를 포함해야 합니다.

`config.json` 갱신 후 Phase 3을 다시 실행합니다. 큰 SPA는 `waitUntil`, `routeTimeout`, `perRouteMs`를 조정할 수 있습니다 (reference A3/I).

---

## Phase 5: 병합과 보고

### 산출물

| 파일 | 단계 | 내용 | |
|---|---|---|
| `js/`, `api_static.txt`, `routes.txt`, `chunkmap.txt` | 1 | 정적 bundle과 경로 |
| `param_candidates.json` | 1b | 정적 매개변수 후보 |
| `config.json` | 2 | 세 확인 단계 + 실행 설정 |
| `runtime_api.json` | 3a | WS/SSE 포함 depth 상세 기록 |
| `param_samples.json`, `scan_raw.json`, `api_detail.json` | 3b | 여러 표본, 클릭 로그, 상세 |
| `route_map.json` 등 | 4 | 권한 트리 중간 파일 (수행 시) |
| `params_merged.json` | 5 | 병합 매개변수 필드 + 신뢰도 |
| `api_merged.txt` | 5 | `METHOD /path [params] [static\|runtime\|both]` |
| `site_map.json` | 5 | 경로, API, 매개변수, 기능, 한계 |
| **insert_assets** | 5 | 모든 서비스/엔드포인트 자산 저장 |

### 5b: 매개변수 병합

`param_samples.json`을 diff합니다. 범용 병합 스크립트는 없습니다. 신뢰도 규칙은 reference J7(높음/중간/낮음/트리거 대기)입니다.

### 5c: 오류 역추론

승인 범위에서 불완전 요청의 400 응답을 매개변수 조사에 사용할 수 있습니다. 취약점 검사가 아닙니다. `field 'x' is required`, 열거 오류 등을 확인하고 `data` 래퍼, `variables`, 암호화 전 `bizData`를 고려합니다.

runtimeMode, 정적/실행 API 수, 매개변수 신뢰도, 미방문 모듈, 템플릿 대비 `CHANGES.md` 요약을 보고합니다.

권장 `site_map.json` 구조:

```json
{
  "site": "https://example.com",
  "runtimeMode": "both",
  "appType": "vue-spa",
  "routeGuardStrategy": ["nav-neutralize", "L1-auth", "L2-patch", "forward"],
  "apisFromStatic": [],
  "apisFromRuntime": [],
  "apis": [],
  "params": [{ "method": "POST", "path": "/api/user/list", "transport": "json", "fields": [] }],
  "frontendRoutes": [],
  "routesVerifiedByClick": [],
  "featuresTriggered": [],
  "limitations": ""
}
```

추가 필드와 grep 예제: [reference](api-recon-reference.md).

---

## 일반 참고

- 프레임워크 독립: webpack/Vite/Angular lazy load에 같은 방법 적용
- 전송: REST/JSON, GraphQL, WebSocket, SSE; gRPC-web 제외
- SSR: 클라이언트 fetch 기록 가능; RSC/Server Actions 전체 열거 불가
- 한계: JSVMP, WASM, 강한 HMAC/mTLS 검증 → 정적 분석과 한계 명시
- 매개변수 한계: 조건 연동, 숨은 매개변수, WASM 조립 → 트리거 대기/접근 불가
- 정적 대안: 실행이 막혀도 정적으로 엔드포인트 수집 가능

---

## 추가 자료

- grep 예제, `config.json` 템플릿, 문제 해결, Hook, 매개변수 역추적(J), site_map 템플릿: **[reference](api-recon-reference.md)**
- 템플릿 경로: [스크립트와 확인 단계](#스크립트와-확인-단계)
