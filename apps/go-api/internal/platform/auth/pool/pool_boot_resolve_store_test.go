package pool

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/platform/auth"
)

// pool_boot_resolve_store_test.go — la résolution parallèle des comptes au démarrage, avec le
// VRAI Resolver et un VRAI MultiUserTokenStore branché comme au boot serveur (rappels
// onRotated, onReauth, onAuthError, onFamilyObserved) : chaque compte retrouve dans SON
// fichier SA rotation de refresh token, sa provenance, son drapeau de reconnexion et son
// dernier échec — aucune écriture concurrente n'en écrase une autre.

// rotatingProvider : refresh OAuth qui fait tourner le jeton (rt → rt+"-rot") et échange
// qui mesure une provenance ; échec de classe « révoqué » pour revokedRT, « config » pour
// configRT. Chaque appel attend un peu pour que les comptes se chevauchent.
type rotatingProvider struct {
	mockTokenProvider
	revokedRT string
	configRT  string
}

func (p *rotatingProvider) TryOAuthRefreshWithRotation(_ context.Context, rt string) (string, string, error) {
	time.Sleep(15 * time.Millisecond)
	switch rt {
	case p.revokedRT:
		return "", "", &auth.OAuthExchangeError{ErrorCode: "invalid_grant", Description: "AADSTS70000: révoqué"}
	case p.configRT:
		return "", "", configError()
	}
	return "at-" + rt, rt + "-rot", nil
}

func (p *rotatingProvider) Exchange(ctx context.Context, _ string) (*auth.ExchangeResult, error) {
	time.Sleep(10 * time.Millisecond)
	auth.RecordObservedTokenFamily(ctx, auth.TokenFamilyXboxNative)
	return &auth.ExchangeResult{}, nil
}

// storeErrors : erreurs d'écriture du store remontées par les rappels (appelés en parallèle).
type storeErrors struct {
	mu   sync.Mutex
	errs []error
}

func (s *storeErrors) add(err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	s.errs = append(s.errs, err)
	s.mu.Unlock()
}

// storeCallbacks : les quatre rappels du resolver écrivent dans le store, comme ceux de
// cmd/server (buildAutoSyncPool) — xuid du rappel, ou résolu par gamertag (map en lecture
// seule) pour la rotation.
func storeCallbacks(store *auth.MultiUserTokenStore, xuidByGT map[string]string, errs *storeErrors) ResolverCallbacks {
	return ResolverCallbacks{
		OnRotated: func(_ context.Context, gamertag, newRT string) error {
			err := store.UpdateOAuthRefreshToken(xuidByGT[gamertag], newRT)
			errs.add(err)
			return err
		},
		OnReauth: func(_ context.Context, gamertag, xuid string, required bool) {
			if required {
				_, err := store.MarkReauthRequired(xuid, gamertag)
				errs.add(err)
				return
			}
			errs.add(store.ClearReauthRequired(xuid))
		},
		OnAuthError: func(_ context.Context, gamertag, xuid, class, msg string) {
			if class == "" {
				errs.add(store.ClearAuthError(xuid))
				return
			}
			errs.add(store.RecordAuthError(xuid, gamertag, class, msg))
		},
		OnFamilyObserved: func(_ context.Context, _, xuid, family string) error {
			err := store.UpdateTokenClientFamily(xuid, family)
			errs.add(err)
			return err
		},
	}
}

func TestNewPool_ResolutionParallele_StoreReelChaqueCompteGardeSesEcritures(t *testing.T) {
	store := auth.NewMultiUserTokenStore(t.TempDir())
	const accounts = 10
	sources := make([]CredentialSource, 0, accounts)
	xuidByGT := make(map[string]string, accounts)
	for i := 0; i < accounts; i++ {
		gt := fmt.Sprintf("Compte%02d", i)
		xuid := fmt.Sprintf("25350000000%02d", i)
		rt := fmt.Sprintf("rt-%02d", i)
		xuidByGT[gt] = xuid
		// État de départ : reconnexion demandée et dernier échec mémorisé ; un refresh
		// réussi doit effacer les deux, dans le fichier du compte.
		if err := store.Upsert(&auth.UserTokens{XUID: xuid, Gamertag: gt, OAuthRefreshToken: rt,
			ReauthRequired: true, LastAuthErrorClass: "config", LastAuthError: "ancien échec"}); err != nil {
			t.Fatalf("Upsert %s : %v", gt, err)
		}
		sources = append(sources, CredentialSource{Gamertag: gt, XUID: xuid, RefreshToken: rt, Source: "test"})
	}
	provider := &rotatingProvider{revokedRT: "rt-03", configRT: "rt-07"}
	errs := &storeErrors{}
	resolver := NewResolverWithCallbacks(provider, time.Hour, storeCallbacks(store, xuidByGT, errs))

	p, err := NewPool(context.Background(), resolver, sources, PoolOptions{})
	if err != nil {
		t.Fatalf("NewPool : %v", err)
	}
	defer p.Close()
	if len(errs.errs) > 0 {
		t.Fatalf("écritures du store en échec : %v", errs.errs)
	}
	if got := p.Size(); got != accounts-2 {
		t.Fatalf("Size() = %d, attendu %d (deux comptes en échec)", got, accounts-2)
	}

	for i := 0; i < accounts; i++ {
		gt := fmt.Sprintf("Compte%02d", i)
		ut, err := store.Load(xuidByGT[gt])
		if err != nil {
			t.Fatalf("Load %s : %v", gt, err)
		}
		rt := fmt.Sprintf("rt-%02d", i)
		switch rt {
		case provider.revokedRT:
			if ut.OAuthRefreshToken != rt || !ut.ReauthRequired || ut.LastAuthErrorClass != string(auth.AuthErrorRevoked) {
				t.Errorf("%s (révoqué) : rt=%q reauth=%v classe=%q, attendu rt inchangé, reauth, classe revoked",
					gt, ut.OAuthRefreshToken, ut.ReauthRequired, ut.LastAuthErrorClass)
			}
		case provider.configRT:
			if ut.OAuthRefreshToken != rt || ut.LastAuthErrorClass != string(auth.AuthErrorConfig) ||
				!strings.Contains(ut.LastAuthError, "AADSTS90023") {
				t.Errorf("%s (config) : rt=%q classe=%q msg=%q, attendu rt inchangé et l'échec config",
					gt, ut.OAuthRefreshToken, ut.LastAuthErrorClass, ut.LastAuthError)
			}
		default:
			if ut.OAuthRefreshToken != rt+"-rot" {
				t.Errorf("%s : refresh token persisté %q, attendu la rotation %q", gt, ut.OAuthRefreshToken, rt+"-rot")
			}
			if ut.TokenClientFamily != auth.TokenFamilyXboxNative || ut.ReauthRequired || ut.LastAuthErrorClass != "" {
				t.Errorf("%s : famille=%q reauth=%v classe=%q, attendu famille mesurée, reauth et échec effacés",
					gt, ut.TokenClientFamily, ut.ReauthRequired, ut.LastAuthErrorClass)
			}
		}
		if ut.Gamertag != gt {
			t.Errorf("fichier de %s porte le gamertag %q : une écriture d'un autre compte l'a écrasé", gt, ut.Gamertag)
		}
	}
}
