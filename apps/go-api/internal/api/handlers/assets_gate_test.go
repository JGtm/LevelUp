package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/assets"
)

// spyFetchResolver est le resolver derrière la porte : stock local vide, compteur
// d'appels sortants.
type spyFetchResolver struct{ fetches atomic.Int32 }

func (s *spyFetchResolver) LookupLocal(context.Context, assets.Ref) (assets.Resolved, bool) {
	return assets.Resolved{}, false
}

func (s *spyFetchResolver) Get(context.Context, assets.Ref) (assets.Resolved, error) {
	s.fetches.Add(1)
	return assets.Resolved{Payload: assets.URLPayload{URL: "https://cdn.example.test/m.png"}}, nil
}

// medal42Catalog ne connaît que la médaille 42 et rend les cartes invérifiables.
type medal42Catalog struct{}

func (medal42Catalog) Classify(_ context.Context, ref assets.Ref) assets.Verdict {
	switch {
	case ref.Kind == assets.KindMedalImage && ref.ID == "42":
		return assets.VerdictKnown
	case ref.Kind == assets.KindChallengeBadge:
		return assets.VerdictUnverifiable
	}
	return assets.VerdictUnknown
}

func newGatedRouter(spy *spyFetchResolver, trusted bool) http.Handler {
	h := handlers.NewAssetHandler(assets.NewGate(spy, medal42Catalog{})).
		WithTrust(func(*http.Request) bool { return trusted })
	r := chi.NewRouter()
	r.Get("/assets/medals/{title_id}/{medal_id}/image", h.GetMedalImage)
	r.Get("/assets/challenge-badge/{title_id}/{badge_id}", h.GetChallengeBadge)
	return r
}

func getPath(r http.Handler, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

// Un identifiant inconnu répond 404 et ne déclenche aucun appel sortant, en boucle.
func TestAssetRoutes_UnknownIDDoesNotCallOut(t *testing.T) {
	spy := &spyFetchResolver{}
	r := newGatedRouter(spy, true)
	for range 100 {
		if w := getPath(r, "/assets/medals/halo_infinite/777777/image"); w.Code != http.StatusNotFound {
			t.Fatalf("statut %d, attendu 404", w.Code)
		}
	}
	if got := spy.fetches.Load(); got != 0 {
		t.Fatalf("appels sortants = %d, attendu 0", got)
	}
}

// Un identifiant connu est résolu.
func TestAssetRoutes_KnownIDIsResolved(t *testing.T) {
	spy := &spyFetchResolver{}
	r := newGatedRouter(spy, false)
	if w := getPath(r, "/assets/medals/halo_infinite/42/image"); w.Code != http.StatusFound {
		t.Fatalf("statut %d, attendu 302", w.Code)
	}
	if got := spy.fetches.Load(); got != 1 {
		t.Fatalf("appels sortants = %d, attendu 1", got)
	}
}

// Sans prédicat de confiance posé, un kind sans catalogue n'est jamais récupéré.
func TestAssetRoutes_UnverifiableKindNeedsTrust(t *testing.T) {
	spy := &spyFetchResolver{}
	r := chi.NewRouter()
	h := handlers.NewAssetHandler(assets.NewGate(spy, medal42Catalog{}))
	r.Get("/assets/challenge-badge/{title_id}/{badge_id}", h.GetChallengeBadge)
	if w := getPath(r, "/assets/challenge-badge/halo_infinite/combat-x"); w.Code != http.StatusNotFound {
		t.Fatalf("statut %d, attendu 404", w.Code)
	}
	if spy.fetches.Load() != 0 {
		t.Fatalf("appels sortants = %d, attendu 0", spy.fetches.Load())
	}

	trusted := newGatedRouter(spy, true)
	if w := getPath(trusted, "/assets/challenge-badge/halo_infinite/combat-x"); w.Code != http.StatusFound {
		t.Fatalf("appelant de confiance : statut %d, attendu 302", w.Code)
	}
	if spy.fetches.Load() != 1 {
		t.Fatalf("appels sortants = %d, attendu 1", spy.fetches.Load())
	}
}
