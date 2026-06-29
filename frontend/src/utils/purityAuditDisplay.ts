import type { PurityProvider, PurityTokenAuditReport, PurityTokenAuditSample } from '@/api/admin/adminPlus'

export type TokenAuditTone = 'neutral' | 'good' | 'warn' | 'bad'

export interface TokenAuditDisplayCell {
  value: number
  delta?: number
  tone: TokenAuditTone
  available: boolean
  display: string
  title?: string
  unavailableReason?: TokenAuditUnavailableReason
}

export type TokenAuditUnavailableReason = 'gemini_cache_creation_missing' | 'gemini_cached_content_missing'
export type TokenAuditCacheDisplayMode = 'dash' | 'zero'

export function normalizeTokenAuditProvider(provider?: PurityProvider | string | null): PurityProvider | string {
  const value = String(provider || '').trim().toLowerCase()
  const alias = value.replace(/[\s-]+/g, '_')
  if (alias === 'claude' || alias === 'anthropic_compatible' || alias === 'claude_compatible' || alias === 'claude_compat') {
    return 'anthropic'
  }
  if (alias === 'google' || alias === 'google_ai' || alias === 'google_ai_studio' || alias === 'ai_studio' || alias === 'aistudio' || alias === 'gemini_compatible' || alias === 'gemini_compat' || alias === 'vertex' || alias === 'vertex_ai' || alias === 'google_vertex' || alias === 'antigravity') {
    return 'gemini'
  }
  if (alias === 'openai_compatible' || alias === 'openai_compat') {
    return 'openai'
  }
  return alias || 'openai'
}

export function isGeminiTokenAuditProvider(provider?: PurityProvider | string | null): boolean {
  return normalizeTokenAuditProvider(provider) === 'gemini'
}

export interface TokenAuditDisplayRow {
  input: TokenAuditDisplayCell
  output: TokenAuditDisplayCell
  cacheCreation: TokenAuditDisplayCell
  cacheRead: TokenAuditDisplayCell
  latency: TokenAuditDisplayCell
  cost: TokenAuditDisplayCell
  multiplier: {
    value: number
    tone: TokenAuditTone
    available: boolean
    display: string
    title?: string
  }
}

export interface TokenAuditRuntimeSummary {
  latencyMS: number
  tokensPerSecond: number
  inputTokens: number
  outputTokens: number
  sampleCount: number
}

export interface TokenAuditSampleRatioCell {
  display: string
  tone: TokenAuditTone
  title: string
}

type AuditRatioInput = Partial<PurityTokenAuditReport> & {
  samples?: AuditSampleInput[]
  rows?: AuditSampleInput[]
  baselineTotalCost?: number
  totalCost?: number
  overallRatio?: number
  billing_multiplier?: number | null
  billingMultiplier?: number | null
}
type AuditSummaryInput = Partial<Pick<PurityTokenAuditReport, 'input_tokens' | 'output_tokens'>> & {
  samples?: AuditSampleInput[]
  rows?: AuditSampleInput[]
}
type AuditSampleInput = Partial<PurityTokenAuditSample>

export function tokenAuditDisplayRatio(audit?: AuditRatioInput | null): number {
  if (!audit) return 0
  const direct = firstPositiveNumber(audit.overall_ratio, audit.overallRatio, audit.multiplier)
  if (direct > 0) return direct
  const totals = tokenAuditCostTotals(audit)
  if (totals.officialBaselineUSD > 0 && totals.actualCostUSD > 0) {
    return roundDisplayRatio(totals.actualCostUSD / totals.officialBaselineUSD)
  }
  return 0
}

export function tokenAuditSampleDisplayRatio(sample: AuditSampleInput): number {
  return firstPositiveNumber(sample.ratio, sample.multiplier)
}

export function tokenAuditSampleRatioDisplayCell(
  sample: AuditSampleInput,
  provider: PurityProvider | string,
  billingMultiplier?: number | null
): TokenAuditSampleRatioCell {
  const normalizedProvider = normalizeTokenAuditProvider(provider)
  const row = tokenAuditSampleDisplayRow(sample, normalizedProvider)
  if (typeof billingMultiplier === 'number' && Number.isFinite(billingMultiplier)) {
    return {
      display: formatMultiplier(billingMultiplier),
      tone: 'good',
      title: `平台计费倍率 ${formatMultiplier(billingMultiplier)}；本轮 Usage 比值 ${formatMultiplier(row.multiplier.value)}。`
    }
  }
  if (isGeminiTokenAuditProvider(normalizedProvider)) {
    return {
      display: row.multiplier.display,
      tone: row.multiplier.tone,
      title: 'Gemini usageMetadata 的 Usage 比值通常只是官方 usage 估算口径，不等同平台计费倍率；需结合账号配置或 /v1/usage 扣费增量确认。'
    }
  }
  return {
    display: row.multiplier.display,
    tone: row.multiplier.tone,
    title: `本轮 Usage 比值 ${row.multiplier.display}。`
  }
}

export function tokenAuditSampleDisplayRow(sample: AuditSampleInput, provider: PurityProvider | string): TokenAuditDisplayRow {
  const normalizedProvider = normalizeTokenAuditProvider(provider)
  const input = auditDisplayInputTokens(sample, normalizedProvider)
  const output = nonNegativeNumber(sample.output_tokens)
  const cacheCreation = auditDisplayCacheCreationTokens(sample, normalizedProvider)
  const cacheRead = auditDisplayCacheReadTokens(sample, normalizedProvider)
  const multiplier = tokenAuditSampleDisplayRatio(sample)
  return {
    input: {
      value: input,
      delta: sample.input_delta_pct,
      tone: deltaTone(sample.input_delta_pct),
      available: true,
      display: formatAuditInteger(input)
    },
    output: {
      value: output,
      delta: sample.output_delta_pct,
      tone: deltaTone(sample.output_delta_pct),
      available: true,
      display: formatAuditInteger(output)
    },
    cacheCreation: {
      value: cacheCreation.value,
      delta: sample.cache_creation_delta_pct,
      tone: cacheCreation.available ? deltaTone(sample.cache_creation_delta_pct) : 'neutral',
      available: cacheCreation.available,
      display: cacheCreation.display,
      title: cacheCreation.title
    },
    cacheRead: {
      value: cacheRead.value,
      delta: sample.cache_read_delta_pct,
      tone: cacheRead.available ? deltaTone(sample.cache_read_delta_pct) : 'neutral',
      available: cacheRead.available,
      display: cacheRead.display,
      title: cacheRead.title
    },
    latency: {
      value: nonNegativeNumber(sample.latency_ms),
      tone: latencyTone(sample.latency_ms),
      available: true,
      display: formatTokenAuditLatencyMS(sample.latency_ms)
    },
    cost: {
      value: firstNonNegativeNumber(sample.actual_cost_usd, sample.cost),
      delta: sample.cost_delta_pct,
      tone: deltaTone(sample.cost_delta_pct),
      available: true,
      display: ''
    },
    multiplier: {
      value: multiplier,
      tone: multiplierTone(multiplier),
      available: multiplier > 0,
      display: formatMultiplier(multiplier)
    }
  }
}

export function hasTokenAuditSampleData(sample: AuditSampleInput): boolean {
  return Boolean(
    (sample.total_tokens || 0) > 0 ||
    (sample.input_tokens || 0) > 0 ||
    (sample.output_tokens || 0) > 0 ||
    (sample.cache_creation_tokens || sample.cache_creation_input_tokens || 0) > 0 ||
    (sample.cached_tokens || sample.cache_read_input_tokens || 0) > 0 ||
    (sample.actual_cost_usd || sample.cost || 0) > 0 ||
    (sample.official_baseline_usd || sample.baseline_cost || 0) > 0
  )
}

export function tokenAuditRuntimeSummary(audit?: AuditSummaryInput | null): TokenAuditRuntimeSummary {
  if (!audit) return emptyRuntimeSummary()
  const samples = audit.rows?.length ? audit.rows : audit.samples || []
  const successfulSamples = samples.filter((sample) => sample.status === 'pass' && hasTokenAuditSampleData(sample))
  const inputTokensFromSamples = successfulSamples.reduce((sum, sample) => sum + nonNegativeNumber(sample.input_tokens), 0)
  const outputTokensFromSamples = successfulSamples.reduce((sum, sample) => sum + nonNegativeNumber(sample.output_tokens), 0)
  const latencySamples = successfulSamples
    .map((sample) => nonNegativeNumber(sample.latency_ms))
    .filter((value) => value > 0)
  const latencyTotalMS = latencySamples.reduce((sum, value) => sum + value, 0)
  const latencyMS = latencySamples.length > 0 ? Math.round(latencyTotalMS / latencySamples.length) : 0
  const outputTokens = Math.round(firstPositiveNumber(audit.output_tokens, outputTokensFromSamples))
  const inputTokens = Math.round(firstPositiveNumber(audit.input_tokens, inputTokensFromSamples))
  const outputTokensForRate = outputTokensFromSamples > 0 ? outputTokensFromSamples : outputTokens
  const tokensPerSecond = latencyTotalMS > 0 && outputTokensForRate > 0 ? outputTokensForRate / (latencyTotalMS / 1000) : 0

  return {
    latencyMS,
    tokensPerSecond,
    inputTokens,
    outputTokens,
    sampleCount: successfulSamples.length
  }
}

export function tokenAuditCostTotals(audit?: AuditRatioInput | null): { officialBaselineUSD: number; actualCostUSD: number } {
  if (!audit) return { officialBaselineUSD: 0, actualCostUSD: 0 }
  const samples = audit.rows?.length ? audit.rows : audit.samples || []
  const sampleTotals = samples.reduce(
    (acc, sample) => {
      if (!hasTokenAuditSampleData(sample)) return acc
      acc.officialBaselineUSD += sample.official_baseline_usd || sample.baseline_cost || 0
      acc.actualCostUSD += sample.actual_cost_usd || sample.cost || 0
      return acc
    },
    { officialBaselineUSD: 0, actualCostUSD: 0 }
  )
  return {
    officialBaselineUSD: roundMoneyTotal(firstPositiveNumber(audit.official_baseline_usd, audit.baseline_total_cost_usd, audit.baselineTotalCost, sampleTotals.officialBaselineUSD)),
    actualCostUSD: roundMoneyTotal(firstPositiveNumber(audit.actual_cost_usd, audit.total_cost, audit.totalCost, sampleTotals.actualCostUSD))
  }
}

function emptyRuntimeSummary(): TokenAuditRuntimeSummary {
  return {
    latencyMS: 0,
    tokensPerSecond: 0,
    inputTokens: 0,
    outputTokens: 0,
    sampleCount: 0
  }
}

export function formatTokenAuditLatencyMS(value: number | undefined | null): string {
  const safe = Math.max(0, Math.round(typeof value === 'number' && Number.isFinite(value) ? value : 0))
  if (safe >= 1000) {
    return `${(safe / 1000).toFixed(1).replace(/\.0$/, '')}s`
  }
  return `${safe}ms`
}

export function multiplierTone(value: number): TokenAuditTone {
  if (!Number.isFinite(value) || value <= 0) return 'neutral'
  if (value >= 1.5) return 'bad'
  if (value >= 1.2) return 'warn'
  return 'good'
}

function auditDisplayInputTokens(sample: AuditSampleInput, provider: PurityProvider | string): number {
  const input = nonNegativeNumber(sample.input_tokens)
  if (normalizeTokenAuditProvider(provider) === 'openai') {
    return Math.max(0, input - auditDisplayCacheReadTokens(sample, provider).value)
  }
  return input
}

function auditDisplayCacheCreationTokens(sample: AuditSampleInput, provider: PurityProvider | string): TokenAuditDisplayCellValue {
  const value = firstDisplayNumber(sample.cache_creation_input_tokens, sample.cache_creation_tokens)
  if (isGeminiTokenAuditProvider(provider) && value <= 0) {
    return unavailableTokenCell('gemini_cache_creation_missing', 'Gemini GenerateContent usageMetadata 没有缓存创建 token 字段，本列无法确认。', 'zero')
  }
  return availableTokenCell(value)
}

function auditDisplayCacheReadTokens(sample: AuditSampleInput, provider: PurityProvider | string): TokenAuditDisplayCellValue {
  const value = firstDisplayNumber(sample.cache_read_input_tokens, sample.cached_tokens)
  if (isGeminiTokenAuditProvider(provider) && value <= 0) {
    return unavailableTokenCell('gemini_cached_content_missing', 'Gemini usageMetadata 本轮未观察到 cachedContentTokenCount 命中，按本轮缓存读取 0 展示。', 'zero')
  }
  return availableTokenCell(value)
}

function firstDisplayNumber(...values: Array<number | undefined | null>): number {
  return firstPositiveNumber(...values) || firstNonNegativeNumber(...values)
}

function firstNonNegativeNumber(...values: Array<number | undefined | null>): number {
  for (const value of values) {
    if (typeof value === 'number' && Number.isFinite(value) && value >= 0) return value
  }
  return 0
}

function firstPositiveNumber(...values: Array<number | undefined | null>): number {
  for (const value of values) {
    if (typeof value === 'number' && Number.isFinite(value) && value > 0) return value
  }
  return 0
}

function nonNegativeNumber(value: number | undefined | null): number {
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? value : 0
}

type TokenAuditDisplayCellValue = Pick<TokenAuditDisplayCell, 'value' | 'available' | 'display' | 'title' | 'unavailableReason'>

function availableTokenCell(value: number): TokenAuditDisplayCellValue {
  return {
    value,
    available: true,
    display: formatAuditInteger(value)
  }
}

function unavailableTokenCell(unavailableReason: TokenAuditUnavailableReason, title: string, displayMode: TokenAuditCacheDisplayMode = 'dash'): TokenAuditDisplayCellValue {
  return {
    value: 0,
    available: false,
    display: displayMode === 'zero' ? '0' : '-',
    title,
    unavailableReason
  }
}

function formatAuditInteger(value: number): string {
  return new Intl.NumberFormat('en-US', { maximumFractionDigits: 0 }).format(value || 0)
}

function formatMultiplier(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '-'
  return `${value.toFixed(2).replace(/0+$/, '').replace(/\.$/, '')}x`
}

function deltaTone(delta: number | undefined): TokenAuditTone {
  if (typeof delta !== 'number' || !Number.isFinite(delta) || delta === 0) return 'neutral'
  return delta > 0 ? 'bad' : 'good'
}

function latencyTone(value: number | undefined): TokenAuditTone {
  if (typeof value !== 'number' || !Number.isFinite(value) || value <= 0) return 'neutral'
  if (value >= 30_000) return 'bad'
  if (value >= 10_000) return 'warn'
  return 'good'
}

function roundDisplayRatio(value: number): number {
  if (!Number.isFinite(value) || value <= 0) return 0
  return Math.round(value * 100) / 100
}

function roundMoneyTotal(value: number): number {
  if (!Number.isFinite(value) || value <= 0) return 0
  return Math.round(value * 1_000_000) / 1_000_000
}
