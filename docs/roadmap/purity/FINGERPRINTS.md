# API 纯度检测 — 指纹与检测手段增补

版本：v0.2.0
日期：2026-06-28
状态：分层落地中。精确 header/error/model/host 指纹已落到 `channels/<channel>/detector.go` 与样本回归；Claude thinking signature / thinking budget / cache_control overflow 负向探针已落地；OpenAI `openai-model`、`reasoning_tokens`、`store:false/include` 正向探针和 Token audit 失败诊断已接入。更重的行为问答指纹、在线模型签名表和真实授权样本校准仍属 P5。
关联：本文件是 `README.md`（PRD v0.1.0）的增补，所有条目按"可直接写进检测器的硬证据"组织。

## 信息来源

本文件指纹来自对以下代码库的黑盒逆向 + 业界检测手段调研：

| 来源 | 类型 | 价值 |
|------|------|------|
| `/Users/coso/Documents/dev/go/new-api` | 中转网关 | 响应头/错误体/桥接痕迹精确常量 |
| `/Users/coso/Documents/dev/go/CLIProxyAPI` | 中转网关 | CORS/路由/管理面/上游伪装头精确化 |
| `/Users/coso/Documents/dev/go/sub2api` | 聚合网关 | 未鉴权配置泄露/主动探针/协议门控文案 |
| `/Users/coso/Documents/dev/rust/codex` | **OpenAI 官方客户端** | Responses API 官方强约束（负向探针金矿） |
| `/Users/coso/Documents/dev/js/claudecode` | **Anthropic 官方客户端** | Messages API 官方强约束（负向探针金矿） |
| `dabaibian/api-model-spy` (GitHub) | 业界检测工具 | 行为指纹/模型身份主动探测 |

## 核心增量（三个全新维度）

1. **官方客户端负向探针**：官方 CLI 揭示了真实 API 会强制校验、而兼容/包装实现会放过的约束。这是判别"是否官方后端"最有力的手段，且不依赖域名。见 §1、§2。
2. **网关黑盒指纹精确化**：把 PRD 里"中等强度"的程序名信号升级为精确常量（精确 header value、精确 JSON 文案、精确路由组合），并新增大量未鉴权即可观测的强指纹。见 §3、§4、§5。
3. **行为指纹检测**：模型身份不再只靠响应 `model` 字段（可被改写），而是用主动行为探针推断真实上游模型。见 §6。

## 当前落地映射

| 维度 | 当前实现 | 仍待推进 |
|------|----------|----------|
| 官方负向/正向探针 | Claude thinking signature、thinking budget、cache_control overflow；OpenAI `store:false/include`、`openai-model`、`reasoning_tokens` | WebSocket Responses、完整行为问答指纹 |
| 网关精确指纹 | CLIProxyAPI、new-api、sub2api、Antigravity、Bedrock、OpenAI/Claude/Gemini/xAI 等 detector 子包 | 更多真实部署样本校准强弱权重 |
| 国产兼容模型 | Qwen、GLM、Doubao、MiniMax、Hunyuan、Kimi、MiMo、DeepSeek detector 与模型身份规则 | 供应商授权样本、模型签名表动态更新 |
| Token audit | OpenAI/Claude usage/cache 统计、独立超时、失败原因透传、OpenAI 参数不兼容 fallback | 真实账单侧交叉校准、更多供应商倍率样本 |

---

## 1. OpenAI / Codex 官方负向探针（来自 codex）

> 原理：codex 是官方 Responses API 客户端。下列约束是官方服务端真实行为，兼容/包装层难以全部正确复刻。检测器应主动发探针，观察目标是否表现出官方行为。

### 1.1 请求侧官方特征（构造探针时模仿官方，观察接受度）

| 特征 | 精确值 | 检测用途 |
|------|--------|----------|
| `OpenAI-Beta: responses_websockets=2026-02-06` | 带日期戳的 WebSocket Responses 协议 | 包装层基本不支持 WS Responses；探测此 beta 的处理可区分 |
| `include: ["reasoning.encrypted_content"]` | 请求加密推理跨轮复用 | ChatGPT 后端独有能力，公共 API key 路径行为不同 |
| `store: false` | ChatGPT 后端路径恒为 false（仅 Azure 为 true） | 许多兼容实现默认/要求 `store:true`，可作负向探针 |
| `tool_choice: "auto"` | 字符串字面量 | 官方请求体稳定指纹 |
| `prompt_cache_key` | = thread UUID，几乎必带 | 官方缓存键格式 |
| `wire_api=chat` 已移除 | 官方绝不打 `/v1/chat/completions` | 若目标只支持 chat completions、不支持 `/v1/responses`，必为包装/降级 |
| originator 条件发送 | originator==默认值时不重复发 header；UA 前缀必须与 originator 一致 | 难复刻的细节 |

### 1.2 响应侧官方特征（观察目标响应是否具备）

完整 SSE 事件名（缺失或顺序错误是阳性信号）：
```
response.created / response.output_item.added / response.output_item.done
response.output_text.delta / response.reasoning_summary_text.delta
response.reasoning_text.delta / response.reasoning_summary_part.added
response.metadata / response.failed / response.incomplete / response.completed
```

| 字段/事件 | 官方特征 | 包装层表现 |
|-----------|----------|------------|
| `response.metadata` 事件 | 含 `openai_chatgpt_moderation_metadata`、`openai_verification_recommendation: trusted_access_for_cyber` | 后端独有，包装层不会产出 |
| `usage.input_tokens_details.cached_tokens` | 官方缓存命中细分 | 包装层常缺 |
| `usage.output_tokens_details.reasoning_tokens` | reasoning 模型独有 | **见 §6.3：reasoning_tokens 是 o 系/思考模型的强身份指纹** |
| `response.completed.end_turn` | 顶层 bool | 包装层常缺 |
| `response.failed` error 结构 | `{type,code,message,plan_type,resets_at}`，code 含 `cyber_policy`/`invalid_prompt`/`context_length_exceeded` | 错误结构是强约束探针 |
| 响应头 `openai-model` | 实际服务模型（安全路由可能 != 请求模型） | **直接暴露真实模型，比 body 的 model 更可信** |
| 响应头 `x-reasoning-included` | 服务端已计入历史推理 token | ChatGPT 后端独有 |

### 1.3 OAuth / 端点真实性

| 项 | 官方值 |
|----|--------|
| ChatGPT 后端 base | `https://chatgpt.com/backend-api/codex` |
| API base | `https://api.openai.com/v1`，endpoint `/responses` |
| OAuth client_id | `app_EMoamEEZ73f0CkXaXp7hrann` |
| OAuth issuer | `https://auth.openai.com`，token `/oauth/token` |
| scope | 含 `api.connectors.read api.connectors.invoke offline_access` |
| JWT claim namespace | `https://api.openai.com/auth` 下含 `chatgpt_plan_type`/`chatgpt_account_id`/`chatgpt_account_is_fedramp` |

### 1.4 官方模型清单（codex 内置，context_window 均 272000）

```
gpt-5.5  gpt-5.4  gpt-5.4-mini  gpt-5.3-codex  gpt-5.2  codex-auto-review
```
reasoning effort 枚举：`none/minimal/low/medium/high/xhigh/ultra`（`ultra` 客户端改写为 `max` 发出）。
base_instructions 以 `You are Codex, a coding agent based on GPT-5...` 开头（~21KB system prompt，本身是强指纹）。

---

## 2. Anthropic / Claude 官方负向探针（来自 claudecode）

> 原理同上：claudecode 是官方 Messages API 客户端。下列是 Claude 兼容包装（Kiro/Antigravity/反代）最难伪造的官方约束。

### 2.1 thinking signature 机制（PRD 已列为重点，此处给精确约束）

| 约束 | 官方行为 | 包装层失败方式 |
|------|----------|----------------|
| thinking budget | API 强制 `max_tokens > thinking.budget_tokens`，违反返回 400 | 兼容层常不校验 → 故意发 `budget_tokens >= max_tokens`，真端 400，假端放过 |
| thinking block 回传 | thinking/redacted_thinking block **不允许带 cache_control**，否则 400 | 转译层无脑透传会触发真端 400，假端可能接受 |
| signature_delta | 流式必发 `signature_delta`，且 thinking block 初始化时 signature 字段必存在 | 转译层常丢失 signature 或把 thinking 当普通 text |
| redacted_thinking | 独立 content type | 包装层常不识别 |
| temperature | thinking 开启时 API 强制 temp=1，官方此时**省略** temperature 字段 | 假端可能仍接受自定义 temperature |

### 2.2 system 数组约束

| 约束 | 官方行为 |
|------|----------|
| system 必须是带 cache_control 的 `TextBlockParam[]` 数组 | 字符串形式可能被官方接受但官方客户端永远发数组 |
| cache_control block 数量上限 | 超量（5+ 个 cache_control block）真端返回 400；转译层常无脑透传或合并 |
| 官方 system prefix（强指纹文本） | `You are Claude Code, Anthropic's official CLI for Claude.` / `...running within the Claude Agent SDK.` |

### 2.3 请求头

```
x-app: cli                                    # 每个推理请求必带，极简单且包装层几乎不模仿
anthropic-version: 2023-06-01                 # 精确日期
User-Agent: claude-cli/{version} (...)
X-Claude-Code-Session-Id: {每 key 稳定 UUID}
anthropic-beta: <见下完整清单>
```

完整 anthropic-beta 清单（官方对特定模型发**确定的 beta 集合**，可双向探针）：
```
claude-code-20250219          interleaved-thinking-2025-05-14   context-1m-2025-08-07
context-management-2025-06-27 structured-outputs-2025-12-15     web-search-2025-03-05
advanced-tool-use-2025-11-20  tool-search-tool-2025-10-19       effort-2025-11-24
task-budgets-2026-03-13       prompt-caching-scope-2026-01-05   fast-mode-2026-02-01
redact-thinking-2026-02-12    token-efficient-tools-2026-03-28  oauth-2025-04-20
ccr-byoc-2025-07-29           advisor-tool-2026-03-01           afk-mode-2026-01-31
```
- opus/sonnet-4 必带 `claude-code-20250219` + `interleaved-thinking-2025-05-14`。
- Bedrock 把 `interleaved-thinking`/`context-1m`/`tool-search-tool` 放进 body 而非 header（路由差异指纹）。

### 2.4 usage 字段（包装层最常漏）

```jsonc
usage: {
  input_tokens, output_tokens,
  cache_creation_input_tokens, cache_read_input_tokens,
  cache_creation: { ephemeral_1h_input_tokens, ephemeral_5m_input_tokens },  // 包装层几乎都缺这个细分
  cache_deleted_input_tokens,
  server_tool_use: { web_search_requests, web_fetch_requests }
}
```
- message_start 给 input 类 token，message_delta 给 output token；message_delta 的显式 0 不应覆盖 message_start 值（官方语义约束）。
- **缺 `cache_creation` 细分对象 = 强阳性信号**（注意：原生 Anthropic usage **无** `total_tokens`，若出现 `total_tokens` 反而是网关拼接痕迹，见 §5）。

### 2.5 OAuth / 鉴权 / 端点

| 项 | 官方值 |
|----|--------|
| OAuth client_id | `9d1c250a-e61b-44d9-88ed-5944d1962f5e` |
| 订阅用户鉴权 | `Authorization: Bearer` + **必带** `anthropic-beta: oauth-2025-04-20` |
| API key 用户 | `x-api-key` |
| token endpoint | `https://platform.claude.com/v1/oauth/token` |
| count_tokens | `/v1/messages/count_tokens`；Vertex 仅接受 3 个特定 beta 否则 400 |
| 遥测端点 | `https://api.anthropic.com/api/event_logging/batch`，payload `{events:[{event_type:'ClaudeCodeInternalEvent',event_data}]}` |
| 反伪造证明头 | `x-anthropic-billing-header: cc_version=...; cch=00000`（cch 由原生 HTTP 栈用 attestation hash 覆写，包装层无法生成有效 cch） |

### 2.6 官方模型清单（claudecode 内置 canonical id）

```
claude-3-5-haiku-20241022   claude-haiku-4-5-20251001   claude-sonnet-4-6
claude-sonnet-4-5-20250929  claude-opus-4-5-20251101    claude-opus-4-6
claude-opus-4-1-20250805    claude-opus-4-20250514      claude-sonnet-4-20250514
```
每个含 firstParty/bedrock/vertex/foundry 四套别名（vertex 用 `@` 分隔日期 `claude-opus-4-1@20250805`，bedrock 用 `us.anthropic.*-v1:0`）。**注意：此清单来自被分析的 claudecode bundle 快照，未收录 `claude-fable-5`（Fable 5）**——但 Fable 5 是真实官方最新模型，只是该快照较旧。因此 `claude-fable-5` 出现在某端点的模型列表里**不能**作为"非官方"反向指纹；判别官方性仍应依赖负向探针（§2.1-2.4）与精确网关指纹，模型清单只用于发现明显的别名/降级，不可仅凭"清单里没有"就判非官方。

---

## 3. new-api / one-api 网关指纹（精确化 + 新增）

> 所有指纹外部 HTTP 可见。带 ★★★ 为最高判别力。

### 3.1 响应头（所有响应必带 = 强指纹）

| 强度 | Header | Value | 说明 |
|------|--------|-------|------|
| ★★★ | `X-New-Api-Version` | 动态版本号（默认 `v0.0.0`，env `VERSION` 可覆盖） | 全局中间件注入，等同 `X-Powered-By: new-api` |
| ★★★ | `X-Oneapi-Request-Id` | `时间戳+8随机前缀+8随机串` | key 名 one-api/new-api 家族独有；即使上游有同名也被覆盖 |
| ★★ | `Auth-Version` | 固定 `864b7076dbcd0a3c01b5520316720ebf` | 后台 `/api/*` session 鉴权成功后 |
| ★★ | `specific_channel_version` | 固定 `701e3ae1dc3f7975556d354e0675168d004891c8` | 指定渠道被拒（403）时 |
| ★ | `X-Accel-Buffering: no` + `Transfer-Encoding: chunked` 组合 | 固定 | SSE 响应；nginx 反代约定头，官方 CDN 不带 |

注意：new-api CORS **未设** `Access-Control-Expose-Headers`，浏览器 JS 读不到上述头，但 curl/抓包全部可见。

### 3.2 错误体（精确文案 = 强指纹）

核心：错误体含 `status_code` 字段（官方响应体内绝无）；`error.type` 取值 `new_api_error`/`upstream_error`（官方无此命名空间）。

| 强度 | 场景 | 精确响应 |
|------|------|----------|
| ★★★ | 中间件层错误 | `error.type == "new_api_error"` |
| ★★★ | 任意错误 | message 尾部恒附 `" (request id: <id>)"` |
| ★★ | 分组无可用渠道 | `分组 X 下模型 Y 无可用渠道（distributor）` + HTTP 503 + `code=model_not_found`（`（distributor）`后缀官方绝无） |
| ★★ | 余额不足 | `用户额度不足, 剩余额度: xxx` + 403 + `insufficient_user_quota`（官方为 429/`insufficient_quota`） |
| ★★ | 令牌异常 | 一律 `无效的令牌`/`Invalid token` + 401 |
| ★★ | 未实现端点 | `{"error":{"message":"API not implemented","type":"new_api_error","code":"api_not_implemented"}}` HTTP 501 |
| ★ | 未知 URL | `Invalid URL (POST /xxx)` + `invalid_request_error` + 404（回显 method+path） |

硬编码中文（任何 Accept-Language 都返回中文 = 强指纹）：`无法解析客户端 IP 地址`、`您的 IP 不在令牌允许访问的列表中`、`无权访问 X 分组`、`普通用户不支持指定渠道`。

### 3.3 路由 / 桥接

| 强度 | 指纹 | 说明 |
|------|------|------|
| ★★★ | OpenAI 接口返回 `id: "msg_..."`（而非 `chatcmpl-`） | 判定经 new-api 中转 Claude：用 OpenAI 协议请求 Claude 渠道时，响应 id 被 Claude 原生 `msg_xxx` 覆盖。AWS Bedrock/Vertex 的 Claude 模型同样继承 |
| ★★ | `/v1/models` 单路径按请求头三协议分流 + 响应含 `"success": true` | 带 `x-api-key`+`anthropic-version`→Claude，带 `x-goog-api-key`→Gemini，否则 OpenAI。官方各家独立域名，绝不会单路径支持三协议头；OpenAI 响应也无 `success` 字段 |
| ★★ | `GET /api/status` 未鉴权返回大 JSON | 含 `version` + `start_time` + 全站配置（OAuth client_id、`turnstile_site_key`、`quota_per_unit`、`price`）|
| ★★ | `system_fingerprint` 恒 null（Claude/Gemini 渠道） | 真 OpenAI(gpt-4o) 几乎总返回 `fp_xxx` |
| ★★ | usage 本地估算特征 | 上游不返 usage 时本地加权字符估算（非真 BPE）：`CompletionTokens += toolCount*7`、prompt padding `MessagesCount*3 + ToolsCount*8 + 3`、image 520/audio 256/file 4096 |
| ★ | finish_reason 映射 | Claude `end_turn→stop`、`max_tokens→length`、`refusal→content_filter`(打标 `claude_stop_reason=refusal`)；Gemini SAFETY/RECITATION 全压成 `content_filter` |
| ★ | Midjourney/Suno 路由 | `/mj/*`、`/suno/submit/:action`（官方 OpenAI/Anthropic 绝无）|
| ★ | 伪模型后缀 | `-thinking`、`-nothinking`、compact 后缀（new-api 专有）|

---

## 4. CLIProxyAPI 指纹（精确化 + 新增）

> 已知项见 PRD §9.2/§11.2。下列为**新增/精确化**。

### 4.1 响应头（重大精确化）

| 强度 | 指纹 | 精确值 |
|------|------|--------|
| ★★★ | **全局 CORS Expose-Headers 8 项**（任意端点、无鉴权、含 OPTIONS 预检/`/healthz` 可见） | `Access-Control-Expose-Headers: X-CPA-VERSION, X-CPA-COMMIT, X-CPA-BUILD-DATE, X-CPA-SUPPORT-PLUGIN, X-CPA-HOME-VERSION, X-CPA-HOME-BUILD-DATE, X-SERVER-VERSION, X-SERVER-BUILD-DATE`（后 4 项是已知 X-CPA 外新发现的头名）|
| ★★★ | **X-CPA 头仅 `/v0/management/*` 注入，且在鉴权前写入** | 未授权请求打 `/v0/management/config` 也能在 401/403 响应里看到 `X-CPA-SUPPORT-PLUGIN: 0/1`（CGO 编译标志）；普通 `/v1/*` 业务响应**不带** X-CPA 头 |
| ★ | 上游网关头清洗（反向指纹） | 透传上游时剥离 `x-litellm-`/`helicone-`/`x-portkey-`/`cf-aig-`/`x-kong-`/`x-bt-` 前缀并丢 `Set-Cookie` |

### 4.2 路由 / 默认页面（新增）

| 强度 | 路由 | 响应 |
|------|------|------|
| ★★★ | `GET /` | `{"message":"CLI Proxy API Server","endpoints":["POST /v1/chat/completions","POST /v1/completions","GET /v1/models"]}`（精确为带 endpoints 数组的 JSON）|
| ★★★ | `GET\|HEAD /healthz` | `{"status":"ok"}` |
| ★★★ | OAuth 回调固定 HTML（4 路径免鉴权：`/anthropic/callback`、`/codex/callback`、`/antigravity/callback`、`/xai/callback`） | `<title>Authentication successful</title>` + `<h1>Authentication successful!</h1>` + `setTimeout(function(){window.close();},5000)` |
| ★★ | 路由拓扑独有组合 | `POST /v1/messages`（Claude 挂在 OpenAI `/v1` 前缀下）、`GET /v1/responses`（走 websocket）、`/v1/responses/compact`、`/backend-api/codex/responses` |
| ★★ | `/v1/models` 按请求头返回不同结构 | 带 `Anthropic-Version` 或 `claude-cli` UA→Claude，带 `client_version` query→Codex，否则 OpenAI |
| ★ | `GET /management.html` | `Home.Enabled` 或 `DisableControlPanel` 时 404，否则静态文件 |

### 4.3 错误体（新增）

| 强度 | 指纹 |
|------|------|
| ★★★ | 中间件鉴权失败返回**扁平** `{"error":"<msg>"}`（非 OpenAI 嵌套结构），与业务端点的嵌套信封并存——双层结构矛盾是强信号 |
| ★★ | 管理面精确文案：`"remote management disabled"`、`"IP banned due to too many failed attempts. Try again in <duration>"`、`"missing management key"`、`"invalid management key"`、`"invalid password"` |
| ★ | 管理 YAML 校验失败：`{"error":"invalid_yaml","message":"..."}` |

### 4.4 桥接字段（精确化）

| 强度 | 字段 | 说明 |
|------|------|------|
| ★★★ | `native_finish_reason` | **标准 OpenAI 无此字段**，出现即几乎确定经过 X→OpenAI-chat 转换 |
| ★★ | `skip_thought_signature_validator` | Gemini 签名占位哨兵，Gemini→Claude 无真实签名时出现 |
| ★★ | `signature_delta`/`thinking_delta` | X→Claude 流式响应包装 |
| ★★ | `prompt_cache_key` 出现在非 ChatGPT 端 | codex/xai executor 主动注入，桥接痕迹 |
| 纠正 | 对外模型 id **不带** `gemini#`/`claude#`/`gpt#` 前缀 | 这些只是内部签名识别用，PRD 描述需修正 |

### 4.5 上游伪装头（反向检测点）

CLIProxyAPI 向上游发的伪装头（若有回环或被反代回显可观测，也指明了它在模仿什么官方客户端）：
- Claude：超长 `Anthropic-Beta: claude-code-20250219,oauth-2025-04-20,interleaved-thinking-2025-05-14,...`、`X-App: cli`、`X-Stainless-*`、**`X-Claude-Code-Session-Id`（每 apiKey 稳定 UUID）**、UA `claude-cli/2.1.63`。
- Codex：UA `codex-tui/0.135.0 (...)`、`Originator: codex-tui`、`Chatgpt-Account-Id`、WS `OpenAI-Beta: responses_websockets=2026-02-06`。
- Kimi：UA `KimiCLI/1.10.6`。
- 入站清洗：移除客户端真实 `X-Stainless-*`/`Sec-Ch-Ua*`/`Sec-Fetch-*`/`X-Forwarded-*` 再替换为伪装值。

---

## 5. sub2api 指纹（精确化 + 新增关键发现）

> PRD §9.3 已建立"多证据组合"原则。下列为精确常量与新增。

### 5.1 ★★★ 极强新发现：`GET /api/v1/settings/public`（无鉴权）

单点泄露 `{"code":0,"message":"success","data":{...}}`，data 含：
- `version`：**精确后端版本号**（如 `0.1.139`，编译期嵌入）→ 可做版本演进追踪
- `turnstile_enabled` + `turnstile_site_key`：**明文 Cloudflare Turnstile site key**
- 全套开关矩阵：`registration_enabled`/`email_verify_enabled`/`totp_enabled`/`payment_enabled`/`backend_mode_enabled`/`risk_control_enabled`/`channel_monitor_enabled` + 各 OAuth provider（`linuxdo`/`github`/`google`/`wechat`/`oidc`/`dingtalk`）enabled
- `site_name`/`api_base_url`/`custom_endpoints`/`custom_menu_items`

### 5.2 ★★★ 强主动探针：预热请求拦截

POST `/v1/messages` body 含 `Please write a 5-10 word title for the following conversation:`，sub2api 本地拦截并秒回固定 SSE：
- message_start id = `msg_mock_warmup`（建议探针）/ `msg_mock_suggestion`
- warmup 文本增量精确为 `["New"," Conversation"]`，事件间 sleep 20ms
- usage 固定 `input_tokens:10`

误判率极低。但注意：这是向生产系统发探测，需遵守 PRD §20 隐私约束，仅对已授权目标使用。

### 5.3 响应头（新增）

| 强度 | 指纹 |
|------|------|
| ★★ | CORS 签名：`Access-Control-Expose-Headers: ETag`（仅 ETag）+ 12 个 `x-stainless-*` allow-headers + `Max-Age:86400` + Methods 精确顺序 `POST, OPTIONS, GET, PUT, DELETE, PATCH` |
| ★★ | 全局安全头三件套（所有响应）：`X-Content-Type-Options: nosniff` + `X-Frame-Options: DENY` + `Referrer-Policy: strict-origin-when-cross-origin`；**API 路径无 CSP、前端页面有 CSP** 的分裂行为本身是指纹 |
| ★★ | 前端页面 CSP 明文暴露第三方栈：`challenges.cloudflare.com`(Turnstile)、`*.stripe.com`、`*.airwallex.com` |
| ★★ | `X-Client-Request-ID`：精确为标准 UUIDv4，仅网关路由注入，`/health`/管理面不带 |
| ★ | SSE 头含 `X-Accel-Buffering: no`；429 带 `Retry-After`（整数秒），无 `X-RateLimit-*` |

### 5.4 错误体（精确常量，中英混排 = 强指纹）

统一格式 `{"code":"<CODE>","message":"<MSG>"}`，code 英文、message 常中文：

| HTTP | code | message |
|------|------|---------|
| 401 | `API_KEY_REQUIRED` | `API key is required in Authorization header (Bearer scheme), x-api-key header, or x-goog-api-key header`（三 header 并提，措辞高度独特）|
| 401 | `INVALID_API_KEY` | `Invalid API key` |
| 403 | `GROUP_DELETED` | `API Key 所属分组已删除` |
| 403 | `GROUP_DISABLED` | `API Key 所属分组已停用` |
| 429 | `API_KEY_QUOTA_EXHAUSTED` | `API key 额度已用完` |
| 403 | `API_KEY_EXPIRED` | `API key 已过期` |
| 400 | `api_key_in_query_deprecated` | `API key in query parameter is deprecated...` |

未分组 Key 文案随协议变形：Anthropic 组 `{"type":"error","error":{"type":"permission_error","message":"API Key is not assigned to any group..."}}`；Gemini 组 `{"error":{"code":403,"message":"...","status":"PERMISSION_DENIED"}}`。

协议门控 404（强）：`"<X> is not supported for Grok groups"`（X∈`Messages API`/`Chat Completions API`/`Responses WebSocket API`）、`"Embeddings API is not supported for this platform"`、`"Token counting is not supported for this platform"`。

### 5.5 路由 / 模型清单（精确化）

- 路由共存（强）：`/backend-api/codex/responses` + `/antigravity/v1beta/models` + `/antigravity/v1/messages` + `/api/v1/auth/oauth/linuxdo/start` 几乎唯一锁定 sub2api。还有无 v1 前缀别名 `POST /responses`、`POST /chat/completions`。
- 内置模型（强）：`claude-fable-5`(created_at RFC3339 `2026-06-09T00:00:00Z`) + `gpt-5.3-codex-spark` + `codex-auto-review` + `gemini-3.1-pro-preview-customtools` 各自端点共存。`/v1/models` Claude shape 用 `created_at` RFC3339（非 unix），自定义模型固定回填 `2024-01-01T00:00:00Z`（与原生 Anthropic 差异点）。OpenAI shape 多出 `type`/`display_name` 字段（原生 OpenAI 无）。
- Antigravity 模型带 `-thinking` 后缀变体（`claude-opus-4-5-thinking` 等，原生 Anthropic 无此命名）。

### 5.6 桥接 / SSE（新增）

| 强度 | 指纹 |
|------|------|
| ★★ | 桥接 id：拦截响应 `msg_bdrk_`+24位混合字母数字（**伪装 AWS Bedrock**）、web 搜索注入 `msg_ws_`、转换 `chatcmpl-`+hex / `resp_`+hex |
| ★★ | usage 块多出 `total_tokens` 字段（原生 Anthropic usage 无）|
| ★ | SSE ping 三态不一致：排队 `data: {"type": "ping"}\n\n`（无 event 行、冒号带空格）vs 转发 `event: ping\ndata:...` vs OpenAI `{"type":"ping"}` |
| ★ | id 校验正则暴露内部约定：`^resp_[A-Za-z0-9_-]{1,256}$`、`^(msg\|message\|item\|chatcmpl)_...$`，可用畸形 `previous_response_id` 触发精确报错探测 |
| ★ | `/api/event_logging/batch` 是全仓唯一用空体 200（无 Content-Type）返回的路由 |

### 5.7 sub2api 高置信判定建议

无鉴权三件组合全中即高置信：(a) `/health`→`{"status":"ok"}`，(b) `/setup/status`→`needs_setup:false`，(c) `/api/v1/settings/public` 返回带 `version`+`turnstile_site_key` 的封套。需 Key 场景再用预热探针(§5.2)和协议门控文案(§5.4)二次确认平台类型。

---

## 6. 行为指纹检测（来自 api-model-spy + codex/claudecode）

> 原理：响应 `model` 字段可被反代改写，模型列表可显示别名。**行为探针**通过模型固有特性推断真实上游，无法被简单改写绕过。这是 PRD §10 模型身份策略目前最缺的"主动证据"维度。

### 6.1 模型身份主动探针套件

| 探针 | 检测内容 | 判别信号 |
|------|----------|----------|
| 知识截止日期 | 训练数据时间范围 | 不同模型/版本截止日期不同；声称高版本但截止日期是旧的 = 降级 |
| Strawberry 字母计数（"strawberry 有几个 r"）| 分词特征 | 弱模型常答"2"而非"3" |
| 9.11 vs 9.9 大小比较 | 基础数值推理 | 旧/弱模型常答错 |
| 多步骤数学 / 逻辑谜题 | 推理质量 | 档位降级信号 |
| 响应延迟分布 | 模型规模代理 | 大模型更慢；声称大模型但延迟极低 = 可疑 |
| 自我身份自述 | 模型是否透露名称 | 辅助信号（常被 system prompt 抑制）|

实现注意：探针应低温度（temperature=0 或最低）以提高确定性，固定 max_tokens，结果与"已知模型特征库"对比。需建立 model_signatures 表（截止日期、分词答案、典型延迟区间）。

### 6.2 价格异常预警

声称高端模型但定价不合理地低（如声称 Opus 却 $0.5/M）→ 结合 token_audit 成本倍率（PRD §12 已有 token_audit）输出价格异常信号。

### 6.3 reasoning_tokens 身份指纹（强）

`usage.output_tokens_details.reasoning_tokens`（OpenAI）/ 思考模型的 reasoning token 字段只有 o 系/思考类模型产出。
- 真实案例：某服务商卖 `gpt-5.5`（不存在的版本），每次响应均含 `reasoning_tokens` + 知识截止 2024-06 + 延迟 4-10s → 实为 o1-mini 套假名。
- 检测器应：请求一个声称非 reasoning 的模型，若响应带 reasoning_tokens（或反之），即模型身份不一致信号。

### 6.4 token usage 交叉验证（来自代理 token 吞噬检测）

用本地 tokenizer（tiktoken/对应 BPE）重算 prompt+completion token 数，与代理上报的 usage 对比：
- 持续正向 delta（上报 > 实算）= token 膨胀/虚报。
- 结合 PRD §12 token_audit 的成本倍率，把"usage 字段完整性"升级为"usage 数值真实性"。
- 注意 new-api 的本地估算特征（§3.3）：非 BPE 的加权字符估算会产生可识别的整数模式。

### 6.5 确定性 / system_fingerprint 一致性

- 同 prompt + temperature=0 + 固定 seed 多次请求，观察输出是否稳定、`system_fingerprint` 是否一致。
- 真 OpenAI(gpt-4o) 几乎总返回 `fp_xxx`；中转/桥接（如 new-api Claude 渠道）`system_fingerprint` 恒 null/缺失。

---

## 7. 落地建议（给实现者）

### 7.1 新增/增强 detector

| detector | 增强点 |
|----------|--------|
| `channels/newapi` | 加 `X-New-Api-Version`/`X-Oneapi-Request-Id` 头、`new_api_error` type、`(request id:` 后缀、`（distributor）`、OpenAI 接口返回 `msg_` id |
| `channels/cliproxyapi` | 加全局 CORS 8 头、`/healthz`、`/` 根 JSON、OAuth 回调 HTML、未授权 X-CPA、扁平 `{"error":...}`、`native_finish_reason` |
| `channels/sub2api` | 加 `/api/v1/settings/public` 解析（version+turnstile）、预热探针、CORS ETag/stainless、中英混排错误码、协议门控文案 |
| `channels/openai` + `openai/probes` | 加 Responses 正向/负向探针（store/include/WS beta）、`openai-model` 响应头读取、reasoning_tokens 检查 |
| `channels/claude` + `claude/probes` | 加 thinking budget 越界负向探针、system cache_control 越量负向探针、cache_creation 细分缺失检查、`x-app:cli`/oauth-beta 行为 |

### 7.2 新增模块建议

- `purity/behavioral_probe.go`（或 channels 外的独立 prober）：实现 §6.1 行为探针套件 + model_signatures 表。受 PRD §20 预算/超时/样本数约束。
- `purity/model_signatures.go`：知识截止日期、分词答案、延迟区间、reasoning_tokens 期望的可更新特征库（对应 PRD 开放问题 #1 在线模型注册表）。
- token usage 交叉验证可并入现有 `token_audit.go`。

### 7.3 评分映射（衔接 PRD §12.3 封顶规则）

| 新证据 | 建议处理 |
|--------|----------|
| 官方负向探针失败（store/thinking budget/signature/cache_control 约束未复刻） | 强烈否定 official verdict，official_score 封顶 ≤45 |
| reasoning_tokens 与声称模型不符 / 行为探针显示降级 | model_identity fail，按 PRD 封顶 50 |
| usage 交叉验证发现膨胀 | token_audit anomaly，扣分 + 报告价格风险 |
| 网关精确头/错误体命中（透明中转） | 仅记录 wrapper_signal，**不封顶**（遵守 PRD AC-06）|
| 桥接字段命中（`native_finish_reason`/伪 id/msg_ 改写） | 混淆风险信号，official_score 封顶 55 |
| `/api/v1/settings/public` 等未鉴权配置泄露 | 强 wrapper_signal（透明中转），不封顶但提高渠道识别置信度 |

### 7.4 隐私 / 合规约束（强制，对齐 PRD §20）

- 行为探针、预热拦截探针、未鉴权管理面探测：仅对已授权目标和用户明确允许的场景执行，未授权目标默认跳过。
- 探针有固定样本数、超时、body 上限，不得变成压测。
- turnstile site key、version 等泄露字段只用于识别，不回显敏感内容。
- 不主动绕过风控；遥测端点探测（如 `/api/event_logging/batch`）避免向未知生产系统频繁发送。

### 7.5 待补充 / 开放项
- codex/claudecode 当前精确版本号无法从源码树取得（构建期注入），UA 版本匹配需运行时校准。
- §6.1 model_signatures 需用授权真实样本校准（对接 PRD P5）。
- api-model-spy 的探针 prompt 原文未能抓取（网络策略限制），§6.1 表格为 README 归纳，落地前建议复现验证各探针的判别力。

---

## 附：PRD 需修正的描述

1. PRD §9.2/§11.2 称 CLIProxyAPI 对外模型带 `gemini#`/`claude#`/`gpt#` 前缀——实测这些仅内部签名识别用，对外 id 不带。应改为"内部签名 provider prefix"。
2. PRD §11.2 称 X-CPA 头是 CLIProxyAPI 通用特征——实测仅 `/v0/management/*` 注入；业务响应靠全局 CORS Expose-Headers（8 项）识别更可靠。
3. 被分析的 claudecode bundle 快照内置清单**未收录** `claude-fable-5`，但 Fable 5 是真实官方最新模型（快照较旧）。**不要**把"清单里没有 claude-fable-5"当作非官方判据；模型清单仅用于发现别名/降级，官方性判别以负向探针（§2）为准。
