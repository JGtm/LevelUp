// sisu.go : le client Xbox natif (login Xbox sans application Azure, dit « SISU ») : son
// identifiant, son scope MSA et ses adresses sur login.live.com. Le device-flow qui les emploie
// vit dans xbox_device_code.go, le repli de rafraîchissement dans oauth_refresh.go ; le
// fournisseur de jetons qui assemble le tout (SISUProvider) reste dans `platform/auth`.

package xblauth

const (
	// IDAppXboxNatif est le client_id Xbox natif utilisé par l'app Xbox sur Windows.
	IDAppXboxNatif = "000000004c20a908"

	// URLConnectMSA : endpoint MSA de démarrage du Device Code Flow natif Xbox.
	// ATTENTION : c'est bien oauth20_connect.srf — l'URL "/oauth20_connect/device" renvoie
	// HTTP 404 chez Microsoft : le login SSO Xbox ne s'amorçait JAMAIS (constaté en local
	// 2026-07-13, vérifié par POST direct : .srf -> 200 + device_code, /device -> 404).
	URLConnectMSA = "https://login.live.com/oauth20_connect.srf"

	// URLTokenMSA : endpoint OAuth 2.0 MSA (login.live.com) du client Xbox natif : polling du
	// device-flow et rafraîchissement des refresh_tokens de cette famille.
	URLTokenMSA = "https://login.live.com/oauth20_token.srf"

	// ScopeMSA est le scope MSA natif demandé avec le client Xbox natif : le ticket rendu
	// (famille MSA) s'échange en User Token XBL avec le préfixe RpsTicket « t= ». Les scopes
	// Azure AD (Xboxlive.signin …) produiraient un JWT AAD hors de cette famille. Sert aussi au
	// rafraîchissement MSA natif.
	ScopeMSA = "service::user.auth.xboxlive.com::MBI_SSL"
)
