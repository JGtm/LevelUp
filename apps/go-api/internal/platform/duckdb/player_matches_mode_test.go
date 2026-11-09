//go:build integration

// player_matches_mode_test.go — chaîne complète de lecture (PlayerMatchesAdapter : requête
// partagée, projection, enrichissement des libellés) du mode d'un match, par titre.
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

func loadModeRows(t *testing.T, slug string, modeSource title.ModeSource) []string {
	t.Helper()
	ctx := context.Background()
	pdb := newTestPlayerDB(t)
	pdb.TitleSlug = slug
	pdb.ModeSource = modeSource
	resetPlayerMatchesTables(t, pdb)
	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS mode_name_tr (lang VARCHAR, mode_en VARCHAR, name VARCHAR)`,
		`CREATE TABLE IF NOT EXISTS map_images_registry (title_id VARCHAR, map_id VARCHAR, local_path VARCHAR,
			fetched_at TIMESTAMPTZ DEFAULT now())`,
	} {
		if _, err := pdb.Metadata.Exec(ctx, q); err != nil {
			t.Fatalf("seed metadata: %v", err)
		}
	}
	seedGameVariantNames(t, pdb.Metadata)

	// m-h5 : type de jeu nommé par le dictionnaire, aucune partie (registre Halo 5).
	// m-pair : une partie ET un type de jeu (registre Halo Infinite).
	for _, m := range []struct{ id, variant, pair, pairFR string }{
		{"m-h5", "gv-slayer", "", ""},
		{"m-pair", "gv-ctf", "Arena:Slayer", "Arène : Slayer"},
	} {
		execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_registry
			(match_id, start_time, map_id, map_name, game_variant_id, pair_name, pair_name_fr, duration_seconds)
			VALUES (?, ?, 'bazaar', 'Bazaar', ?, NULLIF(?, ''), NULLIF(?, ''), 600)`,
			m.id, "2026-01-01T00:00:00Z", m.variant, m.pair, m.pairFR)
		execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_participants
			(match_id, xuid, gamertag, outcome, team_id) VALUES (?, ?, ?, 2, 0)`,
			m.id, pTestXUID, pTestGamertag)
		if _, err := pdb.Player.Exec(ctx, `INSERT INTO player_match_enrichment (match_id) VALUES (?)`, m.id); err != nil {
			t.Fatalf("insert player_match_enrichment: %v", err)
		}
	}
	rows, err := NewPlayerMatchesAdapter(NewPlayerMatchesRepo(pdb), slug, pTestGamertag).
		LoadPlayerMatches(ctx, slug, pTestGamertag, port.PlayerMatchFilters{})
	if err != nil {
		t.Fatalf("LoadPlayerMatches: %v", err)
	}
	out := map[string]string{}
	for _, r := range rows {
		label := "<nil>"
		if m := r.Summary.PairMode; m != nil {
			label = m.Kind + "|" + m.Labels["en"] + "|" + m.Labels["fr"]
		}
		out[r.Summary.MatchID] = label
	}
	return []string{out["m-h5"], out["m-pair"]}
}

// Halo 5 : le mode d'un match sans partie est le type de jeu, nommé EN/FR par le dictionnaire ;
// une partie présente reste le mode.
func TestPlayerMatchesAdapter_Mode_Halo5(t *testing.T) {
	got := loadModeRows(t, "halo_5", title.ModeSourceGameVariant)
	if want := "game_variant|Slayer|Assassin"; got[0] != want {
		t.Errorf("match sans partie : mode = %q, want %q", got[0], want)
	}
	if want := "pair_mode|Arena:Slayer|Arène : Slayer"; got[1] != want {
		t.Errorf("match avec partie : mode = %q, want %q", got[1], want)
	}
}

// Halo Infinite : sans partie, pas de mode (le type de jeu ne s'y substitue jamais) ; la partie reste le mode.
func TestPlayerMatchesAdapter_Mode_HaloInfinite(t *testing.T) {
	got := loadModeRows(t, "halo_infinite", title.ModeSourcePair)
	if want := "<nil>"; got[0] != want {
		t.Errorf("match sans partie : mode = %q, want %q", got[0], want)
	}
	if want := "pair_mode|Arena:Slayer|Arène : Slayer"; got[1] != want {
		t.Errorf("match avec partie : mode = %q, want %q", got[1], want)
	}
}
