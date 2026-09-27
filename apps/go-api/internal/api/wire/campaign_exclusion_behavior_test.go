//go:build integration

// campaign_exclusion_behavior_test.go — les lecteurs post-sync des matchs d'un joueur
// (paliers, jalons, retour de pause, deltas) excluent la Campagne (backlog 2026-09-26,
// lot B4, D-5 ; règle produit du 2026-07-18).
//
// Fixture sur des bases MIGRÉES (setupProgressionEnv) : un match d'arène (01/09, victoire,
// 5 frags) et un match de Campagne PLUS RÉCENT (02/09, défaite, 50 frags). Halo 5 : la
// Campagne disparaît. Halo Infinite (aucune variante masquée) : le résolveur est neutre.

package wire

import (
	"context"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/platform/duckdb"
)

var b4Titres = []struct {
	slug    string
	masquee bool
}{{"halo_5", true}, {"halo_infinite", false}}

func b4Attendu[T any](masquee bool, sans, avec T) T {
	if masquee {
		return sans
	}
	return avec
}

var (
	b4Arene    = time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	b4Campagne = time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
)

// newB4Env : bases migrées, titre du joueur forcé, arène + Campagne semées.
func newB4Env(t *testing.T, titleSlug string) *duckdb.PlayerDB {
	t.Helper()
	pdb := setupProgressionEnv(t).pdb
	pdb.TitleSlug = titleSlug
	ctx := context.Background()
	camp := analysis.CampaignExcludedVariantIDs("halo_5")[0]
	for _, m := range []struct {
		id, variant                 string
		at                          time.Time
		outcome, kills, deaths, dmg int
		acc                         float64
	}{
		{"arena1", "aaaaaaaa-0000-0000-0000-000000000001", b4Arene, 2, 5, 1, 500, 0.6},
		{"camp1", camp, b4Campagne, 3, 50, 0, 5000, 0.9},
	} {
		if _, err := pdb.Shared.Exec(ctx, `INSERT INTO match_registry (match_id, start_time, start_time_utc, game_variant_id)
			VALUES (?, ?, ?, ?)`, m.id, m.at, m.at, m.variant); err != nil {
			t.Fatalf("insert match_registry %s: %v", m.id, err)
		}
		if _, err := pdb.Shared.Exec(ctx, `INSERT INTO match_participants (
				match_id, xuid, gamertag, team_id, outcome, kills, deaths, assists,
				kda, accuracy, personal_score, time_played_seconds, headshot_kills, damage_dealt, damage_taken
			) VALUES (?, ?, ?, 1, ?, ?, ?, 0, 1.0, ?, 1000, 600, 1, ?, 100)`,
			m.id, testXUID, testGT, m.outcome, m.kills, m.deaths, m.acc, m.dmg); err != nil {
			t.Fatalf("insert match_participants %s: %v", m.id, err)
		}
	}
	return pdb
}

// B4.2.6 — loadProgressionSharedMatches (séries et records).
func TestCampaignExclusion_LoadProgressionSharedMatches(t *testing.T) {
	for _, tc := range b4Titres {
		_, ids, err := loadProgressionSharedMatches(context.Background(), newB4Env(t, tc.slug), b4Arene.AddDate(0, -1, 0))
		if err != nil {
			t.Fatalf("%s : %v", tc.slug, err)
		}
		if want := b4Attendu(tc.masquee, "[arena1]", "[arena1 camp1]"); fmt.Sprint(ids) != want {
			t.Errorf("%s : matchs %v, attendu %s", tc.slug, ids, want)
		}
	}
}

// B4.2.6 — loadPlayerStats (jalons) : les trois agrégats sur match_participants.
func TestCampaignExclusion_LoadPlayerStats(t *testing.T) {
	for _, tc := range b4Titres {
		stats, err := loadPlayerStats(context.Background(), newB4Env(t, tc.slug))
		if err != nil {
			t.Fatalf("%s : %v", tc.slug, err)
		}
		want := b4Attendu(tc.masquee, 1.0, 2.0)
		for _, cle := range []string{"matches_played", "accuracy_threshold_days", "combat_precision_matches"} {
			if got := stats.Metrics[cle]; got != want {
				t.Errorf("%s : %s = %v, attendu %v", tc.slug, cle, got, want)
			}
		}
	}
}

// B4.2.6 — loadComebackContext (retour après une pause).
func TestCampaignExclusion_LoadComebackContext(t *testing.T) {
	for _, tc := range b4Titres {
		cb, err := loadComebackContext(context.Background(), newB4Env(t, tc.slug), b4Campagne, time.Hour)
		if err != nil {
			t.Fatalf("%s : %v", tc.slug, err)
		}
		if cb.LastMatchAt == nil || !cb.LastMatchAt.Equal(b4Attendu(tc.masquee, b4Arene, b4Campagne)) {
			t.Errorf("%s : dernier match %v, attendu %v", tc.slug, cb.LastMatchAt, b4Attendu(tc.masquee, b4Arene, b4Campagne))
		}
		if gotPrev := cb.PrevMatchAt != nil; gotPrev == tc.masquee {
			t.Errorf("%s : match précédent %v, attendu présent = %v", tc.slug, cb.PrevMatchAt, !tc.masquee)
		}
	}
}

// B4.2.7 — SnapshotPlayerState : KD/taux de victoire, meilleur KDA, dernier match.
func TestCampaignExclusion_SnapshotPlayerState(t *testing.T) {
	for _, tc := range b4Titres {
		s, err := SnapshotPlayerState(context.Background(), newB4Env(t, tc.slug), nil)
		if err != nil {
			t.Fatalf("%s : %v", tc.slug, err)
		}
		if want := b4Attendu(tc.masquee, 5.0, 55.0); s.KDRatio != want {
			t.Errorf("%s : KD %v, attendu %v", tc.slug, s.KDRatio, want)
		}
		if want := b4Attendu(tc.masquee, 1.0, 0.5); s.Winrate != want {
			t.Errorf("%s : taux de victoire %v, attendu %v", tc.slug, s.Winrate, want)
		}
		if want := b4Attendu(tc.masquee, "arena1", "camp1"); s.BestKDAMatchID != want {
			t.Errorf("%s : meilleur KDA sur %q, attendu %q", tc.slug, s.BestKDAMatchID, want)
		}
		if want := b4Attendu(tc.masquee, b4Arene, b4Campagne); !s.LastMatchStartTime.Equal(want) {
			t.Errorf("%s : dernier match %v, attendu %v", tc.slug, s.LastMatchStartTime, want)
		}
	}
}
