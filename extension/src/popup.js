import { loadConfig, saveConfig } from './lib/storage.js'

const baseURLInput = document.querySelector('#baseURL')
const tokenInput = document.querySelector('#token')
const deviceEl = document.querySelector('#device')
const statusEl = document.querySelector('#status')
const taskTypeInputs = Array.from(document.querySelectorAll('input[type="checkbox"]'))

document.querySelector('#save').addEventListener('click', save)
document.querySelector('#importToken').addEventListener('click', importToken)
document.querySelector('#run').addEventListener('click', run)

init().catch(showError)

async function init() {
  const config = await loadConfig()
  renderConfig(config)
  writeStatus('ready')
}

function renderConfig(config) {
  baseURLInput.value = config.baseURL || ''
  tokenInput.value = config.token || ''
  deviceEl.textContent = config.deviceID
  for (const input of taskTypeInputs) {
    input.checked = config.taskTypes.length === 0 || config.taskTypes.includes(input.value)
  }
}

async function save() {
  const config = await currentConfig()
  await saveConfig(config)
  renderConfig(config)
  writeStatus('saved')
}

async function importToken() {
  const response = await chrome.runtime.sendMessage({ type: 'config:import-token' })
  if (!response.ok) throwMessage(response)
  renderConfig(response.result)
  writeStatus('token imported')
}

async function run() {
  await save()
  writeStatus('claiming task...')
  const response = await chrome.runtime.sendMessage({ type: 'task:claim-run' })
  if (!response.ok) throwMessage(response)
  writeStatus(JSON.stringify(response.result, null, 2))
}

async function currentConfig() {
  const existing = await loadConfig()
  return {
    ...existing,
    baseURL: baseURLInput.value.trim(),
    token: tokenInput.value.trim(),
    taskTypes: taskTypeInputs.filter((input) => input.checked).map((input) => input.value)
  }
}

function throwMessage(response) {
  const error = new Error(response.error?.message || 'operation failed')
  error.reason = response.error?.reason
  throw error
}

function showError(error) {
  writeStatus(`${error.reason || 'ERROR'}: ${error.message || String(error)}`)
}

function writeStatus(message) {
  statusEl.textContent = message
}
