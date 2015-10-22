﻿// Copyright 2015 Richard Wisniewski
(function(){
	'use strict';

	angular
		.module('user')
		.controller('UserController', ['$cookies', '$http', '$scope', '$interval', UserController]);

	function UserController($cookies, $http, $scope, $interval) {
		//gives us an anchor to the outer object from within sub objects or functions
		var self = this;
		//clears the toolbar and such so we can set it up for this view
		$scope.resetHeader();
		$scope.toolbar.title = "USER.USER";
		//set up public fields

		$scope.toolbar.buttons = [{
			label: "LogOut",
			img_src: "assets/svg/share.svg",
		}];
		$scope.toolbar.onclick = function(index) {
			if(index == 0) {
				//self.togleSignup();
			}
		};

		self.username = 'Richard Wisniewski';
		self.email = 'rwisniew@purdue.edu';
		self.sessionID = "0x1234567890";
		
		self.groups = [];
		
		self.reload = function() {
			
		};
		self.createGroup = function() {
			
		};
		self.joinGroup = function() {
			
		};
		
		self.reload();
	}
})();