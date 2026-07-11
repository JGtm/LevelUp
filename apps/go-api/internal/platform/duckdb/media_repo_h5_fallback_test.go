//go:build integration

// Package duckdb — media_repo_h5_fallback_test.go : F1 (résidus H5, DEC-7).
// Le registre H5 porte des colonnes noms TOUTES NULL (map_name, pair_name,
// playlist_name) → la galerie média affichait « Carte inconnue » et ses filtres
// restaient vides alors que l'association média→match existait. Le fallback
// resolveMediaRegistryNameFallbacks résout map/mode/playlist via la cascade
// asset_translations (map par map_id, mode par game_variant_id — même fallback
// data-driven que GetMatchMeta A2 —, playlist par playlist_id).
//
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -run TestMediaH5Fallback -v
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	titlepkg "levelup/go-api/internal/domain/title"
)

// newTestPlayerDBForH5MediaFallback : 1 match H5-like (noms NULL, ids présents),
// 1 média associé, asset_translations peuplées (fr-FR).
func newTestPlayerDBForH5MediaFallback(t *testing.T) *PlayerDB {
	t.Helper()
	player := openMemDB(t)
	shared := openMemDB(t)
	social := openMemDB(t)
	meta := openMemDB(t)

	seedSharedDBSchema(t, shared)
	seedSharedSocialSchema(t, social)
	seedMetaDBSchema(t, meta)

	ctx := context.Background()
	if _, err := shared.Exec(ctx, `DELETE FROM shared.match_registry`); err != nil {
		t.Fatalf("wipe shared: %v", err)
	}
	for _, q := range []string{
		`DELETE FROM media_match_associations_history`,
		`DELETE FROM media_files`,
	} {
		if _, err := social.Exec(ctx, q); err != nil {
			t.Fatalf("wipe social: %v\nSQL: %s", err, q)
		}
	}

	// Match H5-like : noms NULL, seuls les IDs sont présents (cas 100 % du registre H5).
	if _, err := shared.Exec(ctx,
		`INSERT INTO shared.match_registry
			(match_id, start_time, start_time_utc, map_id, game_variant_id, playlist_id, is_ranked)
			VALUES ('mh5', '2019-12-12 21:20:00', '2019-12-12 21:20:00+00',
			        'map-plaza', 'gv-slayer', 'pl-quick', FALSE)`); err != nil {
		t.Fatalf("insert match: %v", err)
	}
	// Traductions (la cascade fr-FR → … existante).
	for _, q := range []string{
		`INSERT INTO asset_translations (asset_id, asset_type, lang, name) VALUES ('map-plaza', 'map', 'fr-FR', 'Plaza')`,
		`INSERT INTO asset_translations (asset_id, asset_type, lang, name) VALUES ('gv-slayer', 'game_variant', 'fr-FR', 'Assassin')`,
		`INSERT INTO asset_translations (asset_id, asset_type, lang, name) VALUES ('pl-quick', 'playlist', 'fr-FR', 'Partie rapide')`,
	} {
		if _, err := meta.Exec(ctx, q); err != nil {
			t.Fatalf("seed translations: %v\nSQL: %s", err, q)
		}
	}
	// 1 média associé au match.
	if _, err := social.Exec(ctx,
		`INSERT INTO media_files
			(id, player_slug, file_path, file_name, file_stem, file_ext, kind, capture_end_utc)
			VALUES ('med-h5', ?, '/Halo_5_Guardians-2019-12-12_22h27.mp4',
			        'Halo_5_Guardians-2019-12-12_22h27.mp4', 'Halo_5_Guardians-2019-12-12_22h27',
			        '.mp4', 'video', '2019-12-12 21:27:00+00')`, mediaTestPlayerSlug); err != nil {
		t.Fatalf("insert media: %v", err)
	}
	if _, err := social.Exec(ctx,
		`INSERT INTO media_match_associations_history (media_file_id, match_id) VALUES ('med-h5', 'mh5')`); err != nil {
		t.Fatalf("insert assoc: %v", err)
	}

	return &PlayerDB{
		Player:       player,
		Shared:       shared,
		SharedSocial: social,
		Metadata:     meta,
		XUID:         mediaTestPlayerXUID,
		Gamertag:     mediaTestPlayerSlug,
		TitleSlug:    titlepkg.DefaultSlug,
	}
}

func TestMediaH5Fallback_LabelsResolvedFromAssetTranslations(t *testing.T) {
	pdb := newTestPlayerDBForH5MediaFallback(t)
	repo := NewMediaRepo(pdb)

	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.MapName == nil || *row.MapName != "Plaza" {
		t.Errorf("MapName = %v, want 'Plaza' (fallback asset_translations map)", row.MapName)
	}
	if row.ModeName == nil || *row.ModeName != "Assassin" {
		t.Errorf("ModeName = %v, want 'Assassin' (fallback game_variant)", row.ModeName)
	}
}

func TestMediaH5Fallback_FilterOptionsPopulated(t *testing.T) {
	pdb := newTestPlayerDBForH5MediaFallback(t)
	repo := NewMediaRepo(pdb)

	opts, err := repo.LoadMediaFilterOptions(context.Background(), domain.MediaFilters{})
	if err != nil {
		t.Fatalf("LoadMediaFilterOptions: %v", err)
	}
	if len(opts.Maps) != 1 {
		t.Fatalf("Maps options = %d, want 1 (fallback → filtre map peuplé)", len(opts.Maps))
	}
	if opts.Maps[0].Label != "Plaza" {
		t.Errorf("Maps[0].Label = %q, want 'Plaza'", opts.Maps[0].Label)
	}
	if len(opts.Playlists) != 1 || opts.Playlists[0].Label != "Partie rapide" {
		t.Errorf("Playlists = %+v, want 1 option 'Partie rapide'", opts.Playlists)
	}
}

// TestMediaH5Fallback_ModeFilterByLabel : titre sans pair (H5) — le filtre mode
// s'applique par égalité de libellé (le label game_variant résolu qui peuple les
// options), pas par catégorie/préfixes pair (taxonomie Infinite inapplicable).
func TestMediaH5Fallback_ModeFilterByLabel(t *testing.T) {
	pdb := newTestPlayerDBForH5MediaFallback(t)
	repo := NewMediaRepo(pdb)

	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{ModeFilter: "Assassin"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles (filtre Assassin): %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("filtre mode 'Assassin' = %d rows, want 1", len(rows))
	}
	rows, err = repo.LoadMediaFiles(context.Background(), domain.MediaFilters{ModeFilter: "Capture du drapeau"}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles (filtre CTF): %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("filtre mode 'Capture du drapeau' = %d rows, want 0", len(rows))
	}
}

// TestMediaH5Fallback_NoRegressionWhenNamesPresent : quand les colonnes noms du
// registre sont remplies (Infinite), le fallback ne s'active pas (aucune requête
// metadata — comportement byte-identique).
func TestMediaH5Fallback_NoRegressionWhenNamesPresent(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)

	rows, err := repo.LoadMediaFiles(context.Background(), domain.MediaFilters{}, 100, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	byMap := map[string]bool{}
	for _, r := range rows {
		if r.MapName != nil {
			byMap[*r.MapName] = true
		}
	}
	for _, want := range []string{"Aquarius", "Bazaar", "Catalyst"} {
		if !byMap[want] {
			t.Errorf("map %q absente des rows (noms registre présents → inchangés)", want)
		}
	}
}
