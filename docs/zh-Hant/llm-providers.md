[English](../llm-providers.md) · [한국어](../ko/llm-providers.md) · [简体中文](../zh-Hans/llm-providers.md) · 繁體中文 · [日本語](../ja/llm-providers.md) · [Español](../es/llm-providers.md)

<a id="llm-provider-templates"></a>

# LLM 供應商範本

各語言版本請見本頁上方導覽。

ScopeWeaver 支援 **Anthropic**、**OpenAI Chat Completions**、**OpenAI Responses**、原生 **Ollama** 和選用的 **OpenCode SDK 橋接器**。原有四種格式繼續保留自訂基底 URL、代理、工作階段標頭、重試、速率與並行設定；雲端、閘道與自行代管的 OpenAI 相容伺服器維持原有行為。SDK 使用獨立憑證及下文的限制，不取代 API／OpenCode Go 路徑。

SDK 範本也不發送請求；模型探索必須明確點選**讀取模型**。

範本只填寫**新設定檔表單**，不儲存憑證、不啟用設定檔、不修改既有設定檔。Ollama 範本亦從所選伺服器讀取已安裝模型；雲端範本不發出請求。套用後格式、基底 URL 與模型均可編輯。

<a id="zai-glm-53"></a>

## Z.ai GLM-5.3

![GLM-5.3 新設定檔範本](../../screenshots/en/llm-templates.png)

預覽未輸入 API 金鑰的範本，並非實際供應商連線。

開啟「LLM → 新增 → 設定範本」。API 計費選「General API（建議）」；只有取得 Z.ai 對 ScopeWeaver 的授權後，才可使用「Coding Plan（參考）」。

| 欄位 | General API | Coding Plan 參考 |
| --- | --- | --- |
| 格式 | OpenAI Chat Completions | OpenAI Chat Completions |
| 基底 URL | `https://api.z.ai/api/paas/v4` | `https://api.z.ai/api/coding/paas/v4` |
| 模型 | `glm-5.3` | `glm-5.3` |
| 思考 | `enabled` | `enabled` |
| 推理強度 | `low` | `low` |
| 上下文視窗（K） | `0`（預設 200K） | `0`（預設 200K） |
| 輸出上限 | `16384` | `16384` |

兩個端點與計費分開：General API 用量不屬於 Coding Plan 訂閱額度。詳見 [Z.ai 連線設定](https://zcode.z.ai/en/docs/configuration)。測試前確認帳號模型權限與計費。

GLM-5.3 接受文字，上下文為 1M token，且必須保持推理啟用。支援 `low`、`high`、`max`；範本採 `low`，在相同發現的小任務[實測](#measured-effect-glm-proxy)中，用量遠低於 `max` 的一半。較難任務可提高強度。模型欄位仍可編輯，選其他模型時請檢查推理與上下文設定。[官方 GLM-5.3 指南](https://docs.z.ai/guides/llm/glm-5.3)。

**Coding Plan 支援邊界：** Z.ai 僅允許官方支援工具使用 Coding Plan，ScopeWeaver 不在清單。端點範本不代表授權或支援；使用前須取得 Z.ai 授權。詳見[工具整合](https://docs.z.ai/devpack/tool/others)與[使用政策](https://docs.z.ai/devpack/usage-policy)。ScopeWeaver 不冒充支援工具，也不自動切換計費端點。

輸入設定檔名稱與自己的 API 金鑰。套用範本保留已輸入名稱、金鑰、代理、工作階段標頭與其他無關偏好。準備完成後儲存；新設定檔須另行啟用。既有「測試連線」與「載入模型」只有點選後才發出真實請求；此變更未進行實際供應商測試。

已於 **2026-10-03** 對照官方文件驗證。供應商可用性、帳號資格與政策可能改變。

<a id="opencode-sdk-bridge"></a>

## OpenCode SDK 橋接器

`opencode-sdk` 是獨立的可選設定格式，不是 OpenCode 的內建 HTTP API。ScopeWeaver 的 Norma 運行時繼續管理對話、工具、審批、範圍、重試與壓縮；橋接器只完成一次模型步驟並收集宿主工具請求。

| `SCOPEWEAVER_OPENCODE_RUNTIME` | 固定的運行時 | 參數與歷史邊界 |
| --- | --- | --- |
| `native`（預設） | 官方 CLI、客戶端、SDK **2.1.4** | 保留響應和宿主歷史中的原始工具參數 JSON，包括 64 位整數；拒絕原始參數之外不安全的數值。 |
| `legacy` | SDK、CLI **1.19.4** | 只支持結構有效、安全可表示的 JSON；整數絕對值不超過 **9,007,199,254,740,991**。空白及等價 HTML／Unicode 轉義可規範化，不保留原始位元組；拒絕不安全整數和不透明／帶簽名思考歷史。需要原始參數保真時用 `native`。 |

原生模式通過實際固定 CLI 的 `serve --stdio`、`run --server --session` 及官方客戶端管理工作階段、事件和中斷。每次調用擁有私有臨時 home／設定／資料路徑和資料庫。保留真實內建工具定義及原始實現；原生權限為 ASK，不使用 `--auto`，公開 `execute.before` 鈎子在執行前阻止所有非宿主工具。只有收集的宿主請求交回 Norma 執行和審批。OpenCode 生成的模型／環境／工具上下文與宿主 system、歷史共同保留。遇到無法受控的系統／MDM 設定來源時拒絕啓動，不讀取或修改它們；細節見適配器指南。

舊版自定義供應商使用顯式 OpenAI 兼容設定。**兩個運行時均不支持 Windows**，因為必需的 POSIX 進程樹清理邊界尚未實現。本機執行環境驗證涵蓋 **macOS arm64**。CI 在 **Linux x64** 執行實際執行環境測試；不宣稱已驗證其他平台。其他 SDK 版本需單獨驗證。

按[適配器啓動指南](../../adapters/opencode-sdk/README.zh-Hant.md)使用 **Bun 1.3.14**。橋接器使用自己的供應商設定，不導入用戶 OpenCode 或雲端已保存憑證。設置至少 **32 個非空白字元**的獨立 `SCOPEWEAVER_OPENCODE_TOKEN`，在 **LLM → 新增 → 設定模板 → OpenCode SDK 橋接器**中填入；不要粘貼 OpenCode Go／Console API 密鑰。

```sh
cd adapters/opencode-sdk
bun install --frozen-lockfile --ignore-scripts
export SCOPEWEAVER_OPENCODE_TOKEN="$(openssl rand -hex 32)"
bun run start
```

保留權杖供設定的**橋接權杖**字段使用。預設 `native`；啓動前設 `SCOPEWEAVER_OPENCODE_RUNTIME=legacy` 可改用舊版。監聽地址為 `127.0.0.1:4097`。通過 `SCOPEWEAVER_OPENCODE_PROVIDER_CONFIG` 指定專用 JSON 可設定顯式供應商，必須使用適配器文檔中的格式，不要指向用戶認證文件。安裝時保留可選平台依賴。

`SCOPEWEAVER_OPENCODE_TIMEOUT_MS` 為 **0–86,400,000 毫秒**，預設 **0** 由宿主取消控制。請求體上限 **16 MiB**；超限明確失敗。其餘輸出與清理限額見適配器指南。

| 字段 | 值 |
| --- | --- |
| 格式 | `opencode-sdk` |
| 橋接 URL | `http://127.0.0.1:4097` |
| 模型 | 點擊**讀取模型**選擇或手動輸入 `providerID/modelID` |
| 橋接權杖 | 橋接器的 `SCOPEWEAVER_OPENCODE_TOKEN`，不是雲端 API 密鑰 |
| 流式傳輸 | 開啓，使用所選固定運行時 |
| 思考／推理強度 | 模板省略，僅設置模型支持的值 |
| 輸出上限 | `0` 使用宿主調用路徑的現有預設值；橋接器按原生模型聲明的較小輸出上限限制請求 |
| 代理／自定義工作階段標頭 | 不支持；模板清除兩者 |

模板清除未保存的雲端密鑰，模型留空，**不發現、不推理、不保存、不啓用**。填入權杖後明確點擊**讀取模型**，可不執行推理而查看目錄；準備好後保存並單獨啓用。**測試連接**只在點擊時發起模型請求，元資料發現不代表推理訪問成功。

**先前免費模型驗證 — 2026-10-25 KST：** 當目錄列出時，可從**讀取模型**中選擇 `opencode/deepseek-v4.1-flash-free`。此前未登錄的 macOS arm64 運行通過真正的原生 CLI 2.1.4 完成 UI 連接測試、手動 ASK 批准後的宿主 Bash 寫入、精確的 `FREE_OK` 加換行文件內容及最終助手答案。共 **3 次推理 POST**，關閉重試，沒有付費回退，也未觀察到內建工具執行。這只證明當時一個小型隔離流程；不代表本次重跑、永久可用或無限免費。當前模型、免費資格及限額仍需確認。舊版托管免費訪問尚未驗證。

編輯時，空權杖只在同一 SDK 地址保留已保存值。更改 URL 會清除輸入的權杖；更改地址或格式後留空不會復用已保存憑證。`localhost`、`127.0.0.1`、`::1` 是不同憑證作用域。模型 ID 與輸出限制以當前元資料為準。思考設置依賴模型／供應商支持，不支持時明確失敗。

**網絡與賬戶限制：** 橋接器必須和 ScopeWeaver 位於同一網絡命名空間；容器中的 localhost 指容器本身。下游仍可能是雲服務，所以**僅本地 LLM** 模式拒絕 SDK 設定。橋接器不支持代理；下游網絡在橋接器側設定。免費模型 ID 不證明訂閱權限、持續提供或無限用量；確認服務條款和當前限額。橋接器不選擇付費回退、不導入賬戶；宿主已設定的故障轉移策略另行管理。現有 API／OpenCode Go 設定保持獨立。

**降級準備：** 資料庫設定格式的 `CHECK` 約束增加 `opencode-sdk`。仍有 SDK 設定時舊二進制不能啓動。降級前備份資料庫，在當前版本將 SDK 設定轉換為舊支持格式或移除。先前驗證使用臨時資料庫，沒有遷移生產資料庫。

<a id="opencode-go-opencode-subscription-gateway"></a>

## OpenCode Go（opencode 訂閱閘道）

開啟「LLM → 新增 → 設定範本」，選擇「OpenCode Go · glm-5.3-flash」。

| 欄位 | 值 |
| --- | --- |
| 格式 | OpenAI Chat Completions |
| 基底 URL | `https://opencode.ai/zen/go/v1` |
| 模型 | `glm-5.3-flash` |
| 思考 | 省略（模型預設） |
| 推理強度 | `max` |
| 上下文視窗（K） | `0`（預設 200K） |
| 輸出上限 | `16384`，以 `max_tokens` 傳送 |
| 工作階段標頭 | `x-opencode-session`（閘道必需；範本自動填入） |

閘道相容 OpenAI：opencode 1.18.5 的 `opencode-go` 定義以 `@ai-sdk/openai-compatible` 連至 `https://opencode.ai/zen/go/v1`，環境金鑰為 `OPENCODE_API_KEY`（2026-10-20 檢查）。GLM 類模型接受 `low`、`high`、`max`；本範本依操作人員要求刻意填 `max`，預期推理 token 用量最高。「載入模型」讀取閘道模型清單；模型欄仍可輸入 `qwen3.7-plus` 等 ID。每個請求必須帶 `x-opencode-session`，否則閘道回 `400 MissingSessionID`。範本填入此標頭名稱，ScopeWeaver 以自己的工作階段 ID 作為值。

**憑證與條款：** 金鑰屬於 opencode 訂閱，opencode 存於自身驗證檔；ScopeWeaver 絕不讀取。請將金鑰貼入設定檔或設定 `OPENCODE_API_KEY`。金鑰受 opencode 訂閱條款約束；儲存前確認允許從其他工具使用。與 Z.ai Coding Plan 一樣，此範本不構成授權。

環境設定：



```bash
SCOPEWEAVER_LLM_PROVIDER=opencode-go
SCOPEWEAVER_LLM_MODEL=glm-5.3-flash
# Key: OPENCODE_API_KEY is used when no generic key is set; SCOPEWEAVER_LLM_API_KEY overrides.
```



`SCOPEWEAVER_LLM_PROVIDER=opencode-go` 解析為 OpenAI Chat Completions 傳輸格式；未設基底 URL 與模型時採上表預設值。

<a id="controlling-token-use"></a>

## 控制 token 用量

每個代理程式回合都重送系統提示詞、工具結構與目前完整對話。因此輸入逐回合增加，直到壓縮縮短歷史；供應商提示詞快取決定多少重複輸入採較低快取費率。本節回應雲端設定檔的 [issue #24](https://github.com/cskwork/scopeweaver/issues/24)。原生 Ollama 另有[執行時保護](#runtime-safeguards)。

**證據狀態：** 測量使用 Coding Plan 的 Z.ai GLM-5.3 作為 DeepSeek 代理。GLM 的 tokenizer、快取計量與價格不同，因此 **DeepSeek 數字未驗證**。DeepSeek、Claude、OpenAI、Qwen、Kimi、OpenRouter、OpenCode Go 支援以官方回應格式建立的測試資料驗證，並非實際呼叫。下文 Claude 5.5 請求與重播亦同，未使用 Claude API 金鑰。

<a id="economy-templates"></a>

### 節省用量範本

在「LLM → 新增 → 設定範本」選 *(economy)* 項目。六種均將輸出上限設 `16384`、上下文視窗設 `0`（200K 壓縮預設）。與 GLM 範本一樣保留已輸入金鑰，不儲存或啟用設定檔。

| 範本 | 格式 | 基底 URL | 模型 | 思考 | 強度 | 輸出欄位 |
| --- | --- | --- | --- | --- | --- | --- |
| DeepSeek | OpenAI Chat Completions | `https://api.deepseek.com` | `deepseek-flash` | 省略（DeepSeek 預設開啟） | `low` | `max_tokens` |
| Anthropic Claude | Anthropic | `https://api.anthropic.com` | `claude-haiku-5-5` | `disabled` | `low` | `max_tokens` |
| OpenAI | OpenAI Responses | `https://api.openai.com/v1` | `gpt-6-luna` | 省略 | `low` | `max_output_tokens` |
| Alibaba Qwen（國際） | OpenAI Chat Completions | `https://dashscope-intl.aliyuncs.com/compatible-mode/v1` | `qwen3.8-flash` | 省略 | `low` | `max_completion_tokens` |
| Moonshot Kimi | OpenAI Chat Completions | `https://api.moonshot.ai/v1` | `kimi-k3` | 省略 | `low` | `max_completion_tokens` |
| OpenRouter | OpenAI Chat Completions | `https://openrouter.ai/api/v1` | `deepseek/deepseek-v4.1-flash` | 省略 | `low` | `max_tokens` |

- 上列供應商的 **`low` 強度皆為未實測的節省預設**。結果截斷或過淺時，先提高至供應商預設附近（DeepSeek `high`、GPT-6 Luna `medium`、Qwen `xhigh`、Kimi K3 `max`），再提高輸出上限。
- Claude：Haiku 範本關閉思考以節省輸出 token。Haiku 僅在 `low`、`medium`、`high` 接受停用思考；Sonnet、Opus 5.5 拒絕，因此無節省範本，請選 **Omit** 或 **Adaptive**，詳見 [Claude 5.5 思考](#claude-55-thinking)。
- OpenAI：GPT-6 Luna 在 Chat Completions 僅於 `none` 強度接受工具，因此範本使用 Responses。
- OpenRouter：模型 ID 與 `reasoning_effort` 支援已於 2026-10-16 查公開模型清單。Claude 需 Chat Completions 無法傳送的快取標記，因此範本選自動快取前綴的 DeepSeek Flash。
- Z.ai GLM-5.3：General 與 Coding Plan 範本現填 `low`、輸出 `16384`、上下文 `0`，即下方實測節省設定。舊版為 `max`、上下文 `1000`、輸出 `0`。GLM-5.3 必須保持思考啟用。

<a id="claude-55-thinking"></a>

### Claude 5.5 思考

規則僅適用官方主機（`https://api.anthropic.com` 或空白基底 URL）上 `claude-…-5-5` 形式的模型 ID，含日期快照 `claude-…-5-5-YYYYMMDD`。

- 思考留空（**Omit**）現傳送 `thinking: {type: adaptive, display: summarized}`；新增 **Adaptive** 同樣傳送此值；**On**（`enabled`）亦加 `display: summarized`。
- 工具迴圈重播工具呼叫及其思考區塊，含空文字及 `redacted_thinking`。各供應商在記憶體保留最近 128 次工具呼叫、最多 4 MiB；含摘要文字的區塊亦儲存至歷史，但工具迴圈中途重啟後，空白／遮蔽區塊無法還原。
- 摘要與核准裁決不再傳 `temperature`。Sonnet、Opus，以及 `xhigh`／`max` 的 Haiku 拒絕 `disabled`，因此這些呼叫移除停用設定，改用設定檔思考值。
- 設定檔 **Off**（`disabled`）在代理程式回合仍送 `disabled`；Sonnet／Opus 5.5 應選 **Omit** 或 **Adaptive**。
- 第三方 Anthropic 相容端點（DeepSeek、Kimi、Z.ai）及其他模型 ID 不變，空白思考仍省略。
- **僅以請求與串流測試資料驗證**；真實 Claude 行為與計費未驗證。

<a id="measured-effect-glm-proxy"></a>

### 實測影響（GLM 代理）

一個小任務：本機測試目標有三條路由與一個植入的未驗證秘密。每種設定檔執行 3 次，各用全新資料庫，並等待報告撰寫對話完成。點數依 Z.ai 公布的 GLM-5.3 倍率計算：（未快取輸入 × 6.9 + 快取輸入 × 1.7 + 輸出 × 24）/ 10,000，並非讀取帳單。

| 設定檔 | 輸入 + 輸出 token 中位數 | 點數中位數 | 找到植入問題 |
| --- | --- | --- | --- |
| 基準：舊 Z.ai Coding 範本（`max`、上下文 `1000`、輸出 `0`） | 209,233（172,576–216,452） | 75.1（64.3–88.6） | 3/3 |
| 節省：`low`、輸出 `16384`、上下文 `0` | 96,023（95,060–105,023） | 26.6（25.0–27.4） | 3/3，皆由報告撰寫代理程式完成報告 |

- 相較基準，節省設定中位數減少 54% 輸入 + 輸出 token 與 65% 點數；每次節省執行皆低於最便宜基準。最大單次呼叫輸入 13,287、輸出 3,829 token，未碰到 200K 視窗或輸出上限，因此此處節省來自 `low` 強度。輸出與視窗對大型任務的限制效果未測量。最大／最小執行比為基準 1.25×、節省 1.10×。
- 各工作類別點數：基準 worker 29.6%、planner 29.3%、report writer 29.1%、goal setup 12.0%；節省為 report writer 38.8%、planner 34.1%、worker 20.2%、goal setup 6.9%。未發生壓縮，無工作階段摘要呼叫；裁決與網頁搜尋關閉。
- 輸入快取比例：基準 85.2%、節省 92.3%。含推理的輸出佔點數基準 39.3%、節省 23.4%；執行當時尚未分開記錄推理 token。

GLM-5.3 一律思考，因此本測量不涵蓋關閉思考的影響。

真實橋接檢查（Claude 經 `claude -p`、OpenAI 經 `codex exec`）僅透過本機橋接測模型行為，不驗證 API 傳輸格式或計費。

`codex exec` 橋接以 OpenAI 範本（gpt-6-luna、Responses）完成一個完整任務：工具迴圈完成、找到植入問題並由報告撰寫代理程式撰文，後續側問亦成功回答，未標記中斷。`llm_usage` 含推理與快取讀取 token，以及 `btw` 側問列。另兩次因橋接緩慢未在測試時間內完成；這是橋接速度限制，不是程式結果。`claude -p` 無法驅動工具協定，因此放棄。Claude／OpenAI 請求與用量處理由測試資料證明。

<a id="recommended-settings"></a>

### 建議設定

1. 從供應商節省範本開始；任務無需更多歷史時保持上下文 `0`。
2. 保持正數輸出上限；`0` 不會省略欄位，詳見下文。
3. 保持低強度，結果變差才提高。
4. 無需時關閉核准裁決、網頁搜尋與 noa 上下文壓縮，三者預設皆關。
5. 小任務可考慮少於預設 3 個工作代理程式；效果未測量。
6. 無需每個發現的詳細報告時，關閉 **Report writer** 的觸發器（**System → Agent**）。
7. 首次任務後檢查各工作類別用量，見[檢查 token 去向](#checking-where-tokens-go)。

<a id="context-window-output-limit-thinking-and-effort"></a>

### 上下文視窗、輸出上限、思考與強度

| 設定 | 雲端設定檔行為 | 指引 |
| --- | --- | --- |
| 上下文視窗（K） | 僅本機壓縮預算，不送供應商。`0` = 200K；值限制 32–1000。 | 越大越晚壓縮，每次攜帶更多歷史。200K 時約 115K token 開始清除舊工具結果，約 179K 摘要歷史。 |
| 輸出上限 | 每次回覆上限。**`0` 不省略**：代理回合送 32768，直接 Anthropic 呼叫送 8192；原生 Ollama 採自身預設。 | 上限只截停長回覆，不使回覆自動變短；太低會截斷。範本使用 16384。 |
| 思考 | `thinking.type`；空白省略，但官方 Claude 5.5 空白送 `adaptive`。推理以輸出計費。 | DeepSeek 預設思考並支援 `disabled`，工具呼叫亦重送舊推理作輸入。GLM-5.3 一律思考。Claude 範本關閉思考。 |
| 推理強度 | `low` 至 `max`；空白省略。 | 範本採 `low`，僅 GLM-5.3 實測；其他供應商為未測節省預設。支援值依供應商而異。 |

<a id="context-optimization"></a>

### 上下文最佳化

- **對話壓縮**（內建、開啟）：接近視窗時摘要舊訊息，保留最新 8 則。摘要多一次呼叫，輸出受上限限制、最多 20000（上限 `0` 時亦為 20000）。
- **舊工具輸出清除**：超過視窗 60% 時，Read、Bash、Grep、Glob、WebFetch、WebSearch、Edit、Write 中超出最近五筆的結果會被替換；圖、證據、自訂工具結果保留至下次摘要。
- **工具輸出上限**：各文字區塊截至 30,000 位元組；完整輸出存磁碟並提供指標。
- **圖壓縮**：探索圖不活躍部分折成摘要，每個合格群組一次呼叫，輸出上限 1500。
- **noa 上下文壓縮**（系統設定、實驗、預設關閉）取代內建壓縮。
- 72% 觸發與 16/32 KiB 輸出上限僅適用原生 Ollama。

**提示詞快取：**

- Claude：planner、worker、main-agent、chat（含 report writer）現在一律標記系統提示詞與其前工具結構可快取。其他含系統提示詞但無邊界的 Anthropic 請求（目標設定、對話／圖摘要、側問準備、核准裁決）會標記完整系統提示詞。低於模型最小可快取長度時，即使標記也不快取。各不同系統提示詞首次寫入以輸入價 1.25× 計費；有效期內後續呼叫依快取價讀取。Worker 提示詞依意圖不同，每個意圖寫一次。標記位置經測試驗證；**真實節省未驗證**，因無 Claude 金鑰。
- DeepSeek、Kimi、Z.ai 自動快取前綴；DeepSeek／Kimi 文件說明 Anthropic 相容端點忽略區塊標記。OpenAI、Qwen、OpenRouter（DeepSeek Flash）也自動快取相同前綴。
- 測試確認三種雲端格式中，同一工作階段連續工具回合保持相同系統提示詞與工具結構，僅附加歷史。壓縮、收尾、新解鎖延後工具會變更前綴。自訂提示詞範本若用 `{{.Now}}`，每次執行都改變系統提示詞。

<a id="workers-retries-and-task-settings"></a>

### Worker、重試與任務設定

- **Worker concurrency · Worker Agent count**（系統設定，每任務預設 3）：各代理程式領不同意圖，執行自身工作階段並攜帶完整系統提示詞與工具結構。
- **Max concurrent LLM requests**（預設 `0` 無上限）與設定檔 **Max concurrent requests** 只調節呼叫速度，不減少每次 token。
- 重試重送完整請求。預設：**Connection retries** 3、**Empty-response retries** 2（僅 OpenAI 格式）、**Same-provider safe-window retries** 2（任何輸出前）、模型錯誤後意圖重跑 2 次、僅推理回覆後最多續接 2 次。在 LLM 頁面、各設定檔或全域政策設 `-1` 可停用一層。失敗是否計費取決供應商。
- 回合／時間預算用盡或任務收束時，收尾預設最多額外 10 回合。
- 供應商池容錯（預設關閉）會在另一設定檔重播完整請求。

<a id="known-causes-of-extra-calls"></a>

### 額外呼叫的已知原因

- 長工作階段每回合重送全部歷史，後期成本較高。
- Planner 每次喚醒（啟動、意圖完成、新發現、閒置心跳）建立新工作階段，各含多回合。
- 目標分解在規劃前執行，最多 8 回合。
- **Report writer** 為預先建立的自訂代理程式，每次 `report_finding` 皆有啟用觸發器。各發現啟動獨立對話，最多 5 個並行。
- **DeepSeek 網頁搜尋**：每次搜尋為獨立 DeepSeek 呼叫，使用目前設定檔金鑰，輸出上限 1024、每次呼叫最多 3 次搜尋。代理搜尋與設定頁搜尋測試皆記入 `llm_usage`，類別 `web_search`，使用目前設定檔名稱與模型；不在 LLM 錄製中。DeepSeek 未說明搜尋回應用量格式，因此讀取 Anthropic 式欄位。**無用量框時記 0 token：呼叫有計數，token 未知。** 請到 DeepSeek 控制台查計費 token。
- 核准裁決預設關閉，無匹配規則的每次工具操作一次呼叫。側問僅依要求執行，回答前最多 12 次摘要。
- **Test connection** 真實呼叫輸出上限 8192，不記錄。
- 已修正 Anthropic 格式串流在 `llm_usage` 高估輸出：累計 10、20 以前誤計 30，現為 20。只影響 ScopeWeaver 數字，不影響供應商帳單。
- 已修正內部工作階段與對話記錄總數同樣 10 + 20 誤計 30，現為 20。這不是 `llm_usage`；高估只可能提早壓縮。
- 已修正 OpenAI Responses 側問因缺停止事件被誤報中斷。`response.completed` 現正常結束；`response.incomplete` 仍標記中斷。

<a id="checking-where-tokens-go"></a>

### 檢查 token 去向

- **Tasks → task → Overview → LLM token usage (by model)**：由常啟 `llm_usage` 表顯示各模型呼叫、輸入、輸出、**Reasoning (in output)**、快取讀取。僅顯示標記該任務的列；report writer 列無任務。DeepSeek 搜尋列在目前設定檔模型下。
- **Dashboard** 與 **Interception rules → Approval token usage** 亦顯示 **Reasoning (in output)**。
- **LLM recordings**：每次呼叫一列，含工作類別、token、延遲、原始請求／回應。預設關閉，任務前開啟 **Recording**，會儲存完整提示詞與回應。
- 對 `llm_usage` 查某時間窗 SQL；類別位於 `worker` 欄：`planner`、`worker`、`goals`、`compactor`、`judge`、側問 `btw`、`mainagent`、DeepSeek 搜尋 `web_search`；report writer、retest、自訂代理對話為空。對話摘要與收尾沿用代理程式類別。



```sql
SELECT COALESCE(worker, '(none)') AS lane, model, COUNT(*) AS calls,
       SUM(input_tokens) AS input, SUM(cache_read) AS cache_read,
       SUM(cache_write) AS cache_write, SUM(output_tokens) AS output,
       SUM(reasoning_tokens) AS reasoning
FROM llm_usage
WHERE ts >= '2026-10-16 09:00' AND ts < '2026-10-16 10:00'  -- your task's time range
GROUP BY 1, 2
ORDER BY SUM(input_tokens) + SUM(output_tokens) DESC;
```



`cache_read`、`cache_write` 已包含於 `input`，`reasoning` 已包含於 `output`，不可再次相加。

快取讀取來自 `prompt_tokens_details.cached_tokens`、DeepSeek 頂層 `prompt_cache_hit_tokens`、Kimi 舊版頂層 `cached_tokens`。OpenAI Chat、OpenAI Responses、Kimi K3、OpenRouter 記 `cache_write_tokens`，Qwen 記 `cache_creation_input_tokens`，亦記 Claude 快取寫入。欄位對應僅以測試資料驗證。

<a id="accounting-gaps-and-follow-ups"></a>

### 計量缺口與後續

- 供應商無獨立推理數字時記 0，不代表未推理。
- DeepSeek 搜尋用量欄位未文件化，見[額外呼叫原因](#known-causes-of-extra-calls)。
- Gemini 尚無範本，需升級 Norma SDK v0.4.4（後續）。
- Claude 歷史快取（最後訊息標記）需 SDK 支援（後續）。
- OpenRouter 部分模型僅接受統一 `reasoning.effort`，可能忽略範本 `reasoning_effort`；SDK 目前僅能送後者（後續）。

<a id="native-ollama-local-inference"></a>

## 原生 Ollama（本機推論）

ScopeWeaver 透過原生 `/api/chat` 與 Ollama 通訊。OpenAI 相容端點無法設定上下文大小（[Ollama OpenAI 相容性](https://docs.ollama.com/api/openai-compatibility)），因此指向 `/v1` 的設定檔無法保證 `num_ctx`。原生格式每次請求皆傳送此值。以 `/v1`、`/api`、`/api/chat` 結尾的基底 URL 會正規化至伺服器根路徑。

Ollama 文件的預設上下文說明不一致：[FAQ](https://docs.ollama.com/faq) 為 4,096 token，[上下文長度頁](https://docs.ollama.com/context-length) 則依 VRAM 決定（低於 24 GiB 約 4K，24–48 GiB 為 32K，更高為 256K），因此 ScopeWeaver 不依賴伺服器預設。

<a id="templates"></a>

### 範本

開啟「LLM → 新增 → 設定範本 → 本機推論（Ollama）」。

| 欄位 | 同一台電腦 | Docker 存取主機 | 內附 Compose 服務 |
| --- | --- | --- | --- |
| 格式 | Ollama（原生 /api/chat） | Ollama（原生 /api/chat） | Ollama（原生 /api/chat） |
| 基底 URL | `http://127.0.0.1:11434` | `http://host.docker.internal:11434` | `http://ollama:11434` |
| 模型 | 此伺服器已安裝模型 | 此伺服器已安裝模型 | 此伺服器已安裝模型 |
| API 金鑰 | 空白 | 空白 | 空白 |
| 上下文視窗（K） | `32`（`num_ctx` 32,768） | `32` | `32` |
| 最大並行請求 | `1` | `1` | `1` |
| Keep-alive | `30m` | `30m` | `30m` |
| 輸出上限 | `4096` | `4096` | `4096` |
| 思考／強度 | 省略（模型預設） | 省略 | 省略 |

選取 Ollama 範本會從端點讀 `/api/tags` 與模型中繼資料，再選已安裝模型，優先宣告 `tools` 者。新設定檔選 Ollama 格式亦同；空白端點或未修改的雲端範本端點改用主機預設，自訂端點保持不變。探索期間模型留空；失敗或無模型時保持空白並顯示錯誤。請檢查端點、選用權杖與代理，再按「載入模型」或手填標籤。編輯模型／連線欄位會取消待選，延遲回應不能覆寫。開啟既有設定檔保留已存模型。選取不會下載／執行模型、儲存或啟用設定檔。「測試連線」僅點選才執行。

**API 金鑰行為：** Ollama 範本與切換至 Ollama 格式會在探索前清除未儲存金鑰，保留名稱、代理、工作階段標頭及其他偏好，但**清除所有未儲存 API 金鑰**，避免將雲端憑證送往本機。GLM 範本仍保留輸入金鑰。本機 Ollama 應留空；僅前置反向代理需要時輸入權杖，以 `Authorization: Bearer` 傳送。免金鑰 Ollama 可測試、儲存、啟用與容錯；卡片顯示「不需要金鑰」。編輯已存原生 Ollama 設定檔時，空白金鑰保留既有權杖；將雲端改為 Ollama 且欄位空白則刪除舊雲端金鑰。

**其他格式金鑰：** Anthropic／OpenAI 需要金鑰，除非基底 URL 明確為本機，即主機只解析至回送或私有位址。因此 vLLM、LM Studio 等免金鑰本機相容伺服器可留空；ScopeWeaver 送內部非秘密預留值，防止 SDK 繼承環境 `OPENAI_API_KEY`／`ANTHROPIC_API_KEY`。非本機且無金鑰會拒絕，錯誤為 *llm: API key required for non-local compatible endpoint*。

<a id="context-output-and-residency-controls"></a>

### 上下文、輸出與駐留控制

| 設定 | Ollama 行為 | 雲端格式 |
| --- | --- | --- |
| 上下文視窗（K） | 二進位 K，以 `options.num_ctx` 傳送（32 = 32,768）；`0` 採 32，低於 16 採 16。亦設定壓縮預算。 | 不變，僅壓縮預算；`0` = 200K |
| 輸出上限 | `options.num_predict`；`0` 採 4,096 | `0` 在代理回合送 32768，直接 Anthropic 送 8192 |
| Keep-alive | `keep_alive`；空白採 `30m`（Ollama 本身預設 5 分鐘） | 不傳送 |
| 最大並行請求 | `0` 採 1。每個 Ollama 伺服器跨設定檔共用准入閘門，採最低已設上限 | `0` 繼承「系統設定 → 最大並行 LLM 請求」 |

排隊中的核准裁決、主要代理程式、側問先於背景工作取得下一名額；各組 FIFO，不中斷進行中推論。診斷與預熱仍為背景請求。

只有伺服器 `OLLAMA_NUM_PARALLEL` 大於 1 才提高並行數。記憶體用量隨 `OLLAMA_NUM_PARALLEL × context length` 增長（[FAQ](https://docs.ollama.com/faq)）。

<a id="runtime-safeguards"></a>

### 執行時保護

- **預熱**：原生設定檔首次推論前，先呼叫 `/api/show`，拒絕未宣告 `tools` 的模型，再以空 `/api/chat`、設定檔 `num_ctx` 與 `keep_alive` 載入模型。預熱亦佔同一並行名額。
- **壓縮**：原生 Ollama 視窗 ≤64K 時，於 72% 觸發歷史壓縮，保留完整工具呼叫／結果配對。即使啟用實驗 noa，此保護仍優先，略過 noa 並警告。≤32K 時給模型的各工具輸出上限 16,384 **位元組**（16 KiB，以 SDK `len(s)` 計，不是 Unicode 字元）；>32K 至 64K 為 32,768 位元組（32 KiB）。完整輸出保留於工具結果指向的溢出檔。雲端壓縮與 noa 不變。
- **核准裁決**：明確裁決設定檔優先，否則優先原始代理／執行實際使用的原生設定檔（含綁定、池、容錯）；雲端或無來源時用目前啟用設定檔。原生裁決預設 **90 秒**、失敗動作 **ask**，已存覆寫優先；使用者核准仍於 300 秒到期並預設拒絕。原生輸出受 decision／comment JSON schema 約束並再次驗證。系統設定成功載入前，表單不得儲存。

<a id="generic-openai-compatible-servers-vllm-lm-studio"></a>

### 一般 OpenAI 相容伺服器（vLLM、LM Studio）

這些伺服器仍選既有 **OpenAI（Chat Completions）** 格式，基底 URL 指向 `/v1`。不得使用原生 Ollama 格式。

- 原生格式「載入模型」讀 `/api/tags`，HTTP 404／405 時備援 `/v1/models`，所以可能顯示一般伺服器模型清單。
- 對話**絕不**默默切換：原生設定檔一律呼叫 `/api/chat`，一般伺服器未提供。因此「找不到模型或端點」及 `ollama pull` 提示不適用；拉模型不能修復 vLLM／LM Studio。請改選 OpenAI 格式。

<a id="load-models-and-test-connection"></a>

### 載入模型與測試連線

「載入模型」讀 `/api/tags`、`/api/show`，各列顯示參數大小、量化與 `tools`／`thinking`／`vision` 標章。未宣告 `tools` 時警告：「此模型未宣告原生工具呼叫支援；ScopeWeaver 代理程式需要工具呼叫以更新資產與探索圖。」中繼資料讀取失敗顯示未知，不當作缺少能力。

「測試連線」以 `/api/show` 檢查模型，使用設定檔 `num_ctx`、`keep_alive`、`num_predict` 做單回合工具呼叫基本測試，再讀 `/api/ps`。面板顯示分配上下文是否涵蓋請求、工具是否回傳、冷啟毫秒與 token/s。缺模型時顯示伺服器上應執行的精確指令，如 `ollama pull qwen2.5-coder:32b`。ScopeWeaver 不拉取模型或建立 Modelfile。

<a id="host-and-container-routing"></a>

### 主機與容器路由

| ScopeWeaver 位置 | Ollama 位置 | 基底 URL |
| --- | --- | --- |
| 主機 | 同一主機 | `http://127.0.0.1:11434` |
| Docker Compose | 主機 | `http://host.docker.internal:11434` |
| Docker Compose | 內附 `ollama` 服務 | `http://ollama:11434` |

容器內 `127.0.0.1` 指容器本身。Compose 將 `host.docker.internal` 對應主機閘道 `host-gateway`。Ollama 預設綁定 `127.0.0.1:11434`；Linux 主機 Ollama 必須監聽容器能到達的位址（如 `OLLAMA_HOST=0.0.0.0:11434`），這會對網路公開服務。請以防火牆限制 11434 連接埠。

**選用 Compose 服務：** `docker compose --profile local-llm up -d` 啟動 `ollama`，設定 `OLLAMA_NO_CLOUD=1`、`OLLAMA_KEEP_ALIVE=30m`、`OLLAMA_NUM_PARALLEL=1`、`OLLAMA_CONTEXT_LENGTH=32768` 與 `ollama_data` 磁碟區。以 `docker compose exec ollama ollama pull qwen2.5-coder:14b` 拉取模型。未啟用 profile 時 Compose 不變。此服務不請求 GPU。**macOS Docker 無法使用 Apple GPU，僅 CPU 執行**；需要實用速度時請在 Mac 原生執行 Ollama，選主機範本。

<a id="network-policy-local_llm_only"></a>

### 網路政策（`local_llm_only`）

預設關閉，雲端供應商仍可用。於「系統設定 → 僅限本機 LLM 網路」或 `SCOPEWEAVER_LOCAL_LLM_ONLY=true` 啟用後：

- LLM HTTP（推論、模型清單、連線測試、預熱、裁決）僅允許**回送位址及明確信任 CIDR**（`local_llm_allowed_cidrs`）。私有 IPv4、IPv6 ULA、LAN、Docker 不再自動信任；僅加入自己控制的推論位址，優先 IPv4 `/32` 或 IPv6 `/128`。
- 鏈路本機、多點傳送、未指定地址即使在信任 CIDR 仍拒絕。連線開啟時檢查 DNS 解析，全部位址均須允許。拒絕所有重新導向與 LLM 設定檔代理。
- **封鎖全部後端的代理程式網頁搜尋、搜尋連線探測、更新檢查與下載。** 更新頁區分刻意政策與網路故障。本機版本、更新進度、復原仍可用；不以快取遠端發行資訊繞過政策。
- 執行時變更無需重啟，含既有工作階段。啟用前已准入下載於下次受保護網路操作／本文讀取停止並移除部分暫存；已送出流量無法收回。
- `SCOPEWEAVER_LOCAL_LLM_ALLOWED_CIDRS`（或保留的 `ARTEX_LOCAL_LLM_ALLOWED_CIDRS`）亦生效，不複製至儲存清單，也無法在介面編輯。環境 `true` 為介面無法停用的政策下限。
- 目標流量、目標工具、全域代理仍可連網。這是應用政策，不是密碼學或整台主機離線保證。

**既有嚴格 LAN／Docker 設定檔遷移：** 同機 `http://127.0.0.1:11434` 無需 CIDR；`http://192.168.1.20:11434` 需 `192.168.1.20/32`；IPv6 `fd00::20` 需 `fd00::20/128`。Docker 必須**從 ScopeWeaver 容器內**解析名稱，因主機與容器可能看到不同位址：



```bash
# Existing Compose service; read its actual name resolution and network addresses.
docker compose exec scopeweaver getent ahosts host.docker.internal
docker compose exec scopeweaver getent ahosts ollama
docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}} {{.GlobalIPv6Address}}{{end}}' "$(docker compose ps -q ollama)"
```



若映像檔缺 `getent`，以 `docker compose exec scopeweaver cat /etc/hosts` 檢查既有主機閘道，並以 `docker inspect` 檢查 Ollama 容器位址。使用 ScopeWeaver 實際到達的推論 IP，勿照抄範例或允許整段 Docker 子網。加入「其他受信任 CIDR」或環境允許清單，如 `SCOPEWEAVER_LOCAL_LLM_ALLOWED_CIDRS=192.168.1.20/32`。`host.docker.internal` 與 `ollama` 名稱均不自動授予信任。Docker 重建服務或位址變更後重查，在重試「載入模型」／「測試連線」前更新狹窄允許清單。

ScopeWeaver 看不到 Ollama 後續行為。**Ollama 可將請求轉送至雲端模型**；內附 Compose 設 `OLLAMA_NO_CLOUD=1`，但主機／遠端 Ollama 須另行停用雲端：`OLLAMA_NO_CLOUD=1` 或依 [FAQ](https://docs.ollama.com/faq) 在 `~/.ollama/server.json` 設 `disable_ollama_cloud`。

<a id="environment-and-configuration"></a>

### 環境與設定

`SCOPEWEAVER_LLM_*` 優先於 `ARTEX_LLM_*`；明確空白的現代變數亦覆寫舊值。兩者都覆寫 `config.json` 的 `llm` 區塊，檔案路徑來自 `SCOPEWEAVER_CONFIG`／`ARTEX_CONFIG`。

| 後綴 | 含義 |
| --- | --- |
| `PROVIDER` | `anthropic`、`openai`、`openai-responses`、`opencode-go`、`ollama` |
| `MODEL`, `BASE_URL`, `PROXY` | 模型、端點、選用 LLM 代理 |
| `API_KEY` | Anthropic／OpenAI 必填，除非基底明確本機；Ollama 為選用權杖 |
| `CONTEXT_WINDOW_K`, `MAX_TOKENS`, `KEEP_ALIVE`, `MAX_CONCURRENT` | 上述控制 |
| `THINKING_TYPE`, `REASONING_EFFORT`, `STREAM` | 既有推理與串流選項 |

政策變數：`SCOPEWEAVER_LOCAL_LLM_ONLY`／`ARTEX_LOCAL_LLM_ONLY`，以及逗號分隔的 `SCOPEWEAVER_LOCAL_LLM_ALLOWED_CIDRS`／`ARTEX_LOCAL_LLM_ALLOWED_CIDRS`。

啟動時 Anthropic／OpenAI 在無通用金鑰時使用 `ANTHROPIC_API_KEY`／`OPENAI_API_KEY`，含需驗證本機閘道。`opencode-go` 改用 `OPENCODE_API_KEY`，絕不使用 `OPENAI_API_KEY`；只有 `OPENCODE_API_KEY` 時會推斷此供應商。若已設供應商金鑰但刻意要免金鑰本機設定檔，明確設 `SCOPEWEAVER_LLM_API_KEY=`（或 `ARTEX_LLM_API_KEY=`），覆寫檔案憑證並阻止供應商金鑰備援。原生 Ollama 僅備援 `OLLAMA_API_KEY`，絕不採 Anthropic、OpenAI、OpenCode 金鑰。

介面以空金鑰儲存新設定檔不會匯入環境憑證。空金鑰「測試連線」若有 profile ID，可重用該已存金鑰。無 ID 時，本機 Anthropic／OpenAI 測試僅在格式、正規化基底 URL、代理全部相符於目前服務時，才重用啟用金鑰，含新草稿測試。不同本機端點、格式、代理不繼承；原生 Ollama 絕不繼承啟用的雲端金鑰。非本機測試保留既有啟用金鑰備援。



```bash
# .env for ScopeWeaver in Docker with Ollama on the host
# Replace your-installed-tool-model with a tool-capable tag shown by Load models.
SCOPEWEAVER_LLM_PROVIDER=ollama
SCOPEWEAVER_LLM_BASE_URL=http://host.docker.internal:11434
SCOPEWEAVER_LLM_MODEL=your-installed-tool-model
SCOPEWEAVER_LLM_CONTEXT_WINDOW_K=32
SCOPEWEAVER_LLM_MAX_CONCURRENT=1
SCOPEWEAVER_LLM_KEEP_ALIVE=30m
SCOPEWEAVER_LLM_MAX_TOKENS=4096
SCOPEWEAVER_LOCAL_LLM_ONLY=false   # add the actual host /32 or /128 before enabling strict mode
# SCOPEWEAVER_LOCAL_LLM_ALLOWED_CIDRS=192.168.1.20/32  # replace with the resolved inference IP
```





```json
{
  "llm": { "provider": "ollama", "base_url": "http://127.0.0.1:11434", "model": "your-installed-tool-model", "context_window_k": 32, "llm_max_concurrent": 1, "keep_alive": "30m", "max_tokens": 4096 },
  "local_llm_only": false,
  "local_llm_allowed_cidrs": []
}
```



<a id="bounded-diagnostics-and-telemetry"></a>

### 有界診斷與遙測

「測試連線」刻意進行推論探測；選範本僅探索安裝中繼資料。除工具／上下文、載入時間／速度外，原生診斷以所選 runner `/api/ps` 的 `size`、`size_vram` 顯示 GPU 記憶體百分比。總大小缺漏／為零或 VRAM 缺漏表示**未知**，不是 0%；總量正數且 VRAM 明確為零才是 0%。此比率為記憶體估算，不是精確權重切分、GPU 容量測試或駐留保證。

百分比 ≤80% 時警告 CPU 工作可能拖慢生成。其他已載模型，以及同端點但不同模型／上下文的明確裁決設定，另有記憶體／重載警告。共用模型／上下文可能減少重載，但不保證駐留。診斷不下載模型、不儲存／啟用設定檔。

接近上下文容量警告是基於回報 token 數的伺服器記錄啟發式，不是截斷偵測器。完整原始 Ollama 時序欄位僅於啟用原始錄製時儲存；token／快取計量獨立。小模型仍需任務特定驗證，下述可靠性限制仍適用。

<a id="hardware-tiers"></a>

### 硬體分級

以下依 GPU VRAM 或 Apple 統一記憶體提供工具能力模型的起點，**僅估算，非基準測試**。上下文為設定檔 **Context window (K)**，須同時符合模型上限與記憶體（權重 + 隨上下文及 `OLLAMA_NUM_PARALLEL` 增長的 KV 快取）。以「測試連線」確認所要求上下文是否獲分配。

| 記憶體 | 模型範例 | 上下文視窗（K） | 角色 |
| --- | --- | --- | --- |
| 8–12 GB | `qwen2.5-coder:7b`, `qwen3:8b` | `16`–`32` | 裁決或輔助設定檔；主要代理能力較弱 |
| 16–24 GB | `qwen2.5-coder:14b`, `qwen3:14b` | `32` | 工作站主要代理 |
| 32–48 GB | `qwen2.5-coder:32b`, `qwen3:32b` | `32` | 主要代理；僅模型上限允許時用 `64`，這些模型列 32K／40K |
| 64–128 GB+ | `llama3.3:70b` | `64`（模型列 128K） | 大記憶體主機主要代理；64K 需要此範圍較高端 |

範本從所選伺服器已安裝模型挑選，使用 32K。硬體範例不是自動模型選擇。

<a id="model-candidates"></a>

### 模型候選

僅列 [Ollama 模型庫](https://ollama.com/library) 頁宣告 `tools` 的模型（2026-10-15 檢查），下載大小為預設量化。**記憶體指引為估算**：權重加隨上下文／並行增長的 KV 快取。ScopeWeaver 未對這些模型／硬體做基準，工具呼叫品質依模型而異。

| 模型 | 下載 | 說明 |
| --- | --- | --- |
| `qwen2.5-coder:7b` | 4.7 GB | 最小實用候選，亦可作裁決 |
| `qwen2.5-coder:14b` | 9.0 GB | 工作站候選 |
| `qwen2.5-coder:32b` | 20 GB | 較大主機候選 |
| `qwen3:8b`, `qwen3:14b`, `qwen3:32b` | 5.2 / 9.3 / 20 GB | 亦宣告 `thinking` |
| `llama3.3:70b` | 43 GB | 大記憶體主機 |

Ollama 模型庫沒有 `qwen3:72b`、`llama3.2:8b`；`llama3.2` 提供 1B／3B。使用前在自己的伺服器以「載入模型」確認 `tools`。

**已觀察限制（2026-10-15）：** 宣告 `tools` 不保證傳輸層工具呼叫。分支測試中，`qwen2.5-coder:7b` 將工具形狀 JSON 當純文字，因此無工具執行；`qwen3:4b` 通過工具診斷卻無法完成長自主任務。小模型作主要代理仍未驗證；見[驗證](VERIFICATION.md#autonomous-task-reliability-is-not-validated)。

無論供應商為何，[README](../../README.zh-Hant.md) 使用限制仍適用：僅在本機隔離環境執行 ScopeWeaver。

[韓文指南](../ko/llm-providers.md)
