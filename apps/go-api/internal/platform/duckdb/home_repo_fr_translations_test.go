//go:build integration

// home_repo_fr_translations_test.go — tests d'intégration de
// enrichHomeMatchTranslations et EnrichCanonicalAssetTranslations.
//
// Couvre les mêmes scénarios de corruption que match_history_fr_translations_test.go
// (asset_translations[fr-FR] == EN raw) appliqués au chemin Home/canonical,
// plus la résolution des playlists depuis asset_translations.
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
)

// seedHomeFRFixtures peuple mode_name_tr + asset_translations dans une meta DB
// de test. Réutilise newHomeRepoTestMetaDB (même package).
func seedHomeFRFixtures(t *testing.T, meta *DB) {
	t.Helper()
	ctx := context.Background()

	for _, row := range [][3]string{
		{"CTF", "fr", "Capture du drapeau"},
		{"Strongholds", "fr", "Bases"},
		{"Slayer", "fr", "Assassin"},
	} {
		if _, err := meta.Exec(ctx,
			`INSERT INTO mode_name_tr (lang, mode_en, name) VALUES (?, ?, ?)`,
			row[1], row[0], row[2],
		); err != nil {
			t.Fatalf("seed mode_name_tr: %v", err)
		}
	}

	// Pair corrompu : toutes les langues retournent l'EN raw.
	for _, lang := range []string{"en-US", "fr-FR", "fr"} {
		if _, err := meta.Exec(ctx,
			`INSERT INTO asset_translations VALUES (?, 'pair', ?, ?, '', now())`,
			"corrupted-pair-id", lang, "Arena:CTF on Shiro",
		); err != nil {
			t.Fatalf("seed corrupted pair (%s): %v", lang, err)
		}
	}

	// Playlist Quick Play : fr-FR correct, en-US correct.
	for _, row := range [][3]string{
		{"qp-playlist-id", "en-US", "Quick Play"},
		{"qp-playlist-id", "fr-FR", "Partie rapide"},
	} {
		if _, err := meta.Exec(ctx,
			`INSERT INTO asset_translations VALUES (?, 'playlist', ?, ?, '', now())`,
			row[0], row[1], row[2],
		); err != nil {
			t.Fatalf("seed playlist: %v", err)
		}
	}

	// Map Aquarius : même nom FR (maps souvent identiques).
	for _, row := range [][3]string{
		{"aquarius-map-id", "en-US", "Aquarius"},
		{"aquarius-map-id", "fr-FR", "Aquarius"},
	} {
		if _, err := meta.Exec(ctx,
			`INSERT INTO asset_translations VALUES (?, 'map', ?, ?, '', now())`,
			row[0], row[1], row[2],
		); err != nil {
			t.Fatalf("seed map: %v", err)
		}
	}
}

// ── enrichHomeMatchTranslations ───────────────────────────────────────────────

// TestEnrichHomeMatchTranslations_CorruptedPairNameFR : pair_name stocké en DB
// mais pair_name_fr == pair_name (EN raw) — le chemin home doit produire "Capture
// du drapeau" via mode_name_tr sans dépendre du cache serveur.
func TestEnrichHomeMatchTranslations_CorruptedPairNameFR(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	seedHomeFRFixtures(t, meta)

	repo := NewHomeRepo(&PlayerDB{Metadata: meta})
	matches := []legacymatch.HomeMatchRow{
		{
			MatchID:    "m1",
			PairID:     "corrupted-pair-id",
			PairName:   "Arena:CTF on Shiro",
			PairNameFR: "Arena:CTF on Shiro", // FR == EN raw → corruption détectée
		},
	}
	repo.enrichHomeMatchTranslations(context.Background(), matches)

	if got := matches[0].PairNameFR; got != "Capture du drapeau" {
		t.Errorf("PairNameFR = %q, want %q", got, "Capture du drapeau")
	}
}

// TestEnrichHomeMatchTranslations_NullPairNameResolvedViaPairID : pair_name NULL
// en DB (pair_name_fr NULL aussi) — home doit résoudre via pair_id →
// asset_translations → re-normaliser → mode_name_tr.
func TestEnrichHomeMatchTranslations_NullPairNameResolvedViaPairID(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	seedHomeFRFixtures(t, meta)

	repo := NewHomeRepo(&PlayerDB{Metadata: meta})
	matches := []legacymatch.HomeMatchRow{
		{
			MatchID:    "m2",
			PairID:     "corrupted-pair-id",
			PairName:   "", // NULL en DB
			PairNameFR: "", // NULL en DB
		},
	}
	repo.enrichHomeMatchTranslations(context.Background(), matches)

	if got := matches[0].PairNameFR; got != "Capture du drapeau" {
		t.Errorf("PairNameFR = %q, want %q", got, "Capture du drapeau")
	}
}

// TestEnrichHomeMatchTranslations_PlaylistFR : playlist dont FR == EN (non traduit
// côté API) — home doit résoudre "Quick Play" → "Partie rapide" via
// asset_translations[fr-FR].
func TestEnrichHomeMatchTranslations_PlaylistFR(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	seedHomeFRFixtures(t, meta)

	repo := NewHomeRepo(&PlayerDB{Metadata: meta})
	matches := []legacymatch.HomeMatchRow{
		{
			MatchID:        "m3",
			PlaylistID:     "qp-playlist-id",
			PlaylistName:   "Quick Play",
			PlaylistNameFR: "Quick Play", // EN raw — doit être remplacé
		},
	}
	repo.enrichHomeMatchTranslations(context.Background(), matches)

	if got := matches[0].PlaylistNameFR; got != "Partie rapide" {
		t.Errorf("PlaylistNameFR = %q, want %q", got, "Partie rapide")
	}
}

// TestEnrichHomeMatchTranslations_AlreadyTranslatedPreserved : un label FR déjà
// correct ne doit pas être écrasé, même si asset_translations a une valeur
// différente.
func TestEnrichHomeMatchTranslations_AlreadyTranslatedPreserved(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	seedHomeFRFixtures(t, meta)

	repo := NewHomeRepo(&PlayerDB{Metadata: meta})
	matches := []legacymatch.HomeMatchRow{
		{
			MatchID:    "m4",
			PairID:     "corrupted-pair-id",
			PairName:   "Arena:Slayer on Live Fire",
			PairNameFR: "Assassin", // déjà traduit correctement — ne pas toucher
		},
	}
	repo.enrichHomeMatchTranslations(context.Background(), matches)

	if got := matches[0].PairNameFR; got != "Assassin" {
		t.Errorf("PairNameFR = %q, want %q (label FR correct ne doit pas être écrasé)", got, "Assassin")
	}
}

// ── EnrichCanonicalAssetTranslations ─────────────────────────────────────────

// TestEnrichCanonicalAssetTranslations_PairModeFRFromModeNameTr : PairMode avec
// DefaultLabel = EN raw corrompu et ID = pair corrompu — Labels["fr"] doit être
// résolu via re-normalisation + mode_name_tr.
func TestEnrichCanonicalAssetTranslations_PairModeFRFromModeNameTr(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	seedHomeFRFixtures(t, meta)

	repo := NewHomeRepo(&PlayerDB{Metadata: meta})
	rows := []canonical.PlayerMatchRow{
		{
			Summary: canonical.MatchSummary{
				MatchID: "m5",
				PairMode: &canonical.AssetReference{
					ID:           "corrupted-pair-id",
					DefaultLabel: "Arena:CTF on Shiro",
					Labels:       map[string]string{},
				},
			},
		},
	}
	if err := repo.EnrichCanonicalAssetTranslations(context.Background(), rows); err != nil {
		t.Fatalf("EnrichCanonicalAssetTranslations: %v", err)
	}

	got := rows[0].Summary.PairMode.Labels["fr"]
	if got != "Capture du drapeau" {
		t.Errorf("Labels[fr] = %q, want %q", got, "Capture du drapeau")
	}
}

// TestEnrichCanonicalAssetTranslations_PlaylistFRFromAssetTranslations : Playlist
// avec DefaultLabel EN et fr-FR dans asset_translations — Labels["fr"] doit être
// "Partie rapide".
func TestEnrichCanonicalAssetTranslations_PlaylistFRFromAssetTranslations(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	seedHomeFRFixtures(t, meta)

	repo := NewHomeRepo(&PlayerDB{Metadata: meta})
	rows := []canonical.PlayerMatchRow{
		{
			Summary: canonical.MatchSummary{
				MatchID: "m6",
				Playlist: &canonical.AssetReference{
					ID:           "qp-playlist-id",
					DefaultLabel: "Quick Play",
					Labels:       map[string]string{},
				},
			},
		},
	}
	if err := repo.EnrichCanonicalAssetTranslations(context.Background(), rows); err != nil {
		t.Fatalf("EnrichCanonicalAssetTranslations: %v", err)
	}

	got := rows[0].Summary.Playlist.Labels["fr"]
	if got != "Partie rapide" {
		t.Errorf("Labels[fr] = %q, want %q", got, "Partie rapide")
	}
}

// TestEnrichCanonicalAssetTranslations_NilMetadataNoOp : sans metadata DB, la
// fonction doit retourner nil et laisser les rows intactes.
func TestEnrichCanonicalAssetTranslations_NilMetadataNoOp(t *testing.T) {
	repo := NewHomeRepo(&PlayerDB{}) // Metadata = nil
	rows := []canonical.PlayerMatchRow{
		{
			Summary: canonical.MatchSummary{
				MatchID: "m7",
				PairMode: &canonical.AssetReference{
					ID:           "some-pair",
					DefaultLabel: "Arena:CTF on X",
					Labels:       map[string]string{},
				},
			},
		},
	}
	if err := repo.EnrichCanonicalAssetTranslations(context.Background(), rows); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := rows[0].Summary.PairMode.Labels["fr"]; got != "" {
		t.Errorf("Labels[fr] = %q, want empty (nil metadata doit être no-op)", got)
	}
}
