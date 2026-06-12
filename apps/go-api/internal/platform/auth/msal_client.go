// Package auth — msal_client.go : wrapper MSAL Go pour le Device Code Flow.
//
// Client ID : e1cb35ab-c41a-4ee5-a7a1-22ea4e94cdca (app Azure "LevelUp Halo")
// Authority : https://login.microsoftonline.com/consumers
// Scopes : Xboxlive.signin, Xboxlive.offline_access
//
// API MSAL v1.7.1 :
//  1. app.AcquireTokenByDeviceCode(ctx, scopes) → (DeviceCode, error)
//  2. deviceCode.Result.UserCode / VerificationURL / ExpiresOn / Message
//  3. deviceCode.AuthenticationResult(ctx) → (AuthResult, error)  — bloquant
//
// Le cache MSAL est maintenu en mémoire pendant le setup initial.
// Une fois le profil créé, il peut être persisté dans sync_meta.duckdb.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

const (
	// LevelUpClientID est le client_id de l'app Azure "LevelUp Halo".
	LevelUpClientID = "e1cb35ab-c41a-4ee5-a7a1-22ea4e94cdca" // pragma: allowlist secret

	// HaloToolsClientID est l'app Azure PUBLIQUE "halo-tools" partagée avec les
	// watchers du parc (client par défaut de cmd/token-capture, cf. defaultClientID).
	// Étant publique, un refresh/échange OAuth NE DOIT PAS lui envoyer de client_secret
	// (Azure rejette : AADSTS90023 "Public clients can't send a client secret").
	HaloToolsClientID = "39829f7a-5262-4d22-a387-795c488f7102" // pragma: allowlist secret

	// MSALAuthority pour les comptes personnels Microsoft (Xbox Live).
	MSALAuthority = "https://login.microsoftonline.com/consumers"
)

// IsPublicAzureClient indique si clientID correspond à une app Azure PUBLIQUE connue
// (LevelUp ou halo-tools). Pour ces clients, ne jamais joindre de client_secret aux
// requêtes token — sinon AADSTS90023. Tout autre client_id est présumé confidentiel
// (un secret défini lui sera transmis). Source unique de la décision secret/no-secret
// partagée par oauth_refresh.go et auth_code.go.
func IsPublicAzureClient(clientID string) bool {
	return clientID == LevelUpClientID || clientID == HaloToolsClientID
}

// XboxScopes sont les scopes requis pour Xbox Live.
var XboxScopes = []string{"Xboxlive.signin", "Xboxlive.offline_access"}

// msalDeviceFlow implémente DeviceFlow via MSAL Device Code Flow.
// Privé — créé uniquement par InitDeviceFlow.
type msalDeviceFlow struct {
	message         string
	userCode        string
	verificationURL string
	expiresIn       int
	dc              public.DeviceCode
	cache           *InMemoryCacheAccessor // conservé pour MSALCacheJSON() après AcquireToken
}

// MSALCacheJSON retourne le cache MSAL sérialisé (contient le refresh_token).
// Vide si AcquireToken n'a pas encore été appelé ou si le cache n'est pas
// disponible. Utilisé par PR 2.5a (SSO Xbox) pour persister un état suffisant
// pour des refresh ultérieurs via AcquireTokenSilent.
func (f *msalDeviceFlow) MSALCacheJSON() string {
	if f.cache == nil {
		return ""
	}
	data, err := f.cache.Serialize()
	if err != nil {
		return ""
	}
	return data
}

func (f *msalDeviceFlow) GetMessage() string         { return f.message }
func (f *msalDeviceFlow) GetUserCode() string        { return f.userCode }
func (f *msalDeviceFlow) GetVerificationURL() string { return f.verificationURL }
func (f *msalDeviceFlow) GetExpiresIn() int          { return f.expiresIn }
func (f *msalDeviceFlow) GetFlowType() string        { return "msal" }

// Vérification compile-time : msalDeviceFlow implémente DeviceFlow.
var _ DeviceFlow = (*msalDeviceFlow)(nil)

// AcquireToken attend la complétion du Device Code Flow et retourne l'access_token.
// Bloquant — doit être appelé dans une goroutine.
func (f *msalDeviceFlow) AcquireToken(ctx context.Context) (string, error) {
	result, err := f.dc.AuthenticationResult(ctx)
	if err != nil {
		return "", fmt.Errorf("MSAL AuthenticationResult: %w", err)
	}
	return result.AccessToken, nil
}

// InitDeviceFlow initie un Device Code Flow Microsoft.
// cacheAccessor peut être nil pour utiliser un cache en mémoire.
//
// Si cacheAccessor est un *InMemoryCacheAccessor, il est conservé sur le flow
// retourné pour que MSALCacheJSON() puisse le sérialiser après AcquireToken
// (utilisé par le SSO Xbox pour persister le refresh_token, cf. PR 2.5a).
func InitDeviceFlow(ctx context.Context, cacheAccessor cache.ExportReplace) (*msalDeviceFlow, error) {
	opts := []public.Option{
		public.WithAuthority(MSALAuthority),
	}
	if cacheAccessor != nil {
		opts = append(opts, public.WithCache(cacheAccessor))
	}

	app, err := public.New(LevelUpClientID, opts...)
	if err != nil {
		return nil, fmt.Errorf("MSAL init: %w", err)
	}

	dc, err := app.AcquireTokenByDeviceCode(ctx, XboxScopes)
	if err != nil {
		return nil, fmt.Errorf("MSAL AcquireTokenByDeviceCode: %w", err)
	}

	expiresIn := int(time.Until(dc.Result.ExpiresOn).Seconds())
	if expiresIn < 0 {
		expiresIn = 0
	}

	flow := &msalDeviceFlow{
		message:         dc.Result.Message,
		userCode:        dc.Result.UserCode,
		verificationURL: dc.Result.VerificationURL,
		expiresIn:       expiresIn,
		dc:              dc,
	}
	// Conserver l'accessor si c'est un InMemoryCacheAccessor — permet à
	// MSALCacheJSON() de le sérialiser après AcquireToken pour persistance.
	if mem, ok := cacheAccessor.(*InMemoryCacheAccessor); ok {
		flow.cache = mem
	}
	return flow, nil
}

// AcquireTokenSilent tente d'obtenir un access_token depuis le cache MSAL.
// Retourne ("", nil) si le cache est vide ou si le refresh échoue.
func AcquireTokenSilent(ctx context.Context, cacheAccessor cache.ExportReplace) (string, error) {
	opts := []public.Option{
		public.WithAuthority(MSALAuthority),
	}
	if cacheAccessor != nil {
		opts = append(opts, public.WithCache(cacheAccessor))
	}

	app, err := public.New(LevelUpClientID, opts...)
	if err != nil {
		return "", fmt.Errorf("MSAL init: %w", err)
	}

	accounts, err := app.Accounts(ctx)
	if err != nil || len(accounts) == 0 {
		return "", nil // Pas de compte en cache
	}

	result, err := app.AcquireTokenSilent(ctx, XboxScopes, public.WithSilentAccount(accounts[0]))
	if err != nil {
		return "", nil // Refresh échoué — nécessite Device Code Flow
	}
	return result.AccessToken, nil
}

// =============================================================================
// InMemoryCacheAccessor — cache MSAL en mémoire (setup avant création du profil)
// =============================================================================

// InMemoryCacheAccessor implémente cache.ExportReplace en mémoire.
// Utilisé pendant le setup initial avant que la player DB soit créée.
type InMemoryCacheAccessor struct {
	mu   sync.Mutex
	data []byte
}

// NewInMemoryCacheAccessorFromJSON crée un InMemoryCacheAccessor pré-chargé
// avec le JSON du cache MSAL lu depuis sync_meta.
// Si jsonData est vide, retourne un cache vide (premier lancement).
func NewInMemoryCacheAccessorFromJSON(jsonData string) *InMemoryCacheAccessor {
	acc := &InMemoryCacheAccessor{}
	if jsonData != "" {
		acc.data = []byte(jsonData)
	}
	return acc
}

// Replace écrit le cache MSAL (appelé par MSAL pour lire le cache).
func (a *InMemoryCacheAccessor) Replace(ctx context.Context, c cache.Unmarshaler, hints cache.ReplaceHints) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.data) == 0 {
		return nil
	}
	return c.Unmarshal(a.data)
}

// Export lit le cache MSAL pour persistance (appelé par MSAL après mise à jour).
func (a *InMemoryCacheAccessor) Export(ctx context.Context, c cache.Marshaler, hints cache.ExportHints) error {
	data, err := c.Marshal()
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.data = data
	return nil
}

// Serialize retourne le cache sérialisé en JSON (pour persistance DuckDB).
func (a *InMemoryCacheAccessor) Serialize() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.data) == 0 {
		return "", nil
	}
	var raw json.RawMessage
	if err := json.Unmarshal(a.data, &raw); err != nil {
		return "", fmt.Errorf("cache MSAL invalide: %w", err)
	}
	return string(a.data), nil
}
