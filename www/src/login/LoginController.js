﻿(function(){
  angular
		.module('login')
		.controller('LoginController', ['$cookies', '$http', '$scope', '$interval', LoginController]); 

	function LoginController($cookies, $http, $scope, $interval) {
		//gives us an anchor to the outer object from within sub objects or functions
		var self = this;
		//clears the toolbar and such so we can set it up for this view
		$scope.resetHeader();
		//adds the public fields and methods of this object to the model ($scope)
		$scope.login = self;
		//set up public fields
		self.username = '';
		self.password = '';
		self.email = "";
		self.comfirmEmail = "t.walters1101@gmail.com";
		self.confirmPassword = "";
		self.isSignup = false;
		//prep the toolbar
		$scope.toolbar.title = 'LOGIN';
		/*$scope.toolbar.buttons = [{
			label: "Signup",
			img_src: "assets/svg/phone.svg",
		}];
		$scope.toolbar.onclick = function(index) {
			if(index == 0) {
				$scope.login.togleSignup();
			}
		} */
		
		//check if already loged in
		var sessionID = $cookies.get("sessionID");
		if(sessionID !== undefined) {
			//show loading spinner
			//validate sessionID
			//if valid take down spinner
			//if invalid (or error)
				//show login screen
				//clear cookie
				//show warning bar
		}
		
		//public functions
		this.login = function() {
			//http login api call
			$scope.loading.is = true;
			$interval(function() {
				$scope.loading.is = false;
			}, 5000, 1);
		}
		
		this.signup = function() {
			//http signup api call
			alert(self.username);
		}
		
		this.togleSignup = function () {
			self.isSignup = !self.isSignup;
			self.username = '';
			self.password = '';
			if(self.isSignup) {
				$scope.toolbar.title = 'SIGNUP.SIGNUP';
			} else {
				$scope.toolbar.title = 'LOGIN';
			}
		}
	}
})();
