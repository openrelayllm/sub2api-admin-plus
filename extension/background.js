chrome.runtime.onMessage.addListener(function (message, sender, sendResponse) {
  handleBackgroundMessage(message, sender).then(function (result) {
    sendResponse({ ok: true, result: result })
  }).catch(function (error) {
    sendResponse({
      ok: false,
      error: {
        reason: error && error.reason ? error.reason : 'EXTENSION_ERROR',
        message: error && error.message ? error.message : String(error)
      }
    })
  })
  return true
})

function handleBackgroundMessage(message, sender) {
  return ensureBackgroundAppLoaded().then(function () {
    if (typeof self.adminPlusHandleMessage !== 'function') {
      throw new Error('extension background app is not available')
    }
    return self.adminPlusHandleMessage(message, sender)
  })
}

function ensureBackgroundAppLoaded() {
  if (typeof self.adminPlusHandleMessage !== 'function') {
    try {
      importScripts('src/background-app.js')
    } catch (error) {
      var message = error && error.message ? error.message : String(error)
      var wrapped = new Error('failed to load extension background app: ' + message)
      wrapped.reason = 'BACKGROUND_APP_LOAD_FAILED'
      wrapped.cause = error
      return Promise.reject(wrapped)
    }
  }
  return Promise.resolve()
}
