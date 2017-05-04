
App.factory '$exceptionHandler', (console) -> (exception, cause) ->
  console.error 'Exception caught\n', exception.stack or exception
  console.error 'Exception cause', cause if cause

App.run ($rootScope, console, $http) ->
  scope = window.root = $rootScope
  console.log 'Init'
  scope.onHeroku = false
  scope.uptime = null
  scope.forwards = 0
  $http.get("/stats")
    .success((data)->
      scope.onHeroku = data.Heroku
      scope.uptime = data.Uptime
      scope.forwards = data.Success
    )
  return
