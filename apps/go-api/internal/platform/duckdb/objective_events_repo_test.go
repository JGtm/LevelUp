//go:build integration

// Package duckdb — objective_events_repo_test.go : tests ObjectiveEventsRepo.
//
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -v
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	titlepkg "levelup/go-api/internal/domain/title"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/migration"
)

const oeTestMatchID = "m_objective_001"

// newObjectiveEventsTestPlayerDB ouvre une mem DB, applique TOUTES les migrations
// shared (dont shared_objective_events_v1 — la "vraie" CREATE sous test), puis
// construit un PlayerDB dont le SharedReader pointe sur cette conn (RW en legacy).
func newObjectiveEventsTestPlayerDB(t *testing.T) *PlayerDB {
	t.Helper()
	sqlDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open mem: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	if err := migration.RunForDB(sqlDB, migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(Shared): %v", err)
	}
	shared := newTestDB(sqlDB, ":memory:")

	return &PlayerDB{
		Shared:       shared,
		SharedReader: LegacySharedReader(shared),
		XUID:         pTestXUID,
		Gamertag:     pTestGamertag,
		TitleSlug:    titlepkg.DefaultSlug,
	}
}

func sampleObjectiveEvents() []domain.ObjectiveEvent {
	team0, team1 := 0, 1
	t1, t2 := 30000, 95000
	cap1, cap2 := 1, 2
	return []domain.ObjectiveEvent{
		{
			MatchID: oeTestMatchID, Seq: 0, TimeMS: &t1,
			ObjectiveType: "flag", EventType: "capture",
			TeamID: &team0, ObjectiveID: nil, Value: &cap1,
			Source: "film_ctf", Confidence: "ms_exact", Details: `{"tiers":6}`,
			Players: []domain.ObjectiveEventPlayer{
				{XUID: pTestXUID, Role: "carrier"},
				{XUID: "xuid_helper_a", Role: "escort"},
			},
		},
		{
			// Event sans joueur + champs NULL-able (team unreliable, objective_id NULL).
			MatchID: oeTestMatchID, Seq: 1, TimeMS: &t2,
			ObjectiveType: "flag", EventType: "capture",
			TeamID: &team1, ObjectiveID: nil, Value: &cap2,
			Source: "film_ctf", Confidence: "ms_exact", Details: "",
			Players: nil,
		},
	}
}

// ---------------------------------------------------------------------------
// Round-trip
// ---------------------------------------------------------------------------

func TestObjectiveEventsRepo_WriteThenLoad_RoundTrip(t *testing.T) {
	pdb := newObjectiveEventsTestPlayerDB(t)
	repo := NewObjectiveEventsRepo(pdb)
	ctx := context.Background()

	if err := repo.WriteMatch(ctx, oeTestMatchID, sampleObjectiveEvents()); err != nil {
		t.Fatalf("WriteMatch: %v", err)
	}

	got, err := repo.LoadMatch(ctx, oeTestMatchID)
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(got))
	}

	// Ordonné par seq ASC.
	if got[0].Seq != 0 || got[1].Seq != 1 {
		t.Fatalf("seq order = [%d,%d], want [0,1]", got[0].Seq, got[1].Seq)
	}

	ev0 := got[0]
	if ev0.ObjectiveType != "flag" || ev0.EventType != "capture" {
		t.Errorf("ev0 type/event = %q/%q", ev0.ObjectiveType, ev0.EventType)
	}
	if ev0.TimeMS == nil || *ev0.TimeMS != 30000 {
		t.Errorf("ev0.TimeMS = %v, want 30000", ev0.TimeMS)
	}
	if ev0.TeamID == nil || *ev0.TeamID != 0 {
		t.Errorf("ev0.TeamID = %v, want 0", ev0.TeamID)
	}
	if ev0.Value == nil || *ev0.Value != 1 {
		t.Errorf("ev0.Value = %v, want 1", ev0.Value)
	}
	if ev0.Source != "film_ctf" || ev0.Confidence != "ms_exact" {
		t.Errorf("ev0 source/conf = %q/%q", ev0.Source, ev0.Confidence)
	}
	if ev0.Details != `{"tiers":6}` {
		t.Errorf("ev0.Details = %q", ev0.Details)
	}
	if len(ev0.Players) != 2 {
		t.Fatalf("ev0 players = %d, want 2", len(ev0.Players))
	}
	// Players ordonnés xuid ASC : xuid_helper_a avant xuid_player_001.
	if ev0.Players[0].XUID != "xuid_helper_a" || ev0.Players[0].Role != "escort" {
		t.Errorf("player[0] = %+v", ev0.Players[0])
	}
	if ev0.Players[1].XUID != pTestXUID || ev0.Players[1].Role != "carrier" {
		t.Errorf("player[1] = %+v", ev0.Players[1])
	}

	// ev1 : objective_id NULL, pas de joueur, details vide.
	ev1 := got[1]
	if ev1.ObjectiveID != nil {
		t.Errorf("ev1.ObjectiveID = %v, want nil", ev1.ObjectiveID)
	}
	if len(ev1.Players) != 0 {
		t.Errorf("ev1 players = %d, want 0", len(ev1.Players))
	}
}

// ---------------------------------------------------------------------------
// Idempotence DELETE-replace
// ---------------------------------------------------------------------------

func TestObjectiveEventsRepo_WriteMatch_ReplaceIdempotent(t *testing.T) {
	pdb := newObjectiveEventsTestPlayerDB(t)
	repo := NewObjectiveEventsRepo(pdb)
	ctx := context.Background()

	if err := repo.WriteMatch(ctx, oeTestMatchID, sampleObjectiveEvents()); err != nil {
		t.Fatalf("WriteMatch #1: %v", err)
	}

	// Ré-écrit avec un set RÉDUIT (1 seul event) : le DELETE doit purger les 2
	// anciens events + leurs joueurs avant l'INSERT.
	t0 := 5000
	replaced := []domain.ObjectiveEvent{{
		MatchID: oeTestMatchID, Seq: 0, TimeMS: &t0,
		ObjectiveType: "zone", EventType: "score",
		Source: "film_strongholds", Confidence: "approx_5s",
		Players: []domain.ObjectiveEventPlayer{{XUID: pTestXUID, Role: "holder"}},
	}}
	if err := repo.WriteMatch(ctx, oeTestMatchID, replaced); err != nil {
		t.Fatalf("WriteMatch #2: %v", err)
	}

	got, err := repo.LoadMatch(ctx, oeTestMatchID)
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(events) = %d, want 1 (replace)", len(got))
	}
	if got[0].ObjectiveType != "zone" || got[0].EventType != "score" {
		t.Errorf("got[0] type/event = %q/%q, want zone/score", got[0].ObjectiveType, got[0].EventType)
	}
	if len(got[0].Players) != 1 || got[0].Players[0].XUID != pTestXUID {
		t.Errorf("got[0].Players = %+v", got[0].Players)
	}

	// Aucun joueur orphelin de l'ancien event seq=1 ne doit subsister.
	var orphanPlayers int
	if err := pdb.Shared.QueryRow(ctx,
		`SELECT COUNT(*) FROM match_objective_event_players WHERE match_id = ? AND seq = 1`,
		oeTestMatchID).Scan(&orphanPlayers); err != nil {
		t.Fatalf("count orphans: %v", err)
	}
	if orphanPlayers != 0 {
		t.Errorf("orphan players seq=1 = %d, want 0", orphanPlayers)
	}
}

// ---------------------------------------------------------------------------
// No-op garde-fou
// ---------------------------------------------------------------------------

func TestObjectiveEventsRepo_WriteMatch_EmptyIsNoOp(t *testing.T) {
	pdb := newObjectiveEventsTestPlayerDB(t)
	repo := NewObjectiveEventsRepo(pdb)
	ctx := context.Background()

	if err := repo.WriteMatch(ctx, oeTestMatchID, sampleObjectiveEvents()); err != nil {
		t.Fatalf("WriteMatch seed: %v", err)
	}
	// nil et [] sont des no-op : ne doivent PAS effacer l'existant.
	if err := repo.WriteMatch(ctx, oeTestMatchID, nil); err != nil {
		t.Fatalf("WriteMatch(nil): %v", err)
	}
	if err := repo.WriteMatch(ctx, oeTestMatchID, []domain.ObjectiveEvent{}); err != nil {
		t.Fatalf("WriteMatch([]): %v", err)
	}

	got, err := repo.LoadMatch(ctx, oeTestMatchID)
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len(events) = %d, want 2 (no-op préserve)", len(got))
	}
}

// LoadMatch sur un match absent retourne un slice vide, pas une erreur.
func TestObjectiveEventsRepo_LoadMatch_Empty(t *testing.T) {
	pdb := newObjectiveEventsTestPlayerDB(t)
	repo := NewObjectiveEventsRepo(pdb)

	got, err := repo.LoadMatch(context.Background(), "no_such_match")
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len(events) = %d, want 0", len(got))
	}
}

// ---------------------------------------------------------------------------
// Capability gating
// ---------------------------------------------------------------------------

func TestObjectiveEventsRepo_CapabilityNotSupported_NoTable(t *testing.T) {
	pdb := newObjectiveEventsTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Shared.Exec(ctx, "DROP TABLE match_objective_event_players"); err != nil {
		t.Fatalf("drop players: %v", err)
	}
	if _, err := pdb.Shared.Exec(ctx, "DROP TABLE match_objective_events"); err != nil {
		t.Fatalf("drop events: %v", err)
	}

	repo := NewObjectiveEventsRepo(pdb)

	if err := repo.WriteMatch(ctx, oeTestMatchID, sampleObjectiveEvents()); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("WriteMatch err = %v, want ErrCapabilityNotSupported", err)
	}
	if _, err := repo.LoadMatch(ctx, oeTestMatchID); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("LoadMatch err = %v, want ErrCapabilityNotSupported", err)
	}
}
