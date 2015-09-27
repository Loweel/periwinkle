﻿(function(){
	var periwinkleApp = angular.module('periwinkleApp', [
		'ngRoute',
		'ngMaterial',
		'ngCookies',
		//periwinkle modules
		'periwinkle',
		'login'
	]);

	periwinkleApp.config(function($mdThemingProvider){
		$mdThemingProvider.theme('default')
			.primaryPalette('deep-purple')
			.accentPalette('teal');
	});

	periwinkleApp.config(['$routeProvider', '$locationProvider',
		function($routeProvider, $locationProvider) {
			$locationProvider
				  .hashPrefix('!');
			
			$routeProvider.
			when('/login', {
				templateUrl:	'src/login/login.html',
				controller:		'LoginController'
			}).
			when('/dashboard', {
				templateUrl:	'src/dashboard/dashboard.html',
				controller:		'DashboardController'
			}).
			when('/messaages/:groupid', {
				templateURL:	'src/messages/messages.html',
				controller:		'MessaagesController'
			}).
			when('/settings', {
				templateUrl:	'src/settings/settings.html',
				controller:		'SettingsController'
			}).
			otherwise({
				redirectTo:	'/login'
			})
		}
	]);
})();