package handlers_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/assets"
)

const (
	mapKnownID      = "298bff6c-79e6-4d2f-a3b5-bd8ec5b02e25"
	mapKnownVersion = "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
	mapNewVersion   = "0f0e0d0c-0b0a-4908-8706-050403020100"
)

// fsSpyResolver est le resolver derrière la porte : stock local réel (LocalFSStore) et
// compteur d'appels sortants.
type fsSpyResolver struct {
	store   *assets.LocalFSStore
	fetches atomic.Int32
}

func (s *fsSpyResolver) LookupLocal(ctx context.Context, ref assets.Ref) (assets.Resolved, bool) {
	bin, err := s.store.LookupBinary(ctx, ref)
	if err != nil || bin == nil {
		return assets.Resolved{}, false
	}
	return assets.Resolved{Payload: *bin, Source: assets.SourceLocalFile}, true
}

func (s *fsSpyResolver) Get(context.Context, assets.Ref) (assets.Resolved, error) {
	s.fetches.Add(1)
	return assets.Resolved{Payload: assets.URLPayload{URL: "https://cdn.example.test/map.jpg"}}, nil
}

// mapCatalogFixture crée une metadata.duckdb où la carte mapKnownID est au catalogue
// avec la version courante mapKnownVersion.
func mapCatalogFixture(t *testing.T) *assets.MetadataCatalog {
	t.Helper()
	path := filepath.Join(t.TempDir(), "metadata.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`CREATE TABLE maps_catalog (title_slug VARCHAR, map_asset_id VARCHAR, current_version_id VARCHAR)`,
		`INSERT INTO maps_catalog VALUES ('halo_infinite', '` + mapKnownID + `', '` + mapKnownVersion + `')`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("%s : %v", stmt, err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	store := assets.NewDuckDBIndexStore(path)
	t.Cleanup(store.Close)
	return assets.NewMetadataCatalog(store, func(id string) bool { return id == "halo_infinite" })
}

// newMapRouter monte la route réelle des images de carte derrière la porte et le
// catalogue de métadonnées.
func newMapRouter(t *testing.T, spy *fsSpyResolver, trusted bool) http.Handler {
	t.Helper()
	h := handlers.NewAssetHandler(assets.NewGate(spy, mapCatalogFixture(t))).
		WithTrust(func(*http.Request) bool { return trusted })
	r := chi.NewRouter()
	r.Get("/assets/maps/{title_id}/{map_id}/image", h.GetMapImage)
	return r
}

func mapImagePath(version string) string {
	return "/assets/maps/halo_infinite/" + mapKnownID + "/image?v=" + version
}

// Route des cartes : une version absente du catalogue ne déclenche aucun appel sortant
// pour un anonyme, quel que soit le nombre de versions inventées ; la version du
// catalogue est récupérée ; un appelant de confiance peut récupérer une version neuve.
func TestAssetRoutes_MapImage_NewVariantCallsOutOnlyForTrustedCallers(t *testing.T) {
	spy := &fsSpyResolver{store: assets.NewLocalFSStore(t.TempDir())}
	anon := newMapRouter(t, spy, false)
	for i := range 50 {
		v := fmt.Sprintf("0f0e0d0c-0b0a-4908-8706-%012x", i)
		if w := getPath(anon, mapImagePath(v)); w.Code != http.StatusNotFound {
			t.Fatalf("anonyme, version inventée %s : statut %d, attendu 404", v, w.Code)
		}
	}
	if got := spy.fetches.Load(); got != 0 {
		t.Fatalf("anonyme : %d appels sortants pour des versions inventées, attendu 0", got)
	}

	if w := getPath(anon, mapImagePath(mapKnownVersion)); w.Code != http.StatusFound {
		t.Fatalf("anonyme, version du catalogue : statut %d, attendu 302", w.Code)
	}
	if got := spy.fetches.Load(); got != 1 {
		t.Fatalf("version du catalogue : %d appel(s) sortant(s), attendu 1", got)
	}

	trusted := newMapRouter(t, spy, true)
	if w := getPath(trusted, mapImagePath(mapNewVersion)); w.Code != http.StatusFound {
		t.Fatalf("appelant de confiance, version neuve : statut %d, attendu 302", w.Code)
	}
	if got := spy.fetches.Load(); got != 2 {
		t.Fatalf("appelant de confiance : %d appels sortants au total, attendu 2", got)
	}
}

// Route des cartes : v n'a qu'une forme, celle d'un UUID. Toute autre valeur est refusée
// (400) avant le stock local et toute source distante.
func TestAssetRoutes_MapImage_MalformedVariantIsRefused(t *testing.T) {
	spy := &fsSpyResolver{store: assets.NewLocalFSStore(t.TempDir())}
	r := newMapRouter(t, spy, true)
	for _, v := range []string{
		"x", "latest", mapKnownVersion + "0", "..", "%2e%2e", "%2F..%2Fsecret",
		"/" + strings.Repeat("../", 4) + "secret", `..\..\secret`, "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4z",
	} {
		if w := getPath(r, mapImagePath(v)); w.Code != http.StatusBadRequest {
			t.Errorf("v=%q : statut %d, attendu 400", v, w.Code)
		}
	}
	if got := spy.fetches.Load(); got != 0 {
		t.Fatalf("%d appels sortants pour des v mal formés, attendu 0", got)
	}
}

// Route des cartes : un v qui remonte l'arborescence ne lit aucun fichier hors du stock,
// même quand le fichier visé existe (aucun proxy devant le serveur).
func TestAssetRoutes_MapImage_TraversalReadsNothing(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "cache")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "secret.png"), []byte("\x89PNG\r\n\x1a\nsecret"), 0o600); err != nil {
		t.Fatal(err)
	}
	spy := &fsSpyResolver{store: assets.NewLocalFSStore(root)}
	r := newMapRouter(t, spy, true)
	w := getPath(r, mapImagePath("/"+strings.Repeat("../", 4)+"secret"))
	if w.Code == http.StatusOK {
		t.Fatalf("fichier hors du stock servi : %q", w.Body.String())
	}
}
