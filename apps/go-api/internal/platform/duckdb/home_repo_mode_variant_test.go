//go:build integration

// home_repo_mode_variant_test.go — enrichissement des libellés d'un mode porté par le
// type de jeu (titre dont le manifeste déclare mode_source = "game_variant") : la référence
// est traduite par les dictionnaires des types de jeu (asset_type='game_variant'), jamais
// par ceux des parties.
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/canonical"
)

func seedGameVariantNames(t *testing.T, meta *DB) {
	t.Helper()
	for _, row := range [][3]string{
		{"gv-slayer", "en-US", "Slayer"},
		{"gv-slayer", "fr-FR", "Assassin"},
		{"gv-ctf", "en-US", "Capture the Flag"},
		{"gv-ctf", "fr-FR", "Capture du drapeau"},
	} {
		if _, err := meta.Exec(context.Background(),
			`INSERT INTO asset_translations VALUES (?, 'game_variant', ?, ?, '', now())`,
			row[0], row[1], row[2]); err != nil {
			t.Fatalf("seed game_variant: %v", err)
		}
	}
}

func h5ModeRow(variantID string) canonical.PlayerMatchRow {
	variant := AssetReference("game_variant", variantID, nil)
	return canonical.PlayerMatchRow{Summary: canonical.MatchSummary{
		MatchID:     "m-" + variantID,
		GameVariant: variant,
		PairMode:    ModeAssetReference(title.ModeSourceGameVariant, nil, variant),
	}}
}

// Le mode d'un match Halo 5 reçoit les noms EN et FR du type de jeu ; ce sont les mêmes que ceux
// de GameVariant (aucune divergence entre le mode affiché et le type de jeu exposé).
func TestEnrichCanonicalAssetTranslations_ModeFromGameVariant(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	seedHomeFRFixtures(t, meta)
	seedGameVariantNames(t, meta)
	repo := NewHomeRepo(&PlayerDB{Metadata: meta})
	rows := []canonical.PlayerMatchRow{h5ModeRow("gv-slayer"), h5ModeRow("gv-ctf")}

	if err := repo.EnrichCanonicalAssetTranslations(context.Background(), rows); err != nil {
		t.Fatalf("EnrichCanonicalAssetTranslations: %v", err)
	}
	want := map[string][2]string{"gv-slayer": {"Slayer", "Assassin"}, "gv-ctf": {"Capture the Flag", "Capture du drapeau"}}
	for _, r := range rows {
		w := want[r.Summary.PairMode.ID]
		if got := r.Summary.PairMode.Labels["en"]; got != w[0] {
			t.Errorf("%s : mode EN = %q, want %q", r.Summary.MatchID, got, w[0])
		}
		if got := r.Summary.PairMode.Labels["fr"]; got != w[1] {
			t.Errorf("%s : mode FR = %q, want %q (la cascade mode_name_tr ne doit pas s'appliquer)", r.Summary.MatchID, got, w[1])
		}
		if r.Summary.GameVariant.Labels["fr"] != w[1] || r.Summary.GameVariant.Labels["en"] != w[0] {
			t.Errorf("%s : GameVariant = %+v, doit porter les mêmes noms", r.Summary.MatchID, r.Summary.GameVariant.Labels)
		}
	}

	// Deuxième passe (Synthèse et Accueil ré-enrichissent des lignes déjà enrichies) : stable.
	if err := repo.EnrichCanonicalAssetTranslations(context.Background(), rows); err != nil {
		t.Fatalf("2e passe: %v", err)
	}
	if got := rows[0].Summary.PairMode.Labels["fr"]; got != "Assassin" {
		t.Errorf("2e passe : mode FR = %q, want Assassin", got)
	}
}

// Un type de jeu que le dictionnaire ne nomme pas reste sans libellé : aucun identifiant brut n'est
// exposé comme nom de mode.
func TestEnrichCanonicalAssetTranslations_ModeFromGameVariantUnnamed(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	seedGameVariantNames(t, meta)
	repo := NewHomeRepo(&PlayerDB{Metadata: meta})
	rows := []canonical.PlayerMatchRow{h5ModeRow("gv-inconnu")}

	if err := repo.EnrichCanonicalAssetTranslations(context.Background(), rows); err != nil {
		t.Fatalf("EnrichCanonicalAssetTranslations: %v", err)
	}
	m := rows[0].Summary.PairMode
	if m.DefaultLabel != "" || m.Labels["fr"] != "" || m.Labels["en"] != "" {
		t.Errorf("mode = %+v, want sans libellé (jamais l'identifiant)", m)
	}
}

// Un mode qui est une partie (Infinite) n'est PAS traduit par les types de jeu, même si un type de jeu
// de même identifiant est nommé.
func TestEnrichCanonicalAssetTranslations_PairModeIgnoresGameVariantDictionary(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	seedGameVariantNames(t, meta)
	repo := NewHomeRepo(&PlayerDB{Metadata: meta})
	rows := []canonical.PlayerMatchRow{{Summary: canonical.MatchSummary{
		MatchID:  "m-pair",
		PairMode: AssetReference("pair_mode", "gv-slayer", nil),
	}}}

	if err := repo.EnrichCanonicalAssetTranslations(context.Background(), rows); err != nil {
		t.Fatalf("EnrichCanonicalAssetTranslations: %v", err)
	}
	if got := rows[0].Summary.PairMode.Labels["en"]; got != "" {
		t.Errorf("mode EN = %q, want vide : une partie ne se traduit pas par le dictionnaire des types de jeu", got)
	}
}

// Un nom de type de jeu porteur d'un préfixe technique (« Conteneur:Mode ») devient un nom de mode : la
// normalisation est faite une fois ici, donc commune à toutes les pages et aux valeurs d'option des filtres.
func TestEnrichCanonicalAssetTranslations_ModeFromGameVariantNormalise(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	for _, row := range [][3]string{
		{"gv-pref", "en-US", "HaloMultiplayer:Slayer"},
		{"gv-pref", "fr-FR", "HaloMultiplayer:Assassin"},
	} {
		if _, err := meta.Exec(context.Background(),
			`INSERT INTO asset_translations VALUES (?, 'game_variant', ?, ?, '', now())`,
			row[0], row[1], row[2]); err != nil {
			t.Fatalf("seed game_variant: %v", err)
		}
	}
	repo := NewHomeRepo(&PlayerDB{Metadata: meta})
	rows := []canonical.PlayerMatchRow{h5ModeRow("gv-pref")}

	for pass := 1; pass <= 2; pass++ { // idempotent : les services ré-enrichissent
		if err := repo.EnrichCanonicalAssetTranslations(context.Background(), rows); err != nil {
			t.Fatalf("passe %d: %v", pass, err)
		}
		m := rows[0].Summary.PairMode
		if m.Labels["en"] != "Slayer" || m.Labels["fr"] != "Assassin" {
			t.Errorf("passe %d : mode = %+v, want Slayer / Assassin", pass, m.Labels)
		}
	}
	// Le type de jeu exposé à part garde son nom d'origine.
	if got := rows[0].Summary.GameVariant.Labels["fr"]; got != "HaloMultiplayer:Assassin" {
		t.Errorf("GameVariant FR = %q, doit rester le nom d'origine", got)
	}
}
