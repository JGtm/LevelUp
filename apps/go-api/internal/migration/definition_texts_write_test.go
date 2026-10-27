//go:build integration

// Tests des écrivains de textes de définitions (definition_texts_write.go) : une ligne par langue,
// mise à jour sur place sans doublon, rien d'inventé ni d'effacé pour une langue sans texte, jalon
// écrit sans colonnes appariées.
package migration

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/progression/milestones"
)

func openDefinitionTextsWriteDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	for _, s := range []string{
		`CREATE TABLE asset_translations (asset_id VARCHAR NOT NULL, asset_type VARCHAR NOT NULL, lang VARCHAR NOT NULL,
			name VARCHAR NOT NULL, description VARCHAR, PRIMARY KEY (asset_id, asset_type, lang))`,
		`CREATE TABLE milestone_catalog (id VARCHAR PRIMARY KEY, title_slug VARCHAR NOT NULL, metric VARCHAR NOT NULL,
			threshold DOUBLE NOT NULL, icon VARCHAR, condition VARCHAR, updated_at TIMESTAMP)`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("schema: %v\n%s", err, s)
		}
	}
	return db
}

func assetRow(t *testing.T, db *sql.DB, assetType, id string, l i18n.Locale) (name string, desc sql.NullString, found bool) {
	t.Helper()
	err := db.QueryRow(`SELECT name, description FROM asset_translations WHERE asset_type = ? AND asset_id = ? AND lang = ?`,
		assetType, id, i18n.HaloTags(l)[0]).Scan(&name, &desc)
	if errors.Is(err, sql.ErrNoRows) {
		return "", desc, false
	}
	if err != nil {
		t.Fatalf("asset_translations %s/%s (%s): %v", assetType, id, l, err)
	}
	return name, desc, true
}

func TestUpsertAssetTexts_UneLigneParLangueSansDoublon(t *testing.T) {
	db := openDefinitionTextsWriteDB(t)
	ctx := context.Background()

	names := i18n.Text{i18n.LocaleEN: "First Blood", i18n.LocaleFR: "Premier sang"}
	descs := i18n.Text{i18n.LocaleEN: "Get a kill"}
	if err := UpsertAssetTexts(ctx, db, "achievement", "a1", names, descs); err != nil {
		t.Fatalf("UpsertAssetTexts: %v", err)
	}
	if n, d, ok := assetRow(t, db, "achievement", "a1", i18n.LocaleFR); !ok || n != "Premier sang" || d.Valid {
		t.Errorf("fr = %q (desc valide=%v, trouvé=%v), attendu nom seul", n, d.Valid, ok)
	}
	if n, d, ok := assetRow(t, db, "achievement", "a1", i18n.LocaleEN); !ok || n != "First Blood" || d.String != "Get a kill" {
		t.Errorf("en = %q / %q (trouvé=%v)", n, d.String, ok)
	}

	// Deuxième écriture : le nom fr change, l'en reste ; aucune ligne en double.
	names[i18n.LocaleFR] = "Premier sang (révisé)"
	if err := UpsertAssetTexts(ctx, db, "achievement", "a1", names, descs); err != nil {
		t.Fatalf("UpsertAssetTexts v2: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM asset_translations`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("lignes = %d (err=%v), attendu 2", n, err)
	}
	if name, _, _ := assetRow(t, db, "achievement", "a1", i18n.LocaleFR); name != "Premier sang (révisé)" {
		t.Errorf("nom fr non mis à jour : %q", name)
	}

	// Une langue sans texte n'écrit rien et n'efface rien.
	if err := UpsertAssetTexts(ctx, db, "achievement", "a1", i18n.Text{i18n.LocaleEN: "First Blood"}, nil); err != nil {
		t.Fatalf("UpsertAssetTexts langue absente: %v", err)
	}
	if name, _, ok := assetRow(t, db, "achievement", "a1", i18n.LocaleFR); !ok || name != "Premier sang (révisé)" {
		t.Errorf("la ligne fr ne doit pas être effacée : %q (trouvé=%v)", name, ok)
	}
}

func TestUpsertAssetTexts_RefuseTypeOuIdentifiantVide(t *testing.T) {
	db := openDefinitionTextsWriteDB(t)
	if err := UpsertAssetTexts(context.Background(), db, "", "a1", i18n.Text{i18n.LocaleEN: "x"}, nil); err == nil {
		t.Error("type vide : erreur attendue")
	}
	if err := UpsertAssetTexts(context.Background(), db, "achievement", " ", i18n.Text{i18n.LocaleEN: "x"}, nil); err == nil {
		t.Error("identifiant vide : erreur attendue")
	}
}

func TestUpsertMilestoneCatalogEntry_LigneEtTraductions(t *testing.T) {
	db := openDefinitionTextsWriteDB(t)
	ctx := context.Background()

	e := milestones.CatalogEntry{
		ID: "m1", TitleSlug: "halo_infinite", Metric: "accuracy", Threshold: 30, Icon: "i",
		Condition:     "accuracy >= 0.5 on 30d",
		Title:         i18n.Text{i18n.LocaleEN: "Aim", i18n.LocaleFR: "Visée"},
		ConditionText: i18n.Text{i18n.LocaleEN: "Hit 50%", i18n.LocaleFR: "Atteindre 50 %"},
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := UpsertMilestoneCatalogEntry(ctx, tx, e); err != nil {
		t.Fatalf("UpsertMilestoneCatalogEntry: %v", err)
	}
	e.Threshold = 40
	e.Title[i18n.LocaleFR] = "Visée (révisée)"
	if err := UpsertMilestoneCatalogEntry(ctx, tx, e); err != nil {
		t.Fatalf("UpsertMilestoneCatalogEntry v2: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var threshold float64
	var condition string
	if err := db.QueryRow(`SELECT threshold, condition FROM milestone_catalog WHERE id = 'm1'`).Scan(&threshold, &condition); err != nil {
		t.Fatalf("select: %v", err)
	}
	if threshold != 40 || condition != "accuracy >= 0.5 on 30d" {
		t.Errorf("ligne = %v / %q", threshold, condition)
	}
	if n, d, _ := assetRow(t, db, AssetTypeMilestone, "m1", i18n.LocaleFR); n != "Visée (révisée)" || d.String != "Atteindre 50 %" {
		t.Errorf("fr = %q / %q", n, d.String)
	}
	var rows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM asset_translations WHERE asset_type = ?`, AssetTypeMilestone).Scan(&rows); err != nil || rows != 2 {
		t.Errorf("traductions de jalon = %d (err=%v), attendu 2", rows, err)
	}
}
