const CONFIG_KEY = 'adminPlusOperatorConfig'
const LAST_CAPTURE_RESULT_KEY = 'adminPlusLastCaptureResult'
const DEFAULT_CONFIG_PATH = 'config/default-config.json'

export async function loadConfig() {
  const stored = await chrome.storage.local.get(CONFIG_KEY)
  const config = stored[CONFIG_KEY] || {}
  const defaultConfig = await loadDefaultConfig()
  if (!config.deviceID) {
    config.deviceID = `admin-plus-chrome-${crypto.randomUUID()}`
    await saveConfig(config)
  }
  return {
    baseURL: config.baseURL || defaultConfig.baseURL || '',
    token: config.token || '',
    deviceID: config.deviceID,
    taskTypes: Array.isArray(config.taskTypes) ? config.taskTypes : [],
    connectedAt: config.connectedAt || ''
  }
}

export async function saveConfig(config) {
  await chrome.storage.local.set({
    [CONFIG_KEY]: {
      baseURL: config.baseURL || '',
      token: config.token || '',
      deviceID: config.deviceID || `admin-plus-chrome-${crypto.randomUUID()}`,
      taskTypes: Array.isArray(config.taskTypes) ? config.taskTypes : [],
      connectedAt: config.connectedAt || ''
    }
  })
}

export async function saveBaseURL(baseURL) {
  const normalized = normalizeBaseURL(baseURL)
  const config = await loadConfig()
  const changed = normalizeBaseURL(config.baseURL) !== normalized
  await saveConfig({
    ...config,
    baseURL: normalized,
    token: changed ? '' : config.token,
    connectedAt: changed ? '' : config.connectedAt
  })
}

export async function loadLastCaptureResult() {
  const stored = await chrome.storage.local.get(LAST_CAPTURE_RESULT_KEY)
  return stored[LAST_CAPTURE_RESULT_KEY] || null
}

export async function saveLastCaptureResult(result) {
  await chrome.storage.local.set({
    [LAST_CAPTURE_RESULT_KEY]: {
      status: result.status || 'failed',
      message: result.message || '',
      supplier: result.supplier || '',
      host: result.host || '',
      taskID: result.taskID || 0,
      summary: result.summary || {},
      recordedAt: result.recordedAt || new Date().toISOString()
    }
  })
}

async function loadDefaultConfig() {
  try {
    const response = await fetch(chrome.runtime.getURL(DEFAULT_CONFIG_PATH), {
      cache: 'no-store'
    })
    if (!response.ok) return {}
    const config = await response.json()
    return {
      baseURL: normalizeBaseURL(config.baseURL)
    }
  } catch {
    return {}
  }
}

export function normalizeBaseURL(value) {
  let raw = String(value || '').trim().replace(/\/+$/, '')
  if (!raw) return ''
  if (!/^[a-z][a-z\d+\-.]*:\/\//i.test(raw)) {
    raw = `${usesLocalHTTPByDefault(raw) ? 'http' : 'https'}://${raw}`
  }
  try {
    const url = new URL(raw)
    if (!['http:', 'https:'].includes(url.protocol)) return ''
    return url.origin
  } catch {
    return ''
  }
}

function usesLocalHTTPByDefault(value) {
  const raw = String(value || '').trim().toLowerCase()
  return raw.startsWith('localhost') ||
    raw.startsWith('127.') ||
    raw.startsWith('0.0.0.0') ||
    raw.startsWith('[::1]') ||
    raw.startsWith('::1')
}
