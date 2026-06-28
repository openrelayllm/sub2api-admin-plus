import type { PurityProvider, PurityTokenAuditReport, PurityTokenAuditSample } from '@/api/admin/adminPlus'

export type TokenAuditTone = 'neutral' | 'good' | 'warn' | 'bad'

export interface TokenAuditDisplayCell {
  value: number
  delta?: number
  tone: TokenAuditTone
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
  }
}

export interface TokenAuditRuntimeSummary {
  latencyMS: number
  tokensPerSecond: number
  inputTokens: number
  outputTokens: number
  sampleCount: number
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

export function tokenAuditSampleDisplayRow(sample: AuditSampleInput, provider: PurityProvider | string): TokenAuditDisplayRow {
  const input = auditDisplayInputTokens(sample, provider)
  const output = nonNegativeNumber(sample.output_tokens)
  const cacheCreation = auditDisplayCacheCreationTokens(sample)
  const cacheRead = auditDisplayCacheReadTokens(sample)
  const multiplier = tokenAuditSampleDisplayRatio(sample)
  return {
    input: {
      value: input,
      delta: sample.input_delta_pct,
      tone: deltaTone(sample.input_delta_pct)
    },
    output: {
      value: output,
      delta: sample.output_delta_pct,
      tone: deltaTone(sample.output_delta_pct)
    },
    cacheCreation: {
      value: cacheCreation,
      delta: sample.cache_creation_delta_pct,
      tone: deltaTone(sample.cache_creation_delta_pct)
    },
    cacheRead: {
      value: cacheRead,
      delta: sample.cache_read_delta_pct,
      tone: deltaTone(sample.cache_read_delta_pct)
    },
    latency: {
      value: nonNegativeNumber(sample.latency_ms),
      tone: latencyTone(sample.latency_ms)
    },
    cost: {
      value: firstNonNegativeNumber(sample.actual_cost_usd, sample.cost),
      delta: sample.cost_delta_pct,
      tone: deltaTone(sample.cost_delta_pct)
    },
    multiplier: {
      value: multiplier,
      tone: multiplierTone(multiplier)
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
  if (provider === 'openai') {
    return Math.max(0, input - auditDisplayCacheReadTokens(sample))
  }
  return input
}

function auditDisplayCacheCreationTokens(sample: AuditSampleInput): number {
  return firstDisplayNumber(sample.cache_creation_input_tokens, sample.cache_creation_tokens)
}

function auditDisplayCacheReadTokens(sample: AuditSampleInput): number {
  return firstDisplayNumber(sample.cache_read_input_tokens, sample.cached_tokens)
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
