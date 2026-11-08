package auth

import (
	"context"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

const (
	corpsAzureOK    = `{"access_token":"at-az","refresh_token":"rt-az-2","expires_in":3600}`
	corpsNatifOK    = `{"access_token":"at-na","refresh_token":"rt-na-2","expires_in":86400}`
	corpsRevoque    = `{"error":"invalid_grant","error_description":"AADSTS70000: jeton étranger ou mort"}`
	corpsErreurNeuf = `boom`
	corpsConfig     = `{"error":"invalid_client","error_description":"client inconnu de ce jeton"}`
)

// endpointsFactices : les deux endpoints /token (Azure et client natif) de l'échange, avec le
// nombre d'appels que chacun a reçus et les refresh_token qui leur ont été présentés.
type endpointsFactices struct {
	mu          sync.Mutex
	azureCorps  string
	natifCorps  string
	natifStatut int
	azure       int
	natif       int
	rtAzure     []string
	rtNatif     []string
}

// poserEndpointsFactices redirige msalTokenURL et msaTokenURL vers deux serveurs de test et vide la
// mémoire des clients avant et après le test.
func poserEndpointsFactices(t *testing.T, azureCorps, natifCorps string) *endpointsFactices {
	t.Helper()
	f := &endpointsFactices{azureCorps: azureCorps, natifCorps: natifCorps, natifStatut: http.StatusOK}
	viderMemoireRefresh()
	t.Cleanup(viderMemoireRefresh)
	withMockTokenEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		f.mu.Lock()
		f.azure++
		f.rtAzure = append(f.rtAzure, r.PostForm.Get("refresh_token"))
		corps := f.azureCorps
		f.mu.Unlock()
		_, _ = w.Write([]byte(corps))
	}, "custom-app-id", "")
	withMockMSAEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		f.mu.Lock()
		f.natif++
		f.rtNatif = append(f.rtNatif, r.PostForm.Get("refresh_token"))
		corps, statut := f.natifCorps, f.natifStatut
		f.mu.Unlock()
		w.WriteHeader(statut)
		_, _ = w.Write([]byte(corps))
	})
	return f
}

func (f *endpointsFactices) appels() (azure, natif int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.azure, f.natif
}

func (f *endpointsFactices) regler(azureCorps, natifCorps string, natifStatut int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.azureCorps, f.natifCorps, f.natifStatut = azureCorps, natifCorps, natifStatut
}

func viderMemoireRefresh() {
	refreshClients.mu.Lock()
	defer refreshClients.mu.Unlock()
	clear(refreshClients.familles)
}

func tailleMemoireRefresh() int {
	refreshClients.mu.RLock()
	defer refreshClients.mu.RUnlock()
	return len(refreshClients.familles)
}

func exigerAppels(t *testing.T, f *endpointsFactices, azure, natif int, contexte string) {
	t.Helper()
	if a, n := f.appels(); a != azure || n != natif {
		t.Fatalf("%s : appels Azure=%d natif=%d, attendu Azure=%d natif=%d", contexte, a, n, azure, natif)
	}
}

// rafraichir : un rafraîchissement du joueur xuid ("" = contexte sans xuid) ; rend la famille.
func rafraichir(t *testing.T, xuid string) (string, error) {
	t.Helper()
	_, _, famille, err := ExchangeRefreshTokenWithRotation(WithRefreshXUID(context.Background(), xuid), "rt-1")
	return famille, err
}

// TestRefreshMemory_SansXUID : sans xuid dans le contexte, ni l'ordre ni la mémoire ne changent :
// Azure d'abord à chaque fois, le natif en repli, rien de retenu.
func TestRefreshMemory_SansXUID(t *testing.T) {
	f := poserEndpointsFactices(t, corpsRevoque, corpsNatifOK)
	for i := 1; i <= 2; i++ {
		if famille, err := rafraichir(t, ""); err != nil || famille != TokenFamilyXboxNative {
			t.Fatalf("appel %d : famille=%q err=%v, attendu le repli natif", i, famille, err)
		}
		exigerAppels(t, f, i, i, "sans xuid")
	}
	if n := tailleMemoireRefresh(); n != 0 {
		t.Errorf("mémoire de %d entrée(s), attendue vide sans xuid", n)
	}
}

// TestRefreshMemory_SecondRafraichissementSansAzure : le premier rafraîchissement d'un compte natif
// passe par Azure puis le repli ; le second, avec le même xuid, ne sollicite plus Azure. Un
// redémarrage (mémoire vide) reprend l'ordre d'origine.
func TestRefreshMemory_SecondRafraichissementSansAzure(t *testing.T) {
	f := poserEndpointsFactices(t, corpsRevoque, corpsNatifOK)
	const xuid = "2533000000000001"

	if famille, err := rafraichir(t, xuid); err != nil || famille != TokenFamilyXboxNative {
		t.Fatalf("1er : famille=%q err=%v", famille, err)
	}
	exigerAppels(t, f, 1, 1, "1er rafraîchissement (Azure puis repli)")

	if famille, err := rafraichir(t, xuid); err != nil || famille != TokenFamilyXboxNative {
		t.Fatalf("2e : famille=%q err=%v", famille, err)
	}
	exigerAppels(t, f, 1, 2, "2e rafraîchissement (natif seul)")

	viderMemoireRefresh() // redémarrage du processus
	if famille, err := rafraichir(t, xuid); err != nil || famille != TokenFamilyXboxNative {
		t.Fatalf("3e : famille=%q err=%v", famille, err)
	}
	exigerAppels(t, f, 2, 3, "après redémarrage (ordre d'origine)")

	for i, rt := range append(append([]string{}, f.rtAzure...), f.rtNatif...) {
		if rt != "rt-1" {
			t.Errorf("présentation %d : refresh_token %q, attendu rt-1", i, rt)
		}
	}
}

// TestRefreshMemory_CompteRedevenuAzure : un compte mémorisé natif dont le jeton vient désormais
// d'Azure (reconnexion web) essuie un refus natif, passe par Azure, et la mémoire suit.
func TestRefreshMemory_CompteRedevenuAzure(t *testing.T) {
	f := poserEndpointsFactices(t, corpsRevoque, corpsNatifOK)
	const xuid = "2533000000000002"
	if _, err := rafraichir(t, xuid); err != nil {
		t.Fatalf("amorce : %v", err)
	}
	exigerAppels(t, f, 1, 1, "amorce")

	f.regler(corpsAzureOK, corpsRevoque, http.StatusOK)
	if famille, err := rafraichir(t, xuid); err != nil || famille != TokenFamilyAzure {
		t.Fatalf("bascule : famille=%q err=%v, attendu Azure", famille, err)
	}
	exigerAppels(t, f, 2, 2, "bascule (natif refusé puis Azure, sans second natif)")
	if refreshClients.natifDabord(xuid) {
		t.Fatal("la mémoire devrait avoir suivi : Azure, plus natif")
	}

	if _, err := rafraichir(t, xuid); err != nil {
		t.Fatalf("suivant : %v", err)
	}
	exigerAppels(t, f, 3, 2, "après bascule (Azure seul)")
}

// TestRefreshMemory_CompteRedevenuAzureNatifEnConfiguration : un compte mémorisé natif se reconnecte
// par le SSO web (jeton Azure) et le client natif répond une erreur de classe configuration : Azure
// répond, le rafraîchissement réussit, la mémoire suit et le suivant commence par Azure.
func TestRefreshMemory_CompteRedevenuAzureNatifEnConfiguration(t *testing.T) {
	f := poserEndpointsFactices(t, corpsRevoque, corpsNatifOK)
	const xuid = "2533000000000008"
	if _, err := rafraichir(t, xuid); err != nil {
		t.Fatalf("amorce : %v", err)
	}
	exigerAppels(t, f, 1, 1, "amorce")

	f.regler(corpsAzureOK, corpsConfig, http.StatusOK)
	if famille, err := rafraichir(t, xuid); err != nil || famille != TokenFamilyAzure {
		t.Fatalf("bascule : famille=%q err=%v, attendu Azure", famille, err)
	}
	exigerAppels(t, f, 2, 2, "bascule (natif en configuration, puis Azure, sans second natif)")
	if refreshClients.natifDabord(xuid) {
		t.Fatal("la mémoire devrait avoir suivi : Azure, plus natif")
	}

	if _, err := rafraichir(t, xuid); err != nil {
		t.Fatalf("suivant : %v", err)
	}
	exigerAppels(t, f, 3, 2, "après bascule (Azure seul)")
}

// TestRefreshMemory_NatifEnErreurTransitoire : un compte mémorisé natif dont le client natif répond
// 500 reçoit cette erreur telle quelle, sans appel à Azure ; elle n'est pas classée révoquée et la
// mémoire reste en place.
func TestRefreshMemory_NatifEnErreurTransitoire(t *testing.T) {
	f := poserEndpointsFactices(t, corpsRevoque, corpsNatifOK)
	const xuid = "2533000000000003"
	if _, err := rafraichir(t, xuid); err != nil {
		t.Fatalf("amorce : %v", err)
	}

	f.regler(corpsAzureOK, corpsErreurNeuf, http.StatusInternalServerError)
	if _, err := rafraichir(t, xuid); err == nil {
		t.Fatal("erreur attendue")
	} else if got := ClassifyAuthError(err); got != AuthErrorTransient {
		t.Errorf("classe = %q, attendue transient (err = %v)", got, err)
	}
	exigerAppels(t, f, 1, 2, "natif en 500 (Azure intact)")
	if !refreshClients.natifDabord(xuid) {
		t.Error("une erreur transitoire ne doit pas effacer la mémoire")
	}
}

// TestRefreshMemory_Compteurs : un essai natif en tête n'est pas compté comme le repli natif
// (retry_msa) ; un échec garde la classe de l'erreur rendue.
func TestRefreshMemory_Compteurs(t *testing.T) {
	f := poserEndpointsFactices(t, corpsRevoque, corpsNatifOK)
	const xuid = "2533000000000004"
	refreshClients.retenir(xuid, TokenFamilyXboxNative)

	type compteurs struct{ total, echec, publique, msa, revoque int64 }
	lire := func() compteurs {
		return compteurs{oauthRefreshTotal.Value(), oauthRefreshFailTotal.Value(),
			oauthRefreshRetryPublicTotal.Value(), oauthRefreshRetryMSATotal.Value(), compteurRevoque()}
	}
	exiger := func(nom string, avant, attendu compteurs) {
		t.Helper()
		if got := lire(); got != attendu {
			t.Errorf("%s : compteurs %+v, avant %+v, attendu %+v", nom, got, avant, attendu)
		}
	}

	avant := lire()
	if _, err := rafraichir(t, xuid); err != nil {
		t.Fatalf("natif OK : %v", err)
	}
	exiger("natif OK en tête", avant, compteurs{avant.total + 1, avant.echec, avant.publique, avant.msa, avant.revoque})

	avant = lire()
	f.regler(corpsRevoque, corpsRevoque, http.StatusOK)
	if _, err := rafraichir(t, xuid); err == nil {
		t.Fatal("erreur attendue (natif et Azure révoqués)")
	}
	exiger("natif puis Azure révoqués", avant,
		compteurs{avant.total + 1, avant.echec + 1, avant.publique, avant.msa, avant.revoque + 1})

	avant = lire()
	f.regler(corpsAzureOK, corpsErreurNeuf, http.StatusInternalServerError)
	if _, err := rafraichir(t, xuid); err == nil {
		t.Fatal("erreur attendue (natif en 500)")
	}
	exiger("natif en 500", avant, compteurs{avant.total + 1, avant.echec + 1, avant.publique, avant.msa, avant.revoque})
}

// TestRefreshMemory_ContexteSansXUIDInchange : un xuid vide ne crée pas de valeur de contexte.
func TestRefreshMemory_ContexteSansXUIDInchange(t *testing.T) {
	ctx := context.Background()
	if got := WithRefreshXUID(ctx, ""); got != ctx {
		t.Error("un xuid vide doit rendre le contexte tel quel")
	}
	if got := RefreshXUIDFromContext(WithRefreshXUID(ctx, "42")); got != "42" {
		t.Errorf("xuid lu = %q, attendu 42", got)
	}
	if got := RefreshXUIDFromContext(ctx); got != "" {
		t.Errorf("xuid lu = %q, attendu vide", got)
	}
}

// TestRefreshMemory_NeRetientQueDesFamilles : seule la famille d'un succès est retenue, jamais un
// échec ni une valeur vide.
func TestRefreshMemory_NeRetientQueDesFamilles(t *testing.T) {
	viderMemoireRefresh()
	t.Cleanup(viderMemoireRefresh)
	refreshClients.retenir("", TokenFamilyAzure)
	refreshClients.retenir("1", "")
	if n := tailleMemoireRefresh(); n != 0 {
		t.Fatalf("mémoire de %d entrée(s) après des appels sans xuid ou sans famille", n)
	}
	refreshClients.retenir("1", TokenFamilyXboxNative)
	refreshClients.retenir("1", "")
	if !refreshClients.natifDabord("1") {
		t.Error("une famille vide ne doit pas effacer la mémoire")
	}
}

// TestRefreshMemory_Concurrence : accès simultanés à la mémoire et à de vrais échanges, sous
// `go test -race`.
func TestRefreshMemory_Concurrence(t *testing.T) {
	poserEndpointsFactices(t, corpsAzureOK, corpsNatifOK)
	xuids := []string{"2533000000000010", "2533000000000011", "2533000000000012"}
	var wg sync.WaitGroup
	for g := 0; g < 12; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				xuid := xuids[(g+i)%len(xuids)]
				if i%2 == 0 {
					refreshClients.retenir(xuid, TokenFamilyXboxNative)
				} else {
					refreshClients.retenir(xuid, TokenFamilyAzure)
				}
				_ = refreshClients.natifDabord(xuid)
				if i%10 == 0 {
					_, _ = rafraichir(t, xuid)
				}
			}
		}(g)
	}
	wg.Wait()
}

// TestRefreshMemory_SISUProviderPorteLeXUID : le contexte posé par l'appelant (le pool) traverse
// TryOAuthRefreshWithRotation jusqu'à l'échange : le client mémorisé répond seul.
func TestRefreshMemory_SISUProviderPorteLeXUID(t *testing.T) {
	f := poserEndpointsFactices(t, corpsRevoque, corpsNatifOK)
	const xuid = "2533000000000005"
	refreshClients.retenir(xuid, TokenFamilyXboxNative)

	at, rt, err := NewSISUProvider().TryOAuthRefreshWithRotation(WithRefreshXUID(context.Background(), xuid), "rt-1")
	if err != nil || at != "at-na" || rt != "rt-na-2" {
		t.Fatalf("(%q, %q, %v), attendu (at-na, rt-na-2, nil)", at, rt, err)
	}
	exigerAppels(t, f, 0, 1, "SISUProvider avec xuid")
}

// TestRefreshMemory_RefreshUserXSTS : le chemin de rafraîchissement du magasin multi-joueur pose
// le xuid du joueur.
func TestRefreshMemory_RefreshUserXSTS(t *testing.T) {
	f := poserEndpointsFactices(t, corpsRevoque, corpsNatifOK)
	const xuid = "2533000000000006"
	refreshClients.retenir(xuid, TokenFamilyXboxNative)

	tokens := &UserTokens{XUID: xuid, OAuthRefreshToken: "rt-1"}
	if got := refreshAccessTokenForUser(context.Background(), nil, tokens); got != "at-na" {
		t.Fatalf("access_token = %q, attendu at-na", got)
	}
	exigerAppels(t, f, 0, 1, "refreshAccessTokenForUser")
}

// TestRefreshMemory_RefreshLoop : la boucle du watcher pose le xuid du tracker.
func TestRefreshMemory_RefreshLoop(t *testing.T) {
	f := poserEndpointsFactices(t, corpsRevoque, corpsNatifOK)
	const xuid = "2533000000000007"
	refreshClients.retenir(xuid, TokenFamilyXboxNative)

	dir := t.TempDir()
	store := NewTokenStore(filepath.Join(dir, "watcher_tokens.json"))
	multi := NewMultiUserTokenStore(filepath.Join(dir, "watcher_tokens"))
	if err := multi.Upsert(&UserTokens{XUID: xuid, Gamertag: "Joueur", OAuthRefreshToken: "rt-1"}); err != nil {
		t.Fatalf("seed multi: %v", err)
	}
	tokens := &StoredTokens{XSTSXUID: xuid, XSTSGamertag: "Joueur", XSTSToken: "xsts",
		XSTSExpiresAt: time.Now().Add(time.Hour)}
	if err := store.Save(tokens); err != nil {
		t.Fatalf("seed tracker: %v", err)
	}

	r := NewRefreshLoop(store, nil).WithMultiUserMirror(multi)
	if err := r.refreshOAuth(context.Background(), tokens, "rt-1"); err != nil {
		t.Fatalf("refreshOAuth: %v", err)
	}
	exigerAppels(t, f, 0, 1, "RefreshLoop.refreshOAuth")
}
