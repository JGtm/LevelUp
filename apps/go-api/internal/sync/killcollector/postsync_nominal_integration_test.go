//go:build integration

package killcollector

// postsync_nominal_integration_test.go — LE CHEMIN NOMINAL DE L ETAPE 1.57, DE BOUT EN BOUT.
//
// POURQUOI CE FICHIER EXISTE. La revue du 2026-08-29 a etabli qu AUCUN test n atteignait le
// corps de `RunPostSync` : les deux tests unitaires s arretaient sur ses gardes d entree
// (dependance manquante, capability absente). Tout ce qui suit — lecture du backlog, ordre de
// travail, resolution de la racine de cache, construction de la source, budget, cablage du
// collecteur, compteurs — n etait couvert par rien. N importe quelle inversion dans ce bloc
// (arguments permutes, `travail` confondu avec `backlog`, inseres ignores) passait au vert.
//
// Ce test-ci fait tourner l etape sur une base migree, avec une source de films en memoire.
// Il ne cherche PAS a decoder un vrai film : ce que le decodeur fait des octets est couvert
// ailleurs (collector_test.go). Ce qu il verrouille, c est que la MECANIQUE atteint le
// collecteur, dans le bon ordre, sur les bons matchs, et rend ce qu elle annonce.

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/haloclient"
	"levelup/go-api/internal/sync/matchflags"
)

// filmsTraces : une source de films qui NOTE l ordre dans lequel on la sollicite. C est cet
// ordre qui porte la correction du 2026-08-29 (les recents d abord) ; sans temoin, il ne se
// verifie qu en relisant le SQL.
type filmsTraces struct {
	demandes []string
}

func (f *filmsTraces) GetFilmChunks(_ context.Context, matchID string) ([]haloclient.FilmChunk, bool, error) {
	f.demandes = append(f.demandes, matchID)
	return nil, false, nil // film absent : etat NORMAL, la passe continue
}

// depsDeTest cable RunPostSync sur une base reelle, avec des segments de lecture et un writer
// qui rendent le meme handle — le process de test est seul, ADR 0013 est satisfaite par ce
// fait, pas par un verrou de plus.
func depsDeTest(db *sql.DB, films *filmsTraces, cache *haloclient.LocalFilmCache) PostSyncDeps {
	return PostSyncDeps{
		Fetcher:       films,
		LocalCache:    cache,
		WithRead:      func(ctx context.Context, _ string, fn func(*sql.DB)) { fn(db) },
		AcquireWriter: func(context.Context) (*sql.DB, func(), error) { return db, func() {}, nil },
		TitleSlug:     "halo_infinite",
		Gamertag:      "GT",
	}
}

// TestRunPostSync_CheminNominal : l etape lit le backlog, sert les inseres d abord puis les
// plus recents, borne son travail, et publie sa jauge.
func TestRunPostSync_CheminNominal(t *testing.T) {
	db := baseBacklog(t)
	t0 := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	// Cinq candidats, du plus vieux au plus recent, plus un exclu par le marqueur terminal.
	for i, id := range []string{"m1", "m2", "m3", "m4", "m5"} {
		inscrireMatch(t, db, id, t0.AddDate(0, i, 0), 0)
	}
	inscrireMatch(t, db, "perdu", t0.AddDate(0, 6, 0), int64(matchflags.MBitFilmAbsent))

	films := &filmsTraces{}
	// perCycle = 3 : la borne doit mordre.
	h := NewPostSyncHook(racineDepot(t), 3)
	// L insere est le PLUS VIEUX du lot : s il passe en tete, c est bien la priorite « insere »
	// qui joue, et non l ordre du backlog.
	ecrits := RunPostSync(context.Background(), h, depsDeTest(db, films, nil), []string{"m1"})

	if ecrits != 0 {
		t.Errorf("ecrits = %d, attendu 0 (aucun film servi par la source)", ecrits)
	}
	attendu := []string{"m1", "m5", "m4"}
	if len(films.demandes) != len(attendu) {
		t.Fatalf("films demandes = %v, attendu %v (la borne perCycle=3 doit mordre)", films.demandes, attendu)
	}
	for i, id := range attendu {
		if films.demandes[i] != id {
			t.Errorf("demande[%d] = %q, attendu %q — l insere d abord, puis du plus RECENT au "+
				"plus vieux", i, films.demandes[i], id)
		}
	}
	for _, id := range films.demandes {
		if id == "perdu" {
			t.Error("un match marque « film absent » a ete redemande : le backlog ne draine pas")
		}
	}

	// La jauge porte le RESTE du backlog total (5 candidats - 3 traites), pas l horizon.
	if v := observability.LoadCounter(CompteurPostSyncRetard); v != 2 {
		t.Errorf("%s = %d, attendu 2", CompteurPostSyncRetard, v)
	}
}

// TestRunPostSync_PrepareLeCacheEtArchiveAuMemeEndroit : LA PROPRIETE QUE TROIS RELECTEURS ONT
// TROUVEE MANQUANTE le meme jour — la racine LUE et la racine ECRITE doivent etre la meme.
//
// Sans elle, un film telecharge est archive dans un repertoire que la lecture du cycle suivant
// ne consulte jamais : le repli disque ne prend plus et chaque cycle repaie le reseau en
// entier. Le test verifie les deux faces : les dossiers sont crees (sinon
// `NewLocalFilmCache` rend nil pour toute la vie du process), et c est la racine du MOTEUR qui
// gagne quand il en porte une.
func TestRunPostSync_PrepareLeCacheEtArchiveAuMemeEndroit(t *testing.T) {
	db := baseBacklog(t)
	inscrireMatch(t, db, "m1", time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), 0)

	// (a) sans cache moteur : le hook prepare SA racine (celle du PathResolver).
	depot := racineDepot(t)
	h := NewPostSyncHook(depot, 1)
	RunPostSync(context.Background(), h, depsDeTest(db, &filmsTraces{}, nil), nil)
	for _, d := range []string{filmcache.ChunksRoot(h.cacheRacine), filepath.Dir(filmcache.ManifestPath(h.cacheRacine, "0"))} {
		if st, err := os.Stat(d); err != nil || !st.IsDir() {
			t.Errorf("dossier de cache non prepare (%s) : %v", d, err)
		}
	}
	if h.cacheRacine == "" {
		t.Fatal("aucune racine de cache resolue")
	}

	// (b) avec un cache moteur : SA racine gagne, en lecture COMME en ecriture.
	//
	// UN NOUVEAU CANDIDAT EST NECESSAIRE. Depuis le 2026-09-01 la passe pose le marqueur
	// terminal « film absent » sur les matchs qu elle a vus sans film : `m1` a donc quitte le
	// backlog en (a), et une passe sans travail sort AVANT de resoudre sa racine de cache.
	inscrireMatch(t, db, "m2", time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC), 0)
	autre := t.TempDir()
	if err := filmcache.EnsureDirs(autre); err != nil {
		t.Fatal(err)
	}
	h2 := NewPostSyncHook(depot, 1)
	RunPostSync(context.Background(), h2, depsDeTest(db, &filmsTraces{}, haloclient.NewLocalFilmCache(autre)), nil)
	if h2.cacheRacine != autre {
		t.Errorf("racine retenue = %q, attendu celle du moteur %q — deux racines et l archivage "+
			"n est jamais relu", h2.cacheRacine, autre)
	}
}
