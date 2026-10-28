[English](../traffic-evidence.md) | [한국어](../ko/traffic-evidence.md) | [简体中文](../zh-Hans/traffic-evidence.md) | [繁體中文](../zh-Hant/traffic-evidence.md) | [日本語](traffic-evidence.md) | [Español](../es/traffic-evidence.md)

<a id="multiple-traffic-records-as-finding-evidence"></a>
# 複数の通信記録を検出事項の証拠として使用する



検出事項の詳細では、ページをまたいで通信を選択し、役割と注記を付け、順序を変更し、紐付けを解除できます。通信のページでも複数の記録を選んで既存の検出事項に添付できます。他のタスクから継承した証拠は読み取り専用です。元のタスクで編集してください。

「エージェントによる通信の自動紐付け」は既定で無効です。`/api/settings` の `agent_traffic_binding` で確認・変更できます。有効にすると、リクエスト・レスポンスの審査、ツール呼び出し、指示の分だけトークン費用が増え、次のエージェントのターンから適用します。無効の場合は、自動紐付けのパラメーターと追加の紐付けツールを隠し、自動紐付けの指示を注入せず、実行中のセッションからの新しい自動紐付け送信も拒否します。手動の紐付け、通信の記録、保存済み証拠の読み取り、エクスポートは引き続き利用できます。

有効な場合の通常の流れは、検出事項の記録、レポートエージェントの自動起動、通信の検証・紐付け、最新の証拠バージョンに対するレポート作成です。報告するエージェントは、検証コマンド、重要な出力、既存の実在する通信 ID と用途を `evidence` に保持します。レポートエージェントは検出事項と実行記録を確認し、`traffic_search` / `traffic_get` で通信を検証して `bind_finding_traffic` を呼び、保存前に最新の `version` を読みます。無効の場合、レポートエージェントには追加の生の通信検索・読み取りツールを渡さず、自動では紐付けませんが、手動で紐付けたスナップショットを読み、レポートを作成できます。

互換性のため、呼び出し元は引き続き `report_finding.traffic_refs` / `evidence_hint_id` で明示的に即時紐付けできます。紐付けは任意で、TCP の問題など HTTP 以外の検出事項、未記録の通信、正確な記録がない場合も報告できます。コマンド、ログ、その他の検証可能な証拠を保持し、有用な場合は通信がない理由を説明してください。新しい必須フィールドはありません。送信する ID はすべて有効で、完全な本文が必要です。失敗すると紐付け操作全体をロールバックし、報告時の明示的な紐付けに失敗すると報告の送信全体をロールバックします。既存のスナップショットを再度追加しても、紐付けを重複させず、注記も上書きしません。

<a id="agent-identifiers-and-report-versions"></a>
## エージェントの識別子とレポートのバージョン

`report_finding` は下記の任意パラメーターを受け取ります。配列の順序が、証拠の初期順序になります。

```json
{
  "traffic_refs": [
    {"traffic_id": "real-traffic-id", "role": "baseline", "note": "Normal account request"},
    {"traffic_id": "another-real-traffic-id", "role": "proof", "note": "Reproduction request"}
  ]
}
```

役割は `baseline`（正常時の比較）、`proof`（検出事項の証明）、`verification`（追加検証）、`supporting`（既定の補助的な証拠）です。先に `traffic_search` / `traffic_get` で実際の記録を検証してください。ドメインと時刻は候補を絞るだけで、タスクへの帰属を確定しません。

指示は [CyberStrikeAI の検出事項報告ガイド](https://github.com/RuoJi6/CyberStrikeAI/blob/54d56774b8bd285817d16d48b70a4a5e6e0963f7/internal/app/vulnerability_tools.go)を参照し、このプロジェクトの任意の紐付け契約に適合させています。通信がない理由は必須ではありません。ID を推測したり、記録を得るだけのために再検査したりしないでください。

応答の最初の行は、引き続き `finding recorded: <exploration node ID>` です。続く JSON に、別の検出事項レコード `finding_id`、探索ノード `finding_node_id`、紐付けの要約を含めます。

- `get_finding_traffic(finding_id)` は、別の検出事項レコード ID を受け取り、順序付き一覧と `version` を返します。最大 8192 バイトずつ読むには、`binding_id`、`side=request|response`、`offset`、`length` を指定してください。
- `update_finding_report.finding_id` は、引き続き探索ノード ID を受け取ります。`evidence_version` は実際に読んだバージョンにしてください。生成中に証拠が変わると、古いバージョンの書き込みは拒否します。読み直して再生成してください。
- バージョンのない従来のレポート呼び出しは、既存の通信の証拠を対象にしたとは主張しません。紐付け、注記、役割、順序の変更は、既存のレポートを更新が必要な状態にします。

自動紐付けが有効なら、`add_hint` / `add_task_hint` は単一のヒントまたは一括の `hints` 配列の各項目に `traffic_refs` を保存できます。他のエージェントに代わって報告するプランナーは、`evidence_hint_id` で、このタスクのヒントから参照を選択できます。継承したヒントは不可です。システムはドメイン、時刻、閲覧履歴から紐付けを推測しません。送信に失敗しても、部分的な検出事項を作成したり、レポート生成を早く開始したりしません。

`bind_finding_traffic(finding_id, traffic_refs)` で、検出事項を再登録せず、既存のものに不足する紐付けを追加できます。`list_findings`、`list_task_findings`、`node_detail`、`get_task_node_detail` は `finding_id` と `finding_node_id` の両方を公開し、従来の `id` は探索ノードの意味を保持します。

起動時は古いツールのスキーマに任意のプロパティを追加するだけです。元の既定の通信ツールの紐付けをレポートエージェントにも拡張し、追加の紐付けツールを既定で渡します。カスタムの紐付け一覧、プロンプト、説明、有効状態は保持します。指示はツールの最終組み立て後に追記し、報告する役割が既存の証拠を引き継ぎ、レポートエージェントが検証・紐付け・作成を行います。タスクのコンテキストがないプラットフォームのチャットでは、直接報告せず、構造化ヒントをタスクのエージェントに引き継いでください。タスクの完了宣言前に既存の証拠を引き継ぎます。記録がないことで待機を強制しません。`report_finding` が失敗しても、レポートエージェントは起動しません。

<a id="api"></a>
## API

ベースパスは、別の検出事項 ID を使う `/api/exploration/findings/{finding_id}/traffic` です。既存の認証を適用し、`context_task` がタスクの可視性と継承の読み取り専用アクセスを確認します。

| メソッド / 相対パス | リクエスト / レスポンス |
| --- | --- |
| `GET` | 順序付きの要約、証拠のバージョン、レポートの証拠バージョン |
| `POST` | `{"traffic_refs":[...]}` のバッチ全体を追記 |
| `PATCH /{binding_id}` | `{"version":1,"role":"proof","note":"Description"}` |
| `DELETE /{binding_id}` | `{"version":1}` |
| `PUT /order` | `{"version":1,"binding_ids":["2","1"]}`。完全な一覧が必要 |
| `GET /{binding_id}` | スナップショットのメタデータと、長さを制限した本文のプレビュー |
| `GET /{binding_id}/body` | `side`、`offset`、`length`。`download=1` は元の全バイトをダウンロード |

バージョン・順序集合の競合とアーカイブ中の書き込みは `409`、継承した内容への書き込みは `403`、存在しない紐付けや検出事項外の紐付けは `404` を返します。通信・添付ファイルの読み取りや検証に失敗した場合は、明示的なエラーを返します。

<a id="storage-and-migration"></a>
## 保存と移行

起動時に PostgreSQL の `traffic_evidence_snapshots`、`finding_traffic_bindings`、`findings.evidence_version` / `report_evidence_version`（既定 0）を冪等に移行します。過去の説明文から紐付けを推測しません。

スナップショットは元の通信 ID、記録時刻、URL、メソッド、ステータス、リクエスト・レスポンスのヘッダー、本文の長さ、SHA-256 を保存します。本文は使い捨ての `data/traffic` とは独立して、`<data>/evidence/blobs/<first-two-characters>/<hash>.bin` に置きます。複数の検出事項でスナップショット・本文を共有できます。スナップショットの内容を更新する API はなく、整合性が一致しなければ読み取りとエクスポートを失敗にします。

大きな Blob や旧方式のディレクトリ記録を含む完全な本文を、元の通信の書き込みロック下で読み取ります。まずファイルを永続化・検証し、次に 1 つの PostgreSQL トランザクションで探索ノード、探索方針の関係、検出事項、スナップショット、紐付けを書き込みます。プランナーへの通知はコミット後のみです。失敗すると参照されないファイルが残る場合はありますが、部分的な業務レコードは残りません。

PostgreSQL のアドバイザリーロック `7337741004` が証拠ファイルと SQL の参照を調整します。タスク行のロックは、アーカイブがキューに入った後の証拠変更を禁止します。復元は本文の設置からメタデータのコミットまで証拠ロックを保持します。検出事項の削除は紐付けに連鎖します。

クリーンアップは毎時動作し、実行中の操作外で、24 時間以上経過した未参照の内容だけを回収します。通常の通信削除は証拠ディレクトリを保持します。ホットデータのバックアップでは、PostgreSQL と `data/evidence` を一緒に保存してください。

<a id="export-and-archives"></a>
## エクスポートとアーカイブ

Markdown は順序付きの証拠一覧とバージョン、JSON はメタデータ、CSV は件数と紐付け ID を含みます。`md-zip` は検出事項の Markdown を保持し、次を含めます:

```text
evidence/<finding_id>/<binding_id>/
  manifest.json
  request.http
  response.http
  request.bin
  response.bin
```

Markdown は相対パスでメッセージにリンクします。ダウンロードの送信前に、サーバーは添付ファイルのコピー、ハッシュ検証、アーカイブの圧縮・同期、全 ZIP 項目の読み取りによる CRC 検査を行います。内容がないか破損していれば、ダウンロード全体を失敗にします。完全な添付ファイルは、元のバイナリのバイト列を保持します。

アーカイブ v3 は、元の通信やドメインと独立して、検出事項の紐付けからスナップショットと本文を収集します。ホットデータを削除するのはアーカイブの検証後のみで、共有する証拠は保持します。復元は、メタデータと紐付けをトランザクションで復元する前に本文を検証・設置し、失敗後の再試行に対応します。v1/v2 も引き続き復元可能で、新しいフィールドがなければ明示的に 0 を設定します。

<a id="verification-and-boundaries"></a>
## 検証と境界

パッケージごとに、`ARTEX_PG_DSN` を通じて個別の新しい PostgreSQL テスト DB を使い、残存するタスク・モデルのテストデータがバックグラウンド作業を開始しないようにしてください。関連パッケージ全体を実行し、設定不足でスキップされていないか確認します:

```sh
# Set ARTEX_PG_DSN to the package's isolated database before each run.
# Explicit configuration failures must fail the test.
go test ./<package> -count=1
go test -race -p 1 ./evidence ./db ./agent ./server -run 'TestEvidence|TestFindingTraffic|TestFindingEvidence|TestReportFindingAtomicContract|TestTaskArchive'
```

フロントエンドは `npx tsc --noEmit`、変更ファイルの Biome、Webpack ビルド、`NEXT_EXPORT=1` の静的エクスポートを確認します。実行中の開発サーバーを上書きしないよう、別のキャッシュディレクトリを使ってください。

ローカル E2E の受け入れ確認は、専用ポート、管理された HTTP・ドメインベースの HTTPS 対象、一時データディレクトリを使います。両方の紐付け入口、ページをまたぐ選択、順序・注記、エラー、継承の読み取り専用アクセス、ダウンロード、元の通信削除後のエクスポート、アーカイブ、ホット本文の回収、復元、ハッシュ検証を対象にします。

初期実装は、証拠全体の調整ロックを使います。大きな紐付け・エクスポートや遅い添付ダウンロードは、他の証拠操作を待たせる場合があります。記録がないものや不完全な本文は捏造できません。この機能は記録設定も、IP アドレスへ直接行う HTTPS 接続の証明書の扱いも変更しません。
