import { describe, expect, it } from 'vitest'

import { tokenAuditSampleDisplayRow, tokenAuditSampleRatioDisplayCell } from '@/utils/purityAuditDisplay'

describe('purityAuditDisplay', () => {
  it('shows missing Gemini cache fields as unavailable instead of confirmed cache values', () => {
    const row = tokenAuditSampleDisplayRow(
      {
        index: 1,
        input_tokens: 90,
        output_tokens: 12,
        uncached_input_tokens: 90,
        cache_creation_tokens: 0,
        cache_creation_tokens_present: false,
        cached_tokens: 0,
        cached_tokens_present: false,
        total_tokens: 102,
        official_baseline_usd: 0.001,
        actual_cost_usd: 0.001,
        multiplier: 1,
        latency_ms: 800,
        status: 'pass'
      },
      'gemini'
    )

    expect(row.cacheCreation.available).toBe(false)
    expect(row.cacheCreation.display).toBe('0')
    expect(row.cacheRead.available).toBe(false)
    expect(row.cacheRead.display).toBe('0')
  })

  it('shows explicit Gemini zero cache fields as unavailable because zero is not a hit', () => {
    const row = tokenAuditSampleDisplayRow(
      {
        index: 1,
        input_tokens: 90,
        output_tokens: 12,
        uncached_input_tokens: 90,
        cache_creation_tokens: 0,
        cache_creation_tokens_present: true,
        cached_tokens: 0,
        cached_tokens_present: true,
        total_tokens: 102,
        official_baseline_usd: 0.001,
        actual_cost_usd: 0.001,
        multiplier: 1,
        latency_ms: 800,
        status: 'pass'
      },
      'gemini'
    )

    expect(row.cacheCreation.available).toBe(false)
    expect(row.cacheCreation.display).toBe('0')
    expect(row.cacheRead.available).toBe(false)
    expect(row.cacheRead.display).toBe('0')
  })

  it('normalizes Gemini provider aliases before displaying cache fields', () => {
    const row = tokenAuditSampleDisplayRow(
      {
        index: 1,
        input_tokens: 90,
        output_tokens: 12,
        cached_tokens: 0,
        cache_creation_tokens: 0,
        latency_ms: 800,
        multiplier: 1,
        status: 'pass'
      },
      'Google AI Studio'
    )

    expect(row.cacheCreation.available).toBe(false)
    expect(row.cacheCreation.display).toBe('0')
    expect(row.cacheRead.available).toBe(false)
    expect(row.cacheRead.display).toBe('0')
  })

  it('shows Gemini usage ratio cells without hiding them when platform billing multiplier is missing', () => {
    const cell = tokenAuditSampleRatioDisplayCell(
      {
        input_tokens: 90,
        output_tokens: 12,
        cached_tokens: 0,
        cache_creation_tokens: 0,
        latency_ms: 800,
        multiplier: 1,
        ratio: 1,
        status: 'pass'
      },
      'Google AI Studio'
    )

    expect(cell.display).toBe('1x')
    expect(cell.tone).toBe('good')
    expect(cell.title).toContain('平台计费倍率')
  })

  it('shows configured platform billing multiplier for Gemini ratio cells', () => {
    const cell = tokenAuditSampleRatioDisplayCell(
      {
        input_tokens: 90,
        output_tokens: 12,
        cached_tokens: 0,
        cache_creation_tokens: 0,
        latency_ms: 800,
        multiplier: 1,
        ratio: 1,
        status: 'pass'
      },
      'gemini-compatible',
      0.11
    )

    expect(cell.display).toBe('0.11x')
    expect(cell.tone).toBe('good')
    expect(cell.title).toContain('Usage 比值 1x')
  })
})
