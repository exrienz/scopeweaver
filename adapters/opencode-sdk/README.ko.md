# OpenCode SDK 브리지

[English](README.md) · [한국어](README.ko.md) · [简体中文](README.zh-Hans.md) · [繁體中文](README.zh-Hant.md) · [日本語](README.ja.md) · [Español](README.es.md)

**네이티브 OpenCode CLI·클라이언트·SDK 2.1.4**와 따로 검증한 **구형 SDK·CLI 1.19.4**를 연결하는 선택 기능입니다. ScopeWeaver가 대화 기록, 도구 실행, 승인, 범위 검사, 재시도와 압축을 계속 관리합니다. 브리지는 모델의 한 단계만 스트리밍하고 도구 요청을 수집합니다. ScopeWeaver 도구나 OpenCode 기본 도구를 직접 실행하지 않습니다. 기존 API·OpenCode Go 공급자는 브리지 없이 사용할 수 있습니다.

## 실행

Bun **1.3.14**가 검증된 CI 런타임입니다. 이후 버전은 아직 검증하지 않았습니다.

```sh
cd adapters/opencode-sdk
bun install --frozen-lockfile --ignore-scripts
export SCOPEWEAVER_OPENCODE_TOKEN="$(openssl rand -hex 32)"
bun run start
```

브리지 토큰은 **공백이 아닌 문자 32자 이상**이어야 합니다. native의 비공개 CLI 서버는 호출마다 별도로 생성한 비밀번호(`OPENCODE_PASSWORD`)와 HTTP Basic 인증을 사용합니다. 이 비밀번호는 브리지 토큰 및 하위 공급자 키와 서로 다릅니다.

ScopeWeaver 공급자 프로필에서 `opencode-sdk`, 주소 `http://127.0.0.1:4097`, 위 **브리지 토큰**을 입력합니다. 클라우드 API 키가 아닙니다. 모델을 조회한 뒤 정확한 `providerID/modelID`를 선택하세요. Go·Console API 키를 브리지 토큰으로 재사용하지 마세요.

기본 런타임은 로그인하지 않은 상태에서 격리된 카탈로그와 공개 모델 목록을 사용합니다. 조회는 공개 카탈로그 GET 요청만 수행하며 **모델 추론을 실행하지 않습니다**. 현재 목록에 있고 고정 SDK가 인식하는 무료 OpenCode 모델만 표시합니다. 조회 실패는 오류로 반환합니다. 목록에 있다고 사용 자격·할당량·추론 성공이 보장되지는 않습니다. 무료 모델은 변경되거나 사라질 수 있으며 기본 설정에 유료 모델·인증정보 대체 경로는 없습니다.

**2026년 10월 25일(KST)의 이전 검증:** `opencode/deepseek-v4.1-flash-free`로 실제 ScopeWeaver 대화를 완료했습니다. 제품의 native CLI·클라이언트 2.1.4 브리지를 통해 실제 추론을 세 번 호출해 UI 연결 테스트, 파일에 영향을 주기 전에 검토·승인한 호스트 도구 요청, 최종 화면의 답변 `FREE_OK`를 확인했습니다. 기본 무료 카탈로그에서 모델을 선택했으며 API 키, 계정 가져오기, 모델·응답 모의 데이터, 식별 위장이나 유료 대체 경로를 사용하지 않았습니다. 기본 도구는 계속 차단했고 호스트 동작은 ScopeWeaver가 실행했습니다.

native 추론은 고정한 실제 CLI와 공식 클라이언트 API를 사용하고 추론 없는 모델 조회는 내장 SDK를 사용합니다. 앞선 내장 SDK의 수집 전용 경로는 HTTP 403 `FreeTierError`로 거부되었습니다. legacy의 실제 무료 접근은 아직 검증하지 않았습니다. 무료 모델과 서비스 사용 자격은 바뀔 수 있으며 이번 날짜의 검증으로 수치화된 할당량이나 이후의 사용 가능성을 보장하지 않습니다. 구조화된 거부는 `free_tier_restricted`로 전달하며 유료 모델로 자동 전환하지 않습니다.

포트는 `SCOPEWEAVER_OPENCODE_PORT`로 변경할 수 있습니다. 기본값은 4097입니다. `SCOPEWEAVER_OPENCODE_HOST`는 `127.0.0.1`만 허용합니다. 브라우저 `Origin` 요청과 루프백이 아닌 주소를 거부하며 모든 경로에 토큰 인증이 필요합니다.

호출 시간 제한은 `SCOPEWEAVER_OPENCODE_TIMEOUT_MS`로 **0~86,400,000ms** 사이에서 설정합니다. 기본값 **0**은 호스트가 취소할 때까지 호출을 허용하며 숨겨진 2분 추론 제한이 없습니다. 취소·종료 시에는 별도의 **1.5초** 강제 종료 제한이 적용됩니다. 공개 카탈로그 네트워크 조회는 10초 제한입니다. 요청은 **16 MiB**, 반환되는 도구 인자 JSON 하나는 **2,000,000자**, 네이티브 모델의 텍스트·사고·인자 합계는 **8,000,000자**, 워커 출력은 **20,000,000바이트**로 제한합니다. 넘으면 오류를 반환하며 데이터를 잘라서 전달하지 않습니다.

## 전용 공급자 설정

명시적인 OpenAI 호환 공급자나 로컬 테스트 모델을 연결하려면 `SCOPEWEAVER_OPENCODE_PROVIDER_CONFIG`에 운영자가 관리하는 JSON 파일 경로를 지정합니다. 파일은 체크아웃 밖에 두거나 Git에서 제외되는 `provider-config.local.json` 이름을 사용하세요. JSON 예시는 [영문 안내](README.md#dedicated-provider-configuration)에 있습니다. 예시 모델 ID는 `local/my-model`입니다.

```json
{
  "providers": {
    "local": {
      "name": "Local model",
      "package": "@opencode/ai/providers/openai-compatible",
      "settings": {
        "baseURL": "http://127.0.0.1:8000/v1",
        "apiKey": "explicit-provider-key-if-needed"
      },
      "models": {
        "my-model": {
          "name": "My model",
          "capabilities": { "tools": true },
          "limit": { "context": 200000, "output": 32000 },
          "compatibility": { "maxTokensField": "max_tokens" }
        }
      }
    }
  }
}
```

이 파일을 지정하면 기본 모델 조회 대신 파일에 명시한 모델만 사용합니다. 사용자·프로젝트 OpenCode 설정, `.env`, 외부 플러그인, MCP 서버, 저장된 인증정보를 불러오지 않습니다. 해당 공급자에는 파일에 명시한 `apiKey`만 전달합니다. 원격 공급자는 추론 비용을 청구할 수 있으며 설정을 추가한다고 무료가 되지는 않습니다. 임의 SDK 패키지와 지원하지 않는 설정 필드를 거부합니다.

## 실행 경계

각 호출은 환경 변수를 허용 목록으로 제한한 별도 Bun 워커와 임시 홈·설정·데이터 경로를 사용합니다. native는 실제 CLI의 비공개 서버, 실제 CLI `run`, 임시 DB, 한 번만 사용할 수 있는 인증된 루프백 추론 전송 서버를 소유합니다. 공식 클라이언트가 모델·세션 조회, 이벤트와 중단을 처리합니다. 클라우드 인증정보·사용자 인증 파일·프록시·사전 로드 설정을 상속하지 않습니다. 정확히 일치하는 공식 선택 의존성의 바이너리를 직접 사용하므로 `--ignore-scripts`가 유효하며 전역 설치가 필요 없습니다. 선택 의존성을 생략하지 마세요.

native는 OpenCode의 실제 기본 도구 정의와 원래 구현을 유지합니다. 기본 도구 권한은 모두 ASK이고 CLI `--auto`는 사용하지 않습니다. 별도의 공개 `execute.before` 정책 훅이 생성된 `execute`를 포함한 모든 비호스트 도구를 실행 전에 거부합니다. 기본 도구를 요청하면 브리지 요청을 실패시키며 ScopeWeaver로 전달하지 않습니다. 등록한 호스트 수집 함수만 대기했다가 호스트 요청이 되고 다른 에이전트는 제거합니다. `sw_tool_N` 별칭으로 호스트 도구가 기본 도구를 덮어쓰지 않게 하며 설명과 반환값에 정확한 원래 이름을 보존합니다.

호스트 system 구간은 두 줄바꿈으로 연결해 공개 사용자 지정 에이전트 system 설정에 그대로 넣습니다. OpenCode가 생성한 모델·환경·도구 문맥도 추가로 유지합니다. 마지막 정책 구간은 설명이 `Host tool:`로 시작하는 함수만 사용할 수 있음을 알립니다. 호스트 대화 역할, 도구 ID·결과, 호출별 제한·설정은 공개 context 훅으로 보존합니다. system 배열에 호스트 구간만 존재한다는 보장은 하지 않습니다. thinking·effort 필드는 공급자에 전달하며 지원하지 않는 설정은 명시적으로 실패합니다. 모델의 출력 한도가 작으면 그 한도까지 제한하고 입력·출력·캐시 사용량은 분리합니다.

워커는 두 번째 모델 요청을 차단합니다. 비공개 전송 서버는 실제 CLI가 만든 요청을 원래 공급자 URL로 보내되 리디렉션을 거부하며 모델·주소·식별·인증정보를 대체하지 않습니다. 호스트 도구 요청은 중단, 세션 삭제, 소유 프로세스 정리가 끝난 뒤에만 전달합니다. 연결 종료·DELETE·시간 초과·서버 종료는 POSIX 워커 프로세스 그룹을 종료하며 1.5초 뒤 강제 종료합니다. 이 프로세스 트리 정리 경계가 구현되지 않아 **Windows에서는 native와 legacy 추론을 모두 거부**합니다.

CLI의 공개 설정은 시스템·MDM 설정 탐색을 완전히 끌 수 없습니다. 따라서 macOS의 `/Library/Application Support/opencode`, 사용자·전역 `/Library/Managed Preferences` 아래 `ai.opencode.managed.plist`, Linux의 `/etc/opencode`가 있으면 native를 시작하지 않고 `configuration_error`로 실패합니다. 접근할 수 없는 경로와 끊어진 심볼릭 링크도 거부합니다. 해당 설정을 읽거나 변경하지 않습니다.

native 모드는 완료·스트림 응답과 호스트 도구 기록 재생에서 64비트 정수를 포함한 도구 인자의 원본 JSON을 보존합니다. JavaScript로 파싱한 도구 인자를 실행하지 않습니다. 원본 도구 인자 밖에서 안전하게 표현할 수 없는 숫자, 지원하지 않는 콘텐츠, 알 수 없는 도구, 중복 ID, 잘못된 응답은 거부합니다. 스키마의 인라인 정의, 로컬 JSON Pointer `$ref`·`$defs`·유니온을 지원하며 원격 참조, `$id`, `$dynamicRef`, 해결되지 않는 참조는 거부합니다. 네이티브 초기화 시 자체 공급자·카탈로그 코드는 로드되며 **운영체제 샌드박스는 아닙니다**. 루프백 브리지도 원격 모델을 사용할 수 있어 ScopeWeaver의 로컬 전용 추론 조건을 충족하지 않습니다.

## 런타임 버전과 구형 경로의 제한

| `SCOPEWEAVER_OPENCODE_RUNTIME` | 고정한 버전 조합 | 기능 응답 `runtime` |
| --- | --- | --- |
| `native`(기본값) | 공식 CLI·`@opencode/client`·SDK **2.1.4** | `native` |
| `legacy` | `@opencode-ai/sdk/v2`·CLI **1.19.4** | `legacy` |

구형 API를 선택하려면 시작 전에 `SCOPEWEAVER_OPENCODE_RUNTIME=legacy`를 지정합니다. 두 조합은 로컬 **macOS arm64**에서 확인했으며 CI는 Linux에서 실제 런타임 테스트를 실행하도록 구성되어 있습니다. 다른 SDK 버전은 추론 전에 거부합니다. 모든 버전·플랫폼의 호환성을 보장하지 않습니다.

구형 경로는 정확히 일치하는 공식 선택 의존성의 플랫폼 바이너리를 직접 실행하므로 `--ignore-scripts` 설치를 사용할 수 있습니다. 초기화되지 않은 래퍼나 전역 `opencode`는 실행하지 않습니다. 선택 의존성을 생략하지 마세요. CLI와 루프백 MCP 수집 서버는 요청 워커의 POSIX 프로세스 그룹과 함께 종료합니다. 이 정리 경계가 구현되지 않은 **Windows에서는 legacy를 거부**합니다. 호출마다 독립된 플러그인 패키지 복사본과 잠금 파일을 사용하며 공유 의존성에 쓰기 가능한 링크를 만들지 않습니다.

legacy 정수의 절대값은 **9,007,199,254,740,991** 이하여야 합니다.

구형 경로는 구조가 유효하고 안전하게 표현할 수 있는 JSON·기록을 지원합니다. 공백이나 같은 값을 나타내는 HTML·Unicode 이스케이프는 정규화될 수 있어 원본 JSON 표기 자체는 보장하지 않습니다. 안전하게 표현할 수 없는 정수, 불투명하거나 서명된 사고 기록, 지원하지 않는 공급자 덮어쓰기는 명시적으로 실패합니다. 원본 64비트 인자는 native 모드를 사용하세요. 구형 사용자 지정 공급자는 위의 OpenAI 호환 설정을 사용하되 `max_tokens` 호환 필드, 지원되는 도구 기능 필드, 컨텍스트·출력 한도만 매핑합니다. 지원하지 않는 `reasoningField`·기능 덮어쓰기를 거부합니다. SDK 경계 밖의 입력에는 기존 API·Go 프로필을 계속 사용할 수 있습니다.

## 프로토콜과 검증

프로토콜 **`scopeweaver-opencode-sdk`, 버전 1**의 인증 경로는 `GET /v1/capabilities`, `GET /v1/models`, `POST /v1/complete`, `POST /v1/stream`, `DELETE /v1/requests/{request_id}`입니다. 스트림은 NDJSON이며 실제 네이티브 텍스트·사고 델타, 수집한 도구 요청, 사용량을 전달하고 성공 시 `message_stop`을 정확히 한 번 보냅니다. 오류는 공급자 원문을 노출하지 않는 제한된 코드·메시지를 사용합니다. 명시적인 구조화 응답으로 확인된 잔액·할당량 소진은 `quota_exhausted`(완료 경로 HTTP 402)로 구분하며 일반 속도 제한·인증·5xx 오류를 할당량 오류로 바꾸지 않습니다. 구조화된 모델 컨텍스트 초과는 `context_length_exceeded`(HTTP 400)로 전달해 호스트의 압축 신호를 보존하며 일반 400 응답이나 메시지 키워드만으로 분류하지 않습니다. 구조화된 `FreeTierError`는 `free_tier_restricted`(HTTP 403)로 구분하며 할당량·일반 금지 응답과 섞지 않습니다. 브리지는 대체 모델을 선택하지 않으며 호스트의 기존 정책이 다음 동작을 결정합니다.

```json
{
  "request_id": "unique-call-uuid",
  "session_id": "stable-host-conversation-uuid",
  "model": "local/my-model",
  "system": ["Host system instructions"],
  "messages": [{ "role": "user", "content": [{ "type": "text", "text": "Hello" }] }],
  "tools": [],
  "max_tokens": 64,
  "thinking": "disabled"
}
```

요청에는 `request_id`, `model`, `system` 배열, `messages` 배열, `tools` 배열, `max_tokens`가 필요합니다. 선택 필드는 `session_id`, `temperature`, 문자열 배열 `stop`, `thinking`, `thinking_type`, `reasoning_effort`입니다. 메시지는 `text`, 어시스턴트의 `thinking`·`tool_use`, 텍스트 결과를 담은 사용자의 `tool_result` 블록을 사용합니다. 도구 선언은 `name`, `description`, `input_schema`를 사용합니다. 완료 응답은 `message`, `stop_reason`(`end_turn`, `tool_use`, `max_tokens`), 입력·출력·캐시 읽기·캐시 쓰기 수치를 분리한 `usage`를 포함합니다. 공개 JSON 예시는 [영문 프로토콜](README.md#protocol-and-checks), 검증·제한은 [protocol.mjs](src/protocol.mjs), 전송 처리는 [bridge.mjs](src/bridge.mjs)에서 확인할 수 있습니다.

`opencode-go` 공급자에서는 `session_id`로 호출별 SDK 세션이 달라도 `x-opencode-session`을 유지합니다. 없으면 `request_id`를 사용합니다. `opencode`는 실제 런타임이 생성한 세션·식별 헤더를 그대로 유지합니다. 일반 공급자에는 게이트웨이 세션 헤더를 추가하지 않으며 실제 CLI의 User-Agent를 유지합니다.

```sh
bun run check
bun test --timeout 30000
```

테스트는 고정한 실제 CLI·클라이언트·SDK와 로컬 SSE 모델을 사용해 추론 없는 조회, 기록·설정, 도구 수집 안전성, 스트리밍·사용량, 원본 숫자 인자, 악성 기본 도구 요청, 동시 호출 격리와 취소를 확인합니다. 실제 무료 모델의 사용 자격을 증명하지는 않습니다.

## ScopeWeaver 프로필과 다운그레이드 경계

빈 토큰은 같은 SDK 주소에서만 저장된 값을 유지합니다. 주소나 형식을 바꾸면 저장된 토큰을 재사용하지 않으며 `localhost`, `127.0.0.1`, `::1`은 서로 다른 자격 증명 범위입니다. SDK 템플릿은 저장하지 않은 클라우드 키, 프록시, 사용자 지정 세션 헤더를 지우고 모델을 빈칸으로 둡니다. 조회·추론·저장·활성화는 수행하지 않습니다. **모델 불러오기**는 명시적으로 실행하고, **연결 테스트**는 클릭할 때만 추론합니다.

프로필 데이터베이스의 `CHECK` 제약에 `opencode-sdk`가 추가됩니다. SDK 프로필이 남아 있으면 이전 바이너리는 시작할 수 없습니다. 다운그레이드 전에 데이터베이스를 백업하고 현재 버전에서 이전에 지원하던 형식으로 변환하거나 SDK 프로필을 제거하세요. 이전 마이그레이션 검증에는 임시 데이터베이스를 사용했으며 운영 데이터베이스를 마이그레이션하지 않았습니다. [제공자 안내](../../docs/ko/llm-providers.md#opencode-sdk-bridge)를 참고하세요.
