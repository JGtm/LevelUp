package haloadapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

const seasonCalendarBody = `{"Seasons":[{"SeasonMetadataPath":"Seasons/Season1.json","StartDate":"2021-11-15T00:00:00Z","EndDate":"2022-05-03T00:00:00Z"}]}`

func providerForSeasons(srvURL string) *Provider {
	p := newTestProvider("", srvURL)
	return p
}

// 401 puis 200 après rafraîchissement du jeton : le calendrier est servi, en un seul nouvel essai.
func TestFetchSeasonCalendar_401ThenFreshToken(t *testing.T) {

	var hits int32
	var dernierJeton atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dernierJeton.Store(r.Header.Get("x-343-authorization-spartan"))
		if atomic.AddInt32(&hits, 1) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(seasonCalendarBody))
	}))
	defer srv.Close()

	var mints int32
	avecRafraichisseur(t, "xuid-saisons", func(_ context.Context, _ string) (*domain.HaloTokens, error) {
		atomic.AddInt32(&mints, 1)
		return &domain.HaloTokens{SpartanToken: "frais", SpartanExpiresAt: time.Now().Add(time.Hour)}, nil
	})

	p := providerForSeasons(srv.URL)
	seasons, _, err := p.FetchSeasonCalendar(ctxWithAuth(testTokens(), "xuid-saisons"), "halo_infinite")
	if err != nil || len(seasons) != 1 {
		t.Fatalf("FetchSeasonCalendar = %d saison(s), err %v ; want 1 saison après le nouvel essai", len(seasons), err)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Errorf("appels serveur = %d, want 2 (401 puis nouvel essai)", got)
	}
	if got := atomic.LoadInt32(&mints); got != 1 {
		t.Errorf("rafraîchissements = %d, want 1", got)
	}
	if j, _ := dernierJeton.Load().(string); j != "frais" {
		t.Errorf("jeton du nouvel essai = %q, want le jeton rafraîchi", j)
	}
}

// 403 persistant (révocation réelle) : un seul nouvel essai, puis l'erreur d'origine.
func TestFetchSeasonCalendar_403Twice_NoLoop(t *testing.T) {

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	avecRafraichisseur(t, "xuid-saisons-2", func(_ context.Context, _ string) (*domain.HaloTokens, error) {
		return &domain.HaloTokens{SpartanToken: "refuse", SpartanExpiresAt: time.Now().Add(time.Hour)}, nil
	})

	_, _, err := providerForSeasons(srv.URL).FetchSeasonCalendar(ctxWithAuth(testTokens(), "xuid-saisons-2"), "halo_infinite")
	if err == nil || !IsAuthError(err) {
		t.Fatalf("err = %v, want un échec d'auth", err)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Errorf("appels serveur = %d, want 2 (jamais de boucle)", got)
	}
}
