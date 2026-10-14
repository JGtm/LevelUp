package deadtoken

import (
	"context"
	"testing"

	"levelup/go-api/internal/platform/auth"
)

// fakeRefreshProvider : TryOAuthRefreshWithRotation compté ; un RT listé dans `morts` est
// refusé par Microsoft (invalid_grant, comme AADSTS70000), les autres rendent un access_token.
type fakeRefreshProvider struct {
	auth.TokenProvider
	appels int
	morts  map[string]bool
}

func (f *fakeRefreshProvider) TryOAuthRefreshWithRotation(_ context.Context, rt string) (string, string, error) {
	f.appels++
	if f.morts[rt] {
		return "", "", &auth.OAuthExchangeError{ErrorCode: "invalid_grant",
			Description: "AADSTS70000: The token was issued for a different client id"}
	}
	return "at-" + rt, "", nil
}

const gateXUID = "2533274000000042"

func gateStore(t *testing.T, rt string) *auth.MultiUserTokenStore {
	t.Helper()
	store := auth.NewMultiUserTokenStore(t.TempDir())
	if err := store.UpdateOAuthRefreshToken(gateXUID, rt); err != nil {
		t.Fatal(err)
	}
	return store
}

// Un jeton refusé une fois n'est plus présenté à Microsoft tant qu'il ne change pas ; l'état
// mort est consigné au store (bannière de reconnexion, dernier échec) ; un jeton importé le
// lève et le succès efface l'état.
func TestPorteJetonsMorts_UnSeulEssaiParJeton(t *testing.T) {
	ctx := context.Background()
	gate := New()
	store := gateStore(t, "rt-mort")
	prov := &fakeRefreshProvider{morts: map[string]bool{"rt-mort": true}}

	// Passe 1 : essai, refus permanent.
	if _, sauté, err := gate.Resolve(ctx, store, prov, gateXUID, "Alice"); err == nil || sauté {
		t.Fatalf("passe 1 : attendu un refus non sauté, obtenu err=%v sauté=%v", err, sauté)
	}
	u, _ := store.Load(gateXUID)
	if !u.ReauthRequired || u.LastAuthErrorClass != string(auth.AuthErrorRevoked) {
		t.Fatalf("état mort non consigné au store : %+v", u)
	}
	// Passes 2 à 5 : sautées, aucun appel à Microsoft.
	for i := 0; i < 4; i++ {
		if _, sauté, err := gate.Resolve(ctx, store, prov, gateXUID, "Alice"); !sauté || err != nil {
			t.Fatalf("passe %d : attendu sautée sans erreur, obtenu sauté=%v err=%v", i+2, sauté, err)
		}
	}
	if prov.appels != 1 {
		t.Fatalf("attendu 1 appel à Microsoft pour un jeton mort, obtenu %d", prov.appels)
	}

	// Import d'un jeton frais : le drapeau du store reste posé, l'empreinte change → essai, succès.
	if err := store.UpdateOAuthRefreshToken(gateXUID, "rt-frais"); err != nil {
		t.Fatal(err)
	}
	at, sauté, err := gate.Resolve(ctx, store, prov, gateXUID, "Alice")
	if err != nil || sauté || at != "at-rt-frais" {
		t.Fatalf("jeton importé : attendu un access_token, obtenu at=%q sauté=%v err=%v", at, sauté, err)
	}
	u, _ = store.Load(gateXUID)
	if u.ReauthRequired || u.LastAuthErrorClass != "" {
		t.Fatalf("succès sans effacement de l'état mort : %+v", u)
	}
}

// Au démarrage, un jeton déjà marqué mort dans le store (par le pool, ou avant le
// redémarrage) n'est pas réessayé ; un jeton remplacé ensuite l'est.
func TestPorteJetonsMorts_EntreeDejaMarquee(t *testing.T) {
	ctx := context.Background()
	gate := New()
	store := gateStore(t, "rt-mort")
	if _, err := store.MarkReauthRequired(gateXUID, "Alice"); err != nil {
		t.Fatal(err)
	}
	prov := &fakeRefreshProvider{morts: map[string]bool{"rt-mort": true}}

	for i := 0; i < 3; i++ {
		if _, sauté, _ := gate.Resolve(ctx, store, prov, gateXUID, "Alice"); !sauté {
			t.Fatalf("passe %d : un jeton marqué mort doit être sauté", i+1)
		}
	}
	if prov.appels != 0 {
		t.Fatalf("aucun appel attendu pour une entrée déjà marquée, obtenu %d", prov.appels)
	}
	if err := store.UpdateOAuthRefreshToken(gateXUID, "rt-frais"); err != nil {
		t.Fatal(err)
	}
	if at, sauté, err := gate.Resolve(ctx, store, prov, gateXUID, "Alice"); sauté || err != nil || at == "" {
		t.Fatalf("jeton remplacé : attendu un essai réussi, obtenu at=%q sauté=%v err=%v", at, sauté, err)
	}
}

// Un échec transitoire (réseau) ne suspend rien : le jeton est réessayé à la passe suivante.
func TestPorteJetonsMorts_EchecTransitoireReessaye(t *testing.T) {
	ctx := context.Background()
	gate := New()
	store := gateStore(t, "rt")
	prov := &transientProvider{}
	for i := 0; i < 2; i++ {
		if _, sauté, err := gate.Resolve(ctx, store, prov, gateXUID, "Alice"); sauté || err == nil {
			t.Fatalf("passe %d : attendu un essai en échec, obtenu sauté=%v err=%v", i+1, sauté, err)
		}
	}
	if prov.appels != 2 {
		t.Fatalf("attendu 2 essais, obtenu %d", prov.appels)
	}
}

type transientProvider struct {
	auth.TokenProvider
	appels int
}

func (p *transientProvider) TryOAuthRefreshWithRotation(context.Context, string) (string, string, error) {
	p.appels++
	return "", "", context.DeadlineExceeded
}
