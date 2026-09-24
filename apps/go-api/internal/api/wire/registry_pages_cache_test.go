package wire

// registry_pages_cache_test.go — câblage du cache des lectures joueur (plan perf
// 2026-09-23, lot L5b, D5b.3 et D5b.4) : les factories rendent les repos cachés.
// Aucune base n'est ouverte (construction seule).

import (
	"context"
	"reflect"
	"testing"

	"levelup/go-api/internal/platform/duckdb"
)

func stubPlayerDB() *duckdb.PlayerDB {
	return &duckdb.PlayerDB{XUID: "2533274800000001", Gamertag: "GT", TitleSlug: "halo_infinite"}
}

func TestPlayerMatchesAdapterFor_IsCached(t *testing.T) {
	reg := &ServiceRegistry{}
	got := reg.playerMatchesAdapterFor(stubPlayerDB())
	if _, ok := got.(*duckdb.CachedPlayerMatchesRepo); !ok {
		t.Errorf("playerMatchesAdapterFor rend %T, want *duckdb.CachedPlayerMatchesRepo (D5b.4)", got)
	}
}

func TestFilters_UsesCachedFiltersRepo(t *testing.T) {
	pdb := stubPlayerDB()
	reg := &ServiceRegistry{resolve: func(context.Context, string) (*duckdb.PlayerDB, error) { return pdb, nil }}
	svc, err := reg.Filters(context.Background(), "gt")
	if err != nil {
		t.Fatalf("Filters : %v", err)
	}
	repo := reflect.ValueOf(svc).Elem().FieldByName("repo")
	if !repo.IsValid() {
		t.Fatal("champ repo introuvable sur FiltersService (renommé ? mettre ce test à jour)")
	}
	if got := repo.Elem().Type(); got != reflect.TypeOf(&duckdb.CachedFiltersRepo{}) {
		t.Errorf("FiltersService.repo = %v, want *duckdb.CachedFiltersRepo (D5b.3)", got)
	}
}
