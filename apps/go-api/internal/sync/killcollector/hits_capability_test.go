//go:build integration

// Package killcollector — hits_capability_test.go : LA PORTE `CapWeaponAccuracy` du numerateur de
// precision par arme, SANS FILM.
//
// MEME MOTIF QUE shots_capability_test (constat J4R-4) : la passe de precision est DIR-BASE (elle
// rejoue un film sur disque), donc son chemin « ecrit des lignes » exige un film reel — non
// versionne. La PORTE, elle, doit etre tenue TOUS LES JOURS. Ce test la verrouille sans film : il
// n observe pas des lignes ecrites mais si le RESOLVEUR DE REPERTOIRE est INVOQUE — la capability
// gate strictement en amont de tout acces au film. Le chemin complet (ecriture de lignes depuis un
// film reel) est couvert quand KILLSOURCE_HITS_FIXTURE_DIR pointe un repertoire de chunks.
package killcollector

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games"
)

// capsAvecPrecision : la CapabilityMap qui autorise le numerateur de precision par arme.
func capsAvecPrecision() games.CapabilityMap {
	return games.CapabilityMap{
		games.CapFilmKillSource: games.CapSupported,
		games.CapWeaponAccuracy: games.CapSupported,
	}
}

func compterLignes(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// TestPorteDePrecisionSansFixture — la capability gate STRICTEMENT en amont du film.
//
// Sans capability, le resolveur de repertoire n est jamais consulte (aucun film n est meme
// cherche) ; avec, il l est. Les deux assertions ensemble interdisent aussi bien une porte fermee
// en permanence (le resolveur jamais appele, donc rien jamais ecrit) qu une porte inexistante (le
// resolveur appele meme capability absente, donc un titre sans precision se verrait decoder).
func TestPorteDePrecisionSansFixture(t *testing.T) {
	ctx := context.Background()

	invoque := func(t *testing.T, caps games.CapabilityMap) bool {
		t.Helper()
		db := openSharedTestDB(t)
		col := NewKillSourceCollector(&fakeFilmClient{}, fakeRoster{}, sharedWriter(db), caps, 0)
		appele := false
		col.ConfigureFilmAccuracy(func(string) string { appele = true; return "" }, "")
		ids, err := fakeRoster{}.IdentitiesForMatch(ctx, "m-hits")
		if err != nil {
			t.Fatalf("identites: %v", err)
		}
		col.collectHits(ctx, "m-hits", nil, ids)
		// Quel que soit le chemin, un repertoire vide n ecrit rien : la porte ne s appuie pas sur
		// l ecriture mais sur l acces au film.
		if n := compterLignes(t, db, "weapon_accuracy"); n != 0 {
			t.Errorf("%d ligne(s) weapon_accuracy ecrites avec un repertoire vide", n)
		}
		return appele
	}

	// (1) SANS la capability : le resolveur de film n est JAMAIS consulte — degradation gracieuse,
	//     la passe des morts reste acquise.
	if invoque(t, games.CapabilityMap{games.CapFilmKillSource: games.CapSupported}) {
		t.Error("repertoire de film consulte SANS `match.weapon.accuracy` — la porte ne garde rien")
	}
	// (2) AVEC la capability : le resolveur EST consulte. Sans cette moitie, une porte fermee dans
	//     les deux cas passerait pour une porte qui fonctionne.
	if !invoque(t, capsAvecPrecision()) {
		t.Error("repertoire de film NON consulte AVEC `match.weapon.accuracy` — la porte est fermee " +
			"dans les deux cas, la precision par arme ne serait jamais produite")
	}
}

// TestPrecisionNonConfigureeNeCassePas — filmDir nil (chemin live sans cache) : best-effort, rien.
func TestPrecisionNonConfigureeNeCassePas(t *testing.T) {
	ctx := context.Background()
	db := openSharedTestDB(t)
	col := NewKillSourceCollector(&fakeFilmClient{}, fakeRoster{}, sharedWriter(db), capsAvecPrecision(), 0)
	ids, _ := fakeRoster{}.IdentitiesForMatch(ctx, "m-hits")
	col.collectHits(ctx, "m-hits", nil, ids) // ConfigureFilmAccuracy jamais appele
	if n := compterLignes(t, db, "match_weapon_hit_distance"); n != 0 {
		t.Errorf("%d ligne(s) distance ecrites sans numerateur configure", n)
	}
}

// TestPrecisionSurFilmReel — le chemin complet, quand un repertoire de chunks est fourni.
// Non versionne (films 107 Mo) : saute sans KILLSOURCE_HITS_FIXTURE_DIR, exactement comme les
// instruments filmdec. La roster fixture doit rattacher au moins un xuid a un indice pour ecrire.
func TestPrecisionSurFilmReel(t *testing.T) {
	dir := os.Getenv("KILLSOURCE_HITS_FIXTURE_DIR")
	if dir == "" {
		t.Skip("KILLSOURCE_HITS_FIXTURE_DIR absent : chemin complet (film reel) saute")
	}
	ctx := context.Background()
	db := openSharedTestDB(t)
	col := NewKillSourceCollector(&fakeFilmClient{}, fakeRoster{}, sharedWriter(db), capsAvecPrecision(), 0)
	col.ConfigureFilmAccuracy(func(string) string { return dir }, os.Getenv("KILLSOURCE_HITS_MAP_BOUNDS"))
	ids, _ := fakeRoster{}.IdentitiesForMatch(ctx, "m-hits")
	col.collectHits(ctx, "m-hits", nil, ids)
	// On ne peut pas garantir de lignes (le roster fixture peut ne rattacher aucun indice), mais la
	// passe doit tourner sans paniquer ; un compte est logue pour l inspection manuelle.
	t.Logf("film %s : weapon_accuracy=%d distance=%d", dir,
		compterLignes(t, db, "weapon_accuracy"), compterLignes(t, db, "match_weapon_hit_distance"))
}
