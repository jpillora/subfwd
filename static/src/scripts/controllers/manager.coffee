App.controller 'ManagerController', ($rootScope, $scope, $window, $http, $timeout, console, storage) ->
	scope = $rootScope.mgr = $scope
	scope.domain = ""

	scope.$watch "domain", ->
		scope.setupOk = false

	scope.loading = false
	scope.setup = ->
		scope.setupOk = false
		scope.setupErr = ""
		scope.loading = true
		$http.get("/setup?domain=#{scope.domain}")
			.success(->
				ga 'send', 'event', 'Setup OK', scope.domain
				scope.setupOk = true
			).error((err)->
				ga 'send', 'event', 'Setup Error', scope.domain, err
				console.error err
				scope.setupErr = err
			).finally(->
				scope.loading = false
			)
		return