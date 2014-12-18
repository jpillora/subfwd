App.controller 'ManagerController', ($rootScope, $scope, $window, $http, $timeout, console, hmac, storage) ->
	scope = $rootScope.mgr = $scope
	scope.hmac = hmac
	scope.id = storage.get "id"
	scope.pass = storage.get "pass"
	scope.subdomains = []

	cancel = null
	scope.$watch "id", (id) ->
		storage.set "id", id
		$timeout.cancel(cancel) if cancel
		return unless /\S\.\S/.test(id)
		cancel = $timeout scope.statusGet, 1000
		return

	scope.$watch "pass", (pass) ->
		storage.set "pass", pass

	scope.statusLoading = false
	scope.statusGet = ->
		scope.statusErr = ""
		scope.statusLoading = true
		$http.post("/status", {ID:scope.id, Pass:scope.pass})
			.success((status)-> scope.status = status if status)
			.error((err)-> console.error err; scope.statusErr = err)
			.finally(->scope.statusLoading = false)
		return

	scope.logout = ->
		scope.status = null
		storage.del "id"
		scope.id = ""
		storage.del "pass"
		scope.pass = ""
		return