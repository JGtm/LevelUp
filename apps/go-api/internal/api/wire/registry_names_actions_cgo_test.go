//go:build cgo

// Package api — registry_names_actions_cgo_test.go : l'action admin « noms du registre »
// (RunRegistryNamesBackfill) et sa garde multi-titre. Bases DuckDB réelles sur disque,
// SharedProvider réel (B-swap RO↔RW) sur la base partagée du titre par défaut.
package wire

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// ddlRegistreNoms : le registre réduit aux colonnes de noms (schéma du balayage).
const ddlRegistreNoms = `CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY,
	playlist_id VARCHAR, playlist_name VARCHAR, playlist_version_id VARCHAR,
	map_id VARCHAR, map_name VARCHAR, map_version_id VARCHAR,
	pair_id VARCHAR, pair_name VARCHAR, pair_version_id VARCHAR,
	game_variant_id VARCHAR, game_variant_name VARCHAR, game_variant_version_id VARCHAR,
	mode_category VARCHAR)`

const ddlTraductions = `CREATE TABLE asset_translations (asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR,
	name VARCHAR, description VARCHAR, fetched_at TIMESTAMP)`

// titreParDefautAvecProvider : base partagée du titre par défaut (m1 nom NULL, m2 nom =
// identifiant), traduction map-x → Streets, provider réel sur la base partagée.
func titreParDefautAvecProvider(t *testing.T) (*ServiceRegistry, sharedprovider.Provider, string) {
	t.Helper()
	repoRoot := t.TempDir()
	pr := titlePkg.NewPathResolver(repoRoot)
	sharedPath := pr.SharedDBPath(titlePkg.DefaultSlug)
	creerBase(t, sharedPath, ddlRegistreNoms,
		`INSERT INTO match_registry (match_id, map_id, map_name, mode_category)
			VALUES ('m1', 'map-x', NULL, 'other'), ('m2', 'map-x', 'map-x', 'other')`)
	creerBase(t, pr.MetadataDBPath(titlePkg.DefaultSlug), ddlTraductions,
		`INSERT INTO asset_translations VALUES ('map-x', 'map', 'en-US', 'Streets', '', now())`)
	prov, err := sharedprovider.New(sharedPath)
	if err != nil {
		t.Fatalf("sharedprovider.New: %v", err)
	}
	t.Cleanup(func() { _ = prov.Close() })
	return &ServiceRegistry{cfg: &config.AppConfig{RepoRoot: repoRoot, SharedProvider: prov}}, prov, repoRoot
}

// nomDeCarte lit map_name d'un match par le provider ("<NULL>" si NULL).
func nomDeCarte(t *testing.T, prov sharedprovider.Provider, matchID string) string {
	t.Helper()
	db, release, err := prov.Get(context.Background())
	if err != nil {
		t.Fatalf("prov.Get: %v", err)
	}
	defer release()
	var nom sql.NullString
	if err := db.QueryRow(`SELECT map_name FROM match_registry WHERE match_id = ?`, matchID).Scan(&nom); err != nil {
		t.Fatalf("lecture %s: %v", matchID, err)
	}
	if !nom.Valid {
		return "<NULL>"
	}
	return nom.String
}

// TestRunRegistryNamesBackfill_SimulationNEcritRien : la simulation compte ce qui serait
// réécrit (deux matchs) et n'écrit rien.
func TestRunRegistryNamesBackfill_SimulationNEcritRien(t *testing.T) {
	reg, prov, _ := titreParDefautAvecProvider(t)

	res, err := reg.RunRegistryNamesBackfill(context.Background(), titlePkg.DefaultSlug, true)
	if err != nil {
		t.Fatalf("simulation: %v", err)
	}
	if !res.DryRun || res.MapsScanned != 2 || res.MapsFixed != 2 || res.TotalFixed != 2 {
		t.Errorf("simulation = %+v, want DryRun, 2 candidats, 2 réparables", res)
	}
	if got := nomDeCarte(t, prov, "m1"); got != "<NULL>" {
		t.Errorf("m1 map_name = %q après simulation, want NULL (aucune écriture)", got)
	}
	if got := nomDeCarte(t, prov, "m2"); got != "map-x" {
		t.Errorf("m2 map_name = %q après simulation, want map-x (aucune écriture)", got)
	}
}

// TestRunRegistryNamesBackfill_ReelEcrit : l'exécution réelle réécrit les deux matchs.
func TestRunRegistryNamesBackfill_ReelEcrit(t *testing.T) {
	reg, prov, _ := titreParDefautAvecProvider(t)

	res, err := reg.RunRegistryNamesBackfill(context.Background(), titlePkg.DefaultSlug, false)
	if err != nil {
		t.Fatalf("exécution: %v", err)
	}
	if res.DryRun || res.MapsFixed != 2 {
		t.Errorf("exécution = %+v, want 2 réécrits", res)
	}
	for _, id := range []string{"m1", "m2"} {
		if got := nomDeCarte(t, prov, id); got != "Streets" {
			t.Errorf("%s map_name = %q, want Streets", id, got)
		}
	}
}

// TestRunRegistryNamesBackfill_RefuseLaBaseDUnAutreTitre : le provider tient la base partagée du
// titre par défaut ; une convergence demandée pour un autre titre (qui a sa propre base et ses
// propres traductions) est refusée — sans la garde, ses noms seraient écrits dans la base du
// titre par défaut, la seule que le provider sait ouvrir en écriture.
func TestRunRegistryNamesBackfill_RefuseLaBaseDUnAutreTitre(t *testing.T) {
	reg, prov, repoRoot := titreParDefautAvecProvider(t)
	const autreTitre = "halo_5"
	pr := titlePkg.NewPathResolver(repoRoot)
	autrePartagee := pr.SharedDBPath(autreTitre)
	creerBase(t, autrePartagee, ddlRegistreNoms,
		`INSERT INTO match_registry (match_id, map_id, map_name) VALUES ('h5', 'map-x', NULL)`)
	creerBase(t, pr.MetadataDBPath(autreTitre), ddlTraductions,
		`INSERT INTO asset_translations VALUES ('map-x', 'map', 'en-US', 'Nom d''un autre titre', '', now())`)

	if _, err := reg.RunRegistryNamesBackfill(context.Background(), autreTitre, false); err == nil {
		t.Fatal("convergence d'un autre titre acceptée, want refus (base partagée non tenue par le provider)")
	}
	if got := nomDeCarte(t, prov, "m1"); got != "<NULL>" {
		t.Errorf("titre par défaut : m1 map_name = %q, want NULL (noms d'un autre titre écrits)", got)
	}
	db, release, err := duckdb.OpenReadForQuery(autrePartagee)
	if err != nil {
		t.Fatalf("lecture base de l'autre titre: %v", err)
	}
	defer release()
	var nom sql.NullString
	if err := db.QueryRow(`SELECT map_name FROM match_registry WHERE match_id = 'h5'`).Scan(&nom); err != nil {
		t.Fatalf("lecture h5: %v", err)
	}
	if nom.Valid {
		t.Errorf("autre titre : h5 map_name = %q, want NULL (aucune écriture)", nom.String)
	}
}
