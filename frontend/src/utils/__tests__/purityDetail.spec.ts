import { describe, expect, it } from 'vitest'
import { formatPurityDetailValue, safePurityDetailEntries } from '@/utils/purityDetail'

describe('purityDetail', () => {
  it('redacts sensitive nested detail while preserving diagnostic evidence', () => {
    const entries = safePurityDetailEntries({
      status_code: 200,
      api_key: 'sk-secret-value',
      nested: {
        authorization: 'Bearer secret',
        result: 'matched'
      },
      value: 'sk-abcdefghijklmnopqrstuvwxyz'
    })

    const output = formatPurityDetailValue(Object.fromEntries(entries))
    expect(output).toContain('status_code')
    expect(output).toContain('matched')
    expect(output).toContain('[redacted]')
    expect(output).not.toContain('sk-secret-value')
    expect(output).not.toContain('Bearer secret')
  })
})
