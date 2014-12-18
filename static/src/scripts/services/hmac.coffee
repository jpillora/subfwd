App.factory 'hmac', () ->
  (msg) ->
    hmac = CryptoJS.algo.HMAC.create CryptoJS.algo.SHA256, "subfwd.com"
    hmac.update msg or ""
    hmac.finalize().toString()