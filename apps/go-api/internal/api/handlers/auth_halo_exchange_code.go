package handlers

import auth_platform "levelup/go-api/internal/platform/auth"

// errCodeHaloExchangeFailed : code de refus de l'échange Halo posé par les deux chemins de connexion
// par le navigateur (site hébergé : /login?oauth_error=, application de bureau : tentative et page de
// retour). Le code d'appareil pose le sien, errCodeHaloExchange. Les valeurs sont servies au web :
// elles ne changent pas.
const errCodeHaloExchangeFailed = "halo_exchange_failed"

// haloExchangeFailureCode choisit le code d'échec d'une chaîne d'échange Xbox -> Halo en échec : les
// services qui ne répondent pas (délai dépassé, réseau coupé, 5xx, 429, 408) rendent
// errCodeHaloUnreachable, à réessayer plus tard ; toute autre cause est un refus du compte et rend le
// code de refus du chemin (refused).
//
// Seul appelant de auth_platform.HaloServicesUnreachable dans internal/api (garde-rail
// archlint/halo_exchange_failure_code_unique_test.go) : les trois chemins de connexion (code d'appareil,
// site hébergé, application de bureau) tranchent donc de la même façon.
func haloExchangeFailureCode(err error, refused string) string {
	if auth_platform.HaloServicesUnreachable(err) {
		return errCodeHaloUnreachable
	}
	return refused
}
