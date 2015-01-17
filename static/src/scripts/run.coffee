
App.factory '$exceptionHandler', (console) -> (exception, cause) ->
  console.error 'Exception caught\n', exception.stack or exception
  console.error 'Exception cause', cause if cause

App.run ($rootScope, console) ->
  window.root = $rootScope
  $rootScope.screen = "manager"
  console.log 'Init'
  return
