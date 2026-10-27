package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/ctxkeys"
)

func seedCommendationDefsMeta(t *testing.T, db *DB, withTiers bool) {
	t.Helper()
	ctx := context.Background()
	tiers := ", tier_targets VARCHAR"
	if !withTiers {
		tiers = ""
	}
	stmts := []string{
		`CREATE TABLE commendation_definitions (commendation_id VARCHAR PRIMARY KEY, icon_url VARCHAR` + tiers + `)`,
		`CREATE TABLE asset_translations (asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR, description VARCHAR)`,
		`INSERT INTO commendation_definitions (commendation_id, icon_url) VALUES ('c1','https://cdn/1.png'), ('c2','https://cdn/2.png'), ('c3', NULL)`,
		`INSERT INTO asset_translations VALUES
			('c1','commendation','fr-FR','Tueur de Spartans',NULL), ('c1','commendation','en-US','Spartan Slayer',NULL),
			('c2','commendation','en-US','Headshot Honcho',NULL),
			('c1','team_color','en-US','Pas une commendation',NULL)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(ctx, s); err != nil {
			t.Fatalf("seed: %v\nSQL: %s", err, s)
		}
	}
	if withTiers {
		if _, err := db.Exec(ctx, `UPDATE commendation_definitions SET tier_targets = '1,41,201' WHERE commendation_id = 'c1'`); err != nil {
			t.Fatal(err)
		}
	}
}

// Le nom suit la langue de la requête (fr, en), se replie sur la langue de repli quand la langue n'a
// pas de ligne (c2 : anglais seul, en français aussi), reste vide sans aucune ligne (c3, l'appelant
// affiche l'identifiant) ; une langue inconnue prend la langue par défaut.
func TestLoadCommendationDefsFromMetadata_NomParLangue(t *testing.T) {
	db := openAssetMemDB(t)
	seedCommendationDefsMeta(t, db, true)
	ids := []string{"c1", "c2", "c3", "inconnu"}

	cases := []struct {
		name string
		ctx  context.Context
		want map[string]string
	}{
		{"fr (contexte par défaut)", context.Background(), map[string]string{"c1": "Tueur de Spartans", "c2": "Headshot Honcho", "c3": ""}},
		{"fr", ctxkeys.WithLocale(context.Background(), "fr"), map[string]string{"c1": "Tueur de Spartans", "c2": "Headshot Honcho", "c3": ""}},
		{"en", ctxkeys.WithLocale(context.Background(), "en"), map[string]string{"c1": "Spartan Slayer", "c2": "Headshot Honcho", "c3": ""}},
		{"langue inconnue", ctxkeys.WithLocale(context.Background(), "zz"), map[string]string{"c1": "Tueur de Spartans", "c2": "Headshot Honcho", "c3": ""}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := loadCommendationDefsFromMetadata(c.ctx, db, ids)
			if len(got) != 3 {
				t.Fatalf("définitions = %d, want 3 (l'identifiant inconnu est absent) : %+v", len(got), got)
			}
			for id, name := range c.want {
				if got[id].name != name {
					t.Errorf("%s : nom = %q, want %q", id, got[id].name, name)
				}
			}
			if got["c1"].iconURL != "https://cdn/1.png" || got["c1"].tierTargets != "1,41,201" || got["c3"].iconURL != "" {
				t.Errorf("colonnes non textuelles : %+v", got)
			}
		})
	}
}

// Une base provisionnée avant tier_targets sert quand même noms et icônes.
func TestLoadCommendationDefsFromMetadata_SansColonnePaliers(t *testing.T) {
	db := openAssetMemDB(t)
	seedCommendationDefsMeta(t, db, false)
	got := loadCommendationDefsFromMetadata(context.Background(), db, []string{"c1"})
	if got["c1"].name != "Tueur de Spartans" || got["c1"].tierTargets != "" || got["c1"].iconURL == "" {
		t.Errorf("c1 = %+v", got["c1"])
	}
}

func TestLoadCommendationDefsFromMetadata_Degradations(t *testing.T) {
	if got := loadCommendationDefsFromMetadata(context.Background(), nil, []string{"c1"}); len(got) != 0 {
		t.Errorf("metadata nil : %v", got)
	}
	db := openAssetMemDB(t)
	if got := loadCommendationDefsFromMetadata(context.Background(), db, []string{"c1"}); len(got) != 0 {
		t.Errorf("table absente : %v", got)
	}
}
