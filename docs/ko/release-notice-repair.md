# v0.2.1 라이선스 고지 패키징 수정

[English](../release-notice-repair.md) · 한국어 · [简体中文](../zh-Hans/release-notice-repair.md) · [繁體中文](../zh-Hant/release-notice-repair.md) · [日本語](../ja/release-notice-repair.md) · [Español](../es/release-notice-repair.md)

`repair-release-notices.yml`은 기존 v0.2.1 릴리스를 한 번 수정하는 수동 워크플로입니다.
배포된 ZIP 5개와 기존 Linux amd64/arm64 이미지에 `THIRD_PARTY_NOTICES.txt`를 추가합니다.
Git 태그, 원본 ZIP의 모든 항목, 이미지의 기존 레이어와 실행 설정을 보존하며 애플리케이션
코드를 빌드하지 않습니다. 공식 이미지 태그를 변경하기 전에는 입력한 불변 멀티아키텍처
이미지 다이제스트가 기존 v0.2.1 및 latest 태그와 일치해야 합니다.

원래 태그로 시작한 릴리스 워크플로와 Docker 작업이 모두 성공적으로 끝난 뒤 실행해야
수정 결과를 다시 덮어쓰지 않습니다. 수정 워크플로를 검토하고 현재 이미지 다이제스트를
확인합니다. 릴리스 자산과 GHCR에는
기존 `GITHUB_TOKEN`을 사용하며 개인 OAuth 권한을 변경하지 않습니다. 별도 고지 파일을
게시하고 원본 체크섬을 `SHA256SUMS.original`로 보존합니다. 원본·수정 ZIP, 실행 파일 해시,
플랫폼별 매니페스트·설정과 변경 전후 이미지 다이제스트를 워크플로 아티팩트에 30일간 보존합니다.
유지관리자는 패키징 체크섬과 이미지 다이제스트 변경을 릴리스 노트에 공개해야 합니다.
애플리케이션 버전과 애플리케이션 해시는 바뀌지 않습니다.

GitHub 자산 교체와 레지스트리 태그 변경은 별도 작업입니다. 일부 게시 후 실패하면
**부분 변경된 릴리스에 워크플로를 다시 실행하지 마세요.** 만료 전에 아티팩트를 내려받아
`archive-verification.json`, `image-verification.json`과 변경 전후 이미지 인덱스를 검토하고
이미 완료된 작업을 확인합니다. 권한이 있는 유지관리자 세션에서 검증된 동일 수정을 완료합니다.

```bash
# 내려받은 아티팩트 루트에서 실행하며, 먼저 JSON 검증 기록을 확인합니다.
set -euo pipefail
assets="$(gh release view v0.2.1 --json assets --jq '.assets[].name')"
if grep -Fxq SHA256SUMS.original <<< "$assets"; then
  mkdir -p recovery-existing
  gh release download v0.2.1 --pattern SHA256SUMS.original --dir recovery-existing
  cmp repaired/SHA256SUMS.original recovery-existing/SHA256SUMS.original || exit 1
else
  gh release upload v0.2.1 repaired/SHA256SUMS.original
fi
gh release upload v0.2.1 THIRD_PARTY_NOTICES.txt --clobber
gh release upload v0.2.1 repaired/scopeweaver-0.2.1-*.zip repaired/SHA256SUMS --clobber
image=ghcr.io/cskwork/scopeweaver
digest="$(cat repaired/corrected-image-digest.txt)"
docker buildx imagetools create --tag "$image:v0.2.1" --tag "$image:latest" "$image@$digest"
```

이후 게시된 ZIP을 다시 내려받아 `sha256sum --check SHA256SUMS`를 실행하고 이미지 태그
2개가 기록된 수정 다이제스트를 가리키는지 확인합니다. 유지관리자가 원본 패키징 복원을
선택하면 아티팩트의 원본 ZIP과 `image-verification.json`의 `original_image_digest`를
사용합니다. 필요하면 아티팩트 만료 이후에도 수정 증거를 별도로 보존합니다.
