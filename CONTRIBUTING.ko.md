# ScopeWeaver 기여 안내

[English](CONTRIBUTING.md) · 한국어 · [简体中文](CONTRIBUTING.zh-Hans.md) · [繁體中文](CONTRIBUTING.zh-Hant.md) · [日本語](CONTRIBUTING.ja.md) · [Español](CONTRIBUTING.es.md)

버그 수정, ARTEX 원본 백포트, 의존성 업그레이드, 문서, 테스트 등 모든 기여를 환영합니다.

## 시작하기 전에

- [사용 제한](README.ko.md#라이선스와-면책-조항)을 읽어 주세요. 실제 온라인 시스템을 대상으로 하는
  기능을 추가하거나 사용 제한을 약화하는 기여는 받지 않습니다.
- 큰 변경은 먼저 이슈를 열어 방향을 합의해 주세요.
- PR 하나에는 한 가지 주제만 담아 주세요. 작은 PR일수록 빨리 리뷰됩니다.

## PR 요구 사항

- **제공 언어와 대응 문서를 모두 완성합니다.** 사용자 문구와 유지 관리하는 사용자·기여자 안내는 영어, 한국어, 간체 중국어(`zh-Hans`), 번체 중국어(`zh-Hant`), 일본어(`ja`), 스페인어(`es`)가 모두 필요합니다.
  - 서버 메시지: `locale.Register(key, en, ko)` 호환성을 유지하고 `locale.RegisterAll`과 `locale/`의 언어별 리소스로 추가 제공 언어를 모두 등록합니다.
  - 웹 UI: EN/KO 테이블과 `web/src/i18n/messages/{zh-hans,zh-hant,ja,es}.ts`에 현재 영어 키를 모두 반영합니다.
  - 문서: `README.md`, `CONTRIBUTING.md`와 유지 관리하는 안내에 `ko`, `zh-Hans`, `zh-Hant`, `ja`, `es` 대응판을 제공합니다. 실행 가능한 영어 스킬 진입점은 유지합니다.
- **바뀐 동작에는 테스트를 추가합니다.** 변경이 없으면 실패하는 테스트를 추가하거나 고쳐 주세요.
  기존 검사를 끄거나, 건너뛰거나, 느슨하게 만들어 통과시키지 마세요.
- **일회용 데이터베이스만 사용합니다.** PostgreSQL에 쓰는 테스트는 `scopeweaver_test_*`
  (에이전트 어댑터 E2E는 `scopeweaver_adapter_*`) 이름의 새 데이터베이스를 사용해야 합니다.
- **원본 작업을 표기합니다.** ARTEX 백포트에는 `Upstream-commit: mhtsec/ARTEX@<sha>`와 원저자의
  `Co-authored-by:` 트레일러를 추가하고, `docs/PROVENANCE.md`와 `docs/ko/PROVENANCE.md`를
  갱신해 주세요.
- **라이선스.** 기여는 저장소의 [AGPL-3.0](LICENSE) 라이선스로 받습니다.

## 언어 추가와 변경

[언어 안내](web/src/i18n/README.ko.md)에 따라 공통 `locale/definitions.json`의 표준 ID,
원어 이름, Intl 매핑과 별칭, 일치 사례, 완전한 프런트엔드/백엔드 카탈로그, date-fns 매핑,
회귀 테스트와 브라우저 스모크를 갱신합니다. `zh`, `zh-CN`, `zh-SG`는 `zh-Hans`에 매핑됩니다.
`zh-TW`, `zh-HK`, `zh-MO`는 `zh-Hant`에 매핑되며 명시된 문자는 충돌하는 지역보다 우선합니다.
영어가 키, 의미, 보간 필드와 rich 태그의 기준입니다. 백엔드의 런타임 영어/키 대체는 호환성을
위한 보호 장치이며 출시 준비의 증거가 아닙니다. 사용자 콘텐츠, 초안, 원시 증거, 식별자와
과거 데이터를 번역하지 않습니다. 번역을 검토하고 우려를 기록하며 증거 없이 원어민 승인을 독립 어댑터의 표준 ID 목록과 루트 셸·배치 메시지 헬퍼도 일치시킵니다. `node e2e/script-locales.mjs`와 실제 Windows의 `node e2e/windows-launcher.mjs`/Windows CI를 실행하며, 배포 어댑터는 실행 시 저장소 루트 JSON에 의존하지 않아야 합니다.
주장하지 않습니다. 영어/한국어 쌍을 포함하여 여섯 언어의 유지 관리하는 기능 안내를 모두 완성합니다.

## 실행할 검사

[.github/workflows/ci.yml](.github/workflows/ci.yml)의 CI 작업과 같습니다.

```bash
# 수정한 Norma SDK (별도 Go 모듈)
(cd third_party/norma && go test ./... -count=1)

# 백엔드: 모든 Go 패키지 (ARTEX_PG_DSN을 일회용 scopeweaver_test_* 데이터베이스로 지정)
go test ./... -count=1

# 프런트엔드
cd web && npm ci && npm test && npm run build:static

# 에이전트 어댑터 (scopeweaver_adapter_* 데이터베이스를 사용하는 실제 백엔드 E2E 포함)
cd adapters/agent && npm ci && npm test && npm run check
SCOPEWEAVER_ADAPTER_E2E=1 go test ./server -run '^TestAgentAdapterE2E$' -count=1
```

`e2e` 워크플로는 모든 PR과 머지 후에 실제 바이너리를 대상으로 핵심 브라우저 스모크 테스트도 실행합니다.
실행 중인 서버와 새 `scopeweaver_test_*` 데이터베이스로 로컬에서 실행하려면
`psql`과 같은 체크아웃에서 빌드한 임베디드 바이너리가 필요합니다.

```bash
cd e2e && npm ci && npx playwright install chromium
SCOPEWEAVER_URL=http://127.0.0.1:8787 ARTEX_PG_DSN=postgres://.../scopeweaver_test_e2e \
  SCOPEWEAVER_BINARY=/absolute/path/to/scopeweaver npm run smoke
```

기존 스모크 검사에서 비밀번호를 초기화한 뒤, 대시보드 회귀 검사는 같은 시험 DB를 사용하는
격리된 바이너리 복사본을 빈 루프백 포트에서 실행합니다. 영어·한국어의 기존·신규 토큰 누적과
프로필 필터를 검사하고, 시험 서버를 종료한 뒤 자신이 만든 데이터를 삭제합니다. 대시보드 증적과
시험 서버 로그는 `e2e/e2e-results/dashboard-token-chart/`에 저장됩니다.

핵심 스모크는 여섯 언어의 설정 복원, 초기 설정/로그인/작업/설정/채팅 초안, 첫 API 언어,
SSE 재연결 시 필터 보존과 화면의 숫자 형식도 검사합니다.

PR 설명에 실행한 검사와 실행하지 못한 검사를 적어 주세요.

## 리뷰와 머지

최신 `dev`에서 `feat/*` 또는 `fix/*` 브랜치를 만들고, PR의 대상은 `dev`로 지정합니다.
통합한 `dev`의 해당 커밋에서 모든 검사와 E2E가 통과하면 이 저장소의 `dev`에서 `main`으로
승격 PR을 엽니다. 승격에는 머지 커밋을 사용해 `dev`가 `main`의 조상으로 남도록 합니다.
승격 후 검사도 확인하고, 다음 작업은 다시 `dev`에서 시작합니다.
`branch-policy` 검사는 이 경로를 따르지 않는 PR을 거부합니다.

1. 현재 후보 커밋의 `branch-policy`, `frontend`, `agent-adapter`, `backend`, `windows`,
   `chat-browser`, `smoke` 필수 검사가 모두 통과해야 합니다. `dev`와 `main` 모두 머지 후
   CI와 E2E를 실행합니다. 처음 기여하는 경우 관리자가 CI 실행을 승인합니다.
2. PR은 diff를 기준으로 정확성, 보안, 제공 언어의 번역 완전성, 여섯 언어 문서 일치 여부를 리뷰합니다.
3. **모든 PR은 머지 전에 관리자가 직접 최종 확인합니다.** 이 확인 없이는 머지하지 않습니다.

릴리스는 머지 전후로 전체 E2E 테스트를 거친 뒤 관리자가 진행합니다.
