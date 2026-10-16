package handlers_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/assets"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync"
)

const (
	// mapPlayedVersion : version de mapKnownID jouée dans un match, distincte de la version
	// courante du catalogue (mapKnownVersion).
	mapPlayedVersion = "1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d"
	// mapUncatalogued : carte jouée, absente des métadonnées (carte toute neuve).
	mapUncatalogued        = "7c6b5a49-3827-4165-9483-726150493827"
	mapUncataloguedVersion = "9e8d7c6b-5a49-4382-9716-151413121110"
)

// dbSharedReader sert une base partagée ouverte par le test (lecteur du provider en service).
type dbSharedReader struct{ db *sql.DB }

func (r dbSharedReader) Get(context.Context) (*sql.DB, func(), error) { return r.db, func() {}, nil }

// playedMatchesFixture crée une base partagée au schéma de production (sync.EnsureSharedSchema)
// dont match_registry enregistre trois matchs : mapKnownID en mapPlayedVersion, la carte hors
// catalogue mapUncatalogued, et un match sans version.
func playedMatchesFixture(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared_matches_v2.duckdb"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	if err := sync.EnsureSharedSchema(ctx, db); err != nil {
		t.Fatalf("schéma partagé : %v", err)
	}
	for _, row := range [][3]string{
		{"m1", mapKnownID, mapPlayedVersion},
		{"m2", mapUncatalogued, mapUncataloguedVersion},
		{"m3", mapKnownID, ""},
	} {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO match_registry (match_id, start_time, map_id, map_version_id) VALUES (?, TIMESTAMP '2026-10-01 20:00:00', ?, ?)`,
			row[0], row[1], row[2]); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

// newPlayedMapRouter monte la route réelle des images de carte derrière la porte, jugée par le
// catalogue de métadonnées et les versions jouées de la base partagée.
func newPlayedMapRouter(t *testing.T, spy *fsSpyResolver, trusted bool) http.Handler {
	t.Helper()
	shared := dbSharedReader{db: playedMatchesFixture(t)}
	played := assets.NewPlayedMapVersions(func(ctx context.Context, _ string, visit func(string, string)) error {
		return platform_duckdb.ForEachPlayedMapVersion(ctx, shared, visit)
	})
	catalog := mapCatalogFixture(t).WithPlayedMapVersions(played)
	h := handlers.NewAssetHandler(assets.NewGate(spy, catalog)).
		WithTrust(func(*http.Request) bool { return trusted })
	r := chi.NewRouter()
	r.Get("/assets/maps/{title_id}/{map_id}/image", h.GetMapImage)
	return r
}

// Route des cartes : une version présente dans de vrais matchs est récupérée pour un anonyme,
// y compris pour une carte encore absente des métadonnées ; une version inventée reste refusée
// sans appel sortant, quel qu'en soit le nombre.
func TestAssetRoutes_MapImage_PlayedVersionIsServedToAnonymous(t *testing.T) {
	spy := &fsSpyResolver{store: assets.NewLocalFSStore(t.TempDir())}
	anon := newPlayedMapRouter(t, spy, false)

	if w := getPath(anon, mapImagePath(mapPlayedVersion)); w.Code != http.StatusFound {
		t.Fatalf("anonyme, version jouée : statut %d, attendu 302", w.Code)
	}
	uncatalogued := "/assets/maps/halo_infinite/" + mapUncatalogued + "/image?v=" + mapUncataloguedVersion
	if w := getPath(anon, uncatalogued); w.Code != http.StatusFound {
		t.Fatalf("anonyme, carte hors métadonnées mais jouée : statut %d, attendu 302", w.Code)
	}
	if got := spy.fetches.Load(); got != 2 {
		t.Fatalf("versions jouées : %d appel(s) sortant(s), attendu 2", got)
	}

	for i := range 50 {
		v := fmt.Sprintf("0f0e0d0c-0b0a-4908-8706-%012x", i)
		if w := getPath(anon, mapImagePath(v)); w.Code != http.StatusNotFound {
			t.Fatalf("anonyme, version inventée %s : statut %d, attendu 404", v, w.Code)
		}
		inventedMap := fmt.Sprintf("/assets/maps/halo_infinite/%s/image?v=%s", mapUncatalogued, v)
		if w := getPath(anon, inventedMap); w.Code != http.StatusNotFound {
			t.Fatalf("anonyme, carte hors métadonnées, version inventée %s : statut %d, attendu 404", v, w.Code)
		}
	}
	// Version jouée d'une AUTRE carte : le couple (carte, version) n'a jamais été joué.
	if w := getPath(anon, mapImagePath(mapUncataloguedVersion)); w.Code != http.StatusNotFound {
		t.Fatalf("anonyme, version jouée sur une autre carte : statut %d, attendu 404", w.Code)
	}
	if got := spy.fetches.Load(); got != 2 {
		t.Fatalf("versions inventées : %d appel(s) sortant(s) au total, attendu 2", got)
	}
}
