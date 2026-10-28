# 여러 트래픽 기록을 취약점 증거로 연결하기

[English](../traffic-evidence.md) · 한국어 · [简体中文](../zh-Hans/traffic-evidence.md) · [繁體中文](../zh-Hant/traffic-evidence.md) · [日本語](../ja/traffic-evidence.md) · [Español](../es/traffic-evidence.md)

취약점 상세 화면의 연결된 트래픽에서는 여러 페이지의 기록 선택, 용도와 설명 지정, 순서 변경, 연결 해제를 지원합니다. 트래픽 화면에서도 여러 기록을 선택해 기존 취약점 하나에 연결할 수 있습니다. 다른 작업에서 상속한 증거는 읽기 전용이며, 원본 작업에서 수정해야 합니다.

시스템 설정의 ‘에이전트 자동 트래픽 연결’은 기본적으로 꺼져 있습니다. `/api/settings`의 `agent_traffic_binding`으로 조회하거나 변경합니다. 켜면 요청/응답 검토, 도구 호출, 지침에 필요한 토큰 사용량이 늘어나며 다음 에이전트 턴부터 적용됩니다. 끄면 자동 연결 매개변수와 추가 연결 도구를 숨기고 관련 지침을 넣지 않으며, 이미 실행 중인 세션이 새로 제출한 자동 연결도 거부합니다. 수동 연결, 캡처, 저장된 증거 조회와 내보내기는 계속 사용할 수 있습니다.

켜진 상태의 기본 흐름은 취약점 저장, 보고서 에이전트 자동 실행, 트래픽 확인 및 연결, 최신 증거 버전에 맞춘 보고서 작성입니다. 보고자는 `evidence`에 검증 명령, 주요 출력, 이미 존재하는 실제 트래픽 ID와 용도를 남깁니다. 보고서 에이전트는 취약점 상세와 실행 기록을 참고하고 `traffic_search` / `traffic_get`으로 확인한 뒤 `bind_finding_traffic`을 호출합니다. 저장 전에는 최신 `version`을 다시 읽습니다. 꺼진 상태에서는 보고서 에이전트에 원시 트래픽 검색/조회 도구를 추가로 주지 않고 자동 연결도 하지 않습니다. 수동으로 연결한 스냅샷을 읽고 보고서를 작성하는 것은 가능합니다.

기존 호출과의 호환성을 위해 `report_finding`의 `traffic_refs` / `evidence_hint_id`를 명시하면 즉시 연결할 수 있습니다. 연결은 선택 사항입니다. TCP 등 HTTP가 아닌 취약점, 캡처하지 않은 트래픽, 정확한 기록을 찾을 수 없는 경우에도 신고와 보고서 작성이 가능합니다. 명령 출력이나 로그 등 검증 가능한 다른 증거를 보존하고 필요하면 미연결 이유를 설명하세요. 새 필수 필드는 없습니다. 제출한 ID는 모두 유효하고 본문이 완전해야 합니다. 하나라도 실패하면 이번 연결 전체를 롤백하며, 신고와 함께 명시한 연결이 실패하면 신고 전체를 롤백합니다. 같은 스냅샷을 다시 추가해도 연결 수나 기존 설명은 바뀌지 않습니다.

## 에이전트 식별자와 보고서 버전

`report_finding`에 다음 선택 매개변수를 사용할 수 있습니다. 배열 순서가 최초 증거 순서입니다.

```json
{
  "traffic_refs": [
    {"traffic_id": "real-traffic-id", "role": "baseline", "note": "정상 계정 요청"},
    {"traffic_id": "another-real-traffic-id", "role": "proof", "note": "재현 요청"}
  ]
}
```

용도는 `baseline`(정상 비교), `proof`(취약점 입증), `verification`(추가 검증), `supporting`(보조 증거, 기본값)입니다. 먼저 `traffic_search` / `traffic_get`으로 실제 기록을 확인하세요. 도메인과 시간은 후보를 좁히는 조건이며 작업 소속을 입증하지 않습니다.

지침은 [CyberStrikeAI 취약점 신고 도구 안내](https://github.com/RuoJi6/CyberStrikeAI/blob/54d56774b8bd285817d16d48b70a4a5e6e0963f7/internal/app/vulnerability_tools.go)를 참고하되 이 프로젝트의 선택적 연결 규칙을 따릅니다. 캡처가 없는 이유를 필수로 요구하지 않습니다. ID를 추측하거나 캡처를 채우기 위해서만 탐사를 반복하지 마세요.

응답 첫 줄은 기존처럼 `finding recorded: <탐색 노드 ID>`입니다. 이후 JSON에는 독립 취약점 레코드 `finding_id`, 탐색 노드 `finding_node_id`, 연결 요약이 포함됩니다.

- `get_finding_traffic(finding_id)`는 독립 취약점 레코드 ID를 받으며 정렬된 목록과 `version`을 반환합니다. `binding_id`, `side=request|response`, `offset`, `length`를 지정하면 한 번에 최대 8192바이트씩 읽습니다.
- `update_finding_report`의 `finding_id`는 계속 탐색 노드 ID를 사용합니다. `evidence_version`에는 실제 읽은 버전을 넣습니다. 생성 중 증거가 바뀌면 이전 버전의 저장을 거부하므로 다시 읽고 작성해야 합니다.
- 버전 없이 호출한 기존 보고서는 저장된 트래픽 증거를 반영했다고 간주하지 않습니다. 연결, 설명, 용도, 순서를 바꾸면 기존 보고서에 갱신 필요 상태가 표시됩니다.

자동 연결을 켜면 `add_hint` / `add_task_hint`의 단일 힌트 또는 배치 `hints`의 각 항목에 `traffic_refs`를 저장할 수 있습니다. Planner가 대신 신고할 때는 `evidence_hint_id`로 현재 작업의 힌트에서 참조를 선택할 수 있으며 상속된 힌트는 사용할 수 없습니다. 시스템은 도메인, 시간, 방문 기록으로 연결을 추측하지 않습니다. 제출 실패 시 일부 취약점을 만들거나 보고서 생성을 미리 시작하지 않습니다.

기존 취약점의 누락된 연결은 `bind_finding_traffic(finding_id, traffic_refs)`으로 추가하며 다시 신고할 필요가 없습니다. `list_findings`, `list_task_findings`, `node_detail`, `get_task_node_detail`은 `finding_id`와 `finding_node_id`를 구분해 반환합니다. 기존 `id`는 탐색 노드 의미를 유지합니다.

시작 시 기존 도구 스키마에는 선택 속성만 추가합니다. 기본 트래픽 도구 연결을 보고서 에이전트까지 확장하고 추가 연결 도구도 기본 제공하지만, 사용자 지정 연결 목록, 프롬프트, 설명, 활성화 상태는 보존합니다. 지침은 도구 구성을 마친 뒤 일괄 적용합니다. 신고 역할은 기존 증거를 인계하고 보고서 에이전트는 확인, 연결, 작성을 담당합니다. 작업 맥락이 없는 플랫폼 대화는 직접 신고하는 대신 구조화된 힌트로 작업 에이전트에 인계해야 합니다. 완료 판단 전에 기존 증거를 인계하되 캡처가 없다고 강제로 기다리지 않습니다. 실패한 `report_finding`은 보고서 에이전트를 실행하지 않습니다.

## API

기본 경로는 `/api/exploration/findings/{finding_id}/traffic`이며 독립 취약점 ID를 사용합니다. 기존 인증을 적용하고 `context_task`로 작업 접근 권한과 상속 읽기 전용 여부를 확인합니다.

| 메서드 / 상대 경로 | 요청 / 응답 |
| --- | --- |
| `GET` | 정렬된 요약, 증거 버전, 보고서에 적용한 버전 |
| `POST` | `{"traffic_refs":[...]}` 배치 전체 추가 |
| `PATCH /{binding_id}` | `{"version":1,"role":"proof","note":"설명"}` |
| `DELETE /{binding_id}` | `{"version":1}` |
| `PUT /order` | `{"version":1,"binding_ids":["2","1"]}`; 전체 목록 필요 |
| `GET /{binding_id}` | 스냅샷 메타데이터와 제한된 본문 미리보기 |
| `GET /{binding_id}/body` | `side`, `offset`, `length`; `download=1`은 원본 전체 바이트 다운로드 |

버전/정렬 집합 충돌과 보관 중 쓰기는 `409`, 상속 증거 수정은 `403`, 없거나 해당 취약점 소속이 아닌 연결은 `404`를 반환합니다. 트래픽/첨부 읽기와 검증 실패는 명시적 오류로 반환합니다.

## 저장 및 마이그레이션

시작 시 PostgreSQL에 `traffic_evidence_snapshots`, `finding_traffic_bindings`, `findings.evidence_version` / `report_evidence_version`(기본값 0)을 멱등적으로 적용합니다. 과거 텍스트에서 연결을 추측하지 않습니다.

스냅샷에는 원본 트래픽 ID, 캡처 시각, URL, 메서드, 상태, 요청/응답 헤더, 본문 길이와 SHA-256을 저장합니다. 본문 경로는 `<data>/evidence/blobs/<해시 앞 두 글자>/<hash>.bin`이며 정리 가능한 `data/traffic`과 분리됩니다. 여러 취약점이 스냅샷/본문을 공유할 수 있습니다. 스냅샷 내용 수정 API는 없으며 무결성이 맞지 않으면 조회와 내보내기에 실패합니다.

원본 트래픽 쓰기 잠금 아래에서 대용량 blob과 이전 디렉터리 기록을 포함한 전체 본문을 읽습니다. 파일을 먼저 저장하고 검증한 다음, 단일 PostgreSQL 트랜잭션으로 탐색 노드, 의도 관계, 취약점, 스냅샷과 연결을 기록합니다. 커밋 후에만 Planner에 알립니다. 실패하면 참조 없는 파일이 남을 수 있지만 업무 레코드 일부만 저장되지는 않습니다.

PostgreSQL advisory lock `7337741004`가 증거 파일과 SQL 참조를 조정합니다. 작업 행 잠금은 보관 대기열에 들어간 뒤 증거 변경을 막습니다. 복원은 본문 설치부터 메타데이터 커밋까지 증거 잠금을 유지합니다. 취약점 삭제 시 연결도 연쇄 삭제합니다.

정리기는 매시간 실행하며 진행 중인 작업에 속하지 않는 미참조 내용만 최소 24시간 뒤 회수합니다. 일반 트래픽 정리는 증거 디렉터리를 건드리지 않습니다. 운영 중 데이터를 백업할 때는 PostgreSQL과 `data/evidence`를 함께 백업하세요.

## 내보내기와 보관

Markdown에는 정렬된 증거 목록과 버전, JSON에는 메타데이터, CSV에는 개수와 연결 ID가 포함됩니다. `md-zip`은 취약점 Markdown과 다음 파일을 제공합니다.

```text
evidence/<finding_id>/<binding_id>/
  manifest.json
  request.http
  response.http
  request.bin
  response.bin
```

Markdown은 상대 경로로 메시지를 참조합니다. 다운로드 전 첨부 복사, 해시 검증, 압축, 디스크 동기화, 모든 ZIP 항목을 읽는 CRC 검증을 마칩니다. 누락이나 손상이 있으면 다운로드 전체가 실패합니다. 전체 첨부는 원본 바이너리 바이트를 보존합니다.

보관 v3은 원본 트래픽이나 도메인 대신 취약점 연결 관계로 스냅샷과 본문을 수집합니다. 패키지 검증 후에만 운영 데이터를 정리하며 공유 증거는 남깁니다. 복원은 본문 검증과 설치 후 메타데이터와 연결을 트랜잭션으로 복구하고 실패 시 재시도할 수 있습니다. v1/v2도 복원 가능하며 없는 새 필드는 명시적으로 0으로 채웁니다.

## 검증과 경계

각 테스트 패키지에 새 독립 PostgreSQL 데이터베이스를 만들고 `ARTEX_PG_DSN`으로 지정하세요. 남은 작업/모델 테스트 데이터가 백그라운드 실행을 일으키는 일을 방지합니다. 관련 패키지 전체를 실행하고 설정 누락으로 건너뛴 테스트가 없는지 확인합니다.

```sh
# 각 패키지 실행 전에 해당 독립 테스트 DB의 ARTEX_PG_DSN을 설정합니다.
# 명시한 연결 설정에 문제가 있으면 테스트가 실패해야 합니다.
go test ./<package> -count=1
go test -race -p 1 ./evidence ./db ./agent ./server -run 'TestEvidence|TestFindingTraffic|TestFindingEvidence|TestReportFindingAtomicContract|TestTaskArchive'
```

프런트엔드는 `npx tsc --noEmit`, 변경 파일 Biome 검사, Webpack 빌드, `NEXT_EXPORT=1` 정적 내보내기로 검증합니다. 실행 중인 개발 서버를 덮어쓰지 않도록 별도 캐시 디렉터리를 사용하세요.

로컬 종단 간 검증에는 독립 포트, 통제된 HTTP/도메인 기반 HTTPS 대상, 임시 데이터 디렉터리를 사용합니다. 두 연결 진입점, 여러 페이지 선택, 정렬/설명, 오류, 상속 읽기 전용, 다운로드, 원본 트래픽 삭제 후 내보내기, 보관, 운영 본문 회수, 복원, 해시 검증을 확인합니다.

초기 구현은 전역 증거 조정 잠금을 사용합니다. 대량 연결/내보내기나 느린 첨부 다운로드 중에는 다른 증거 작업이 기다릴 수 있습니다. 기록이나 완전한 본문이 없으면 증거를 만들어 낼 수 없습니다. 이 기능은 캡처 설정이나 IP 직접 접속 HTTPS의 인증서 문제를 바꾸지 않습니다.
