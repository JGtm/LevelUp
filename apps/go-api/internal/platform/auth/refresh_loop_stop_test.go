// Package auth — refresh_loop_stop_test.go : l'arrêt du serveur pendant un rafraîchissement de jetons
// (ADR 0045, lot L12.6). La boucle doit rendre la main vite, sans rien perdre : un échange en vol
// est interrompu AVANT la réponse (le refresh_token stocké reste valable), un jeton déjà obtenu est
// écrit même si l'annulation arrive juste après.
package auth

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"testing"
	"time"
)

// stopTestBudget : la boucle doit rendre la main bien avant le délai d'un échange (oauthTimeout,
// 15 s) ; la phase scheduler de l'arrêt n'attend que 3 s.
const stopTestBudget = 3 * time.Second

// nouvelleBoucleExpiree prépare une boucle dont l'access_token est expiré (le premier contrôle
// lance donc un échange OAuth) et un refresh_token "rt-1" dans le magasin multi-utilisateur.
func nouvelleBoucleExpiree(t *testing.T) (*RefreshLoop, *TokenStore, *MultiUserTokenStore) {
	t.Helper()
	dir := t.TempDir()
	store := NewTokenStore(filepath.Join(dir, "watcher_tokens.json"))
	multi := NewMultiUserTokenStore(filepath.Join(dir, "watcher_tokens"))
	if err := store.Save(&StoredTokens{XSTSXUID: "111", XSTSGamertag: "Alice"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := multi.Upsert(&UserTokens{XUID: "111", Gamertag: "Alice", OAuthRefreshToken: "rt-1"}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	return NewRefreshLoop(store, nil).WithMultiUserMirror(multi), store, multi
}

// attendreFin attend la fin de Run ; échoue si elle dépasse stopTestBudget.
func attendreFin(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(stopTestBudget):
		t.Fatalf("la boucle n'a pas rendu la main en %s après l'annulation : la phase scheduler de l'arrêt "+
			"l'attendrait jusqu'à son délai (clean=false)", stopTestBudget)
	}
}

// Un échange OAuth lent (le serveur de Microsoft ne répond pas) est interrompu par l'annulation ;
// le refresh_token stocké n'a pas bougé.
func TestRefreshLoop_ArretPendantUnEchangeOAuthLent(t *testing.T) {
	entered := make(chan struct{}, 4)
	release := make(chan struct{})
	withMockTokenEndpoint(t, func(_ http.ResponseWriter, r *http.Request) {
		// Le serveur ne voit la fermeture du client qu'une fois le corps lu.
		_, _ = io.Copy(io.Discard, r.Body)
		entered <- struct{}{}
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}, "custom-app-id", "s3cret")
	// Enregistré APRÈS le serveur de test : exécuté AVANT sa fermeture, qui attend les handlers.
	t.Cleanup(func() { close(release) })
	rl, _, multi := nouvelleBoucleExpiree(t)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); rl.Run(ctx) }()

	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("l'échange OAuth n'a jamais atteint le serveur de test")
	}
	cancel()
	attendreFin(t, done)

	u, err := multi.Load("111")
	if err != nil || u == nil {
		t.Fatalf("Load: %v %v", u, err)
	}
	if u.OAuthRefreshToken != "rt-1" {
		t.Errorf("refresh_token = %q, attendu rt-1 inchangé (échange interrompu avant la réponse)", u.OAuthRefreshToken)
	}
}

// L'annulation arrive APRÈS l'échange OAuth, pendant l'acquisition du XSTS : le refresh_token tourné
// et l'access_token obtenus sont écrits, et la boucle rend quand même la main tout de suite.
func TestRefreshLoop_ArretApresLEchange_LeJetonObtenuEstEcrit(t *testing.T) {
	withMockTokenEndpoint(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"at-2","refresh_token":"rt-2","expires_in":3600}`))
	}, "custom-app-id", "s3cret")
	rl, store, multi := nouvelleBoucleExpiree(t)

	inXSTS := make(chan struct{})
	rl.acquireXSTSFn = func(ctx context.Context, _ string) (*XSTSResult, error) {
		close(inXSTS)
		<-ctx.Done()
		return nil, ctx.Err()
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); rl.Run(ctx) }()

	select {
	case <-inXSTS:
	case <-time.After(5 * time.Second):
		t.Fatal("l'acquisition XSTS n'a jamais commencé")
	}
	cancel()
	attendreFin(t, done)

	u, err := multi.Load("111")
	if err != nil || u == nil {
		t.Fatalf("Load multi: %v %v", u, err)
	}
	if u.OAuthRefreshToken != "rt-2" {
		t.Errorf("refresh_token = %q, attendu rt-2 : le jeton tourné obtenu avant l'annulation doit être écrit", u.OAuthRefreshToken)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load store: %v", err)
	}
	if got.AccessToken != "at-2" {
		t.Errorf("access_token = %q, attendu at-2 écrit avant l'annulation", got.AccessToken)
	}
}
