# App.factory 'hmac', () ->
#   (key) ->
#     return "" unless key
#     hmac = CryptoJS.algo.HMAC.create CryptoJS.algo.SHA256, key
#     hmac.update "SubFwd requires you to hash this sentence with the correct key."
#     return hmac.finalize().toString()