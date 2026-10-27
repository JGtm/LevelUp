//go:build integration

// Tests d'intégration de MilestoneCatalogRepo (metadata.duckdb).

package duckdb

import (
	"context"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/progression/milestones"

	"levelup/go-api/internal/i18n"
)

func newMilestoneCatalogRepoForTest(t *testing.T) *MilestoneCatalogRepo {
	t.Helper()
	db := setupPrestigeDB(t, migration.TargetMetadata)
	return NewMilestoneCatalogRepo(db)
}

// bilingue range un même texte dans les deux langues de config/locales.toml.
func bilingue(en, fr string) i18n.Text {
	return i18n.Text{i18n.LocaleEN: en, i18n.LocaleFR: fr}
}

func TestMilestoneCatalogRepo_UpsertAndList(t *testing.T) {
	repo := newMilestoneCatalogRepoForTest(t)
	ctx := context.Background()

	entries := []milestones.CatalogEntry{
		{ID: "h.matches.100", TitleSlug: "halo_infinite", Metric: "matches_played", Threshold: 100, Title: bilingue("Centurion", "Centurion"), Icon: "icon_100"},
		{ID: "h.matches.500", TitleSlug: "halo_infinite", Metric: "matches_played", Threshold: 500, Title: bilingue("Veteran", "Vétéran")},
		{ID: "h.wins.50", TitleSlug: "halo_infinite", Metric: "wins", Threshold: 50, Title: bilingue("Winner", "Vainqueur")},
		// Autre titre (devrait être exclu)
		{ID: "other.x.1", TitleSlug: "other_title", Metric: "x", Threshold: 1, Title: bilingue("X", "X")},
	}
	for _, e := range entries {
		if err := repo.Upsert(ctx, e); err != nil {
			t.Fatalf("Upsert %s: %v", e.ID, err)
		}
	}

	got, err := repo.ListByTitle(ctx, "halo_infinite")
	if err != nil {
		t.Fatalf("ListByTitle: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("ListByTitle len = %d, want 3", len(got))
	}
	// Tri attendu : (metric, threshold) — matches_played avant wins, 100 avant 500.
	if got[0].ID != "h.matches.100" || got[1].ID != "h.matches.500" || got[2].ID != "h.wins.50" {
		t.Errorf("ordre incorrect : %s %s %s", got[0].ID, got[1].ID, got[2].ID)
	}
	// Icon round-trip
	if got[0].Icon != "icon_100" {
		t.Errorf("Icon = %q, want icon_100", got[0].Icon)
	}
}

func TestMilestoneCatalogRepo_UpsertOverwrites(t *testing.T) {
	repo := newMilestoneCatalogRepoForTest(t)
	ctx := context.Background()

	original := milestones.CatalogEntry{
		ID: "h.matches.100", TitleSlug: "halo_infinite", Metric: "matches_played", Threshold: 100,
		Title: bilingue("Centurion", "Centurion"),
	}
	if err := repo.Upsert(ctx, original); err != nil {
		t.Fatalf("Upsert v1: %v", err)
	}

	// Update libellé FR.
	updated := original
	updated.Title = bilingue("Centurion", "Centurion (révisé)")
	if err := repo.Upsert(ctx, updated); err != nil {
		t.Fatalf("Upsert v2: %v", err)
	}

	got, err := repo.ListByTitle(ctx, "halo_infinite")
	if err != nil || len(got) != 1 {
		t.Fatalf("ListByTitle: got=%v err=%v", got, err)
	}
	if got[0].Title[i18n.LocaleFR] != "Centurion (révisé)" {
		t.Errorf("TitleFR = %q, want updated", got[0].Title[i18n.LocaleFR])
	}
	if got[0].Title[i18n.LocaleEN] != "Centurion" {
		t.Errorf("TitleEN = %q, want unchanged", got[0].Title[i18n.LocaleEN])
	}
	var n int
	if err := repo.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM asset_translations WHERE asset_type = ? AND asset_id = ?`,
		AssetTypeMilestone, "h.matches.100").Scan(&n); err != nil {
		t.Fatalf("count asset_translations: %v", err)
	}
	if n != 2 {
		t.Errorf("asset_translations milestone = %d lignes, want 2 (une par langue, pas de doublon à la réécriture)", n)
	}
}

// Le titre et la condition lisible se lisent par langue : fr et en leur ligne, une langue sans
// traduction (jalon sans fr, langue hors liste) se replie sur l'anglais, rien n'est inventé.
func TestMilestoneCatalogRepo_TextesParLangue(t *testing.T) {
	repo := newMilestoneCatalogRepoForTest(t)
	ctx := context.Background()

	both := milestones.CatalogEntry{
		ID: "h.acc.30d", TitleSlug: "halo_infinite", Metric: "accuracy", Threshold: 30,
		Title:         bilingue("Consistent Aim", "Visée régulière"),
		Condition:     "accuracy >= 0.50 on 30 distinct days",
		ConditionText: bilingue("Hit 50% accuracy on 30 days", "Atteindre 50 % de précision sur 30 jours"),
	}
	enOnly := milestones.CatalogEntry{
		ID: "h.wins.50", TitleSlug: "halo_infinite", Metric: "wins", Threshold: 50,
		Title: i18n.Text{i18n.LocaleEN: "Winner"},
	}
	for _, e := range []milestones.CatalogEntry{both, enOnly} {
		if err := repo.Upsert(ctx, e); err != nil {
			t.Fatalf("Upsert %s: %v", e.ID, err)
		}
	}
	got, err := repo.ListByTitle(ctx, "halo_infinite")
	if err != nil || len(got) != 2 {
		t.Fatalf("ListByTitle: got=%v err=%v", got, err)
	}
	byID := map[string]milestones.CatalogEntry{got[0].ID: got[0], got[1].ID: got[1]}

	cases := []struct {
		name         string
		id           string
		loc          i18n.Locale
		wantTitle    string
		wantConditon string
	}{
		{"fr", "h.acc.30d", i18n.LocaleFR, "Visée régulière", "Atteindre 50 % de précision sur 30 jours"},
		{"en", "h.acc.30d", i18n.LocaleEN, "Consistent Aim", "Hit 50% accuracy on 30 days"},
		{"langue hors liste : repli en", "h.acc.30d", i18n.Locale("es"), "Consistent Aim", "Hit 50% accuracy on 30 days"},
		{"fr sans traduction : repli en", "h.wins.50", i18n.LocaleFR, "Winner", ""},
	}
	for _, c := range cases {
		e := byID[c.id]
		if g := e.Title.Pick(c.loc); g != c.wantTitle {
			t.Errorf("%s : titre = %q, attendu %q", c.name, g, c.wantTitle)
		}
		if g := e.ConditionText.PickOptional(c.loc); g != c.wantConditon {
			t.Errorf("%s : condition lisible = %q, attendu %q", c.name, g, c.wantConditon)
		}
	}
	// La formule technique n'est pas localisée : elle reste dans la table de jalons.
	if byID["h.acc.30d"].Condition != "accuracy >= 0.50 on 30 distinct days" {
		t.Errorf("Condition technique = %q", byID["h.acc.30d"].Condition)
	}
	if byID["h.wins.50"].Title.Has(i18n.LocaleFR) {
		t.Errorf("aucune traduction ne doit être inventée pour fr")
	}
}
