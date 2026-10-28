[English](../release-notice-repair.md) · [한국어](../ko/release-notice-repair.md) · 简体中文 · [繁體中文](../zh-Hant/release-notice-repair.md) · [日本語](../ja/release-notice-repair.md) · [Español](../es/release-notice-repair.md)

<a id="v021-license-notice-packaging-repair"></a>

# v0.2.1 授权声明封装修复

各语言版本请见本页上方导览。

`repair-release-notices.yml` 是对既有 v0.2.1 发行版运行一次的手动修正。它将 `THIRD_PARTY_NOTICES.txt` 加入五个已发布 ZIP 及既有 Linux amd64/arm64 镜像，保留 Git 标签、原始封存项目、全部原始映像层及运行配置，不构建应用程序代码。变更正式标签前，输入映像摘要必须与既有 v0.2.1 及 latest 标签一致。

仅在原标签触发的发行工作流（含 Docker 工作）成功完成后才可启动，以免覆盖修正。请审查修复工作流并确认目前不可变的多架构映像摘要。工作流使用既有 `GITHUB_TOKEN` 存取发行资产与 GHCR，无须修改个人 OAuth 权限。它会发布独立声明、将原始校验档保留为 `SHA256SUMS.original`，并在工作流产物中保留原始与替换 ZIP、可执行文件哈希、各平台 manifest／配置及两个映像摘要 30 天。维护者必须在版本说明揭露变更后的封装校验值与映像摘要。应用程序版本与哈希保持不变。

GitHub 资产替换与容器镜像仓库标签更新是分开的操作。任一操作后若发布失败，**不得针对已部分变更的版本重新运行工作流**。请在产物到期前下载保留的工作流产物，检查 `archive-verification.json`、`image-verification.json` 及原始／修正映像索引，确认哪一端已完成。使用获授权的维护者会话完成同一项已验证修正：



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



然后重新下载已发布封存档并运行 `sha256sum --check SHA256SUMS`，验证两个容器镜像仓库标签都解析至记录的修正摘要。若维护者改选恢复原始封装，产物中的原始 ZIP 与 `image-verification.json` 的 `original_image_digest` 可作为回滚输入。依需要在产物到期后继续保留修正证据。
