(function() {
  var App;

  App = window.App = angular.module('subfwd', []);

  App.controller('ManagerController', function($rootScope, $scope, $window, $http, $timeout, console, hmac, storage) {
    var cancel, scope;
    scope = $rootScope.mgr = $scope;
    scope.hmac = hmac;
    scope.id = storage.get("id");
    scope.pass = storage.get("pass");
    scope.subdomains = [];
    cancel = null;
    scope.$watch("id", function(id) {
      storage.set("id", id);
      if (cancel) {
        $timeout.cancel(cancel);
      }
      if (!/\S\.\S/.test(id)) {
        return;
      }
      cancel = $timeout(scope.statusGet, 1000);
    });
    scope.$watch("pass", function(pass) {
      return storage.set("pass", pass);
    });
    scope.statusLoading = false;
    scope.statusGet = function() {
      scope.statusErr = "";
      scope.statusLoading = true;
      $http.post("/status", {
        ID: scope.id,
        Pass: scope.pass
      }).success(function(status) {
        if (status) {
          return scope.status = status;
        }
      }).error(function(err) {
        console.error(err);
        return scope.statusErr = err;
      })["finally"](function() {
        return scope.statusLoading = false;
      });
    };
    return scope.logout = function() {
      scope.status = null;
      storage.del("id");
      scope.id = "";
      storage.del("pass");
      scope.pass = "";
    };
  });

  App.factory('console', function($window) {
    var c, str;
    ga('create', 'UA-38709761-15', 'auto');
    ga('send', 'pageview');
    setInterval((function() {
      return ga('send', 'event', 'Ping');
    }), 60 * 1000);
    str = function(args) {
      return Array.prototype.slice.call(args).join(' ');
    };
    c = $window.console;
    return {
      log: function() {
        c.log.apply(c, arguments);
        return ga('send', 'event', 'Log', str(arguments));
      },
      error: function() {
        c.error.apply(c, arguments);
        return ga('send', 'event', 'Error', str(arguments));
      }
    };
  });

  App.factory('hmac', function() {
    return function(msg) {
      var hmac;
      hmac = CryptoJS.algo.HMAC.create(CryptoJS.algo.SHA256, "subfwd.com");
      hmac.update(msg || "");
      return hmac.finalize().toString();
    };
  });

  App.factory('storage', function() {
    var storage, wrap;
    wrap = function(ns, fn) {
      return function() {
        arguments[0] = [ns, arguments[0]].join('-');
        return fn.apply(null, arguments);
      };
    };
    storage = {
      create: function(ns) {
        var fn, k, s;
        s = {};
        for (k in storage) {
          fn = storage[k];
          s[k] = wrap(ns, fn);
        }
        return s;
      },
      get: function(key) {
        var str;
        str = localStorage.getItem(key);
        if (str && str.substr(0, 4) === "J$ON") {
          return JSON.parse(str.substr(4));
        }
        return str;
      },
      set: function(key, val) {
        if (typeof val === 'object') {
          val = "J$ON" + (JSON.stringify(val));
        }
        return localStorage.setItem(key, val);
      },
      del: function(key) {
        return localStorage.removeItem(key);
      }
    };
    return window.storage = storage;
  });

  App.factory('$exceptionHandler', function(console) {
    return function(exception, cause) {
      console.error('Exception caught\n', exception.stack || exception);
      if (cause) {
        return console.error('Exception cause', cause);
      }
    };
  });

  App.run(function($rootScope, console) {
    window.root = $rootScope;
    console.log('Init');
    $("#loading-cover").fadeOut(500, function() {
      return $(this).remove();
    });
  });

}).call(this);
