// Package duckdb — home_repo_locales_test.go : les enrichissements de l'Accueil et la lecture de l'Escouade
// travaillent sur TOUTES les langues connues (config/locales.toml), sans en coder aucune (ADR 0040 D-15).
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/legacymatch"
)

// seedLocalizedMap pose le nom d'une carte dans chaque langue connue (`<langue> <nom>`) : le test ne
// suppose donc aucune liste de langues.
func seedLocalizedMap(t *testing.T, meta *DB, id string) {
	t.Helper()
	for _, l := range i18n.Locales() {
		if _, err := meta.Exec(context.Background(),
			`INSERT INTO asset_translations VALUES (?, 'map', ?, ?, '', now())`,
			id, i18n.HaloTags(l)[0], string(l)+" Carte "+id,
		); err != nil {
			t.Fatalf("seed carte (%s): %v", l, err)
		}
	}
}

// TestEnrichCanonicalAssetTranslations_UnLibelleParLangue : chaque langue connue reçoit le nom de la
// carte dans sa langue, la langue de repli garde le nom canonique d'asset_translations.
func TestEnrichCanonicalAssetTranslations_UnLibelleParLangue(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	seedLocalizedMap(t, meta, "m1")
	repo := NewHomeRepo(&PlayerDB{Metadata: meta})
	rows := []canonical.PlayerMatchRow{{Summary: canonical.MatchSummary{
		Map: &canonical.AssetReference{ID: "m1", DefaultLabel: "registre", Labels: map[string]string{}},
	}}}
	if err := repo.EnrichCanonicalAssetTranslations(context.Background(), rows); err != nil {
		t.Fatal(err)
	}
	for _, l := range i18n.Locales() {
		if got, want := rows[0].Summary.Map.Labels[string(l)], string(l)+" Carte m1"; got != want {
			t.Errorf("Labels[%s] = %q, want %q", l, got, want)
		}
	}
}

// TestEnrichHomeMatchTranslations_UnNomParLangue : même chose pour les tuiles legacy ; un nom de la
// langue de repli déjà porté par le registre n'est pas écrasé.
func TestEnrichHomeMatchTranslations_UnNomParLangue(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	seedLocalizedMap(t, meta, "m1")
	repo := NewHomeRepo(&PlayerDB{Metadata: meta})
	matches := []legacymatch.HomeMatchRow{{
		MatchID: "x", MapID: "m1", MapName: i18n.Text{i18n.Fallback: "registre"},
	}}
	repo.enrichHomeMatchTranslations(context.Background(), matches)
	for _, l := range i18n.Locales() {
		want := string(l) + " Carte m1"
		if l == i18n.Fallback {
			want = "registre"
		}
		if got := matches[0].MapName[l]; got != want {
			t.Errorf("MapName[%s] = %q, want %q", l, got, want)
		}
	}
}

// TestSquadRepo_LoadAssetTexts : le nom de l'actif est rendu dans chaque langue qui en porte un ;
// base ou table absente = (nil, nil).
func TestSquadRepo_LoadAssetTexts(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	seedLocalizedMap(t, meta, "m1")
	got, err := NewSquadRepo(&PlayerDB{Metadata: meta}).LoadAssetTexts(context.Background(), "map", []string{"m1", "inconnue"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("1 actif attendu, vu %d : %v", len(got), got)
	}
	for _, l := range i18n.Locales() {
		if want := string(l) + " Carte m1"; got["m1"][l] != want {
			t.Errorf("m1[%s] = %q, want %q", l, got["m1"][l], want)
		}
	}
	if none, err := NewSquadRepo(&PlayerDB{}).LoadAssetTexts(context.Background(), "map", []string{"m1"}); err != nil || none != nil {
		t.Errorf("sans metadata : (%v, %v), want (nil, nil)", none, err)
	}
}

// TestModeKeySet : un libellé non normalisable est ignoré, le nom de chaque langue d'un actif est
// inscrit.
func TestModeKeySet(t *testing.T) {
	keys := modeKeySet{}
	keys.addLabel("Arena:Slayer on Aquarius")
	keys.addLabel("")
	keys.addTexts(i18n.Text{i18n.Default: "Arena:CTF on Shiro"})
	got := map[string]bool{}
	for _, k := range keys.list() {
		got[k] = true
	}
	if !got["Slayer"] || !got["CTF"] || len(got) != 2 {
		t.Errorf("clés = %v, want {Slayer, CTF}", got)
	}
}
