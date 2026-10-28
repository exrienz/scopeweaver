# 추적

[English](../../../../../skills/playwright-cli/references/tracing.md) · 한국어 · [简体中文](../../../../zh-Hans/skills/playwright-cli/references/tracing.md) · [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/tracing.md) · [日本語](../../../../ja/skills/playwright-cli/references/tracing.md) · [Español](../../../../es/skills/playwright-cli/references/tracing.md)

디버깅/분석용 상세 실행 기록을 수집합니다. DOM 스냅샷, 스크린샷, 네트워크 활동, 콘솔 로그가 포함됩니다.

## 기본 사용

```bash
# Start trace recording
playwright-cli tracing-start

# Perform actions
playwright-cli open https://example.com
playwright-cli click e1
playwright-cli fill e2 "test"

# Stop trace recording
playwright-cli tracing-stop
```

## 추적 출력 파일

추적을 시작하면 Playwright는 여러 파일이 담긴 `traces/`를 만듭니다:

### `trace-{timestamp}.trace`

주 추적 파일인 작업 로그에는 다음이 포함됩니다:
- 수행한 모든 작업 (클릭, 입력, 이동)
- 각 작업 전후 DOM 스냅샷
- 단계별 스크린샷
- 시간 정보
- 콘솔 메시지
- 소스 위치

### `trace-{timestamp}.network`

네트워크 로그에는 전체 활동이 포함됩니다:
- 모든 HTTP 요청/응답
- 요청 헤더/본문
- 응답 헤더/본문
- 시간 (DNS, 연결, TLS, TTFB, 다운로드)
- 리소스 크기
- 실패 요청과 오류

### `resources/`

리소스 디렉터리의 캐시 내용:
- 이미지, 글꼴, 스타일시트, 스크립트
- 재생용 응답 본문
- 페이지 상태 재구성에 필요한 파일

## 추적 수집 범위

| 분류 | 상세 |
|----------|---------|
| 작업 | 클릭, 입력, hover, 키보드, 이동 |
| DOM | 각 작업 전후 전체 DOM 스냅샷 |
| 스크린샷 | 각 단계 시각 상태 |
| 네트워크 | 모든 요청, 응답, 헤더, 본문, 시간 |
| 콘솔 | 모든 console.log, warn, error |
| 시간 | 작업별 정확한 시간 |

## 사용 사례

### 실패 작업 디버깅

```bash
playwright-cli tracing-start
playwright-cli open https://app.example.com

# This click fails - why?
playwright-cli click e5

playwright-cli tracing-stop
# Open trace to see DOM state when click was attempted
```

### 성능 분석

```bash
playwright-cli tracing-start
playwright-cli open https://slow-site.com
playwright-cli tracing-stop

# View network waterfall to identify slow resources
```

### 증거 수집

```bash
# Record a complete user flow for documentation
playwright-cli tracing-start

playwright-cli open https://app.example.com/checkout
playwright-cli fill e1 "4111111111111111"
playwright-cli fill e2 "12/25"
playwright-cli fill e3 "123"
playwright-cli click e4

playwright-cli tracing-stop
# Trace shows exact sequence of events
```

## 추적, 영상, 스크린샷 비교

| 항목 | 추적 | 영상 | 스크린샷 |
|---------|-------|-------|------------|
| 형식 | .trace 파일 | .webm 영상 | .png/.jpeg 이미지 |
| DOM 조회 | 가능 | 불가 | 불가 |
| 네트워크 상세 | 가능 | 불가 | 불가 |
| 단계별 재생 | 가능 | 연속 | 단일 프레임 |
| 파일 크기 | 중간 | 큼 | 작음 |
| 적합한 용도 | 디버깅 | 시연 | 빠른 캡처 |

## 권장 방법

### 1. 문제 발생 전에 추적 시작

```bash
# Trace the entire flow, not just the failing step
playwright-cli tracing-start
playwright-cli open https://example.com
# ... all steps leading to the issue ...
playwright-cli tracing-stop
```

### 2. 오래된 추적 정리

추적은 많은 디스크 공간을 사용할 수 있습니다:

```bash
# Remove traces older than 7 days
find .playwright-cli/traces -mtime +7 -delete
```

## 한계

- 추적은 자동화에 추가 부하를 줍니다
- 큰 추적은 많은 디스크 공간을 사용합니다
- 일부 동적 내용은 완전히 재생되지 않을 수 있습니다
