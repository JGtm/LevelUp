//go:build integration

// Package ops — seed_citation_reconcile_integration_test.go : la réconciliation des citations au
// démarrage (ReconcileCitationMappings) sur une base de référentiels migrée. Elle insère une
// citation ajoutée après l'installation, réécrit seulement ce qui diffère, et n'écrit rien sur une
// base déjà alignée (le compte `Inserted == 0 && Updated == 0` en fait foi).
package ops

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/migration"
)

// reconcileTestDB rend une base metadata migrée (asset_translations et citation_mappings à leur
// schéma de production), sur un seul handle comme l'écrivain du serveur.
func reconcileTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "metadata.duckdb"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	db.SetMaxOpenConns(1)
	if err := migration.RunForDB(db, migration.TargetMetadata); err != nil {
		t.Fatalf("migrations metadata: %v", err)
	}
	return db
}

func reconcileOK(t *testing.T, db *sql.DB) SeedResult {
	t.Helper()
	res, err := ReconcileCitationMappings(context.Background(), db)
	if err != nil {
		t.Fatalf("ReconcileCitationMappings: %v", err)
	}
	return res
}

func reconcileCount(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

func TestReconcileCitationMappings_InsereUneCitationAjouteeApresInstallation(t *testing.T) {
	db := reconcileTestDB(t)
	total := len(defaultCitationMappings())
	if first := reconcileOK(t, db); first.Inserted != total || first.Updated != 0 {
		t.Fatalf("installation : Inserted=%d Updated=%d, want %d et 0", first.Inserted, first.Updated, total)
	}

	// Une base installée avant la citation : ni sa règle ni ses textes.
	const norm = citationNormSkullGrabs
	if _, err := db.Exec(`DELETE FROM citation_mappings WHERE citation_name_norm = ?`, norm); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM asset_translations WHERE asset_type = ? AND asset_id = ?`, migration.AssetTypeCitation, norm); err != nil {
		t.Fatal(err)
	}

	res := reconcileOK(t, db)
	if res.Inserted != 1 || res.Updated != 0 || res.Skipped != total-1 {
		t.Errorf("Inserted=%d Updated=%d Skipped=%d, want 1, 0, %d", res.Inserted, res.Updated, res.Skipped, total-1)
	}
	rows, err := loadCitationRows(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	var want CitationMapping
	for _, m := range defaultCitationMappings() {
		if m.Norm == norm {
			want = m
		}
	}
	if got, ok := rows[norm]; !ok || got != citationRowOf(want) {
		t.Errorf("ligne insérée = %+v (présente %v), want %+v", got, ok, citationRowOf(want))
	}
	if n := reconcileCount(t, db, `SELECT COUNT(*) FROM asset_translations WHERE asset_type = ? AND asset_id = ?`,
		migration.AssetTypeCitation, norm); n != len(i18n.Locales()) {
		t.Errorf("textes de %s = %d lignes, want une par langue (%d)", norm, n, len(i18n.Locales()))
	}
}

func TestReconcileCitationMappings_DeuxiemePassageNEcritRien(t *testing.T) {
	db := reconcileTestDB(t)
	total := len(defaultCitationMappings())
	first := reconcileOK(t, db)
	if first.Inserted != total {
		t.Fatalf("premier passage : Inserted = %d, want %d", first.Inserted, total)
	}
	second := reconcileOK(t, db)
	if second.Inserted != 0 || second.Updated != 0 {
		t.Errorf("deuxième passage : Inserted=%d Updated=%d, want 0 et 0", second.Inserted, second.Updated)
	}
	if second.Skipped != total {
		t.Errorf("deuxième passage : Skipped = %d, want %d", second.Skipped, total)
	}
}

// Une valeur vide (chaîne vide, 0) n'est pas un NULL : l'écart est réécrit une fois, puis la base
// est stable. Garde la comparaison distincte entre NULL et valeur par défaut.
func TestReconcileCitationMappings_DistingueNullEtVide(t *testing.T) {
	db := reconcileTestDB(t)
	reconcileOK(t, db)

	// `flag_captures` n'a pas de sous-catégorie : la colonne est NULL.
	var sub sql.NullString
	if err := db.QueryRow(`SELECT subcategory FROM citation_mappings WHERE citation_name_norm = ?`, citationNormFlagCaptures).Scan(&sub); err != nil {
		t.Fatal(err)
	}
	if sub.Valid {
		t.Fatalf("pré-requis : sous-catégorie de %s = %q, want NULL", citationNormFlagCaptures, sub.String)
	}
	if _, err := db.Exec(`UPDATE citation_mappings SET subcategory = '' WHERE citation_name_norm = ?`, citationNormFlagCaptures); err != nil {
		t.Fatal(err)
	}
	if res := reconcileOK(t, db); res.Updated != 1 || res.Inserted != 0 {
		t.Errorf("'' à la place de NULL : Updated=%d Inserted=%d, want 1 et 0", res.Updated, res.Inserted)
	}
	if err := db.QueryRow(`SELECT subcategory FROM citation_mappings WHERE citation_name_norm = ?`, citationNormFlagCaptures).Scan(&sub); err != nil || sub.Valid {
		t.Errorf("sous-catégorie après réconciliation = %+v (%v), want NULL", sub, err)
	}
	if res := reconcileOK(t, db); res.Updated != 0 || res.Inserted != 0 {
		t.Errorf("passage suivant : Updated=%d Inserted=%d, want 0 et 0", res.Updated, res.Inserted)
	}
}

// Une colonne qui a bougé et un texte qui a bougé sont réécrits, et seulement eux.
func TestReconcileCitationMappings_ReecritSeulementCeQuiDiffere(t *testing.T) {
	db := reconcileTestDB(t)
	reconcileOK(t, db)

	if _, err := db.Exec(`UPDATE citation_mappings SET tier_targets = '1,2,3' WHERE citation_name_norm = ?`, citationNormCharge); err != nil {
		t.Fatal(err)
	}
	tag := i18n.HaloTags(i18n.Default)[0]
	if _, err := db.Exec(`UPDATE asset_translations SET name = 'périmé' WHERE asset_type = ? AND asset_id = ? AND lang = ?`,
		migration.AssetTypeCitation, citationNormGotYou, tag); err != nil {
		t.Fatal(err)
	}

	res := reconcileOK(t, db)
	if res.Updated != 2 || res.Inserted != 0 {
		t.Fatalf("Updated=%d Inserted=%d, want 2 et 0", res.Updated, res.Inserted)
	}
	names, _, err := citationTextsFor(citationNormGotYou)
	if err != nil {
		t.Fatal(err)
	}
	var name string
	if err := db.QueryRow(`SELECT name FROM asset_translations WHERE asset_type = ? AND asset_id = ? AND lang = ?`,
		migration.AssetTypeCitation, citationNormGotYou, tag).Scan(&name); err != nil || name != names[i18n.Default] {
		t.Errorf("nom restauré = %q (%v), want %q", name, err, names[i18n.Default])
	}
	var tiers string
	if err := db.QueryRow(`SELECT tier_targets FROM citation_mappings WHERE citation_name_norm = ?`, citationNormCharge).Scan(&tiers); err != nil || tiers != tierTargets10_20_30_50_100 {
		t.Errorf("paliers restaurés = %q (%v), want %q", tiers, err, tierTargets10_20_30_50_100)
	}
	if again := reconcileOK(t, db); again.Updated != 0 || again.Inserted != 0 {
		t.Errorf("passage suivant : Updated=%d Inserted=%d, want 0 et 0", again.Updated, again.Inserted)
	}
}

func TestReconcileCitationMappings_SansAssetTranslationsRefuse(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := ReconcileCitationMappings(context.Background(), db); err == nil {
		t.Fatal("attendu : refus explicite quand asset_translations est absente")
	}
}
