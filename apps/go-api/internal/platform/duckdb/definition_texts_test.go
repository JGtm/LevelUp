package duckdb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/i18n"
)

type sqlQuerier struct{ db *sql.DB }

func (s sqlQuerier) Query(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, q, args...)
}

func TestLoadDefinitionTexts(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, s := range []string{
		`CREATE TABLE asset_translations (asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR, description VARCHAR)`,
		`CREATE TABLE medal_translations (medal_name_id BIGINT, lang VARCHAR, name VARCHAR, description VARCHAR)`,
		`INSERT INTO asset_translations VALUES
			('c1','commendation','fr-FR','Tireur','Touche'),('c1','commendation','en-US','Shooter',NULL),
			('c1','commendation','zz-YY','Inconnu','x'),('c1','commendation','fr','Tireur (court)','y'),
			('c2','commendation','en-US','Solo',''),('t1','team_color','fr-FR','Rouge',NULL)`,
		`INSERT INTO medal_translations VALUES (7,'fr-FR','Tueur',NULL),(7,'en-US','Killer','One kill'),(7,'ja-JP','x','y')`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	q := sqlQuerier{db}
	ctx := context.Background()
	got, err := LoadAssetTexts(ctx, q, AssetTypeCommendation, []string{"c1", "c2", "absent"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("2 commendations attendues, got %d", len(got))
	}
	if n := got["c1"].Name.Pick(i18n.LocaleFR); n != "Tireur" {
		t.Errorf("fr = %q (l'étiquette principale doit l'emporter sur « fr »)", n)
	}
	if d := got["c1"].Description.Pick(i18n.LocaleFR); d != "Touche" {
		t.Errorf("description fr = %q", d)
	}
	if n := got["c1"].Name.Pick(i18n.LocaleEN); n != "Shooter" {
		t.Errorf("en = %q", n)
	}
	if got["c1"].Description.Has(i18n.LocaleEN) {
		t.Error("une description NULL ne doit pas être inscrite")
	}
	if n := got["c2"].Name.Pick(i18n.LocaleFR); n != "Solo" {
		t.Errorf("repli fr -> en = %q, want Solo", n)
	}
	all, err := LoadAllAssetTexts(ctx, q, AssetTypeTeamColor)
	if err != nil || all["t1"].Name.Pick(i18n.LocaleFR) != "Rouge" {
		t.Errorf("LoadAllAssetTexts = %v, %v", all, err)
	}
	medals, err := LoadMedalTexts(ctx, q, []int64{7, 8})
	if err != nil {
		t.Fatal(err)
	}
	if len(medals) != 1 || medals[7].Name.Pick(i18n.LocaleEN) != "Killer" || medals[7].Description.Pick(i18n.LocaleEN) != "One kill" {
		t.Errorf("medals = %v", medals)
	}
}
