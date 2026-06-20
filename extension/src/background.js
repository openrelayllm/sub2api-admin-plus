import { AdminPlusClient } from './lib/admin-plus-client.js'
import { loadConfig, saveConfig } from './lib/storage.js'

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  handleMessage(message)
    .then((result) => sendResponse({ ok: true, result }))
    .catch((error) => sendResponse({
      ok: false,
      error: {
        reason: error.reason || 'EXTENSION_ERROR',
        message: error.message || String(error)
      }
    }))
  return true
})

async function handleMessage(message) {
  switch (message?.type) {
    case 'config:get':
      return loadConfig()
    case 'config:save':
      await saveConfig(message.config || {})
      return loadConfig()
    case 'config:import-token':
      return importTokenFromActiveTab()
    case 'task:claim-run':
      return claimAndRunTask()
    default:
      throw new Error('Unsupported extension message')
  }
}

async function importTokenFromActiveTab() {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true })
  if (!tab?.id) {
    throw new Error('No active tab')
  }
  const injected = await chrome.scripting.executeScript({
    target: { tabId: tab.id },
    func: () => ({
      token: window.localStorage.getItem('auth_token') || '',
      origin: window.location.origin
    })
  })
  const result = injected[0]?.result || {}
  if (!result.token) {
    throw new Error('Current tab does not contain Admin Plus auth_token')
  }
  const config = await loadConfig()
  config.token = result.token
  if (!config.baseURL) {
    config.baseURL = result.origin
  }
  await saveConfig(config)
  return loadConfig()
}

async function claimAndRunTask() {
  const config = await loadConfig()
  const client = new AdminPlusClient(config)
  const task = await client.claimTask(config.deviceID, config.taskTypes)
  await client.heartbeat(task)
  const credential = await client.browserCredential(task)
  const result = await runTaskInSupplierTab(task, credential)
  if (!result.ok) {
    await client.fail(task, result.error_code || 'SUPPLIER_PAGE_UNSUPPORTED', result.error_message || 'supplier page is not supported')
    return { task, credential: redactCredential(credential), status: 'failed', result }
  }
  await client.complete(task, result.result)
  return { task, credential: redactCredential(credential), status: 'succeeded', result: result.result }
}

async function runTaskInSupplierTab(task, credential) {
  if (!credential.dashboard_url) {
    return { ok: false, error_code: 'SUPPLIER_DASHBOARD_URL_REQUIRED', error_message: 'supplier dashboard url is required' }
  }
  const tab = await chrome.tabs.create({ url: credential.dashboard_url, active: false })
  try {
    for (let attempt = 0; attempt < 4; attempt++) {
      await waitForTabComplete(tab.id)
      const response = await sendTaskToContent(tab.id, task, credential)
      if (response?.status === 'login_applied' || response?.status === 'login_submitted') {
        await sleep(1800)
        continue
      }
      return response
    }
    return { ok: false, error_code: 'LOGIN_OR_PAGE_STUCK', error_message: 'login did not reach a supported supplier page' }
  } finally {
    await chrome.tabs.remove(tab.id).catch(() => {})
  }
}

function sendTaskToContent(tabId, task, credential) {
  return chrome.tabs.sendMessage(tabId, {
    type: 'admin-plus:run-task',
    task,
    credential
  })
}

function waitForTabComplete(tabId) {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      chrome.tabs.onUpdated.removeListener(listener)
      reject(new Error('supplier page load timed out'))
    }, 30000)
    const listener = (updatedTabId, changeInfo) => {
      if (updatedTabId !== tabId || changeInfo.status !== 'complete') return
      clearTimeout(timer)
      chrome.tabs.onUpdated.removeListener(listener)
      resolve()
    }
    chrome.tabs.onUpdated.addListener(listener)
    chrome.tabs.get(tabId, (tab) => {
      if (chrome.runtime.lastError) {
        clearTimeout(timer)
        chrome.tabs.onUpdated.removeListener(listener)
        reject(new Error(chrome.runtime.lastError.message))
        return
      }
      if (tab.status === 'complete') {
        clearTimeout(timer)
        chrome.tabs.onUpdated.removeListener(listener)
        resolve()
      }
    })
  })
}

function redactCredential(credential) {
  return {
    supplier_id: credential.supplier_id,
    supplier_name: credential.supplier_name,
    supplier_type: credential.supplier_type,
    dashboard_url: credential.dashboard_url,
    api_base_url: credential.api_base_url,
    username_configured: Boolean(credential.username),
    password_configured: Boolean(credential.password),
    token_configured: Boolean(credential.token)
  }
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}
