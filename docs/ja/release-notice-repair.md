[English](../release-notice-repair.md) | [한국어](../ko/release-notice-repair.md) | [简体中文](../zh-Hans/release-notice-repair.md) | [繁體中文](../zh-Hant/release-notice-repair.md) | [日本語](release-notice-repair.md) | [Español](../es/release-notice-repair.md)

<a id="v021-license-notice-packaging-repair"></a>
# v0.2.1 のライセンス告知のパッケージ修正



`repair-release-notices.yml` は、既存の v0.2.1 リリースに対する、手動の 1 回限りの修正です。公開済みの ZIP 5 件と既存の Linux amd64/arm64 イメージに `THIRD_PARTY_NOTICES.txt` を追加します。Git タグ、元のアーカイブのすべての項目、元のすべてのイメージレイヤー、イメージの実行時設定を保持し、アプリケーションのコードはビルドしません。公式タグを変更する前に、入力イメージのダイジェストが既存の v0.2.1 と latest タグに一致する必要があります。

修正が上書きされないよう、元のタグ起動のリリースワークフロー（Docker ジョブを含む）が正常完了した後にのみ実行してください。修正ワークフローをレビューし、現在の不変なマルチアーキテクチャのイメージダイジェストを確認します。リリース資産と GHCR には既存の `GITHUB_TOKEN` を使い、個人の OAuth スコープ変更は不要です。告知を単独でも公開し、元のチェックサムファイルを `SHA256SUMS.original` として保持します。元の ZIP、置き換えた ZIP、実行ファイルのハッシュ、プラットフォーム別のマニフェスト・設定、両イメージダイジェストは、ワークフローの成果物に 30 日間保持します。メンテナーはリリースノートで、変更したパッケージのチェックサムとイメージダイジェストを開示する必要があります。既存のアプリケーションのバージョンとハッシュは変わりません。

GitHub 資産の置き換えとレジストリーのタグ更新は別の操作です。いずれかの後で公開が失敗した場合、**一部変更済みのリリースに対してワークフローを再実行しないでください**。期限が切れる前に、保持されたワークフローの成果物をダウンロードします。`archive-verification.json`、`image-verification.json`、元と修正版のイメージインデックスを確認し、どちらが完了済みか特定してください。許可されたメンテナーのセッションで、同じ検証済みの修正を完了します:

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

その後、公開したアーカイブを再度ダウンロードして `sha256sum --check SHA256SUMS` を実行し、両方のレジストリータグが記録した修正版ダイジェストを指すと検証してください。メンテナーが元のパッケージへの復元を選ぶ場合、成果物内の元の ZIP と `image-verification.json` の `original_image_digest` がロールバックの入力になります。必要に応じて、成果物の期限後も修正の証拠を保持してください。
