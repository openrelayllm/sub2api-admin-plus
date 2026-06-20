(() => {
  chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
    if (message?.type !== 'admin-plus:run-task') return false
    Promise.resolve(runTask(message.task, message.credential))
      .then((result) => sendResponse(result))
      .catch((error) => sendResponse({
        ok: false,
        error_code: error.reason || 'CONTENT_SCRIPT_ERROR',
        error_message: error.message || String(error)
      }))
    return true
  })

  async function runTask(task, credential) {
    const login = ensureLogin(credential)
    if (login) return login

    switch (task.type) {
      case 'fetch_rates':
        return collectRates()
      case 'fetch_balance':
        return collectBalance()
      case 'fetch_promotions':
        return collectPromotions()
      case 'export_bills':
        return collectBills()
      case 'fetch_health':
        return collectHealth()
      default:
        return fail('UNSUPPORTED_TASK_TYPE', `unsupported task type: ${task.type}`)
    }
  }

  function ensureLogin(credential) {
    const passwordInput = document.querySelector('input[type="password"]')
    const loginLike = passwordInput || /login|signin|auth/i.test(location.pathname)
    if (!loginLike) return null

    if (credential.token) {
      window.localStorage.setItem('auth_token', credential.token)
      window.localStorage.setItem('token_expires_at', String(Date.now() + 24 * 60 * 60 * 1000))
      location.reload()
      return { ok: false, status: 'login_applied' }
    }

    if (!credential.username || !credential.password || !passwordInput) {
      return fail('LOGIN_CREDENTIAL_REQUIRED', 'supplier login page requires username and password')
    }

    const userInput = document.querySelector('input[type="email"], input[name*="email" i], input[name*="user" i], input[type="text"]')
    if (!userInput) {
      return fail('LOGIN_FORM_UNSUPPORTED', 'supplier login form username input was not found')
    }
    setInputValue(userInput, credential.username)
    setInputValue(passwordInput, credential.password)
    const submit = document.querySelector('button[type="submit"], input[type="submit"], button')
    if (!submit) {
      return fail('LOGIN_FORM_UNSUPPORTED', 'supplier login form submit button was not found')
    }
    submit.click()
    return { ok: false, status: 'login_submitted' }
  }

  function collectRates() {
    const rows = tableRows()
    const entries = []
    for (const row of rows) {
      const cells = row.cells.map((value) => value.trim()).filter(Boolean)
      if (cells.length < 2) continue
      const model = findModel(cells)
      const price = findMoneyOrNumber(cells)
      if (!model || price === null) continue
      entries.push({
        model,
        billing_mode: inferBillingMode(cells),
        price_item: inferPriceItem(cells),
        unit: inferUnit(cells),
        currency: inferCurrency(cells),
        price_micros: Math.round(price * 1000000),
        raw_payload: { cells, url: location.href }
      })
    }
    if (entries.length === 0) {
      return fail('RATE_TABLE_NOT_FOUND', 'no supported rate table rows were found')
    }
    return ok({
      source: 'chrome',
      captured_at: new Date().toISOString(),
      threshold_percent: 1,
      entries
    })
  }

  function collectBalance() {
    const text = visibleText()
    const balance = findLabeledAmount(text, ['balance', '余额', '剩余', '可用'])
    if (!balance) {
      return fail('BALANCE_NOT_FOUND', 'no supported balance value was found')
    }
    return ok({
      source: 'chrome',
      captured_at: new Date().toISOString(),
      runtime_status: 'monitor_only',
      balance_cents: Math.round(balance.amount * 100),
      currency: balance.currency,
      raw_payload: { url: location.href, evidence: balance.evidence }
    })
  }

  function collectPromotions() {
    const text = visibleText()
    const keywords = ['优惠', '折扣', '赠送', 'bonus', 'discount', 'promotion', 'recharge']
    const matched = text
      .split(/\n+/)
      .map((line) => line.trim())
      .filter((line) => line && keywords.some((keyword) => line.toLowerCase().includes(keyword.toLowerCase())))
      .slice(0, 20)
    if (matched.length === 0) {
      return fail('PROMOTION_PAGE_UNSUPPORTED', 'no supported promotion entries were found')
    }
    return ok({
      source: 'chrome',
      promotions: matched.map((line) => ({
        type: inferPromotionType(line),
        title: line.slice(0, 120),
        description: line,
        currency: 'CNY',
        runtime_status: 'monitor_only',
        balance_cents: 0,
        raw_payload: { url: location.href, line }
      }))
    })
  }

  function collectBills() {
    const rows = tableRows()
    const lines = []
    for (const row of rows) {
      const cells = row.cells.map((value) => value.trim()).filter(Boolean)
      const cost = findMoneyOrNumber(cells)
      const model = findModel(cells)
      const requestID = cells.find((value) => /req|chatcmpl|cmpl|[a-f0-9-]{16,}/i.test(value)) || ''
      if (cost === null || !model) continue
      lines.push({
        external_bill_id: requestID || `${location.host}-${row.index}`,
        external_request_id: requestID,
        model,
        currency: inferCurrency(cells),
        cost_cents: Math.round(cost * 100),
        input_tokens: findTokenValue(cells, ['input', 'prompt', '输入']),
        output_tokens: findTokenValue(cells, ['output', 'completion', '输出']),
        started_at: findDate(cells) || new Date().toISOString(),
        raw_payload: { cells, url: location.href }
      })
    }
    if (lines.length === 0) {
      return fail('BILL_TABLE_NOT_FOUND', 'no supported billing rows were found')
    }
    return ok({ source: 'chrome', lines })
  }

  function collectHealth() {
    const text = visibleText()
    const concurrency = findLabeledPair(text, ['并发', 'concurrency'])
    if (!concurrency) {
      return fail('HEALTH_METRICS_NOT_FOUND', 'no supported health metrics were found')
    }
    return ok({
      source: 'chrome',
      captured_at: new Date().toISOString(),
      model: 'unknown',
      first_token_latency_ms: 0,
      total_latency_ms: 0,
      status_code: 200,
      observed_concurrency: concurrency.current,
      available_concurrency: Math.max(0, concurrency.limit - concurrency.current),
      concurrency_limit: concurrency.limit,
      concurrency_saturation_percent: concurrency.limit > 0 ? (concurrency.current / concurrency.limit) * 100 : 0,
      raw_payload: { url: location.href, evidence: concurrency.evidence }
    })
  }

  function tableRows() {
    return Array.from(document.querySelectorAll('tr')).map((tr, index) => ({
      index,
      cells: Array.from(tr.querySelectorAll('th,td')).map((cell) => cell.textContent || '')
    }))
  }

  function visibleText() {
    return (document.body?.innerText || '').replace(/\r/g, '\n')
  }

  function findModel(cells) {
    return cells.find((value) => /\b(gpt|claude|gemini|o[0-9]|text-|embedding|rerank|dall-e|whisper)[\w.-]*\b/i.test(value)) || ''
  }

  function findMoneyOrNumber(values) {
    for (const value of values) {
      const match = value.replace(/,/g, '').match(/(?:[$￥¥]|USD|CNY|RMB)?\s*(-?\d+(?:\.\d+)?)/i)
      if (!match) continue
      const parsed = Number(match[1])
      if (Number.isFinite(parsed)) return parsed
    }
    return null
  }

  function findLabeledAmount(text, labels) {
    const lines = text.split(/\n+/)
    for (const line of lines) {
      const normalized = line.trim()
      if (!labels.some((label) => normalized.toLowerCase().includes(label.toLowerCase()))) continue
      const amount = findMoneyOrNumber([normalized])
      if (amount === null) continue
      return {
        amount,
        currency: inferCurrency([normalized]),
        evidence: normalized.slice(0, 300)
      }
    }
    return null
  }

  function findLabeledPair(text, labels) {
    const lines = text.split(/\n+/)
    for (const line of lines) {
      const normalized = line.trim()
      if (!labels.some((label) => normalized.toLowerCase().includes(label.toLowerCase()))) continue
      const match = normalized.match(/(\d+)\s*\/\s*(\d+)/)
      if (!match) continue
      return {
        current: Number(match[1]),
        limit: Number(match[2]),
        evidence: normalized.slice(0, 300)
      }
    }
    return null
  }

  function inferBillingMode(cells) {
    return cells.some((value) => /request|请求/i.test(value)) ? 'request' : 'token'
  }

  function inferPriceItem(cells) {
    const text = cells.join(' ').toLowerCase()
    if (/output|completion|输出/.test(text)) return 'output'
    if (/input|prompt|输入/.test(text)) return 'input'
    return 'mixed'
  }

  function inferUnit(cells) {
    const text = cells.join(' ').toLowerCase()
    if (/1m|million|百万/.test(text)) return '1m_tokens'
    if (/1k|thousand|千/.test(text)) return '1k_tokens'
    if (/request|请求/.test(text)) return 'request'
    return 'token'
  }

  function inferCurrency(cells) {
    const text = cells.join(' ')
    if (/USD|\$/.test(text)) return 'USD'
    if (/CNY|RMB|￥|¥|元/.test(text)) return 'CNY'
    return 'USD'
  }

  function inferPromotionType(line) {
    const lower = line.toLowerCase()
    if (/bonus|赠送|返/.test(lower)) return 'recharge_bonus'
    if (/discount|折扣|优惠/.test(lower)) return 'rate_discount'
    return 'other'
  }

  function findTokenValue(cells, labels) {
    for (const cell of cells) {
      const lower = cell.toLowerCase()
      if (!labels.some((label) => lower.includes(label.toLowerCase()))) continue
      const match = cell.replace(/,/g, '').match(/(\d+)/)
      if (match) return Number(match[1])
    }
    return 0
  }

  function findDate(cells) {
    for (const cell of cells) {
      const date = new Date(cell)
      if (!Number.isNaN(date.getTime())) return date.toISOString()
    }
    return ''
  }

  function setInputValue(input, value) {
    const setter = Object.getOwnPropertyDescriptor(input.constructor.prototype, 'value')?.set
    setter?.call(input, value)
    input.dispatchEvent(new Event('input', { bubbles: true }))
    input.dispatchEvent(new Event('change', { bubbles: true }))
  }

  function ok(result) {
    return { ok: true, result }
  }

  function fail(errorCode, errorMessage) {
    return { ok: false, error_code: errorCode, error_message: errorMessage }
  }
})()
