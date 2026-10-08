package api

// IsServerRoutePath expose isServerRoutePath aux tests externes (package api_test), qui
// seuls peuvent construire le vrai routeur (cf. spa_handler_routes_test.go).
var IsServerRoutePath = isServerRoutePath
