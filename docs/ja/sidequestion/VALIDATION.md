[English](../../../sidequestion/VALIDATION.md) | [한국어](../../ko/sidequestion/VALIDATION.md) | [简体中文](../../zh-Hans/sidequestion/VALIDATION.md) | [繁體中文](../../zh-Hant/sidequestion/VALIDATION.md) | [日本語](VALIDATION.md) | [Español](../../es/sidequestion/VALIDATION.md)

<a id="btw-verification-record"></a>
# `/btw` の検証記録



上流の過去の記録です。日付: 2026-09-10。ブランチ: `codex/btw-side-question`。基準: `8dae851b9b622f2ff2631f332fde9719d0b16fba`。新しい ScopeWeaver の結果ではありません。

隔離した PostgreSQL のテスト DB とデータディレクトリを使用しました。実際のモデルの認証情報は、その環境にのみ注入し、ソースやこの記録には含めていません。製品の既定モデルは変更していません。Go 1.26.3、norma v0.3.6、Next.js 16.2.9。

モデルとの会話、戻り値のオブジェクト、実装上のアサーション、Qwen のレビューは、API 認証情報を除いて [validation-2026-09-10.json](../../../sidequestion/validation-2026-09-10.json) にあります。説明文は翻訳し、識別子と測定値は保持しています。

<a id="engineering-checks"></a>
## 実装の検査

| 範囲 | 結果 | 証拠 |
| --- | --- | --- |
| 構造化メッセージとツール引数の深いコピー | 合格 | `TestCheckpointDeepCopyAndBoundaries` |
| 要約・圧縮要求は上書きしない。完了した応答と最終状態は公開し、部分応答は除外 | 合格 | `TestCheckpointDeepCopyAndBoundaries`, `TestSnapshotExcludesPartialStreamAndSelectsPoolMember` |
| 実際のモデルプールのメンバーの識別 | 合格 | `TestSnapshotExcludesPartialStreamAndSelectsPoolMember` |
| ツールの対、20 件のやり取りの再生、予算内への削減、超過エラー | 合格 | `TestBuildRequestCompactionToolPairingAndBudget` |
| メイン・補足の並列実行と、双方向の中止の隔離 | 合格 | ブロックするプロバイダー、`TestMainSideConcurrencyAndIndependentCancellation` |
| ツール未実行、ストリーミング・非ストリーミング、失敗時に取得可能な使用量 | 合格 | `TestServiceNoToolsAndUsageOnFailure` |
| 実際の norma ChatAgent とローカル Read ツール。メインの会話記録・アクティビティの隔離 | 合格 | `TestSideActualChatCheckpointToolResultAndTranscriptIsolation`。ストリーミング・非ストリーミングのサブテスト |
| 永続化、ページ送り、冪等性、再起動後の部分回答 | 合格 | `TestSideHistoryIdempotencyPagingAndRecovery` |
| 消去と遅い書き込みの競合、親の削除、バージョン比較 | 合格 | `TestSideClearLateWritersAndDeletedParent` |
| MainAgent/Worker のアーカイブと復元、v1/v2/v3 | 合格 | `TestSideTaskArchiveVersions` |
| 3 種の親 API、認証、所有権、ワーカーの論理削除 | 合格 | `TestSideHTTPGlobalLimitTaskWorkerAndDeletion`, `TestSideCheckpointPersistsBeforeAdmissionAndRestart` |
| メインセッションの処理中の補足の質問。独立した SSE の再接続・切断、中止、消去 | 合格 | `TestSideHTTPBusyIsolationClearAndReconnect` |
| 親ごとに同時要求 1 件、全体で 4 件 | 合格 | 2 つの `TestSideHTTP…` テスト |
| 受け入れ前のスナップショット保存、再起動後の追加質問、古いセッションのスナップショットを捏造しない | 合格 | `TestSideCheckpointPersistsBeforeAdmissionAndRestart` |
| 削除済みのキャッシュされたプロファイルや、変更されたモデルの拒否 | 合格 | `TestSideRejectsDeletedOrChangedCachedProfile` |
| アーカイブ前に中止し、最終回答・使用量の永続化を待機 | 合格 | `TestSideTaskDrainPersistsBeforeArchive` |
| ストリーミング利用側の早い中止でも、使用量と補足への帰属を 1 回記録 | 合格 | `TestSideUsageRecordedOnceOnConsumerCancellation` |
| 再起動から復元したワーカー・期限付き実行のコンテキストが、新しいスナップショットを公開 | 合格 | `TestSideRestoredWorkerRuntimePublishesNewCheckpoint` |
| 関連パッケージの競合検査 | 合格 | 下記のコマンド |
| TypeScript と本番ビルド | 合格 | `npx tsc --noEmit`, `npm run build` |
| 新しいフロントエンドモジュールの Biome | 合格 | `biome check`、新しい 3 モジュール |

本番ではない個別の使い捨て DB を `ARTEX_PG_DSN` に設定して、自動検査を再現してください:

```sh
go test -race ./agent ./db ./server ./sidequestion ./llmrec ./llmpool \
  -run 'Test(Side|Checkpoint|Snapshot|BuildRequest|Service|MainSide|CaptureRun|TaskArchive|CompleteForwards|StopIntent|CancelIntent)' -count=1
cd web
npx tsc --noEmit
npx biome check src/lib/side-questions.ts src/hooks/use-side-questions.ts src/components/side-question-workspace.tsx
npm run build
```

Go の全回帰テストは全件合格ではありませんでした。既存の `server` テスト 2 件が、一時ディレクトリの削除中に `TempDir RemoveAll … directory not empty` で失敗しました:

- `TestInheritedActivityDetailAndRelationDeletion`
- `TestTaskMetadataPatchReturnsRenameAndPin`

未変更の基準をエクスポートし、同じ隔離環境で `server` を再実行すると、両方の削除失敗を再現しました。基準では `TestCoreTaskLifecyclePG` の対象ノード数のアサーションにも失敗しましたが、最終の変更済みサーバーの回帰テストでは失敗しませんでした。他のパッケージ、補足の質問のケース、競合検査は合格しました。基準の問題は受け入れ済みと数えず、隠すために既存のアサーションを変更していません。

Next.js は既存の複数ロックファイル・ワークスペースルート推定の警告を出しましたが、ビルドとすべてのページ生成は完了しました。

<a id="browser-checks"></a>
## ブラウザーの確認

Codex のアプリ内ブラウザーを、隔離したローカル Go・Next.js 開発サーバーに接続しました。デスクトップと 390 × 844 の狭い画面で、ブラウザー自動操作とスクリーンショット・ログの確認を行いました:

- 通常のチャットで、`/btw` がデスクトップのサイドバーにメインと補足の内容を同時に表示しました。
- 追加質問が機能し、停止時は補足の部分出力を保持したまま、メインの処理を継続しました。
- パネルを閉じても要求は継続し、開き直すと完了済み回答を復元しました。更新後も空の `/btw` で履歴を復元しました。
- 狭い画面の Drawer の入力、ボタン、履歴、閉じる操作は、横方向のはみ出しなく動作しました。
- 消去は確認を要求し、履歴を削除してもメインの会話記録とスナップショットを保持しました。
- MainAgent と 2 つのワーカーの間で質問を切り替えても、ラベルと履歴は分離されていました。
- ブロックするローカルモデルのテスト用実装でワーカーを実行中に保ちました。メイン入力から `/btw` を送って補足の要求を停止した後も、ワーカーは実行中の表示と独自の一時停止ボタンを保ち、補足の部分回答は永続化されました。
- ブラウザーのエラー・警告ログは空でした。

管理されたテスト用実装で、モデルの速度と独立して正確な同時実行のタイミングを検証しました。最初のワーカー確認 2 回は、タスク・回答がすでに終了していたため、有効な重複実行の期間がありませんでした。テスト用実装を修正後、繰り返しの確認に合格しました。最初の試行は合格に数えていません。

<a id="real-model-conversations"></a>
## 実際のモデルとの会話

優先候補の `grok-4.6` を、OpenAI 互換の `http://127.0.0.1:12580/tingly/openai` で検査し、2.82 秒で HTTP 200、モデル `grok-4.6`、`READY` が返りました。利用可能だったため、Tingly の `glm` も Zhipu の `glm-5.3` フォールバックも使用・検証していません。

| シナリオ | 実際の結果 |
| --- | --- |
| メイン実行中に資産、目標、マーカーを質問 | `redhaze.top`、ホームページの読み取り・要約という目標、`BTW-REAL-0910` を返し、16.97 秒で完了 |
| ホームページの読み取り後にツールの証拠を質問 | 正しい WebFetch 200、curl 301 → 302 → 200 のリダイレクト、タイトル。7.24 秒 |
| 補足の要求に Bash によるファイル作成を指示 | 拒否し、対象ファイルは未作成。7.74 秒 |
| 補足の要求完了後もメインコンテキストを保持 | メイン会話記録の SHA-256 とアクティビティは未変更。補足のツール実行は 0 回 |
| 実際に Go を停止・再起動した後に追加質問 | 以前の補足記録 3 件を保持。メインエージェントを再実行せず、保存済みスナップショットから資産、マーカー、タイトルを回答 |
| 非ストリーミング Grok の新しい会話 | 正しい資産と `ATOMIC-0910`。返却・保存された使用量: input 11734、output 138、cache_read 11520 |

メインの会話は WebFetch と Bash/curl で公開ホームページを読み、最終的に `https://id.redhaze.top/home` に到達しました。タイトルは「RedHaze Technology RedHaze Group · Global integrated group portal」（翻訳）です。Bash はローカルの一時レスポンスファイルを保存し、リモートへの書き込みはありませんでした。補足の質問のツール未実行とは別に、この点を検証しました。

メイン会話記録のチェックサム: `e7e61f135a4a120954b539f357e8c4205d7d5cd7460dcaf3dc0fd066463e1d00`。

使用量の制限: Tingly Grok のストリーミング応答は使用量を返しませんでした。`stream_options.include_usage=true` を直接試すと、HTTP 200、データフレーム 12、使用量フレーム 0 でした。ストリーミングの 0 は、課金なしではなく、エンドポイントから使用量が提供されなかったことを意味します。非ストリーミングの使用量と、テスト用実装の失敗・中止時の使用量は正しく永続化されました。

<a id="qwen-review"></a>
## Qwen のレビュー

OpenAI 互換の `https://dashscope.aliyuncs.com/compatible-mode/v1` にあるレビュー用モデル `qwen-flash` は HTTP 200 を返しました。最初の 3 件の実際の補足会話、メインのツールの証拠、実装上のアサーションを受け取り、`verdict: accept`、`concerns: []` を返しました。回答が資産、マーカー、ページの証拠と整合し、ツールの拒否も制約に整合すると判断しました。使用量: prompt 6625、completion 312、total 6937。

このレビューは、その後の再起動と非ストリーミングのテストを含みません。Qwen の「書き込みなし」という表現は広すぎました。上記のとおり、メインセッションの curl はローカルのレスポンスファイルを作成しています。同時実行、補足のツール実行 0、会話記録の隔離を立証するのは実装上のアサーションであり、モデルのレビューは回答品質の評価を補助するだけです。
