# api-recon 참고서

[English](../../../skills/api-recon/reference.md) · 한국어 · [简体中文](../../zh-Hans/skills/api-recon-reference.md) · [繁體中文](../../zh-Hant/skills/api-recon-reference.md) · [日本語](../../ja/skills/api-recon-reference.md) · [Español](../../es/skills/api-recon-reference.md)

grep 예제, `config.json` 템플릿과 문제 해결입니다. grep은 `js/`에서 실행합니다. 한 줄 bundle은 필요하면 `js-beautify` 또는 `sed 's/}/}\n/g'`를 사용하되 보통 주변 범위를 포함한 raw grep이면 충분합니다.

## 스크립트 안내

`scripts/`의 모든 파일은 참고 템플릿입니다. 실행 전에 대상에 맞게 수정하세요. 일반 수정 사항:

| 스크립트 | 일반 수정 사항 |
|---|---|
| `harvest_static.py` | endpoint 정규식, webpack/Vite manifest, 마이크로프런트엔드 publicPath, 재시도/병행 |
| `runtime_harvest.js` | neutralize 필드/성공 값, stub 일치/본문 구조, 경로 출처, WS 기록, `waitUntil`/`routeTimeout`/`proxy` |
| `preload.js` | `loginPathRe`, L1 stubs, `neutralize.fields`, `apiPattern`, L3 활성화 여부, `recordDetail`, `observe.*`, `neutralizeVueRouter` |
| `spider_mpa.py` | 파괴적 링크 `--exclude`, cookie, depth/max, 동일 출처 필터 |
| `extract_route_map.py` | `routeMap` / `routeLink` 정규식, KEY 이름 패턴 |
| `build_perm_tree.py` | `userRouteAuth` 분석, `ROOTS`/`PREFIX_PARENT` 계층 추정, stub 외부 필드 이름 |
| `config.json` | 위 대상별 매개변수의 통합 설정 |

수정 파일은 `recon/` 등 작업 디렉터리에 두고 템플릿 대비 변경 사항을 보고하세요.

---

## A. 세 확인 단계 역추적

### A1. 렌더링: 로그인 상태 판단 방법

```bash
grep -rhoaE '.{0,40}(isLogin|isAuthenticated|loggedIn|hasLogin|requireAuth)\b.{0,80}' js | head
grep -rhoaE 'function (getUser|getToken|getAuth)[0-9]?\([^)]*\)\{.{0,200}' js | head
grep -rhoaE '(localStorage|sessionStorage)\.getItem\("[^"]+"\)' js | sort -u
grep -rhoaE '(Cookies?|cookie)\.(get|load)\("[^"]+"\)' js | sort -u
grep -rhoaE '\batob\(|JSON\.parse\(|jwt|decode' js | head
```

`isLogin = f(getUser())` → `getUser = decode(storage.read(KEY))`를 추적해 저장 키, 저장소(Cookie/localStorage), 인코딩을 확인합니다:

| 인코딩 | config mock 값 |
|---|---|
| 평문 문자열 / `"1"` / token | `"value": "anything-truthy"` |
| `JSON.parse(x)` | `"value": "json:{\"id\":1,\"username\":\"admin\"}"` |
| `JSON.parse(atob(x))` | `"value": "b64json:{\"id\":1,\"username\":\"admin\"}"` |
| JWT | 서명 없는/`alg:none` JWT 또는 bundle 내 키 서명, 클라이언트 mock 가드에만 사용 |
| 암호화 (SM2/AES/RSA) | 하드코딩 키 확인; 렌더링 가드가 해독 가능한 blob만 필요할 때 mock, 아니면 정적 분석 |

`cookies` / `localStorage`에 기록합니다.

### A2. 인터셉터: /login 이동을 유발하는 조건

```bash
grep -rhoaE '.{0,60}(interceptors\.response|axios|request\.use).{0,120}' js | head
grep -rhoaE '.{0,40}(response_code|errcode|errno|\bcode\b|\bret\b|\bstatus\b)\s*[=!]==?\s*[\-0-9]{1,4}.{0,60}' js | head -20
grep -rhoaE '.{0,40}(未登录|请重新登录|登录已过期|unauthorized|登录失效|授权|token.{0,10}invalid).{0,40}' js | head
grep -rhoaE '.{0,30}(location\.href|router\.(push|replace)|navigate)\([^)]*login[^)]*\)' js | head
```

필드 이름, 성공 값(보통 `0` 또는 `200`), 이동을 유발하는 실패 값을 확인합니다. 가짜 세션으로 확인:

```bash
curl -sk -X POST -H 'Cookie: <fakekey>=junk' https://target/api/<protected> -d '{}' -H 'Content-Type: application/json'
```

`neutralize.fields` + `neutralize.success`에 기록합니다.

### A3. 내용: 메뉴와 권한의 출처

```bash
grep -rhoaE '"/api[^"]*(permission|perm|role|menu|acl|resource|nav)[^"]*"' js | sort -u
grep -rhoaE '.{0,30}(menus|permissions|menuList|routeList|authList|role_permissions)\b.{0,120}' js | head
grep -rhoaE 'userRouteAuth|getResultTree|routeMap|routeLink|hasPermission|checkAuth' js | head
grep -rhoaE '([A-Z_][A-Z0-9_]*):\{name:"[^"]*",link:"/[^"]+"\}' js | head
```

관리 화면에서 흔한 두 데이터 계층:

| API | 일반 payload | 소비 측 |
|---|---|---|
| `.../role_permissions` | `{ permissions: string[], role_type }` | 라우트 가드, 버튼 ACL |
| `.../permissions/all` | `tree[{ code, position, children }]` | 사이드바 메뉴 렌더링 |
| bundle 내 `userRouteAuth` | `{ CODE: { url, name? } }` | code → 프런트엔드 경로 |
| bundle 내 `routeMap` | `{ KEY: { name, link } }` | 별칭 해석 (webpack `o.DASHBOARD`) |

소비 측 코드에서 `getResultTree(tree, permissions)` 필터링과 `v-if` / `hasAuth(code)` 확인 필드를 찾습니다.

작은 사이트는 허용형 mock payload를 직접 구성해 `stubs`에 넣습니다.

큰 사이트의 사이드바/하위 모듈이 비면 I절에서 전체 권한 트리를 복원합니다.

---

## B. config.json 템플릿

```json
{
  "baseUrl": "https://target/",
  "runtimeMode": "both",
  "chromium": "/usr/bin/chromium",

  "cookies": [
    { "name": "auth", "value": "b64json:{\"id\":1,\"username\":\"admin\",\"role\":\"admin\",\"func\":{},\"permissions\":[\"*\"]}" }
  ],
  "localStorage": { "token": "faketoken", "isLogin": "1" },

  "neutralize": {
    "fields": ["response_code", "code", "errno", "ret", "status"],
    "success": 0,
    "flags": { "success": true, "message": "ok" }
  },
  "forward": true,
  "loginUrlPattern": "/login",
  "apiPattern": "/api/|/rest/|/graphql",

  "mockTier": "L1+L2",
  "recordDetail": true,
  "observe": {
    "storageReads": false,
    "cookieReads": false,
    "xhrHeaders": true
  },
  "neutralizeVueRouter": true,
  "stubs": [
    {
      "match": "permissions/all|/menu|role_permissions",
      "body": {
        "response_code": 0, "code": 0,
        "data": {
          "permissions": ["*"],
          "menus": [
            { "name": "dashboard", "path": "/dashboard", "show": true, "children": [] },
            { "name": "alert", "path": "/alert", "show": true, "children": [] }
          ]
        }
      }
    }
  ],

  "explore": {
    "clickTabs": true,
    "clickTables": true,
    "pushStateFallback": true,
    "maxMenuItems": 50
  },

  "routes": ["/dashboard", "/alert", "/asset", "/device", "/report", "/config", "/system"],
  "waitMs": 1500, "perRouteMs": 900, "headless": true,
  "waitUntil": "domcontentloaded",
  "routeTimeout": 12000,
  "proxy": "",

  "captureResponses": true, "recordWs": true, "respMax": 600
}
```

필드:
- `runtimeMode`: `depth` (Puppeteer), `coverage` (browser MCP), `both`
- `cookies[].value` 접두사: `b64json:` → base64(JSON), `json:` → 원시 JSON, 없음 → 리터럴
- `forward: true`는 실제 요청을 전달하고 코드 필드를 수정, `false`는 완전 오프라인 stub
- `mockTier`: coverage preload 계층, 예: `L1+L2`, `L1+L2+L3`
- `routes`는 `routes.txt`에서 가져옴; mock 메뉴 후 harness가 `<a href>` 경로 추가
- `captureResponses` / `recordWs`는 depth 전용
- `waitUntil`: 큰 SPA는 `networkidle2` 대기 방지를 위해 `domcontentloaded`
- `routeTimeout`: 경로별 `page.goto` 제한 시간(ms)
- `proxy`: Puppeteer `--proxy-server`; `HTTP_PROXY` / `HTTPS_PROXY`도 가능

### B1. 이중 stub 템플릿 (role_permissions + permissions/all)

```json
"stubs": [
  {
    "match": "role_permissions",
    "body": {
      "response_code": 0,
      "data": {
        "permissions": ["MONITOR", "MONITOR_ALERT", "THREAT", "ASSETS_RISK"],
        "role_type": "SUPER_ADMIN"
      }
    }
  },
  {
    "match": "permissions/all",
    "body": {
      "response_code": 0,
      "data": [
        {
          "code": "MONITOR",
          "position": 1,
          "children": [
            { "code": "MONITOR_ALERT", "position": 1, "children": [] }
          ]
        }
      ]
    }
  }
]
```

외부 필드(`response_code` / `code` / `data`)는 A2 조건과 일치하고 `permissions`는 tree의 모든 leaf code를 포함해야 합니다.

---

## C. coverage preload 설정

복사한 `preload.js` 상단의 `CONFIG`를 수정하거나 CDP 주입 전에 교체합니다:

```javascript
const CONFIG = {
  loginPathRe: /\/(login|signin)(\/|$|\?)/i,
  mockTier: 'L1+L2',
  forward: true,
  recordDetail: true,
  extractUrlsFromResponse: true,
  neutralizeVueRouter: true,
  observe: { storageReads: false, cookieReads: false, xhrHeaders: true },
  neutralize: { fields: ['response_code', 'code'], success: 0 },
  stubs: [ /* config.json stubs와 동일 */ ],
  apiPattern: /\/(api|apis|v\d+|dev|internal|graphql)\//i,
};
```

`window.__API_RECON_PRELOAD__ === true`와 안정된 pathname을 확인합니다.

기록 내보내기:

```javascript
JSON.stringify({
  apis: [...window.__API_RECON_LOG__],
  detail: window.__API_RECON_DETAIL__,
  routes: [...(window.__API_RECON_ROUTES__ || [])],
  observe: window.__API_RECON_OBSERVE__,
}, null, 2)
```

---

## D. preload/runtime Hook 범위

preload(coverage)와 runtime_harvest(depth)의 내장 브라우저 Hook:

| Hook | 탐색 가치 | 지원 |
|---|---|---|
| Hook fetch / XHR.open | URL/메서드 기록 | ✅ `recordDetail` + `__API_RECON_LOG__` |
| Hook XHR.setRequestHeader | Authorization 등 헤더 확인 | ✅ `observe.xhrHeaders` |
| localStorage/cookie 읽기 Hook | 세션 키 확인 | 선택 `observe.storageReads/cookieReads` |
| Vue 경로 수집 | frontendRoutes 보완 | ✅ `__API_RECON_ROUTES__`(불러온 경로) |
| Vue 가드 처리 / 로그인 이동 차단 | 모듈 렌더링으로 API 유발 | ✅ `neutralizeVueRouter` + 기본 이동 처리 |
| React 경로 수집 | 경로 보완 | 정적 + 클릭; 전용 Hook 없음 |
| 페이지 이동 차단 (login 경로) | 페이지에 남아 분석 | 로그인 경로만 차단하고 업무 이동 보존 |
| 암호화 라이브러리 Hook (CryptoJS/SM 등) | 암호화 매개변수 → 평문 API 본문 | 암호화 입력 수동 Hook 필요; config에 기록 |
| 디버깅 방지 처리 | 처리하지 않으면 runtime API 기록 불가 | 수동 처리 필요; 정적 분석 가능 |

---

## E. 정적 결과가 적을 때 endpoint 정규식

`harvest_static.py`의 `extract_endpoints`를 넓히거나 필수 harvest 이후 수동 확인합니다:

```bash
grep -rhoaE '"/[a-z][A-Za-z0-9_/\-]{3,}"' js | sort -u
grep -rhoaE '/api/[a-zA-Z0-9_./-]+' js | sort -u
```

---

## F. 문제 해결

| 증상 | 원인과 조치 |
|---|---|
| 정적 API 적음 | endpoint 문법 불일치: 정규식 확장 (E절) |
| chunk 수가 manifest보다 훨씬 적음 | CSS 전용 또는 미배포 chunk; 404 재시도 확인 |
| runtime이 로그인 화면에 머묾 | 렌더링 가드: A1 키, 저장소, 인코딩, domain 재확인 |
| 화면 틀 진입, 모듈 비어 있음 | 내용 조건: mock 메뉴(A3), `routes` 경로 확인 |
| 경로마다 bootstrap/locale만 존재 | 권한 코드 부족: 트리 복원(I), `role_permissions`와 `permissions/all` 이중 stub 확인 |
| 사이드바 있음, 하위 화면 비어 있음 | 중간 tree 노드 누락 또는 `userRouteAuth` 코드 불일치 |
| 모든 API가 로그인으로 이동 | 인터셉터: `neutralize` 확인; 중첩 필드는 탐색 로직 확장 |
| WS 프레임 0 | 상호작용 후 구독 필요; `perRouteMs` 증가 |
| 응답 본문 없음 | 실제 응답은 `forward: true` 필요 |
| Chromium 없음 | Chromium 설치 또는 `config.chromium` / `CHROMIUM` 지정 |
| mock 이후에도 로그인 이동 | Hook이 늦거나 `location.href` setter 없음: document-start preload |
| 목록이 모두 비어 있음 | L3 빈 배열은 정상; 탭/설정/상세 계속 확인 |
| Redux action을 경로로 오인 | get/set/change/clear/toggle/upload 포함 내부 경로 제외 |
| Vue가 계속 로그인 이동 | document-start에 preload 주입; `neutralizeVueRouter: false`면 클라이언트 가드 수동 처리 |
| 응답 URL이 로그에 없음 | `extractUrlsFromResponse` 활성화 또는 `__API_RECON_DETAIL__`에서 추출 |
| 인증 헤더 이름 모름 | `observe.xhrHeaders` 활성화 또는 DevTools 요청 헤더 확인 |
| runtime 지연/시간 초과 | `waitUntil: domcontentloaded`, `routeTimeout` 감소, `networkidle2` 피하기 |
| 프록시 연결 실패 | `proxy`/환경 변수 확인; Puppeteer/curl 프록시 포트 일치 |

---

## G. 보호가 강화된 대상

서버가 전체 흐름에서 세션을 검증하면(mock 불가능한 서명 cookie, stub 불가능한 서버 렌더링 메뉴) runtime은 화면 틀에서 멈춥니다. 예상 동작:

- 모듈 경로가 코드에 남아 정적으로 endpoint 열거 가능
- 원본 참고서는 명시적으로 승인된 실제 세션을 같은 harness(`forward: true`, neutralize 없음)로 사용해 메서드/매개변수/응답을 수집하는 경우도 설명합니다. 이는 이 스킬의 자격 증명 없는 기본 범위 밖이며 자격 증명 요청과 실제 로그인 금지를 무효화하지 않습니다.

---

## H. 작업별 확인 목록

1. 승인 범위 확인
2. `scripts/harvest_static.py` 읽기, 수정, 실행, `api_static.txt`/`routes.txt` 검토
3. Phase 1b: 경로 주변 확장 + 연결 계층 → `param_candidates.json` (J)
4. A1/A2/A3 역추적 → 대상별 `config.json`
5. 실행 전에 `runtime_harvest.js` / `preload.js` 읽고 수정
6. `runtimeMode=depth`: `npm install` → 수정 harvest 실행
7. `runtimeMode=coverage/both`: document-start에 수정 preload 주입 → browser MCP 열거 + 매개변수 동작 매트릭스
8. 모듈 렌더링 안 됨 → 권한 트리 복원(I) → stub 수정 → 재실행
9. 여러 매개변수 표본 diff + 오류 역추론 → `params_merged.json`
10. 병합 → `site_map.json` + `api_merged.txt`; 범위, 누락, 스크립트 변경을 정확히 표시

---

## I. 권한 트리 복원 (Phase 4 상세)

단순 `menus: [{ path, show: true }]` mock으로 하위 모듈이 열리지 않을 때 사용합니다.

### I1. auth 모듈 찾기

```bash
grep -l 'userRouteAuth' js/*.js
grep -l 'routeMap\|routeLink' js/*.js
grep -rhoaE 'getResultTree|role_permissions|permissions/all' js | head
```

권한 API 경로, 응답 필드 이름, 소비 chunk 파일 이름을 기록합니다.

### I2. routeMap 추출

```bash
python3 scripts/extract_route_map.py recon/js recon/
# recon/route_map.json 생성
```

`[!] no routeMap pattern found`이면 `extract_route_map.py` 정규식을 넓히거나 grep으로 확인합니다:

```bash
grep -rhoaE '([A-Z_][A-Z0-9_]*):\{name:"[^"]*",link:"/[^"]+"\}' js | head -20
```

### I3. 권한 트리와 stub 생성

```bash
python3 scripts/build_perm_tree.py recon/js recon/ --config recon/config.json
```

스크립트 로직:
1. webpack 별칭 `He=o.DASHBOARD`를 포함하여 `userRouteAuth={MONITOR:{url:...},...}` 분석
2. `route_map.json`으로 별칭을 실제 경로로 변환
3. code 접두사로 부모 추정 (`MONITOR_ALERT` → `MONITOR`)
4. `permissions_tree.json`, `permissions_all_stub.json`, `role_permissions_stub.json` 생성
5. `--config` 사용 시 `config.json`의 `stubs` 갱신 및 `routes` 확장

복사한 스크립트 상단에서 대상별 수정:
- `DEFAULT_ROOTS`: 최상위 모듈 code 목록
- `DEFAULT_PREFIX_PARENT`: `PREFIX_` → 부모 매핑
- `DEFAULT_EXTRA_PARENT`: 접두사 관계 없는 고아 노드

### I4. stub 일관성 검증

```bash
# permission 수는 userRouteAuth 항목 수와 유사해야 함
wc -l recon/perm_codes_all.txt
# routes는 route_map의 모든 link를 포함해야 함
python3 -c "import json; m=json.load(open('recon/route_map.json')); r=set(json.load(open('recon/config.json'))['routes']); print('missing', [v['link'] for v in m.values() if v['link'] not in r])"
```

### I5. runtime 재실행 및 비교

```bash
node recon/runtime_harvest.js recon/config.json
# mock 전후 runtime_api.json 개수 비교; /attack, /asset 등의 모듈 API 확인
```

| mock 전 | mock 성공 후 |
|---|---|
| 경로마다 같은 bootstrap 3~5개 | 경로별 다른 모듈 API |
| `/api/locale/language`만 있음 | `/api/web/...` 모듈 endpoint 등장 |
| `routes.txt` 경로 한 자릿수 | route_map에서 80~110개 이상의 `routes` |

### I6. 계속 실패하는 경우

- coverage: 사이드바/탭 클릭; 권한 요청에 상호작용이 필요할 수 있음
- stub 필드: 이미 승인된 API 응답과 중첩 구조 비교. 원본 예시는 실제 세션 curl이지만 이 스킬에서 자격 증명 취득/로그인 금지
- 추가 가드: `hasPermission|checkRole|func.` 등 버튼 확인 grep, `role_permissions.permissions` 확장
- 정적 대안: 모듈 경로는 `api_static.txt`에 유지; runtime은 메서드/본문 보완. `param_candidates.json`과 기록 표본 유지

---

## J. 매개변수 역추적 (Phase 1b / 5b / 5c)

범용 스크립트가 아닌 방법론입니다. 경로는 정규식, 매개변수는 주변 확장, UI 연결, 표본 diff, 오류 역추론을 사용합니다.

### J1. 주변 확장: 경로에서 요청 조립 찾기

```bash
# Phase 1의 알려진 경로를 기준점으로 사용
grep -n '"/api/user/list"' js/*.js
grep -rhoaE '.{0,120}("/api[^"]+").{0,200}' js | head
grep -rhoaE '(params|data|body|payload)\s*:\s*\{' js | head
grep -rhoaE '(get|post|put|delete|patch)\([^,]+,\s*\{' js | head
```

### J2. 래퍼와 전송 형식

```bash
# axios / 공통 request 래퍼
grep -rhoaE '(axios|request)\.(get|post|put|delete|patch)\(' js | head
grep -rhoaE 'interceptors\.(request|response)' js | head

# GraphQL
grep -rhoaE '(query|mutation)\s+\w+|gql`|graphql\(' js | head
grep -rhoaE '\$[a-zA-Z_]+\s*:\s*(Int|String|Boolean|\[)' js | head

# FormData / multipart
grep -rhoaE 'FormData|\.append\(' js | head

# 경로 매개변수
grep -rhoaE 'path:\s*"/[^"]*:[^"]+"' js | head
grep -rhoaE 'useParams|route\.params|\$route\.params' js | head
```

### J3. 검증: 필수, 형식, 열거

```bash
grep -rhoaE '(required|message|pattern|enum|validator)\s*:' js | head
grep -rhoaE 'yup\.|zod\.|async-validator|Form\.Item|a-form-item|el-form-item' js | head
grep -rhoaE 'rules\s*:\s*\[|name:\s*["\'][a-zA-Z_]+["\']' js | head
grep -rhoaE 'label.*value|options\s*:\s*\[' js | head
```

### J4. 연결 계층: 폼에서 API

```bash
grep -rhoaE 'onFinish|handleSubmit|getFieldsValue|validateFields' js | head
grep -rhoaE '(pick|omit|transform|dayjs|moment)\(' js | head
```

실행 증거: DevTools → Network → 요청 → Initiator(호출 스택)에서 `fetch`/`send`부터 요청 조립을 역추적합니다.

### J5. 암호화 매개변수

```bash
grep -rhoaE 'encrypt|decrypt|sign|CryptoJS|sm2|sm3|sm4|RSA|AES' js | head
```

암호문에서 필드를 추측하지 마세요. 암호화 함수 입력을 Hook해 암호화 전 평문 payload를 기록하고 `config.json` / `param_candidates.json`에 결론을 저장합니다.

### J6. 매개변수 동작 매트릭스 (Phase 3 필수)

모듈별 각 조작을 기록하고 요청 body/query를 비교합니다:

| 조작 | 확인 |
|---|---|
| 목록 첫 화면 | 페이지 기본값 |
| 검색 | keyword, filters |
| 고급 필터 | 선택 필드 |
| 생성/편집 | 전체 entity |
| 일괄/내보내기 | `ids[]`, `exportType` |
| 정렬/페이지 전환 | `sortField`, `order` |

`param_samples.json` 생성: `[{ "path", "method", "action": "search", "body", "query", "headers" }]`

### J7. 신뢰도 규칙

| 신뢰도 | 조건 |
|---|---|
| 높음 | 정적 호출 지점과 runtime 표본 두 개 이상 일치 |
| 중간 | 정적만 또는 runtime 표본 하나 |
| 낮음 | 응답/오류 추정, 재검증 없음 |
| 트리거 대기 | 정적 필드 확인, UI/권한 경로 미도달 |

### J8. 시나리오별 순서

| 시나리오 | 순서 |
|---|---|
| REST 목록 화면 | J1 요청 객체 → J6 네 번 diff → J3 rules |
| 생성/편집 폼 | J3 폼 이름 → J4 submit 흐름 → 승인된 비파괴적 runtime 제출과 빈 필드 400 확인 |
| GraphQL | J2 변수 선언 → runtime operation별 variables 기록 |
| 암호화 body | J5 입력 Hook → 암호화 전 필드가 실제 매개변수 |

### J9. api-recon 단계 매핑

| api-recon | 매개변수 조사 |
|---|---|
| Phase 1 정적 | J1 주변 확장 |
| Phase 2 A2 인터셉터 | 전역 삽입 필드 (tenantId, sign) |
| Phase 3 runtime | J6 동작 매트릭스 + `param_samples.json` |
| Phase 4 권한 트리 | 모듈마다 다른 폼; 충분한 mock 권한으로 전체 필드 노출 |
| Phase 5 병합 | `params_merged.json` + 신뢰도; 표본 하나로 필수 판단 금지 |

### J10. 문제 해결

| 증상 | 조치 |
|---|---|
| 정적 필드가 runtime에 없음 | 트리거 대기 표시; 권한 트리/고급 필터/연동 select 옵션 보완 |
| 같은 경로, 다른 body 구조 | 정상: `action`별 기록, 스키마 강제 병합 금지 |
| mock 응답에서 실제 매개변수 확인 필요 | 나가는 요청 body/headers 확인; stub 응답 역추정 금지 |
| 400에 중첩 필드 표시 | 외부 `data`/`bizData`/`variables` 래퍼 확인 |
| GraphQL operation 이름만 보임 | `variables` JSON 확장; 정적 `$var: Type` 확인 |

---
