# 官方负向探针实现规格（给 codex）

状态：Claude thinking signature / thinking budget / cache_control overflow 负向探针已落地；OpenAI `openai-model` 响应头、`reasoning_tokens` 身份信号和 `store:false` + `include:["reasoning.encrypted_content"]` 正向探针已接入 `model_identity` / `openai` 检查链路；更重的行为问答指纹仍待后续实现。本规格继续作为 OpenAI / Claude 官方约束探针与行为指纹扩展清单。
关联：`FINGERPRINTS.md` §1、§2、§7.1、§7.3。
背景：本会话由 Claude 完成调研 + detector 增强；openai/claude flow 区域（`claude_payload.go` 等）由 codex 接手，故官方负向探针以规格交付，避免并发写同一批文件。

## 0. 核心原理

官方客户端（codex/claudecode）揭示了真实 API **强制校验、而兼容/包装实现常放过**的约束。检测器主动发"故意违反官方约束"的探针：
- 真官方端 → 返回 4xx（按官方错误形态拒绝）。
- 包装/转译层 → 常放过（2xx）或错误形态不符。

判别"是否官方后端"，且**不依赖域名**。

## 1. 现有可复用范式（已验证）

`claude.go` 已有一个负向探针，照抄其结构即可：
- payload：`claudeInvalidThinkingSignaturePayload`（`claude_payload.go`）
- probe：`probeClaudeInvalidThinkingSignature`（`claude_probe.go`）—— `s.doJSONWithHeaders(... POST /v1/messages ... claudeHeaders(...) ...)`
- check：`buildClaudeThinkingSignatureCheck`（`claude_checks.go`）—— 2xx=fail（被接受=非官方），400+signature 文案=pass，其他 4xx=warn
- 接线：`claude.go` 第 163-168 行，归入 `signature` validation 分组

## 2. 评分约定（重要，避免污染兼容分）

- Claude 类负向探针 check **必须**有 `MaxScore>0` 并归入 **signature** validation 分组（不要新建独立 validation）。
- 原因：`report.go` 把 `MaxScore>0` 且 `ID!="base_url"` 的 check 同时计入 official_score 和 compatibility_score。官方专属约束**不应**影响兼容分——但归入 signature 分组后，已有的 `wrapperPurityScoreCap`（`evaluation.go`）会在 `validationStatus(report,"signature")==Fail` 时把 official_score 封顶，**无需改评分逻辑**。
- OpenAI `store:false` + `include:["reasoning.encrypted_content"]` 是正向探针，当前实现刻意采用 `MaxScore=0`，只驱动 `signature` validation 和官方分封顶，不污染兼容分。

## 3. 探针清单

### 3.1 Claude — thinking budget 越界（FINGERPRINTS §2.1）

实现状态：已落地到 `claudeThinkingBudgetViolationPayload` / `probeClaudeThinkingBudgetViolation` / `buildClaudeThinkingBudgetCheck`，归入 `signature` validation。

官方约束：`max_tokens > thinking.budget_tokens`，违反返回 400。

- payload：`claudeThinkingBudgetViolationPayload(model, probeCtx)`——`max_tokens: 64`，`thinking: {"type":"enabled","budget_tokens": 64}`（budget >= max，故意违反），其余沿用 `claudeSystemBlocks` + 一条简单 user 消息。
- check 判定：
  - 2xx → fail（"thinking budget 越界请求被接受，未观察到 Claude 官方 max_tokens>budget 约束"）
  - 400 且错误文案含 `budget`/`max_tokens`/`thinking` → pass
  - 其他 4xx → warn（错误形态不完全匹配）
  - StatusCode==0 / 余额不足 → fail（探测无法执行）
- 归入 `signature` validation。

### 3.2 Claude — system cache_control 越量（FINGERPRINTS §2.2）

实现状态：已落地到 `claudeCacheControlOverflowPayload` / `probeClaudeCacheControlOverflow` / `buildClaudeCacheControlOverflowCheck`，归入 `signature` validation。

官方约束：system block 的 cache_control 数量有上限，超量（5+ 个 ephemeral block）返回 400。

- payload：`claudeCacheControlOverflowPayload(model, probeCtx)`——system 数组放 6 个均带 `cache_control:{type:"ephemeral"}` 的 text block。
- check 判定：2xx→fail；400→pass；其他 4xx→warn。归入 `signature` validation。
- 注意：此探针消耗一次请求，注意 PRD §20 预算约束。可与 3.1 二选一或都做。

### 3.3 OpenAI — Responses store/include 行为（FINGERPRINTS §1.1）

官方 codex 走 Responses，`store:false` + `include:["reasoning.encrypted_content"]`。许多兼容实现默认/要求 `store:true` 或不识别 include。

- 这是**正向探针**（模仿官方行为，观察是否被正常接受），已落在 openai flow；当前实现刻意采用 `MaxScore=0`，只驱动 `signature` validation 和官方分封顶，不计入兼容分。
- 更稳的负向信号是读响应头 `openai-model`（已加入 `http_probe.go` 白名单并接入 `model_identity`）：若存在且 != body/request model，按更可信响应模型来源参与身份校验，`model_identity.evidence` 会记录 `openai_model_header`、`response_body_model`、`response_model_source`。
- reasoning_tokens 身份探针见 §3.4。

### 3.4 行为指纹 — reasoning_tokens 身份（FINGERPRINTS §6.3）

`usage.output_tokens_details.reasoning_tokens`（OpenAI）/ Claude thinking token：
- 请求一个声称非 reasoning 的模型，若响应带 reasoning_tokens（或反之）→ model_identity 不一致信号。
- 实现状态：OpenAI Responses 非流式 usage 已解析 `reasoning_tokens` 并接入 `model_identity`；当普通 GPT/Chat 声称模型返回 `reasoning_tokens>0` 时触发 `reasoning_tokens_mismatch`，Codex/o 系/显式 reasoning 模型不触发。更细的模型签名表仍待 §6.1/P5 校准。

## 4. 测试要求（CLAUDE.md Definition of Done）

每个新探针配 mock server 测试（参考 `service_test.go` 现有 `TestServiceRunPublicCheck_*`）：
- 官方行为 mock（返回 400 拒绝）→ check pass，signature validation pass。
- 兼容/包装 mock（返回 200 接受）→ check fail，signature validation fail，official_score 被封顶到 45，verdict 不为 official_claude。
- 余额不足 / 连接失败 → check fail 但不误判为"非官方"。

## 5. 已完成部分（Claude 已做，勿重复）

- `http_probe.go` 响应头白名单已含 `openai-model`、`x-reasoning-included`、`x-models-etag`、`auth-version` 等（§3.3/§1.2 所需）。
- newapi/cliproxyapi/sub2api/antigravity/gemini detector 已按 FINGERPRINTS §3-§5 增强，含测试。
- `claude-fable-5` 误判已修正（它是真实官方模型）。
- Claude flow 已接入三类官方负向探针：伪造 thinking signature、thinking budget 越界、system cache_control 越量；官方 mock 400 拒绝为 pass，兼容/包装 mock 2xx 接受为 fail。
