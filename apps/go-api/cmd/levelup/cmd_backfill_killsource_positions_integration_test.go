//go:build integration

package main

// cmd_backfill_killsource_positions_integration_test.go — UNE METADATA ILLISIBLE NE COUPE PAS LA
// RESOLUTION DE CARTE (revue finale, P1-a, 2026-10-02).
//
// Depuis la carte obligatoire (2026-09-27), un collecteur sans resolution de carte met TOUS ses
// films de cote. `positionCaptureDeps` rendait une capture vide des que `metadata.duckdb` ne
// s ouvrait pas : `backfill-killsource` ne decodait alors plus aucun film, alors que les cartes
// nommees en clair dans `match_registry.map_name` se resolvent sans metadonnees.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : rendre `killcollector.DepsCapture{}` quand l ouverture de
// la metadata echoue (le comportement d avant le correctif).

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/sync/killcollector"
	"levelup/go-api/internal/testutil"
)

// racineSansMetadata : une racine de depot qui porte le catalogue de bornes VERSIONNE du titre
// et, a la place de `metadata.duckdb`, un fichier qui n est pas une base DuckDB.
func racineSansMetadata(t *testing.T) string {
	t.Helper()
	vraie, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	racine := t.TempDir()
	src, dst := titlePkg.NewPathResolver(vraie), titlePkg.NewPathResolver(racine)
	octets, err := os.ReadFile(src.MapQuantBoundsPath(titlePkg.DefaultSlug))
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	for chemin, contenu := range map[string][]byte{
		dst.MapQuantBoundsPath(titlePkg.DefaultSlug): octets,
		dst.MetadataDBPath(titlePkg.DefaultSlug):     []byte("pas une base duckdb"),
	} {
		if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
			t.Fatalf("mkdir %s : %v", chemin, err)
		}
		if err := os.WriteFile(chemin, contenu, 0o644); err != nil {
			t.Fatalf("ecriture %s : %v", chemin, err)
		}
	}
	return racine
}

func TestPositionCaptureDeps_MetadataIllisible_LesCartesEnClairSeResolvent(t *testing.T) {
	ctx := context.Background()
	db := baseDeSelection(t)
	executer(t, db, `INSERT INTO match_registry (match_id, map_name) VALUES ('m-clair', 'Bazaar')`)
	executer(t, db, `INSERT INTO match_registry (match_id, map_name) VALUES ('m-forge', 'Forge Inconnue')`)

	cfg := &config.AppConfig{RepoRoot: racineSansMetadata(t)}
	capture, fermer := positionCaptureDeps(ctx, cfg, titlePkg.DefaultSlug, db, nil)
	defer fermer()
	if !capture.Cablee() {
		t.Fatalf("capture non cablee alors que seule la metadata est illisible : la resolution de carte " +
			"ne depend pas des metadonnees (les cartes nommees en clair se resolvent sans elles)")
	}

	col := killcollector.NewKillSourceCollector(nil, nil, nil,
		games.CapabilityMap{games.CapFilmKillSource: games.CapSupported}, 0).AvecCapture(capture)
	retenus, ecartes := col.RetenirLesMatchsAvecCarte(ctx, []string{"m-clair", "m-forge"}, 0)
	if len(retenus) != 1 || retenus[0] != "m-clair" {
		t.Errorf("retenus = %v, attendu [m-clair] : une carte nommee en clair au registre se resout sans metadonnees", retenus)
	}
	if len(ecartes) != 1 || ecartes[0] != "m-forge" {
		t.Errorf("ecartes = %v, attendu [m-forge] (hors catalogue de bornes)", ecartes)
	}
}
