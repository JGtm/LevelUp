package auth

// user_token_store.go — interface de découplage (Axe 3 refactor).
//
// UserTokenStore abstrait la persistance multi-user des tokens auth telle que
// consommée par la couche API (ServiceRegistry, XboxOAuthHandler). Les
// consommateurs dépendent de l'interface plutôt que du type concret
// *MultiUserTokenStore, ce qui rend la persistance des tokens mockable en test
// (sans répertoire sur disque).
//
// NB : à ne pas confondre avec le type concret `TokenStore` (token_store.go),
// qui est le store mono-user legacy (data/auth/tokens.json).
//
// Surface minimale (interface segregation) : uniquement les méthodes réellement
// appelées via les champs injectés — lire les tokens d'un joueur (Load) et
// persister la rotation du refresh token Microsoft (UpdateOAuthRefreshToken,
// ADR 0023). Les autres méthodes (Upsert, LoadByGamertag, LoadAll, Remove…)
// restent accessibles via le type concret au composition root / CLI.
type UserTokenStore interface {
	// Load retourne les tokens persistés d'un joueur par xuid (nil si absent).
	Load(xuid string) (*UserTokens, error)
	// UpdateOAuthRefreshToken persiste atomiquement un refresh token rotaté.
	UpdateOAuthRefreshToken(xuid, refreshToken string) error
}

// Garantie compile-time : le store multi-user concret satisfait l'interface.
var _ UserTokenStore = (*MultiUserTokenStore)(nil)
