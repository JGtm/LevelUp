//go:build integration

package duckdb

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"levelup/go-api/internal/campaign"
)

// createCampaignTable crée la table des campagnes (schéma de la migration du joueur, index compris) sur
// la base joueur du harnais.
func createCampaignTable(t *testing.T, pdb *PlayerDB) {
	t.Helper()
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
	if _, err := pdb.Player.Exec(ctx,
		`CREATE INDEX IF NOT EXISTS idx_campaign_user_title ON improvement_campaign(user_id, title_slug, status)`); err != nil {
		t.Fatalf("create idx_campaign_user_title: %v", err)
	}
}

// memRevisionStore : révision rangée en mémoire (la base du harnais est en mémoire, sans fichier pour le
// lease de WriteSyncMeta).
type memRevisionStore struct{ rev string }

func (m *memRevisionStore) Stored(context.Context) (string, error) { return m.rev, nil }
func (m *memRevisionStore) Store(_ context.Context, rev string) error {
	m.rev = rev
	return nil
}

// seedEvaluatedCampaign insère une campagne déjà évaluée à un ancien départ (valeur 1, 4 échantillons).
func seedEvaluatedCampaign(t *testing.T, repo *CampaignRepo, id, axis string, kind campaign.AxisKind,
	status campaign.CampaignStatus, startedAt time.Time) {
	t.Helper()
	ctx := context.Background()
	err := repo.Insert(ctx, campaign.ImprovementCampaign{
		ID: id, UserID: pTestXUID, TitleSlug: "halo_infinite", Axis: axis, AxisKind: kind,
		StartedAt: startedAt, Status: campaign.StatusActive, PlaylistGroup: "all",
		SnapshotValue: 1, SnapshotSample: 4, MatchesSinceStart: 7,
	})
	if err != nil {
		t.Fatalf("insert %s: %v", id, err)
	}
	raw, lowess, p := 61.0, 62.0, 0.01
	if err := repo.UpdateEvaluation(ctx, id, campaign.Evaluation{
		CurrentRaw: sql.NullFloat64{Float64: raw, Valid: true}, CurrentLOWESS: sql.NullFloat64{Float64: lowess, Valid: true}, MannWhitneyP: sql.NullFloat64{Float64: p, Valid: true},
		MatchesSinceStart: 7, EvaluatedAt: startedAt.AddDate(0, 0, 10),
		ProgressionConfirmed: true, AutoClosureSuggested: true, AutoClosureReason: "plateau_60d",
	}); err != nil {
		t.Fatalf("évaluation de %s: %v", id, err)
	}
	if status != campaign.StatusActive {
		ended := startedAt.AddDate(0, 0, 20)
		var endedAt *time.Time
		if status != campaign.StatusPaused {
			endedAt = &ended
		}
		if err := repo.UpdateStatus(ctx, id, status, endedAt); err != nil {
			t.Fatalf("statut de %s: %v", id, err)
		}
	}
}

// ListOpen ne rend que les campagnes en cours (active, paused) du joueur et du titre.
func TestCampaignRepo_ListOpen(t *testing.T) {
	pdb := newTestPlayerDB(t)
	createCampaignTable(t, pdb)
	repo := NewCampaignRepo(pdb.Player)
	start := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	seedEvaluatedCampaign(t, repo, "c_active", "objective", campaign.AxisKindRadar, campaign.StatusActive, start)
	seedEvaluatedCampaign(t, repo, "c_paused", "combat", campaign.AxisKindRadar, campaign.StatusPaused, start)
	seedEvaluatedCampaign(t, repo, "c_done", "objective", campaign.AxisKindRadar, campaign.StatusCompleted, start)
	seedEvaluatedCampaign(t, repo, "c_gone", "objective", campaign.AxisKindRadar, campaign.StatusAbandoned, start)

	got, err := repo.ListOpen(context.Background(), pTestXUID, "halo_infinite")
	if err != nil {
		t.Fatalf("ListOpen: %v", err)
	}
	ids := map[string]bool{}
	for _, c := range got {
		ids[c.ID] = true
	}
	if len(got) != 2 || !ids["c_active"] || !ids["c_paused"] {
		t.Fatalf("ListOpen = %v, attendu c_active et c_paused seulement", ids)
	}
	if other, _ := repo.ListOpen(context.Background(), pTestXUID, "halo_5"); len(other) != 0 {
		t.Errorf("ListOpen d'un autre titre = %d campagnes, attendu 0", len(other))
	}
}

// RestartMeasure remplace le départ et vide l'évaluation d'une campagne en cours ; une campagne close
// n'est pas touchée.
func TestCampaignRepo_RestartMeasure(t *testing.T) {
	pdb := newTestPlayerDB(t)
	createCampaignTable(t, pdb)
	repo := NewCampaignRepo(pdb.Player)
	ctx := context.Background()
	start := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	seedEvaluatedCampaign(t, repo, "c_active", "objective", campaign.AxisKindRadar, campaign.StatusActive, start)
	seedEvaluatedCampaign(t, repo, "c_done", "objective", campaign.AxisKindRadar, campaign.StatusCompleted, start)

	for _, id := range []string{"c_active", "c_done"} {
		if err := repo.RestartMeasure(ctx, id, campaign.MeasureStart{SnapshotValue: 75, SnapshotSample: 2, MatchesSinceStart: 1}); err != nil {
			t.Fatalf("RestartMeasure %s: %v", id, err)
		}
	}
	a, err := repo.GetByID(ctx, "c_active")
	if err != nil {
		t.Fatal(err)
	}
	if a.SnapshotValue != 75 || a.SnapshotSample != 2 || a.MatchesSinceStart != 1 || a.Status != campaign.StatusActive {
		t.Errorf("départ de c_active = %v / %d / %d (statut %s), attendu 75 / 2 / 1 (active)",
			a.SnapshotValue, a.SnapshotSample, a.MatchesSinceStart, a.Status)
	}
	if a.CurrentValueRaw != nil || a.CurrentValueLOWESS != nil || a.MannWhitneyP != nil || a.LastEvaluatedAt != nil ||
		a.ProgressionConfirmed || a.AutoClosureSuggested || a.AutoClosureReason != "" {
		t.Errorf("évaluation de c_active non remise à vide : %+v", a)
	}
	d, err := repo.GetByID(ctx, "c_done")
	if err != nil {
		t.Fatal(err)
	}
	if d.SnapshotValue != 1 || d.SnapshotSample != 4 || d.CurrentValueRaw == nil || !d.ProgressionConfirmed {
		t.Errorf("campagne close modifiée : %+v", d)
	}
}

// De bout en bout sur le dépôt et la mesure réels : le départ d'une campagne sur l'axe Objectif est
// recalé avec la note d'objectif par match, celui d'une campagne sur un autre axe ne bouge pas, et un
// second passage n'écrase pas l'évaluation faite depuis.
func TestApplyObjectiveAxisRev_SurLaMesureReelle(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	createCampaignTable(t, pdb)
	type seed struct {
		matchID, startTS string
		me, mate         int
	}
	for _, m := range []seed{
		{"m_obj_1", "2025-06-15 10:00:00+00", 2, 2}, // 50
		{"m_obj_2", "2025-07-01 10:00:00+00", 3, 0}, // 100
		{"m_obj_3", "2025-09-10 10:00:00+00", 1, 3}, // 25, après le départ
	} {
		execOnSharedDBs(t, pdb, ctx,
			`INSERT INTO shared.match_registry (match_id, start_time) VALUES (?, ?)`, m.matchID, m.startTS)
		for _, xuid := range []string{pTestXUID, "xuid_mate"} {
			execOnSharedDBs(t, pdb, ctx,
				`INSERT INTO shared.match_participants (match_id, xuid, kills, deaths, team_id, time_played_seconds)
				 VALUES (?, ?, 1, 1, 0, 600)`, m.matchID, xuid)
		}
		mustExecPlayerShared(t, pdb, ctx,
			`INSERT INTO shared.match_objective_stats (match_id, xuid, flag_captures) VALUES (?, ?, ?)`,
			m.matchID, pTestXUID, m.me)
		mustExecPlayerShared(t, pdb, ctx,
			`INSERT INTO shared.match_objective_stats (match_id, xuid, flag_captures) VALUES (?, ?, ?)`,
			m.matchID, "xuid_mate", m.mate)
	}
	repo := NewCampaignRepo(pdb.Player)
	start := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	seedEvaluatedCampaign(t, repo, "c_obj", "objective", campaign.AxisKindRadar, campaign.StatusActive, start)
	seedEvaluatedCampaign(t, repo, "c_combat", "combat", campaign.AxisKindRadar, campaign.StatusPaused, start)
	svc := campaign.NewService(repo, NewCampaignSampleProvider(pdb))
	store := &memRevisionStore{}
	now := time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC)

	n, err := svc.ApplyObjectiveAxisRev(ctx, store, pTestXUID, "halo_infinite", now)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v, attendu 1 campagne recalée", n, err)
	}
	o, _ := repo.GetByID(ctx, "c_obj")
	if o.SnapshotValue != 75 || o.SnapshotSample != 2 || o.MatchesSinceStart != 1 || o.CurrentValueRaw != nil {
		t.Errorf("c_obj : départ %v sur %d, %d match depuis, courant %v ; attendu 75 sur 2, 1 match, courant vide",
			o.SnapshotValue, o.SnapshotSample, o.MatchesSinceStart, o.CurrentValueRaw)
	}
	c, _ := repo.GetByID(ctx, "c_combat")
	if c.SnapshotValue != 1 || c.SnapshotSample != 4 || c.CurrentValueRaw == nil {
		t.Errorf("c_combat modifiée alors qu'elle n'est pas sur l'axe Objectif : %+v", c)
	}
	if store.rev != campaign.ObjectiveAxisRev {
		t.Errorf("révision rangée = %q, attendu %q", store.rev, campaign.ObjectiveAxisRev)
	}
	if n, err := svc.ApplyObjectiveAxisRev(ctx, store, pTestXUID, "halo_infinite", now); err != nil || n != 0 {
		t.Errorf("second passage : n=%d err=%v, attendu 0, nil", n, err)
	}
}

// SyncMetaRevisionStore range et relit la révision dans sync_meta d'un fichier de base joueur, sous lease.
func TestSyncMetaRevisionStore_RangeEtRelit(t *testing.T) {
	db, err := OpenReadWrite(t.TempDir() + "/stats.duckdb")
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	defer db.Close()
	ctx := context.Background()
	if _, err := db.Exec(ctx, `CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR, updated_at TIMESTAMP)`); err != nil {
		t.Fatalf("create sync_meta: %v", err)
	}
	store := NewSyncMetaRevisionStore(&PlayerDB{Player: db}, campaign.SyncMetaObjectiveAxisRev)

	if got, err := store.Stored(ctx); err != nil || got != "" {
		t.Fatalf("avant rangement : %q, %v ; attendu vide", got, err)
	}
	for _, rev := range []string{"r1", "r2"} { // INSERT puis UPDATE (SELECT-then-UPDATE-or-INSERT)
		if err := store.Store(ctx, rev); err != nil {
			t.Fatalf("Store(%s): %v", rev, err)
		}
		if got, err := store.Stored(ctx); err != nil || got != rev {
			t.Fatalf("après Store(%s) : %q, %v", rev, got, err)
		}
	}
}

// Le fournisseur d'échantillons et le recalage désignent le même axe : la constante du fournisseur ne
// dérive pas de celle du service.
func TestCampaignObjectiveAxis_MemeAxeQueLeService(t *testing.T) {
	if campaignObjectiveAxis != campaign.AxisObjective {
		t.Fatalf("campaignObjectiveAxis = %q, campaign.AxisObjective = %q", campaignObjectiveAxis, campaign.AxisObjective)
	}
}
