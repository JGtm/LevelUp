package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/i18n"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestLoadTeamColorNames_ReadsColor prouve que LoadTeamColorNames lit bien la colonne
// `color` de team_colors (couleur d'identité hex exposée jusqu'au scoreboard front) en
// plus des noms par langue de asset_translations. Utilise openAssetMemDB (helper in-memory partagé, non taggué
// integration).
func TestLoadTeamColorNames_ReadsColor(t *testing.T) {
	db := openAssetMemDB(t)
	ctx := context.Background()

	for _, q := range []string{
		`CREATE TABLE team_colors (team_id INTEGER, color VARCHAR)`,
		`CREATE TABLE asset_translations (asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR, description VARCHAR)`,
		`INSERT INTO team_colors VALUES (0, '#b00000'), (1, '#178dd8'), (2, NULL), (3, '#000000')`,
		`INSERT INTO asset_translations VALUES
			('0','team_color','fr-FR','Rouge',NULL), ('0','team_color','en-US','Red',NULL),
			('1','team_color','fr-FR','Bleu',NULL), ('1','team_color','en-US','Blue',NULL),
			('2','team_color','en-US','Yellow',NULL),
			('9','team_color','en-US','Sans couleur',NULL)`,
	} {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("seed: %v\nSQL: %s", err, q)
		}
	}

	m := LoadTeamColorNames(ctx, db)
	// L'équipe 3 n'a aucun nom : omise ; la ligne "9" n'a pas de couleur : omise aussi.
	if len(m) != 3 {
		t.Fatalf("LoadTeamColorNames: attendu 3 entrées, obtenu %d", len(m))
	}
	if _, ok := m[3]; ok {
		t.Error("une équipe sans nom ne doit pas être servie")
	}
	if got := m[0].Color; got != "#b00000" {
		t.Errorf("team 0 color = %q, want #b00000", got)
	}
	if got := m[1].Color; got != "#178dd8" {
		t.Errorf("team 1 color = %q, want #178dd8", got)
	}
	// COALESCE(color, '') → couleur absente dégrade en "" (jamais fatal).
	if got := m[2].Color; got != "" {
		t.Errorf("team 2 color (NULL) = %q, want \"\"", got)
	}
	// Noms par langue : la langue demandée, sinon la langue de repli (équipe 2 : anglais seul).
	for _, c := range []struct {
		id   int
		loc  i18n.Locale
		want string
	}{
		{0, i18n.LocaleFR, "Rouge"}, {0, i18n.LocaleEN, "Red"},
		{1, i18n.LocaleFR, "Bleu"}, {1, i18n.LocaleEN, "Blue"},
		{2, i18n.LocaleFR, "Yellow"}, {2, i18n.LocaleEN, "Yellow"},
	} {
		if got := m[c.id].Name.Pick(c.loc); got != c.want {
			t.Errorf("team %d [%s] = %q, want %q", c.id, c.loc, got, c.want)
		}
	}
}
