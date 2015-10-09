﻿(function(){
	'use strict';

	var periwinkleApp = angular.module('periwinkle', [
		'ngRoute',
		'ngMaterial',
		'ngMessages',
		'ngCookies',
		'pascalprecht.translate',
		'validation.match',
		'ngSanitize',
		//periwinkle modules
		'validation.xregex',
		'login',
		'dashboard',
		'user'
	]);

	//user this plugin instead of the browsers regex (for unicode support)
	XRegExp.install('natives');

	periwinkleApp.config(function($mdThemingProvider){
		$mdThemingProvider.theme('default')
			.primaryPalette('deep-purple')
			.accentPalette('teal');
	});
	
	periwinkleApp.config(['$translateProvider', function($translateProvider) {
		$translateProvider
			.translations('en', localised.en)
			.translations('it', localised.it)
			.translations('es', localised.es);
		$translateProvider.fallbackLanguage('en');
		$translateProvider.use(lang);
		$translateProvider.useSanitizeValueStrategy('escape');
	}]);

	periwinkleApp.config(['$routeProvider', '$locationProvider',
		function($routeProvider, $locationProvider) {
			$locationProvider
				  .hashPrefix('!');
			
			$routeProvider.
				when('/login', {
					templateUrl:	'src/login/login.html',
					controller:		'LoginController as login'
				}).
				when('/dashboard', {
					templateUrl:	'src/dashboard/dashboard.html',
					controller:		'DashboardController'
				}).
				when('/user', {
					templateURL:	'src/user/user1.html',
					controller:		'UserController as user'
				}).
				otherwise({
					redirectTo:	'/login'
				});
		}
	]);
})();
