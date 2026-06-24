export function formatMoneyCompact(cents: number, currency: string): string {
  return new Intl.NumberFormat(undefined, { style: 'currency', currency: currency || 'USD', currencyDisplay: 'narrowSymbol', minimumFractionDigits: 4, maximumFractionDigits: 4 }).format((cents || 0) / 100)
}

export function formatInteger(value: number): string {
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 }).format(value || 0)
}

export function formatRate(value?: number | null): string {
  return `${(value ?? 1).toFixed(2)}x`
}

export function formatDateTime(value?: string | null): string {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '-' : date.toLocaleString()
}

export function formatRelativeDateTime(value?: string | null): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  const diff = Date.now() - date.getTime()
  if (diff < 60 * 1000) return '刚刚'
  if (diff < 60 * 60 * 1000) return `${Math.floor(diff / (60 * 1000))}分钟前`
  if (diff < 24 * 60 * 60 * 1000) return `${Math.floor(diff / (60 * 60 * 1000))}小时前`
  if (diff < 30 * 24 * 60 * 60 * 1000) return `${Math.floor(diff / (24 * 60 * 60 * 1000))}天前`
  return date.toLocaleDateString()
}

export function accountStatusLabel(value: string): string {
  return { active: '正常', disabled: '停用', error: '异常', rate_limited: '限流' }[value] || value || '-'
}

export function accountStatusClass(value: string): string {
  if (value === 'active') return 'badge-success'
  if (value === 'disabled') return 'badge-gray'
  if (value === 'error' || value === 'rate_limited') return 'badge-danger'
  return 'badge-gray'
}

export function platformBadgeClass(platform: string): string {
  const normalized = normalizePlatform(platform)
  if (normalized === 'openai') return 'badge-primary'
  if (normalized === 'anthropic') return 'badge-warning'
  if (normalized === 'gemini') return 'badge-success'
  return 'badge-gray'
}

export function platformLabel(platform: string): string {
  const normalized = normalizePlatform(platform)
  if (normalized === 'openai') return 'OpenAI'
  if (normalized === 'anthropic') return 'Claude'
  if (normalized === 'gemini') return 'Gemini'
  return platform || '-'
}

export function typeShortLabel(type: string): string {
  const normalized = (type || '').toLowerCase()
  if (normalized.includes('api') || normalized.includes('key')) return 'Key'
  if (normalized.includes('oauth')) return 'OAuth'
  return type || '-'
}

export function normalizeType(type: string): string {
  const normalized = (type || '').toLowerCase()
  if (normalized.includes('oauth')) return 'oauth'
  if (normalized.includes('api') || normalized.includes('key')) return 'apikey'
  if (normalized.includes('setup')) return 'setup-token'
  return normalized
}

export function normalizePlatform(platform: string): string {
  const value = (platform || '').toLowerCase()
  if (value.includes('anthropic') || value.includes('claude')) return 'anthropic'
  if (value.includes('gemini') || value.includes('google')) return 'gemini'
  if (value.includes('openai') || value.includes('gpt')) return 'openai'
  return value
}

export function providerLabel(value?: string): string {
  const provider = (value || 'mixed').toLowerCase()
  if (provider.includes('anthropic') || provider.includes('claude')) return 'Anthropic / Claude'
  if (provider.includes('gemini') || provider.includes('google')) return 'Gemini'
  if (provider.includes('openai') || provider.includes('gpt')) return 'OpenAI'
  return provider === 'mixed' ? '混合渠道' : value || '混合渠道'
}
