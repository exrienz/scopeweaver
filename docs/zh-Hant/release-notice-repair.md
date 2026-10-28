[English](../release-notice-repair.md) · [한국어](../ko/release-notice-repair.md) · [简体中文](../zh-Hans/release-notice-repair.md) · 繁體中文 · [日本語](../ja/release-notice-repair.md) · [Español](../es/release-notice-repair.md)

<a id="v021-license-notice-packaging-repair"></a>

# v0.2.1 授權聲明封裝修復

各語言版本請見本頁上方導覽。

`repair-release-notices.yml` 是對既有 v0.2.1 發行版執行一次的手動修正。它將 `THIRD_PARTY_NOTICES.txt` 加入五個已發布 ZIP 及既有 Linux amd64/arm64 映像檔，保留 Git 標籤、原始封存項目、全部原始映像層及執行設定，不建置應用程式程式碼。變更正式標籤前，輸入映像摘要必須與既有 v0.2.1 及 latest 標籤一致。

僅在原標籤觸發的發行工作流程（含 Docker 工作）成功完成後才可啟動，以免覆蓋修正。請審查修復工作流程並確認目前不可變的多架構映像摘要。工作流程使用既有 `GITHUB_TOKEN` 存取發行資產與 GHCR，無須修改個人 OAuth 權限。它會發布獨立聲明、將原始校驗檔保留為 `SHA256SUMS.original`，並在工作流程成品中保留原始與替換 ZIP、執行檔雜湊、各平台 manifest／設定及兩個映像摘要 30 天。維護者必須在版本說明揭露變更後的封裝校驗值與映像摘要。應用程式版本與雜湊保持不變。

GitHub 資產替換與容器映像檔登錄庫標籤更新是分開的操作。任一操作後若發布失敗，**不得針對已部分變更的版本重新執行工作流程**。請在成品到期前下載保留的工作流程成品，檢查 `archive-verification.json`、`image-verification.json` 及原始／修正映像索引，確認哪一端已完成。使用獲授權的維護者工作階段完成同一項已驗證修正：



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



然後重新下載已發布封存檔並執行 `sha256sum --check SHA256SUMS`，驗證兩個容器映像檔登錄庫標籤都解析至記錄的修正摘要。若維護者改選還原原始封裝，成品中的原始 ZIP 與 `image-verification.json` 的 `original_image_digest` 可作為復原輸入。依需要在成品到期後繼續保留修正證據。
