//go:build integration

// Package duckdb — campaign_repo_test.go : tests CampaignSampleProvider.
// Sprint follow-up B1 Phase 1 (9g.3) : couverture des méthodes migrées
// sprint B1 (LoadAxisSamples, loadLUSRComponentSamples).
package duckdb

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/campaign"
)

// TestCampaignRepo_ListEnded vérifie le filtre closes (completed/abandoned),
// le scope user_id+title_slug et le tri ended_at desc.
func TestCampaignRepo_ListEnded(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS improvement_campaign (
			id VARCHAR PRIMARY KEY, user_id VARCHAR NOT NULL, title_slug VARCHAR NOT NULL,
			axis VARCHAR NOT NULL, axis_kind VARCHAR NOT NULL,
			started_at TIMESTAMP NOT NULL, ended_at TIMESTAMP,
			status VARCHAR NOT NULL DEFAULT 'active', playlist_group VARCHAR NOT NULL DEFAULT 'all',
			snapshot_value DOUBLE NOT NULL, snapshot_sample INTEGER NOT NULL DEFAULT 0,
			current_value_raw DOUBLE, current_value_lowess DOUBLE,
			matches_since_start INTEGER NOT NULL DEFAULT 0, last_evaluated_at TIMESTAMP,
			mann_whitney_p DOUBLE, progression_confirmed BOOLEAN NOT NULL DEFAULT FALSE,
			auto_closure_suggested BOOLEAN NOT NULL DEFAULT FALSE, auto_closure_reason VARCHAR
		)`); err != nil {
		t.Fatalf("create improvement_campaign: %v", err)
	}

	repo := NewCampaignRepo(pdb.Player)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mk := func(id, title string, snap float64) campaign.ImprovementCampaign {
		return campaign.ImprovementCampaign{
			ID: id, UserID: "u1", TitleSlug: title, Axis: "combat",
			AxisKind: campaign.AxisKindRadar, StartedAt: base, Status: campaign.StatusActive,
			PlaylistGroup: "all", SnapshotValue: snap,
		}
	}
	seed := []campaign.ImprovementCampaign{
		mk("c_completed_old", "halo_infinite", 1.0),
		mk("c_abandoned_new", "halo_infinite", 2.0),
		mk("c_active", "halo_infinite", 3.0),
		mk("c_other_title", "halo_5", 4.0),
	}
	for _, c := range seed {
		if err := repo.Insert(ctx, c); err != nil {
			t.Fatalf("insert %s: %v", c.ID, err)
		}
	}
	t1 := base.AddDate(0, 0, 10)
	t2 := base.AddDate(0, 0, 20) // plus récent
	if err := repo.UpdateStatus(ctx, "c_completed_old", campaign.StatusCompleted, &t1); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateStatus(ctx, "c_abandoned_new", campaign.StatusAbandoned, &t2); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateStatus(ctx, "c_other_title", campaign.StatusCompleted, &t2); err != nil {
		t.Fatal(err)
	}

	got, err := repo.ListEnded(ctx, "u1", "halo_infinite")
	if err != nil {
		t.Fatalf("ListEnded: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("attendu 2 campagnes closes (scope halo_infinite), obtenu %d : %+v", len(got), got)
	}
	// Tri ended_at desc : l'abandonnée (t2) avant la complétée (t1).
	if got[0].ID != "c_abandoned_new" || got[1].ID != "c_completed_old" {
		t.Errorf("ordre attendu [c_abandoned_new, c_completed_old], obtenu [%s, %s]", got[0].ID, got[1].ID)
	}
}

// TestCampaignSampleProvider_LoadAxisSamples_RadarCombat : seed
// match_participants + match_registry, charge l'axe "combat" (= kills).
func TestCampaignSampleProvider_LoadAxisSamples_RadarCombat(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	// 3 matchs dans la fenêtre 2025-01-01 → 2025-12-31, avec kills variables.
	// Le seed a déjà m1 avec kills=10. Ajoutons m2 (kills=20), m3 (kills=5).
	for _, m := range []struct {
		matchID string
		startTS string
		kills   int
	}{
		{"m_camp_2", "2025-06-15 10:00:00+00", 20},
		{"m_camp_3", "2025-09-01 10:00:00+00", 5},
	} {
		execOnSharedDBs(t, pdb, ctx,
			`INSERT INTO shared.match_registry (match_id, start_time) VALUES (?, ?)`,
			m.matchID, m.startTS)
		execOnSharedDBs(t, pdb, ctx,
			`INSERT INTO shared.match_participants (match_id, xuid, kills, deaths)
			 VALUES (?, ?, ?, ?)`,
			m.matchID, pTestXUID, m.kills, 5)
	}

	provider := NewCampaignSampleProvider(pdb)
	since, _ := time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")
	until, _ := time.Parse(time.RFC3339, "2025-12-31T23:59:59Z")

	samples, err := provider.LoadAxisSamples(ctx, pTestXUID, "halo_infinite",
		"combat", campaign.AxisKindRadar, "all", since, until)
	if err != nil {
		t.Fatalf("LoadAxisSamples: %v", err)
	}
	// seed m1 (kills=10) + m_camp_2 (20) + m_camp_3 (5) → 3 samples
	if len(samples) != 3 {
		t.Fatalf("attendu 3 samples, obtenu %d : %v", len(samples), samples)
	}
	// Ordre chronologique : m1 (2025-01-10) → 10, m_camp_2 (2025-06-15) → 20, m_camp_3 → 5
	if samples[0] != 10 || samples[1] != 20 || samples[2] != 5 {
		t.Errorf("ordre chronologique attendu [10, 20, 5], obtenu %v", samples)
	}
}

// TestCampaignSampleProvider_LoadAxisSamples_UnsupportedAxis : axis non mappé → nil.
func TestCampaignSampleProvider_LoadAxisSamples_UnsupportedAxis(t *testing.T) {
	pdb := newTestPlayerDB(t)
	provider := NewCampaignSampleProvider(pdb)
	since, _ := time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")
	until, _ := time.Parse(time.RFC3339, "2025-12-31T23:59:59Z")

	samples, err := provider.LoadAxisSamples(context.Background(), pTestXUID,
		"halo_infinite", "unknown_axis", campaign.AxisKindRadar, "", since, until)
	if err != nil {
		t.Fatalf("LoadAxisSamples unsupported: %v", err)
	}
	if samples != nil {
		t.Errorf("attendu nil pour axis non supporté, obtenu %v", samples)
	}
}

// TestCampaignSampleProvider_loadLUSRComponentSamples : split cross-DB.
// Seed lusr_component_history (player) + match_registry (shared, déjà seed).
func TestCampaignSampleProvider_loadLUSRComponentSamples(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	// Ajouter 2 matchs supplémentaires dans la fenêtre.
	for _, m := range []struct {
		matchID, startTS string
	}{
		{"lusr_m2", "2025-06-15 10:00:00+00"},
		{"lusr_m3", "2025-09-01 10:00:00+00"},
	} {
		execOnSharedDBs(t, pdb, ctx,
			`INSERT INTO shared.match_registry (match_id, start_time) VALUES (?, ?)`,
			m.matchID, m.startTS)
	}

	// lusr_component_history : valeurs sur les 3 matchs pour "kills_vs_expected".
	for _, h := range []struct {
		matchID string
		value   float64
	}{
		{"m1", 0.5},
		{"lusr_m2", 0.8},
		{"lusr_m3", 0.3},
	} {
		if _, err := pdb.Player.Exec(ctx,
			`INSERT INTO lusr_component_history (match_id, component_name, value)
			 VALUES (?, ?, ?)`,
			h.matchID, "kills_vs_expected", h.value); err != nil {
			t.Fatalf("seed lusr_component_history: %v", err)
		}
	}

	provider := NewCampaignSampleProvider(pdb)
	since, _ := time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")
	until, _ := time.Parse(time.RFC3339, "2025-12-31T23:59:59Z")

	samples, err := provider.LoadAxisSamples(ctx, pTestXUID, "halo_infinite",
		"kills_vs_expected", campaign.AxisKindLUSRComponent, "all", since, until)
	if err != nil {
		t.Fatalf("LoadAxisSamples LUSRComponent: %v", err)
	}
	// 3 matchs avec values → 3 samples ordonnés par start_time
	if len(samples) != 3 {
		t.Fatalf("attendu 3 samples, obtenu %d : %v", len(samples), samples)
	}
	if samples[0] != 0.5 || samples[1] != 0.8 || samples[2] != 0.3 {
		t.Errorf("samples ordonnés chronologiquement [0.5, 0.8, 0.3], obtenu %v", samples)
	}
}
