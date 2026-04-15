// Package auth — msal_client.go : wrapper MSAL Go pour le Device Code Flow.
//
// Client ID : e1cb35ab-c41a-4ee5-a7a1-22ea4e94cdca (app Azure "LevelUp Halo")
// Authority : https://login.microsoftonline.com/consumers
// Scopes : Xboxlive.signin, Xboxlive.offline_access
//
// API MSAL v1.7.1 :
//   1. app.AcquireTokenByDeviceCode(ctx, scopes) → (DeviceCode, error)
//   2. deviceCode.Result.UserCode / VerificationURL / ExpiresOn / Message
//   3. deviceCode.AuthenticationResult(ctx) → (AuthResult, error)  — bloquant
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

	// MSALAuthority pour les comptes personnels Microsoft (Xbox Live).
	MSALAuthority = "https://login.microsoftonline.com/consumers"
)

// XboxScopes sont les scopes requis pour Xbox Live.
var XboxScopes = []string{"Xboxlive.signin", "Xboxlive.offline_access"}

// DeviceCodeFlow regroupe les données d'un flow en cours.
type DeviceCodeFlow struct {
	// Message contient le texte à afficher à l'utilisateur (localisé par Microsoft).
	Message string
	// UserCode est le code court à entrer sur microsoft.com/devicelogin.
	UserCode string
	// VerificationURL est l'URL de vérification.
	VerificationURL string
	// ExpiresIn est le nombre de secondes avant expiration.
	ExpiresIn int
	// internal field for AuthenticationResult
	dc  public.DeviceCode
}

// AcquireToken attend la complétion du Device Code Flow et retourne l'access_token.
// Bloquant — doit être appelé dans une goroutine.
func (f *DeviceCodeFlow) AcquireToken(ctx context.Context) (string, error) {
	result, err := f.dc.AuthenticationResult(ctx)
	if err != nil {
		return "", fmt.Errorf("MSAL AuthenticationResult: %w", err)
	}
	return result.AccessToken, nil
}

// InitDeviceFlow initie un Device Code Flow Microsoft.
// cacheAccessor peut être nil pour utiliser un cache en mémoire.
func InitDeviceFlow(ctx context.Context, cacheAccessor cache.ExportReplace) (*DeviceCodeFlow, error) {
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

	return &DeviceCodeFlow{
		Message:         dc.Result.Message,
		UserCode:        dc.Result.UserCode,
		VerificationURL: dc.Result.VerificationURL,
		ExpiresIn:       expiresIn,
		dc:              dc,
	}, nil
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
