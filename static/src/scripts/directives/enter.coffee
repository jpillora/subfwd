App.directive "enter", ->
  (scope, element, attrs) ->
    element.bind "keydown keypress", (event) ->
      return  if event.which isnt 13
      scope.$apply ->
        scope.$eval attrs.enter
        return
      event.preventDefault()
    return
