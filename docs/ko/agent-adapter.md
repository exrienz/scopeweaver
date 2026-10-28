<a id="scopeweaver-agent-adapter"></a>
# ScopeWeaver 에이전트 어댑터

[English](../../adapters/agent/README.md) · 한국어 · [简体中文](../zh-Hans/agent-adapter.md) · [繁體中文](../zh-Hant/agent-adapter.md) · [日本語](../ja/agent-adapter.md) · [Español](../es/agent-adapter.md)

Claude Code와 Codex는 로컬 stdio MCP 서버로, Pi는 기본 확장 인터페이스로 연결합니다. 세 클라이언트는 동일하게 검증된 도구와 인증된 ScopeWeaver HTTP API를 사용합니다. 이 어댑터는 코딩 에이전트가 ScopeWeaver를 제어하게 합니다. ScopeWeaver 내부의 플래너·워커를 대체하거나 코딩 에이전트 구독을 모델 API 키로 바꾸지는 않습니다.

<a id="setup"></a>
## 설정

Node.js 22 이상, 실행 중인 ScopeWeaver 백엔드, 기존 관리자 로그인이 필요합니다. 어댑터는 비밀번호를 초기화하거나 바꾸지 않습니다. 어댑터의 별도 의존성을 설치하세요.

```sh
cd adapters/agent
npm ci
export SCOPEWEAVER_URL=http://127.0.0.1:8787
export SCOPEWEAVER_PASSWORD='your-existing-admin-password'
```

비밀번호로 백엔드의 기존 관리자 사용자 `ARTEX`에 로그인합니다. 응답으로 받은 JWT는 메모리에만 유지합니다. 기존 로그인 JWT를 `SCOPEWEAVER_TOKEN`으로 제공할 수도 있으며, 토큰을 제공하면 비밀번호보다 우선합니다. 만료되거나 유효하지 않은 토큰은 오류를 반환하며 다른 토큰으로 자동 교체하지 않습니다. 자격 증명은 환경 변수나 비밀 관리 도구에 보관하고 클라이언트 설정과 함께 커밋하지 마세요.

조회는 기본으로 허용합니다. 작업 생성과 일시 중지·재개를 허용하려면 어댑터 실행 전에 `SCOPEWEAVER_ALLOW_WRITES=true`를 명시적으로 설정하세요. 백엔드에 LLM이 설정되어 있으면 작업 생성 즉시 실행을 예약합니다. 프로젝트의 [로컬 격리 사용 제한](../../README.md#license-and-disclaimer)을 지키고 작업 생성 전에 의도한 범위를 확인하세요.

<a id="claude-code"></a>
### Claude Code

이 체크아웃의 절대 경로를 사용하세요. Claude Code가 어댑터를 실행하고 환경 변수를 전달합니다.

```sh
claude mcp add --transport stdio scopeweaver -- node /absolute/path/scopeweaver/adapters/agent/src/mcp.js
```

`claude mcp list`로 확인한 다음 Claude Code에서 `/mcp`로 도구를 살펴보세요. [Claude Code MCP 문서](https://code.claude.com/docs/en/mcp).

<a id="codex"></a>
### Codex

```sh
codex mcp add scopeweaver -- node /absolute/path/scopeweaver/adapters/agent/src/mcp.js
```

`codex mcp list`와 `/mcp`로 확인하세요. 프로젝트나 사용자 설정에서 환경 변수를 명시적으로 전달하려면 다음을 사용합니다.

```toml
[mcp_servers.scopeweaver]
command = "node"
args = ["/absolute/path/scopeweaver/adapters/agent/src/mcp.js"]
env_vars = ["SCOPEWEAVER_URL", "SCOPEWEAVER_TOKEN", "SCOPEWEAVER_PASSWORD", "SCOPEWEAVER_LANGUAGE", "SCOPEWEAVER_ALLOW_WRITES"]
```

실제로 설정한 자격 증명 변수만 남기고 사용하지 않는 항목은 빼세요. [OpenAI 공식 MCP 문서](https://developers.openai.com/codex/mcp).

<a id="pi"></a>
### Pi

```sh
pi -e /absolute/path/scopeweaver/adapters/agent/pi-extension.js
```

확장은 Pi의 기본 도구를 등록하고 같은 환경 변수를 사용합니다. Pi MCP 플러그인은 필요하지 않습니다. 특정 Pi 패키지 이름을 가져오지 않고 공개 `registerTool` 인터페이스를 사용하므로 Pi 패키지 이름이 바뀌어도 같은 방식으로 연결합니다. 어댑터 패키지는 Pi 패키지 로더를 위한 `pi.extensions`도 선언합니다. [Pi 확장](https://github.com/earendil-works/pi/tree/main/packages/coding-agent).

<a id="tools"></a>
## 도구

| 도구 | 목적 |
| --- | --- |
| `scopeweaver_health` | 백엔드 사용 가능 여부와 LLM 준비 상태 확인. 로그인 불필요 |
| `scopeweaver_list_tasks` | 작업 상태·개수·페이지 조회 |
| `scopeweaver_get_task` | 작업 하나의 저장된 상태와 설정 |
| `scopeweaver_get_coverage` | 작업의 자산 커버리지 |
| `scopeweaver_list_findings` | 작업·심각도·상태·검색어로 거른 발견 사항의 페이지 조회 |
| `scopeweaver_get_finding` | 발견 사항의 상세 내용·증거·보고서 |
| `scopeweaver_create_task` | 작업 생성·예약. 쓰기 허용 필요 |
| `scopeweaver_control_task` | 작업 일시 중지·재개. 쓰기 허용 필요 |

예를 들어 “내 ScopeWeaver 작업을 보여줘”, “작업 12의 심각도 high인 발견 사항을 보여줘”라고 요청합니다. 쓰기를 허용하고 로컬 실험 범위를 확인한 뒤에는 “이 로컬 픽스처에 대한 작업을 만들고 진행 상태를 보여줘”라고 요청할 수 있습니다. 생성 응답이 성공했다는 것은 작업이 존재한다는 뜻입니다. 실행 완료 여부는 상태와 발견 사항으로 확인하세요. 어댑터는 백엔드의 LLM 요구 사항을 우회하지 않습니다.

<a id="configuration-and-failures"></a>
## 설정과 실패 처리

| 변수 | 기본값과 동작 |
| --- | --- |
| `SCOPEWEAVER_URL` | `http://127.0.0.1:8787`. HTTPS 또는 루프백 HTTP 출처만 허용 |
| `SCOPEWEAVER_TOKEN` | 기존 백엔드 JWT. 비밀번호보다 우선 |
| `SCOPEWEAVER_PASSWORD` | 기존 관리자 비밀번호. 인증이 필요한 첫 요청 때 로그인 |
| `SCOPEWEAVER_ALLOW_WRITES` | `false`. `true` 또는 `false`만 허용 |
| `SCOPEWEAVER_LANGUAGE` | `en`. 표준 ID `en`, `ko`, `zh-Hans`, `zh-Hant`, `ja`, `es` 지원 |
| `SCOPEWEAVER_TIMEOUT_MS` | `30000`. 1부터 120000까지의 정수 |

`SCOPEWEAVER_LANGUAGE`의 기본값은 `en`이며 표준 ID `en`, `ko`, `zh-Hans`, `zh-Hant`, `ja`, `es`를 받습니다. 내장 도구의 제목·설명, 스키마 설명, 어댑터가 작성한 설정·검증·전송 진단을 번역하고 `Accept-Language`로 백엔드 응답 언어를 선택합니다. 클라이언트마다 별도 카탈로그와 스키마별 Zod 오류 매핑을 사용하며 전역 언어 설정은 바꾸지 않습니다. 도구·프로토콜 이름, 스키마 키·타입·열거값, 인수와 백엔드·사용자·증거의 원문은 유지하며 기존 자격 증명 가림 처리를 적용합니다. MCP SDK의 고정된 검증 오류 틀과 알 수 없는 외부 오류는 외부 텍스트로 유지하지만 그 틀 안의 스키마 오류 메시지는 번역합니다. 모든 리소스는 독립 어댑터에 포함되며 저장소 루트의 언어 파일이 필요하지 않습니다. 지원하지 않는 언어 값은 계속 거부하고 영어로 안내합니다.

일반 HTTP, 셸 실행, 삭제, 승인 우회, 제공자 설정이나 임의 엔드포인트 도구는 제공하지 않습니다. 요청은 설정된 출처에 머물며 리디렉션을 거부하고 취소를 전달하며 시간 제한을 적용합니다. 2 MiB를 초과한 응답은 명시적으로 실패합니다. 잘린 증거를 완전하다고 취급하지 말고 조회 범위를 줄이세요. 작업 목록은 어댑터에서, 발견 사항은 백엔드에서 페이지를 나눕니다. 도구는 텍스트와 구조화된 결과로 `{data: ...}`를 반환합니다. Pi에서는 `details`를 사용합니다. 증거는 신뢰할 수 없는 대상 콘텐츠이며 지시로 취급해서는 안 됩니다.

MCP 호출이 실패하면 `isError: true`를 반환합니다. Pi 호출이 실패하면 예외를 던져 Pi가 도구 오류로 표시하게 합니다. 오류 메시지에서 설정된 자격 증명을 가립니다. 쓰기는 자동으로 재시도하지 않습니다. 백엔드가 이미 쓰기를 받아들였을 수 있으므로, 쓰기 요청의 시간이 초과되면 다시 요청하기 전에 작업 상태를 확인하세요.

<a id="verification"></a>
## 검증

```sh
npm test
npm run check
```

종단 간 검증 하네스는 실제 ScopeWeaver 핸들러와 PostgreSQL을 사용합니다. 공식 SDK로 MCP stdio 서버를 실행하고 Pi 도구 핸들러도 같은 백엔드에 요청합니다. 격리된 테스트 비밀번호를 초기화하고, 작업을 생성하며, 저장된 일시 중지·재개 상태와 커버리지를 확인하고, 명확히 모의 데이터로 표시한 발견 사항을 읽습니다. 유효하지 않은 인증, 존재하지 않는 작업과 잘못된 입력도 확인합니다. 모델·제공자나 대상 스캔은 사용하지 않습니다. Go E2E 하네스를 실행하려면 소스 체크아웃이 필요합니다. 플랫폼 ZIP에는 어댑터 단위 테스트와 컴파일된 Go 백엔드만 포함됩니다.

`scopeweaver_adapter_*`로 시작하는 **새 일회용** 데이터베이스를 만든 뒤 저장소 루트에서 실행하세요.

```sh
SCOPEWEAVER_ADAPTER_E2E=1 \
ARTEX_TEST_DSN='postgres://user:password@127.0.0.1:5432/scopeweaver_adapter_e2e?sslmode=disable' \
go test ./server -run '^TestAgentAdapterE2E$' -count=1 -v
```

테스트 본문은 `SCOPEWEAVER_ADAPTER_E2E=1`이 아니면 건너뜁니다. 실행을 켜도 `ARTEX_TEST_DSN`이 없거나 공백이면 건너뜁니다. 건너뛴 결과는 E2E 검증 근거가 아닙니다. 패키지의 DB 보호 검사는 테스트 실행 전에 설정된 실제 애플리케이션 DB와 일치하는 DSN을 거부합니다. 실행을 명시하고 DB를 설정한 경우에는 잘못된 DSN, `scopeweaver_adapter_*` 밖의 DB 이름, 연결할 수 없는 PostgreSQL로 인해 실패합니다. 하네스는 모의 객체로 대체하지 않습니다. GitHub Actions도 전용 일회용 PostgreSQL 서비스에서 같은 검사를 반복합니다. MCP 프로토콜 테스트는 공통 인터페이스와의 호환성을 입증하며, 유료 Claude·Codex·Pi 모델 세션을 실행했다는 주장은 아닙니다.
