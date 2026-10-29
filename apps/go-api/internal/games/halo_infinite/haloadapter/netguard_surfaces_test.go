package haloadapter

// netguard_surfaces_test.go — les HUIT surfaces sortantes de l'adaptateur sont gardées en mode
// démo. Pour chacune, une méthode publique qui la traverse est appelée contre un serveur témoin :
// aucune requête ne doit l'atteindre, la garde `netguard` doit avoir été consultée SOUS LE NOM de
// la surface (journal de `netguard`, une ligne par surface), et l'erreur, quand la méthode la
// rend, doit être `netguard.ErrOffline` marquée définitive. Un `*http.Client` nu passé à la
// brique à la place du `Doer` gardé laisse partir la requête : ce test rougit.

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/haloapi"
	"levelup/go-api/internal/platform/netguard"
	"levelup/go-api/internal/sync/filmfetch"

	"github.com/JGtm/LevelUp/libs/film/filmcache"
)

// surfacesJournal capture les surfaces que `netguard` journalise (une ligne par surface coupée).
type surfacesJournal struct {
	mu       sync.Mutex
	surfaces []string
}

func (j *surfacesJournal) Enabled(context.Context, slog.Level) bool { return true }
func (j *surfacesJournal) WithAttrs([]slog.Attr) slog.Handler       { return j }
func (j *surfacesJournal) WithGroup(string) slog.Handler            { return j }
func (j *surfacesJournal) Handle(_ context.Context, r slog.Record) error {
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "surface" {
			j.mu.Lock()
			j.surfaces = append(j.surfaces, a.Value.String())
			j.mu.Unlock()
		}
		return true
	})
	return nil
}

func (j *surfacesJournal) vues() []string {
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]string(nil), j.surfaces...)
}

// modeDemoJournalise arme netguard à neuf (journal par surface remis à zéro) et capture le
// journal par défaut ; tout est rétabli en fin de test.
func modeDemoJournalise(t *testing.T) *surfacesJournal {
	t.Helper()
	j := &surfacesJournal{}
	avant := slog.Default()
	slog.SetDefault(slog.New(j))
	netguard.SetOffline(false)
	netguard.SetOffline(true)
	t.Cleanup(func() {
		netguard.SetOffline(false)
		slog.SetDefault(avant)
	})
	return j
}

// temoin : un serveur qui compte les requêtes reçues (il ne doit en recevoir aucune).
func temoin(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, &n
}

// manifesteFinaliseEnCache écrit, pour matchTest, un manifeste FINALISÉ dans un cache disque neuf
// (aucun chunk en cache) : la lecture du film saute le manifeste de l'API et télécharge les blobs.
func manifesteFinaliseEnCache(t *testing.T, prefixe string) *filmfetch.LocalFilmCache {
	t.Helper()
	dir := t.TempDir()
	m := filmfetch.CachedManifest{BlobPrefix: prefixe, Chunks: []filmfetch.CachedChunk{
		{Index: 1, ChunkType: filmfetch.FilmChunkTypeReplicationData, FileRelativePath: "c1.bin"},
		{Index: 2, ChunkType: filmfetch.FilmChunkTypeHighlightEvents, FileRelativePath: "c2.bin"},
	}}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	chemin := filmcache.ManifestPath(dir, title.FilmShortMatchID(matchTest))
	if err := os.MkdirAll(filepath.Dir(chemin), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chemin, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return filmfetch.NewLocalFilmCache(dir)
}

// providerVers : un fournisseur dont toute requête aboutit à srv.
func providerVers(srv *httptest.Server) *Provider {
	p := NewHaloProvider()
	p.client = &http.Client{Transport: &redirectTransport{host: strings.TrimPrefix(srv.URL, "http://")}}
	p.limiter = newRateLimiter(600)
	return p
}

// defaultClientVers fait aboutir à srv les requêtes de `http.DefaultClient` (client des plaques,
// relu à chaque appel) le temps du test.
func defaultClientVers(t *testing.T, srv *httptest.Server) {
	t.Helper()
	avant := http.DefaultClient.Transport
	http.DefaultClient.Transport = &redirectTransport{host: strings.TrimPrefix(srv.URL, "http://")}
	t.Cleanup(func() { http.DefaultClient.Transport = avant })
}

func TestNetguard_HuitSurfacesGardees(t *testing.T) {
	const emblemeTest = "Inventory/Spartan/Emblems/104-001-test-emblem-abcdef.json"
	cas := []struct {
		surface string
		// erreurRendue : la méthode rend l'erreur de la garde (sinon elle dégrade sans erreur, par
		// contrat : la plaque rend "").
		erreurRendue bool
		appel        func(t *testing.T, srv *httptest.Server) error
	}{
		{surfaceAPIGet, true, func(_ *testing.T, srv *httptest.Server) error {
			_, err := clientVers(srv, "s").GetMatchStats(context.Background(), matchTest)
			return err
		}},
		{surfaceAPIBlob, true, func(t *testing.T, srv *httptest.Server) error {
			c := clientVers(srv, "s")
			c.localFilmCache = manifesteFinaliseEnCache(t, srv.URL+"/blobs/")
			_, _, err := c.GetFilmChunks(context.Background(), matchTest)
			return err
		}},
		{surfaceProviderGet, true, func(_ *testing.T, srv *httptest.Server) error {
			_, err := providerVers(srv).FetchMatchStats(context.Background(), matchTest, testTokens())
			return err
		}},
		{surfaceDiscoveryGet, true, func(_ *testing.T, srv *httptest.Server) error {
			_, err := providerVers(srv).FetchAsset(context.Background(), AssetTypeMap, "", matchTest, matchTest, "en-US")
			return err
		}},
		{surfaceNameplateMapping, false, func(t *testing.T, srv *httptest.Server) error {
			resetEmblemMappingCacheForTest()
			t.Cleanup(resetEmblemMappingCacheForTest)
			defaultClientVers(t, srv)
			ResolveNameplateURL(context.Background(), emblemeTest, 372285867, "s", "c")
			return nil
		}},
		{surfaceNameplateEmblem, false, func(t *testing.T, srv *httptest.Server) error {
			// Table fraîche sans l'emblème : pas de relecture de la table, lecture du descripteur.
			seedEmblemMappingCacheForTest(map[string]map[string]emblemMappingEntry{})
			t.Cleanup(resetEmblemMappingCacheForTest)
			defaultClientVers(t, srv)
			if got := ResolveNameplateURL(context.Background(), emblemeTest, -5, "s", "c"); got != "" {
				t.Errorf("URL = %q, attendu \"\" (descripteur non lu)", got)
			}
			return nil
		}},
		{surfaceClassementPage, true, func(_ *testing.T, srv *httptest.Server) error {
			s := NewLeaderboardScraper(0)
			s.host = srv.URL
			_, err := s.FetchCSRLeaderboard(context.Background(), "csrseason13-3", "playlist", 10)
			return err
		}},
		{surfaceClassementSaison, true, func(_ *testing.T, srv *httptest.Server) error {
			s := NewLeaderboardScraper(0)
			s.host = srv.URL
			_, err := s.FetchActiveSeasonByRedirect(context.Background())
			return err
		}},
	}
	for _, tc := range cas {
		t.Run(tc.surface, func(t *testing.T) {
			srv, recues := temoin(t)
			j := modeDemoJournalise(t)
			err := tc.appel(t, srv)
			if n := recues.Load(); n != 0 {
				t.Fatalf("%d requête(s) reçue(s) par le témoin en mode démo", n)
			}
			if tc.erreurRendue && (!errors.Is(err, netguard.ErrOffline) || !errors.Is(err, haloapi.ErrDefinitif)) {
				t.Errorf("erreur = %v, attendu netguard.ErrOffline marquée haloapi.ErrDefinitif", err)
			}
			vues := j.vues()
			trouve := false
			for _, s := range vues {
				trouve = trouve || s == tc.surface
			}
			if !trouve {
				t.Errorf("garde non consultée sous la surface %q (surfaces journalisées : %v)", tc.surface, vues)
			}
		})
	}
}

// TestNetguard_RefusAvantLimiteur : une sortie refusée en mode démo ne consomme aucun jeton du
// limiteur du client (la garde est consultée avant toute attente).
func TestNetguard_RefusAvantLimiteur(t *testing.T) {
	srv, recues := temoin(t)
	modeDemoJournalise(t)
	c := clientVers(srv, "s")
	c.limiter = rate.NewLimiter(rate.Every(time.Hour), 1)
	// Échéance courte : si la garde passait après le limiteur, le second appel attendrait un jeton
	// (une heure) ; l'échéance le fait échouer au lieu de bloquer le test.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for i := 0; i < 2; i++ {
		if _, err := c.GetMatchStats(ctx, matchTest); !errors.Is(err, netguard.ErrOffline) {
			t.Fatalf("appel %d : erreur = %v, attendu netguard.ErrOffline", i+1, err)
		}
	}
	if jetons := c.limiter.Tokens(); jetons < 0.99 {
		t.Errorf("jetons du limiteur = %.2f après deux refus, attendu 1 (aucun consommé)", jetons)
	}
	if recues.Load() != 0 {
		t.Errorf("%d requête(s) reçue(s)", recues.Load())
	}
}
