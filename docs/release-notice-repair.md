# v0.2.1 license notice packaging repair

English · [한국어](ko/release-notice-repair.md) · [简体中文](zh-Hans/release-notice-repair.md) · [繁體中文](zh-Hant/release-notice-repair.md) · [日本語](ja/release-notice-repair.md) · [Español](es/release-notice-repair.md)

`repair-release-notices.yml` is a manual, one-time correction for the existing v0.2.1
release. It adds `THIRD_PARTY_NOTICES.txt` to the five published ZIPs and to the
existing Linux amd64/arm64 image. It preserves the Git tag, every original archive
entry, all original image layers, and the image runtime configuration. It builds
no application code. The input image digest must match the existing v0.2.1 and
latest tags before official tags are changed.

Dispatch only after the original tag-triggered release workflow (including its
Docker job) has completed successfully, so it cannot overwrite the correction.
Review the repair workflow and confirm the current immutable multiarch image
digest. The workflow uses its existing `GITHUB_TOKEN` for release
assets and GHCR; no personal OAuth scope changes are required. It publishes the
standalone notice, retains the original checksum file as `SHA256SUMS.original`,
and retains original ZIPs, replacement ZIPs, executable hashes, per-platform
manifests/configuration and both image digests in a workflow artifact for 30 days.
The maintainer must disclose the changed packaging checksums and image digests in
the release notes. Existing application versions and application hashes stay unchanged.

GitHub asset replacement and registry tag updates are separate operations. If
publication fails after either operation, **do not rerun the workflow against the
partially changed release**. Download the retained workflow artifact before it
expires. Review `archive-verification.json`, `image-verification.json` and the
original/corrected image indexes to identify which side has already completed.
Using an authorized maintainer session, finish the same validated correction:

```bash
# Run from the downloaded artifact root; inspect the JSON evidence first.
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

Then download the published archives again and run `sha256sum --check SHA256SUMS`;
verify both registry tags resolve to the recorded corrected digest. The original
ZIPs in the artifact and `original_image_digest` in `image-verification.json`
provide rollback inputs if the maintainer instead chooses to restore the original
packaging. Preserve the correction evidence beyond artifact expiry as needed.
