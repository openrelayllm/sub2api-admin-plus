const CONFIG_KEY = 'adminPlusOperatorConfig'

export async function loadConfig() {
  const stored = await chrome.storage.local.get(CONFIG_KEY)
  const config = stored[CONFIG_KEY] || {}
  if (!config.deviceID) {
    config.deviceID = `admin-plus-chrome-${crypto.randomUUID()}`
    await saveConfig(config)
  }
  return {
    baseURL: config.baseURL || '',
    token: config.token || '',
    deviceID: config.deviceID,
    taskTypes: Array.isArray(config.taskTypes) ? config.taskTypes : []
  }
}

export async function saveConfig(config) {
  await chrome.storage.local.set({
    [CONFIG_KEY]: {
      baseURL: config.baseURL || '',
      token: config.token || '',
      deviceID: config.deviceID || `admin-plus-chrome-${crypto.randomUUID()}`,
      taskTypes: Array.isArray(config.taskTypes) ? config.taskTypes : []
    }
  })
}
