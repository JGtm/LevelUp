//go:build integration

package main

// cmd_backfill_killsource_match_integration_test.go — `--match` dans la SELECTION de production
// (`filmsACollecter`), sur un shared migre par les vraies migrations (plan Emprise vies, V5.2a).
// La base de fixture et les helpers viennent de
// cmd_backfill_killsource_selection_placement_integration_test.go.

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/sync/killcollector"
)

// registreAvecFilms inscrit `ids` au registre ET leur pose un manifeste de cache (un chunk), de
// sorte que `compterChunks` les voie : sans film en cache, la selection les ecarterait.
func registreAvecFilms(t *testing.T, ids ...string) (*sql.DB, string) {
	t.Helper()
	db, cache := baseDeSelection(t), t.TempDir()
	for _, id := range ids {
		executer(t, db, `INSERT INTO match_registry (match_id, map_name) VALUES (?, 'streets')`, id)
		court := strings.SplitN(id, "-", 2)[0]
		chemin := filmcache.ManifestPath(cache, court)
		if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
			t.Fatalf("mkdir manifestes: %v", err)
		}
		if err := os.WriteFile(chemin, []byte(`{"chunks":[{}]}`), 0o644); err != nil {
			t.Fatalf("manifeste %s: %v", id, err)
		}
	}
	return db, cache
}

func TestSelectionMatch_Filtre(t *testing.T) {
	db, cache := registreAvecFilms(t, registreDeTest...)
	ctx := context.Background()

	// Sans --match : tout le registre (garde contre un filtre toujours actif).
	tous, bilan, err := filmsACollecter(ctx, db, cache, killsourceOptions{})
	if err != nil || len(tous) != 4 || bilan.TotalRegistre != 4 {
		t.Fatalf("sans --match : %d films, registre %d, %v — attendu 4", len(tous), bilan.TotalRegistre, err)
	}

	// Un prefixe univoque de 8 caracteres et un identifiant complet.
	o := killsourceOptions{match: "aaaaaaaa," + registreDeTest[3]}
	got, bilan, err := filmsACollecter(ctx, db, cache, o)
	if err != nil {
		t.Fatalf("filmsACollecter: %v", err)
	}
	var ids []string
	for _, c := range got {
		ids = append(ids, c.matchID)
	}
	sort.Strings(ids)
	if len(ids) != 2 || ids[0] != registreDeTest[0] || ids[1] != registreDeTest[3] {
		t.Errorf("selection = %v, attendu les matchs 0 et 3", ids)
	}
	if bilan.TotalRegistre != 2 {
		t.Errorf("TotalRegistre = %d, attendu 2 (le denominateur est le perimetre nomme)", bilan.TotalRegistre)
	}

	// --limit s applique apres le filtre.
	o.limit = 1
	if got, _, _ = filmsACollecter(ctx, db, cache, o); len(got) != 1 {
		t.Errorf("--match + --limit 1 : %d films, attendu 1", len(got))
	}
}

func TestSelectionMatch_PrefixeAmbiguOuInconnuRefuse(t *testing.T) {
	db, cache := registreAvecFilms(t, registreDeTest...)
	for nom, c := range map[string]struct{ match, fragment string }{
		"ambigu":  {"abcd1234", "AMBIGU"},
		"inconnu": {"deadbeef", "aucun match du registre"},
	} {
		t.Run(nom, func(t *testing.T) {
			got, _, err := filmsACollecter(context.Background(), db, cache, killsourceOptions{match: c.match})
			if err == nil || !strings.Contains(err.Error(), c.fragment) {
				t.Fatalf("erreur = %v, attendu %q", err, c.fragment)
			}
			if len(got) != 0 {
				t.Errorf("un refus ne doit rien selectionner, %d films", len(got))
			}
		})
	}
}

// `matchsAJour` reste applique sous `--match`, sauf `--force`.
func TestSelectionMatch_AJourEtForce(t *testing.T) {
	db, cache := registreAvecFilms(t, registreDeTest...)
	fait := registreDeTest[0]
	matchAJourDeSesVies(t, db, fait)
	placementA(t, db, fait, killcollector.PlacementRev)
	ctx := context.Background()

	o := killsourceOptions{match: "aaaaaaaa,bbbbbbbb"}
	got, bilan, err := filmsACollecter(ctx, db, cache, o)
	if err != nil {
		t.Fatalf("filmsACollecter: %v", err)
	}
	if len(got) != 1 || got[0].matchID != registreDeTest[1] || bilan.DejaAJour != 1 {
		t.Errorf("sans --force : %v, deja a jour %d — attendu le seul match 1 et 1 saute", got, bilan.DejaAJour)
	}

	o.force = true
	got, bilan, err = filmsACollecter(ctx, db, cache, o)
	if err != nil {
		t.Fatalf("filmsACollecter --force: %v", err)
	}
	if len(got) != 2 || bilan.DejaAJour != 0 {
		t.Errorf("--force : %d films, deja a jour %d — attendu les 2 matchs nommes, 0 saute", len(got), bilan.DejaAJour)
	}
}
