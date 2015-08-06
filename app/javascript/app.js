var indexOf = function(needle) {
    if(typeof Array.prototype.indexOf === 'function') {
        indexOf = Array.prototype.indexOf;
    } else {
        indexOf = function(needle) {
            var i = -1, index = -1;

            for(i = 0; i < this.length; i++) {
                if(this[i] === needle) {
                    index = i;
                    break;
                }
            }

            return index;
        };
    }

    return indexOf.call(this, needle);
};

function sleep(milliseconds) {
  var start = new Date().getTime();
  for (var i = 0; i < 1e7; i++) {
    if ((new Date().getTime() - start) > milliseconds){
      break;
    }
  }
}

Array.prototype.remove = function() {
    var what, a = arguments, L = a.length, ax;
    while (L && this.length) {
        what = a[--L];
        while ((ax = this.indexOf(what)) !== -1) {
            this.splice(ax, 1);
        }
    }
    return this;
};

(function() {
  var app = angular.module('S3Bench', ['ngAnimate', 'chart.js']);

  app.value('loadingService', {
    loadingCount: 0,
    isLoading: function() { return loadingCount > 0; },
    requested: function() { loadingCount += 1; },
    responded: function() { loadingCount -= 1; }
  });

  app.factory('loadingInterceptor', ['$q', 'loadingService', function($q, loadingService) {
    return {
      request: function(config) {
        loadingService.requested();
        return config;
      },
      response: function(response) {
        loadingService.responded();
        return response;
      },
      responseError: function(rejection) {
        loadingService.responded();
        return $q.reject(rejection);
      },
    }
  }]);

  app.config(["$httpProvider", function ($httpProvider) {
    $httpProvider.interceptors.push('loadingInterceptor');
  }]);

  app.config(['ChartJsProvider', function (ChartJsProvider) {
    // Configure all charts
    ChartJsProvider.setOptions({
      animation: false,
      showTooltips: false//,
      //scaleOverride: true,
      //scaleFontSize: 6
    });
    // Configure all line charts
    ChartJsProvider.setOptions('Line', {
      datasetFill: false
    });
  }])

  app.controller('S3BenchController', ['$http', '$animate', '$scope', 'loadingService', function($http, $animate, $scope, loadingService) {
    loadingCount = 0;
    $scope.s3bench.noreset = false;
    $scope.s3bench.stopped = false;
    this.changestate = function(action) {
      $scope.s3bench.stopped = true;
      if($scope.s3bench.requests != null) {
        for(i=0; i<$scope.s3bench.requests.length; i++) {
          clearTimeout($scope.s3bench.requests[i]);
        }
      }
      if(action == "start") {
        $scope.s3bench.started = {};
        $scope.s3bench.requests = [];
        $scope.s3bench.threads = 1;
        operations = [];
        operations["listkeys"] = "List keys";
        operations["putupdate"] = "Put";
        operations["putcreate"] = "Put";
        operations["getsame"] = "Get";
        operations["getrandom"] = "Get";
        operations["deleterandom"] = "Delete";
        $scope.s3bench.responsetime = {};
        for(var key in operations) {
          $scope.s3bench.responsetime[key] = {}
          $scope.s3bench.responsetime[key]["series"] = []
          $scope.s3bench.responsetime[key]["labels"] = [];
          $scope.s3bench.responsetime[key]["data"] = [];
          $scope.s3bench.responsetime[key]["series"][0] = operations[key];
          $scope.s3bench.responsetime[key]["data"][0] = [];
          $scope.s3bench.statuscode = {};
        }
        $scope.s3bench.noreset = true;
        $scope.s3bench.stopped = false;
        for(i=0; i<$scope.s3bench.threads; i++) {
          for(var key in operations) {
            requests($http, $scope, operations, key);
          }
        }
      }
    }
  }]);

  app.directive("s3benchMessage", function() {
    return {
      restrict: 'E',
      templateUrl: "app/html/s3bench-message.html"
    };
  });

  app.directive("s3benchDashboard", function() {
    return {
      restrict: 'E',
      templateUrl: "app/html/s3bench-dashboard.html"
    };
  });
})();

function requests($http, $scope, operations, key) {
  sample = $scope.s3bench.sample;
  $http.post('/api/v1/requests/' + key, {
    accesskey: $scope.s3bench.accesskey,
    secretkey: $scope.s3bench.secretkey,
    endpoint: $scope.s3bench.endpoint,
    bucket: $scope.s3bench.bucket,
    }).success(function(data) {
    if(data[key]["statuscode"] != -1) {
      $scope.s3bench.responsetime[key]["labels"] = $scope.s3bench.responsetime[key]["labels"].concat(data["timestamp"]);
      if($scope.s3bench.responsetime[key]["labels"].length > sample) {
        $scope.s3bench.responsetime[key]["labels"].shift();
      }
      $scope.s3bench.responsetime[key]["data"][0] = $scope.s3bench.responsetime[key]["data"][0].concat(data[key]["duration"]);
      if($scope.s3bench.responsetime[key]["data"][0].length > sample) {
        $scope.s3bench.responsetime[key]["data"][0].shift();
      }
    }
    $scope.s3bench.statuscode[data["timestamp"] + "-" + key] = data;
    if(!$scope.s3bench.stopped) {
      request = setTimeout(function () {
        $scope.s3bench.started[key] = true;
        if(Object.keys($scope.s3bench.started).length >= 6) {
          $scope.s3bench.noreset = false;
        }
        requests($http, $scope, operations, key);
      }, $scope.s3bench.delay);
      $scope.s3bench.requests.push(request);
    }
  }).
  error(function(data, status, headers, config) {
    if(!$scope.s3bench.stopped) {
      request = setTimeout(function () {
        $scope.s3bench.started[key] = true;
        if(Object.keys($scope.s3bench.started).length >= 6) {
          $scope.s3bench.noreset = false;
        }
        requests($http, $scope, operations, key);
      }, $scope.s3bench.delay);
      $scope.s3bench.requests.push(request);
    }
    $('#message').modal('show');
  });
}
