App.factory 'console', ($window) ->

  ga('create', 'UA-38709761-15', 'auto')
  ga('send', 'pageview')

  setInterval (-> ga 'send', 'event', 'Ping'), 60*1000

  str = (args) ->
    Array::slice.call(args).join(' ')

  c = $window.console

  log: ->
    c.log.apply c, arguments
    ga 'send', 'event', 'Log', str arguments

  error: ->
    c.error.apply c, arguments
    ga 'send', 'event', 'Error', str arguments
