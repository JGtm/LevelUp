package haloadapter

// politiques_et_absences_test.go — ce que l'adaptateur décide pour chaque appel de la brique :
// la politique de reprise de chaque chemin, la lecture « absent = vide » de chaque appel qui la
// portait, et les en-têtes et hôtes par défaut.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/haloapi/infinite"
	"levelup/go-api/internal/i18n"
)

// serveurStatut : un serveur qui rend toujours `statut`.
func serveurStatut(t *testing.T, statut int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statut)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// politiqueObservee déduit la politique de reprise des issues d'un même appel contre un serveur qui
// rend toujours 429, puis toujours 503, sous une échéance plus courte que la première attente :
//   - RetryProvider retente tout, 429 compris, et poursuit jusqu'au dernier essai : « N tentatives »
//     dans les deux cas ;
//   - RetryDiscovery rend 429 aussitôt et s'arrête sur la fin de contexte pendant l'attente ;
//   - RetryHaloClient rend 429 aussitôt et poursuit jusqu'au dernier essai (« N tentatives »).
func politiqueObservee(err429, err503 error) string {
	var he *infinite.HTTPError
	immediat429 := errors.As(err429, &he) && he.StatusCode == http.StatusTooManyRequests
	switch {
	case err429 == nil || err503 == nil:
	case immediat429 && strings.Contains(err503.Error(), "tentatives"):
		return "RetryHaloClient"
	case immediat429 && errors.Is(err503, context.DeadlineExceeded):
		return "RetryDiscovery"
	case !immediat429 && strings.Contains(err429.Error(), "tentatives") && strings.Contains(err503.Error(), "tentatives"):
		return "RetryProvider"
	}
	return fmt.Sprintf("inconnue (429 : %v ; 503 : %v)", err429, err503)
}

// TestPolitiqueParChemin : chaque chemin de l'adaptateur porte la politique de reprise de son
// ancien consommateur (fournisseur, Discovery, client de synchronisation). GetMatchPrivacy n'y est
// pas : par contrat il rend un repli sans erreur.
func TestPolitiqueParChemin(t *testing.T) {
	tokens := testTokens()
	cas := []struct {
		chemin  string
		attendu string
		appel   func(ctx context.Context, srv *httptest.Server) error
	}{
		{"Provider.FetchServiceRecord", "RetryProvider", func(ctx context.Context, srv *httptest.Server) error {
			_, err := providerVers(srv).fetchServiceRecordOnce(ctx, "Gamertag", "halo_infinite")
			return err
		}},
		{"Provider.FetchSeasonServiceRecord", "RetryProvider", func(ctx context.Context, srv *httptest.Server) error {
			_, err := providerVers(srv).fetchSeasonServiceRecordOnce(ctx, "Gamertag", "Season1", nil)
			return err
		}},
		{"Provider.FetchMatchStats", "RetryProvider", func(ctx context.Context, srv *httptest.Server) error {
			_, err := providerVers(srv).FetchMatchStats(ctx, matchTest, tokens)
			return err
		}},
		{"Provider.FetchMedalsMetadata", "RetryProvider", func(ctx context.Context, srv *httptest.Server) error {
			_, _, err := providerVers(srv).FetchMedalsMetadata(ctx, "hi")
			return err
		}},
		{"Provider.fetchBattlePass", "RetryProvider", func(ctx context.Context, srv *httptest.Server) error {
			_, _, err := providerVers(srv).fetchBattlePass(ctx, tokens, "1")
			return err
		}},
		{"Provider.fetchChallenges", "RetryProvider", func(ctx context.Context, srv *httptest.Server) error {
			_, _, err := providerVers(srv).fetchChallenges(ctx, tokens, "1")
			return err
		}},
		{"Provider.FetchSeasonCalendar", "RetryProvider", func(ctx context.Context, srv *httptest.Server) error {
			_, _, err := providerVers(srv).FetchSeasonCalendar(ctx, "hi")
			return err
		}},
		{"Provider.FetchCSRSeasonCalendar", "RetryProvider", func(ctx context.Context, srv *httptest.Server) error {
			_, _, err := providerVers(srv).FetchCSRSeasonCalendar(ctx, "hi")
			return err
		}},
		{"Provider.FetchAsset", "RetryDiscovery", func(ctx context.Context, srv *httptest.Server) error {
			_, err := providerVers(srv).FetchAsset(ctx, AssetTypeMap, "", matchTest, matchTest, "")
			return err
		}},
		{"Client.GetMatchStats", "RetryHaloClient", func(ctx context.Context, srv *httptest.Server) error {
			_, err := clientVers(srv, "s").GetMatchStats(ctx, matchTest)
			return err
		}},
		{"Client.GetFilmChunks (manifeste)", "RetryHaloClient", func(ctx context.Context, srv *httptest.Server) error {
			_, _, err := clientVers(srv, "s").GetFilmChunks(ctx, matchTest)
			return err
		}},
	}
	for _, tc := range cas {
		t.Run(tc.chemin, func(t *testing.T) {
			t.Parallel()
			issue := func(statut int) error {
				srv := serveurStatut(t, statut)
				ctx, cancel := context.WithTimeout(ctxWithAuth(tokens, "1"), 300*time.Millisecond)
				defer cancel()
				return tc.appel(ctx, srv)
			}
			if got := politiqueObservee(issue(http.StatusTooManyRequests), issue(http.StatusServiceUnavailable)); got != tc.attendu {
				t.Errorf("politique = %s, attendu %s", got, tc.attendu)
			}
		})
	}
}

// manifesteTempsFortsJSON : un manifeste dont l'unique fragment est celui des temps forts.
func manifesteTempsFortsJSON(prefixe string) []byte {
	b, _ := json.Marshal(map[string]any{
		"BlobStoragePathPrefix": prefixe,
		"CustomData": map[string]any{"FilmMajorVersion": 2, "Chunks": []map[string]any{
			{"Index": 0, "ChunkType": 3, "FileRelativePath": "c0.bin"},
		}},
	})
	return b
}

// TestAbsenceRendVide : une ressource absente côté service rend un résultat VIDE sans erreur, comme
// avant la brique — skill (map vide), CSR d'un joueur (tranche vide), CSR d'une playlist (nil), blob
// des temps forts (absent) sur 404/410 ; carrière (nil) sur refus 401/403 (portée joueur).
func TestAbsenceRendVide(t *testing.T) {
	ctx := context.Background()
	for _, statut := range []int{http.StatusNotFound, http.StatusGone} {
		c := clientVers(serveurStatut(t, statut), "s")
		if m, err := c.GetMatchSkill(ctx, matchTest, []string{"1"}); err != nil || m == nil || len(m) != 0 {
			t.Errorf("%d GetMatchSkill = %v, %v ; attendu map vide, nil", statut, m, err)
		}
		if s, err := c.GetPlayerCSRs(ctx, "1", "Season1"); err != nil || s == nil || len(s) != 0 {
			t.Errorf("%d GetPlayerCSRs = %v, %v ; attendu tranche vide, nil", statut, s, err)
		}
		if p, err := c.GetPlaylistCsr(ctx, "pl", "1", ""); err != nil || p != nil {
			t.Errorf("%d GetPlaylistCsr = %v, %v ; attendu nil, nil", statut, p, err)
		}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/spectate") {
				_, _ = w.Write(manifesteTempsFortsJSON("http://blobs.test/"))
				return
			}
			w.WriteHeader(statut)
		}))
		data, version, trouve, err := clientVers(srv, "s").GetHighlightEventsChunk(ctx, matchTest)
		srv.Close()
		if data != nil || version != 0 || trouve || err != nil {
			t.Errorf("%d blob des temps forts = (%v, %d, %v, %v) ; attendu (nil, 0, false, nil)", statut, data, version, trouve, err)
		}
	}
	for _, statut := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		c := clientVers(serveurStatut(t, statut), "s")
		if r, err := c.GetCareerProgress(ctx, "1"); err != nil || r != nil {
			t.Errorf("%d GetCareerProgress = %v, %v ; attendu nil, nil", statut, r, err)
		}
	}
}

// TestClassementEtHotesParDefaut : les pages du classement portent la langue de repli de
// `config/locales.toml` (« en-US,en;q=0.9 », la valeur d'avant la brique), l'agent de navigateur et
// le type accepté d'avant ; l'hôte des défis par défaut est celui d'avant.
func TestClassementEtHotesParDefaut(t *testing.T) {
	var langue, agent, accepte string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		langue, agent, accepte = r.Header.Get("Accept-Language"), r.Header.Get("User-Agent"), r.Header.Get("Accept")
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	s := NewLeaderboardScraper(0)
	s.host = srv.URL
	_, _ = s.FetchCSRLeaderboard(context.Background(), "csrseason13-3", "playlist", 10)

	if langue != "en-US,en;q=0.9" || langue != i18n.HaloTags(i18n.Fallback)[0]+","+string(i18n.Fallback)+";q=0.9" {
		t.Errorf("Accept-Language = %q, attendu \"en-US,en;q=0.9\" (langue de repli de config/locales.toml)", langue)
	}
	if agent != "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36" {
		t.Errorf("User-Agent = %q", agent)
	}
	if accepte != "text/html,application/xhtml+xml" {
		t.Errorf("Accept = %q", accepte)
	}
	if got := NewHaloProvider().hostFor(context.Background(), infinite.CleChallenges); got != "https://halostats.svc.halowaypoint.com" {
		t.Errorf("hôte des défis par défaut = %q, attendu https://halostats.svc.halowaypoint.com", got)
	}
}
