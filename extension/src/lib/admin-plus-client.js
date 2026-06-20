export class AdminPlusClient {
  constructor(config) {
    this.baseURL = trimTrailingSlash(config.baseURL || '')
    this.token = config.token || ''
  }

  ready() {
    return this.baseURL !== '' && this.token !== ''
  }

  async claimTask(deviceID, types = []) {
    return this.request('/api/v1/admin-plus/extension/tasks/claim', {
      method: 'POST',
      body: {
        device_id: deviceID,
        types,
        lease_ttl_seconds: 300
      }
    })
  }

  async heartbeat(task, leaseTTLSeconds = 300) {
    return this.request(`/api/v1/admin-plus/extension/tasks/${task.id}/heartbeat`, {
      method: 'POST',
      body: {
        device_id: task.device_id,
        lease_token: task.lease_token,
        lease_ttl_seconds: leaseTTLSeconds
      }
    })
  }

  async browserCredential(task) {
    return this.request(`/api/v1/admin-plus/extension/tasks/${task.id}/browser-credential`, {
      method: 'POST',
      body: {
        device_id: task.device_id,
        lease_token: task.lease_token
      }
    })
  }

  async complete(task, result) {
    return this.request(`/api/v1/admin-plus/extension/tasks/${task.id}/complete`, {
      method: 'POST',
      body: {
        device_id: task.device_id,
        lease_token: task.lease_token,
        result
      }
    })
  }

  async fail(task, errorCode, errorMessage) {
    return this.request(`/api/v1/admin-plus/extension/tasks/${task.id}/fail`, {
      method: 'POST',
      body: {
        device_id: task.device_id,
        lease_token: task.lease_token,
        error_code: errorCode,
        error_message: errorMessage
      }
    })
  }

  async request(path, options = {}) {
    if (!this.ready()) {
      throw new Error('Admin Plus URL and token are required')
    }
    const response = await fetch(`${this.baseURL}${path}`, {
      method: options.method || 'GET',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${this.token}`
      },
      body: options.body === undefined ? undefined : JSON.stringify(options.body)
    })
    const text = await response.text()
    const json = parseJSON(text)
    if (!response.ok || json.code !== 0) {
      const reason = json.reason || `HTTP_${response.status}`
      const message = json.message || text || 'Admin Plus request failed'
      const error = new Error(message)
      error.reason = reason
      throw error
    }
    return json.data
  }
}

export function trimTrailingSlash(value) {
  return String(value || '').replace(/\/+$/, '')
}

function parseJSON(text) {
  try {
    return JSON.parse(text)
  } catch {
    throw new Error('Admin Plus did not return JSON')
  }
}
