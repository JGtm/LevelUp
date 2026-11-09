package ops

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/i18n"
)

func TestUpsertDefinitionTexts(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, s := range []string{
		`CREATE TABLE asset_translations (asset_id VARCHAR NOT NULL, asset_type VARCHAR NOT NULL, lang VARCHAR NOT NULL, name VARCHAR NOT NULL, description VARCHAR, PRIMARY KEY (asset_id, asset_type, lang))`,
		`CREATE TABLE medal_translations (medal_name_id BIGINT NOT NULL, lang VARCHAR NOT NULL, name VARCHAR NOT NULL, description VARCHAR, PRIMARY KEY (medal_name_id, lang))`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	names := i18n.Text{i18n.LocaleFR: "Tireur", i18n.LocaleEN: "Shooter"}
	descs := i18n.Text{i18n.LocaleEN: "Hit things"}
	for i := 0; i < 2; i++ { // 2e passe : mise à jour, aucune ligne en double
		if err := UpsertAssetTexts(ctx, db, "commendation", "c1", names, descs); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM asset_translations`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("lignes = %d (err %v), want 2", n, err)
	}
	var d sql.NullString
	if err := db.QueryRow(`SELECT description FROM asset_translations WHERE lang = 'fr-FR'`).Scan(&d); err != nil || d.Valid {
		t.Errorf("description fr = %v (err %v), want NULL", d, err)
	}
	// Une langue absente n'efface rien.
	if err := UpsertAssetTexts(ctx, db, "commendation", "c1", i18n.Text{i18n.LocaleFR: "Tireur d'élite"}, nil); err != nil {
		t.Fatal(err)
	}
	var en string
	if err := db.QueryRow(`SELECT name FROM asset_translations WHERE lang = 'en-US'`).Scan(&en); err != nil || en != "Shooter" {
		t.Errorf("en = %q (err %v), want Shooter", en, err)
	}
	if err := UpsertMedalTexts(ctx, db, 7, names, descs); err != nil {
		t.Fatal(err)
	}
	var mn string
	if err := db.QueryRow(`SELECT name FROM medal_translations WHERE medal_name_id = 7 AND lang = 'fr-FR'`).Scan(&mn); err != nil || mn != "Tireur" {
		t.Errorf("médaille fr = %q (err %v)", mn, err)
	}
	if err := UpsertAssetTexts(ctx, db, "", "x", names, nil); err == nil {
		t.Error("un type vide doit être refusé")
	}
}
