const sensitiveDetailKeyPattern = /api[_-]?key|authorization|cookie|signature/i
const sensitiveDetailValuePattern = /^(?:bearer\s+|sk-[a-z0-9_-]{12,}|[a-z0-9+/]{80,}={0,2}$)/i

export function safePurityDetailEntries(details?: Record<string, unknown>): Array<[string, unknown]> {
  if (!details) return []
  return Object.entries(details)
    .filter(([key]) => !sensitiveDetailKeyPattern.test(key))
    .map(([key, value]) => [key, sanitizePurityDetailValue(value, 0)])
}

export function formatPurityDetailValue(value: unknown): string {
  if (value === null || value === undefined) return '-'
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  try {
    return JSON.stringify(value)
  } catch {
    return '[unavailable]'
  }
}

export function purityDetailKeyLabel(key: string): string {
  return key
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .replace(/[_-]+/g, ' ')
    .trim()
}

function sanitizePurityDetailValue(value: unknown, depth: number): unknown {
  if (depth >= 4) return '[truncated]'
  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (sensitiveDetailValuePattern.test(trimmed)) return '[redacted]'
    return value.length > 500 ? `${value.slice(0, 500)}...` : value
  }
  if (Array.isArray(value)) {
    return value.slice(0, 30).map((item) => sanitizePurityDetailValue(item, depth + 1))
  }
  if (value && typeof value === 'object') {
    return Object.fromEntries(
      Object.entries(value as Record<string, unknown>)
        .filter(([key]) => !sensitiveDetailKeyPattern.test(key))
        .slice(0, 50)
        .map(([key, child]) => [key, sanitizePurityDetailValue(child, depth + 1)])
    )
  }
  return value
}
