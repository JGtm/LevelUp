//go:build cgo

package skill

import (
	"context"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestScanLUSRGaps_HeterogeneousDataset valide la classification du détecteur sur
// un dataset mêlant les cas réels : match ac313879-like (éligible, non noté, sous
// watermark → trou d'intérieur), match noté (rated), FFA + BTB déséquilibré (non
// éligibles, exclus), match récent au-dessus du watermark (pending), plus un bot
// (xuid vide, filtré) et un quitter (n'altèrent pas l'éligibilité).
func TestScanLUSRGaps_HeterogeneousDataset(t *testing.T) {
	shared := openShadowTestDB(t)
	player := openCanonicalPlayerTestDB(t)
	ctx := context.Background()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	type part struct {
		xuid          string
		team, outcome int
		quitter       bool
	}
	insMatch := func(id string, ts time.Time, pair string, parts []part) {
		if _, err := shared.Exec(`INSERT INTO match_registry
			(match_id, start_time, start_time_utc, pair_name, is_ranked, is_firefight, duration_seconds)
			VALUES (?, ?, ?, ?, FALSE, FALSE, 600)`, id, ts, ts, pair); err != nil {
			t.Fatalf("insert match %s: %v", id, err)
		}
		for _, p := range parts {
			if _, err := shared.Exec(`INSERT INTO match_participants
				(match_id, xuid, team_id, outcome, kills, deaths, present_at_beginning, left_in_progress, time_played_seconds)
				VALUES (?, ?, ?, ?, 10, 8, TRUE, ?, ?)`,
				id, p.xuid, p.team, p.outcome, p.quitter, map[bool]float64{true: 120, false: 600}[p.quitter]); err != nil {
				t.Fatalf("insert participant %s/%s: %v", id, p.xuid, err)
			}
		}
	}

	// Trou d'intérieur (10:00) : 2v2 éligible + 1 bot (xuid vide → filtré des rosters),
	// SANS note, SOUS le watermark (12:00). C'est le cas ac313879.
	insMatch("m_interior", base.Add(10*time.Hour), "Slayer", []part{
		{xuid: "owner", team: 0, outcome: 2}, {xuid: "mate", team: 0, outcome: 2},
		{xuid: "opp1", team: 1, outcome: 3}, {xuid: "opp2", team: 1, outcome: 3},
		{xuid: "", team: 0, outcome: 2}, // bot
	})
	// Noté (11:00) : 2v2 éligible AVEC ligne LUSR.
	insMatch("m_rated", base.Add(11*time.Hour), "Slayer", []part{
		{xuid: "owner", team: 0, outcome: 2}, {xuid: "mate", team: 0, outcome: 2},
		{xuid: "opp1", team: 1, outcome: 3}, {xuid: "opp2", team: 1, outcome: 3},
	})
	// Pending (13:00) : 2v2 éligible + quitter, SANS note, AU-DESSUS du watermark.
	insMatch("m_pending", base.Add(13*time.Hour), "Slayer", []part{
		{xuid: "owner", team: 0, outcome: 2}, {xuid: "mate", team: 0, outcome: 2, quitter: true},
		{xuid: "opp1", team: 1, outcome: 3}, {xuid: "opp2", team: 1, outcome: 3},
	})
	// FFA (09:00) : 4 équipes → non éligible.
	insMatch("m_ffa", base.Add(9*time.Hour), "Slayer", []part{
		{xuid: "owner", team: 0, outcome: 2}, {xuid: "x1", team: 1, outcome: 3},
		{xuid: "x2", team: 2, outcome: 3}, {xuid: "x3", team: 3, outcome: 3},
	})
	// Déséquilibré 4v2 (09:30) : |4-2| > 1 → non éligible.
	insMatch("m_imbalance", base.Add(9*time.Hour+30*time.Minute), "Slayer", []part{
		{xuid: "owner", team: 0, outcome: 2}, {xuid: "a2", team: 0, outcome: 2},
		{xuid: "a3", team: 0, outcome: 2}, {xuid: "a4", team: 0, outcome: 2},
		{xuid: "b1", team: 1, outcome: 3}, {xuid: "b2", team: 1, outcome: 3},
	})

	if _, err := player.Exec(`INSERT INTO match_skill_rank
		(match_id, rating_type, rating_value, playlist_group)
		VALUES ('m_rated', 'LUSR', 1600, 'arena_slayer')`); err != nil {
		t.Fatalf("insert LUSR row: %v", err)
	}
	wm := base.Add(12 * time.Hour)
	if _, err := shared.Exec(`INSERT INTO player_skill_state_v2
		(xuid, playlist_group, mu, sigma, experience, last_match_id, last_match_at)
		VALUES ('owner', 'arena_slayer', 25, 5, 2, 'm_rated', ?)`, wm); err != nil {
		t.Fatalf("insert watermark: %v", err)
	}

	rep, err := ScanLUSRGaps(ctx, player, shared, "owner")
	if err != nil {
		t.Fatalf("ScanLUSRGaps: %v", err)
	}
	if len(rep.Groups) != 1 || rep.Groups[0].Group != "arena_slayer" {
		t.Fatalf("Groups = %+v, want 1 groupe arena_slayer", rep.Groups)
	}
	g := rep.Groups[0]
	if g.Eligible != 3 {
		t.Errorf("Eligible = %d, want 3 (FFA + imbalance exclus)", g.Eligible)
	}
	if g.Rated != 1 {
		t.Errorf("Rated = %d, want 1 (m_rated)", g.Rated)
	}
	if len(g.InteriorGaps) != 1 || g.InteriorGaps[0].MatchID != "m_interior" {
		t.Errorf("InteriorGaps = %+v, want [m_interior]", g.InteriorGaps)
	}
	if g.PendingRecent != 1 {
		t.Errorf("PendingRecent = %d, want 1 (m_pending au-dessus du watermark)", g.PendingRecent)
	}
	if rep.TotalEligible != 3 || rep.TotalRated != 1 || rep.TotalInteriorGaps != 1 || rep.TotalPendingRecent != 1 {
		t.Errorf("agrégats = %+v, want eligible=3 rated=1 interior=1 pending=1", rep)
	}
}

// TestLUSRInteriorGapsGauge vérifie que le setter publie bien la valeur lue par
// l'accesseur (source du badge d'onglet monitoring). Les cumuls held/owner sont
// des compteurs process-wide : on vérifie juste qu'ils sont lisibles (≥ 0).
func TestLUSRInteriorGapsGauge(t *testing.T) {
	SetLUSRInteriorGapsGauge(7)
	if got := LUSRInteriorGapsGaugeValue(); got != 7 {
		t.Errorf("LUSRInteriorGapsGaugeValue() = %d, want 7", got)
	}
	SetLUSRInteriorGapsGauge(0)
	if got := LUSRInteriorGapsGaugeValue(); got != 0 {
		t.Errorf("après reset, LUSRInteriorGapsGaugeValue() = %d, want 0", got)
	}
	// Accesseurs de compteurs cumulés : lisibles sans panique (valeur non négative).
	if v := LUSRCanonicalWriteHeldWatermarkValue(); v < 0 {
		t.Errorf("LUSRCanonicalWriteHeldWatermarkValue() = %d, want >= 0", v)
	}
	if v := LUSRCanonicalOwnerMissingValue(); v < 0 {
		t.Errorf("LUSRCanonicalOwnerMissingValue() = %d, want >= 0", v)
	}
}

// TestScanLUSRGaps_NoWatermark_AllPending : sans watermark de groupe (jamais scoré),
// aucun éligible non noté n'est un trou — tous sont « en attente ».
func TestScanLUSRGaps_NoWatermark_AllPending(t *testing.T) {
	shared := openShadowTestDB(t)
	player := openCanonicalPlayerTestDB(t)
	ctx := context.Background()
	base := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	if _, err := shared.Exec(`INSERT INTO match_registry
		(match_id, start_time, start_time_utc, pair_name, is_ranked, is_firefight, duration_seconds)
		VALUES ('m1', ?, ?, 'Slayer', FALSE, FALSE, 600)`, base, base); err != nil {
		t.Fatalf("insert match: %v", err)
	}
	for _, p := range []struct {
		xuid          string
		team, outcome int
	}{{"owner", 0, 2}, {"mate", 0, 2}, {"opp1", 1, 3}, {"opp2", 1, 3}} {
		if _, err := shared.Exec(`INSERT INTO match_participants
			(match_id, xuid, team_id, outcome, kills, deaths, present_at_beginning)
			VALUES ('m1', ?, ?, ?, 10, 8, TRUE)`, p.xuid, p.team, p.outcome); err != nil {
			t.Fatalf("insert participant: %v", err)
		}
	}

	rep, err := ScanLUSRGaps(ctx, player, shared, "owner")
	if err != nil {
		t.Fatalf("ScanLUSRGaps: %v", err)
	}
	if len(rep.Groups) != 1 {
		t.Fatalf("Groups = %+v, want 1", rep.Groups)
	}
	g := rep.Groups[0]
	if g.Eligible != 1 || len(g.InteriorGaps) != 0 || g.PendingRecent != 1 {
		t.Errorf("sans watermark : got eligible=%d interior=%d pending=%d, want 1/0/1",
			g.Eligible, len(g.InteriorGaps), g.PendingRecent)
	}
}
