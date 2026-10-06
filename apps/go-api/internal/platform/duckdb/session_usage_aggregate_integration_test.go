//go:build integration

// Package duckdb_test — session_usage_aggregate_integration_test.go : le résumé
// d'usage DE BOUT EN BOUT sur une DB migrée par les VRAIES migrations et
// peuplée par le VRAI persister (persist.UsageSummaryPersister) — jamais de DDL
// recopiée, jamais d'INSERT direct dans les tables d'usage.
//
// En package duckdb_test (modèle csr_pipeline_e2e_integration_test.go) : le
// package duckdb ne peut pas importer persist (cycle persist→duckdb).
//
// Le scénario est le témoin miniature du plan S2 : 3 matchs dont 2 mesurés
// (couverture partielle), deux camps, effectifs INÉGAUX (2v2 puis 3v2), une
// re-passe sur m1 (la vue _latest doit servir la DERNIÈRE
// passe), lus par le repo puis assemblés comme le font l'Emprise et les formes.
package duckdb_test

import (
	"context"
	"math"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/persist"
	ddb "levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

// setupUsageSharedDB : DB shared temporaire, schéma réel + migrations réelles.
func setupUsageSharedDB(t *testing.T) (*ddb.DB, *ddb.PlayerDB) {
	t.Helper()
	sharedDB, err := ddb.OpenReadWrite(filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatalf("open shared: %v", err)
	}
	t.Cleanup(func() { _ = sharedDB.Close() })
	// Migrations réelles D'ABORD (sur DB vierge, comme openUsageSummaryTestDB —
	// les steps fr-cols échouent si match_registry préexiste sans ces colonnes),
	// puis le schéma sync (IF NOT EXISTS) qui apporte match_participants.
	if err := migration.RunForDB(sharedDB.SQLDb(), migration.TargetShared); err != nil {
		t.Fatalf("RunForDB shared: %v", err)
	}
	if err := syncpkg.EnsureSharedSchema(context.Background(), sharedDB.SQLDb()); err != nil {
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	pdb := &ddb.PlayerDB{
		Shared:       sharedDB,
		SharedReader: ddb.LegacySharedReader(sharedDB),
		XUID:         "P",
		TitleSlug:    "halo_infinite",
	}
	return sharedDB, pdb
}

func seedUsageParticipant(t *testing.T, db *ddb.DB, matchID, xuid, gamertag string, teamID int) {
	t.Helper()
	if _, err := db.SQLDb().Exec(`
		INSERT INTO match_participants (match_id, xuid, gamertag, team_id, present_at_completion)
		VALUES (?, ?, ?, ?, TRUE)`, matchID, xuid, gamertag, teamID); err != nil {
		t.Fatalf("seed participant %s/%s: %v", matchID, xuid, err)
	}
}

// passeM1 / passeM2 : les résumés écrits par le persister réel. La PREMIÈRE
// passe de m1 porte des valeurs fausses (99) que la re-passe doit supplanter.
func passeM1Fausse() *replay.UsageSummary {
	return &replay.UsageSummary{
		Match: replay.UsageMatchSummary{DurationMS: 1, PadUnnamed: 99},
		Players: []replay.UsagePlayerSummary{
			{XUID: "P", PadPickups: 99},
		},
	}
}

func passeM1() *replay.UsageSummary {
	return &replay.UsageSummary{
		Match: replay.UsageMatchSummary{
			DurationMS: 600000, PadNamed: 6, PadUnnamed: 3,
			PowerupPadPickups: map[string]int{"powerup_camo": 2},
		},
		Players: []replay.UsagePlayerSummary{
			{XUID: "P", PadPickups: 1, PadPickupsByWeapon: map[string]int{"aabbccdd": 1}},
			{XUID: "A", PadPickups: 2, DeployedByFamily: map[string]int{"wall": 1}},
			{XUID: "E1", PadPickups: 3, PadPickupsByWeapon: map[string]int{"aabbccdd": 2, "eeff0011": 1}},
		},
	}
}

func passeM2() *replay.UsageSummary {
	return &replay.UsageSummary{
		Match: replay.UsageMatchSummary{
			DurationMS: 300000, PadNamed: 6, PadUnnamed: 1,
			PowerupPadPickups: map[string]int{"powerup_camo": 1, "powerup_overshield": 1},
		},
		Players: []replay.UsagePlayerSummary{
			{XUID: "P", PadPickups: 4},
			{XUID: "B", PadPickups: 1},
			{XUID: "E1", PadPickups: 1},
		},
	}
}

func proche(got *float64, want float64) bool {
	return got != nil && math.Abs(*got-want) < 1e-6
}

func TestSessionUsageAggregate_DePersisterAuBloc(t *testing.T) {
	sharedDB, pdb := setupUsageSharedDB(t)
	ctx := context.Background()

	// Participants : m1 2v2, m2 3v2 (effectifs inégaux), m3 sans film.
	for _, p := range []struct {
		match, xuid, gt string
		team            int
	}{
		{"m1", "P", "Papa", 0}, {"m1", "A", "Alpha", 0}, {"m1", "E1", "Echo", 1}, {"m1", "E2", "Ezra", 1},
		{"m2", "P", "Papa", 0}, {"m2", "A", "Alpha", 0}, {"m2", "B", "Bravo", 0}, {"m2", "E1", "Echo", 1}, {"m2", "E2", "Ezra", 1},
		{"m3", "P", "Papa", 0}, {"m3", "A", "Alpha", 0}, {"m3", "E1", "Echo", 1},
	} {
		seedUsageParticipant(t, sharedDB, p.match, p.xuid, p.gt, p.team)
	}

	// Écritures par le persister RÉEL : deux passes sur m1 (la re-passe fait foi).
	persister := persist.NewUsageSummaryPersister(sharedDB.SQLDb())
	if err := persister.PersistPass(ctx, "m1", passeM1Fausse()); err != nil {
		t.Fatalf("PersistPass m1 (passe A): %v", err)
	}
	if err := persister.PersistPass(ctx, "m1", passeM1()); err != nil {
		t.Fatalf("PersistPass m1 (passe B): %v", err)
	}
	if err := persister.PersistPass(ctx, "m2", passeM2()); err != nil {
		t.Fatalf("PersistPass m2: %v", err)
	}

	// Lecture par le repo réel (vues _latest uniquement).
	repo := ddb.NewSessionUsageRepo(pdb)
	ids := []string{"m1", "m2", "m3"}
	films, err := repo.LoadUsageFilms(ctx, ids)
	if err != nil {
		t.Fatalf("LoadUsageFilms: %v", err)
	}
	players, err := repo.LoadUsagePlayers(ctx, ids)
	if err != nil {
		t.Fatalf("LoadUsagePlayers: %v", err)
	}
	participants, err := repo.LoadParticipants(ctx, ids)
	if err != nil {
		t.Fatalf("LoadParticipants: %v", err)
	}
	if len(films) != 2 {
		t.Fatalf("films = %v, attendu m1 et m2 seulement (m3 non mesuré)", films)
	}

	// ── La vue _latest sert la DERNIÈRE passe de m1 (la passe A portait 1 ms et 99 prises) ──
	if films["m1"].DurationMS != 600000 || films["m2"].DurationMS != 300000 {
		t.Errorf("durées = %d / %d ms, attendu 600000 / 300000", films["m1"].DurationMS, films["m2"].DurationMS)
	}
	if films["m1"].PowerupPickups["powerup_camo"] != 2 || films["m2"].PowerupPickups["powerup_overshield"] != 1 {
		t.Errorf("prises de bonus = %v / %v", films["m1"].PowerupPickups, films["m2"].PowerupPickups)
	}

	// ── Assemblage (le chemin de l'Emprise et des formes) : mesure, camps, effectifs inégaux ──
	tc := sessionusage.BuildTeamContext("P", participants)
	matchs := sessionusage.BuildMatchInputs(ids, films, players, tc)
	if len(matchs) != 3 || !matchs[0].Measured || !matchs[1].Measured || matchs[2].Measured {
		t.Fatalf("matchs = %+v, attendu m1 et m2 mesurés, m3 non", matchs)
	}
	if matchs[0].TeamSize != 2 || matchs[1].TeamSize != 3 || matchs[0].LobbySize != 4 || matchs[1].LobbySize != 5 {
		t.Errorf("effectifs = (%d, %d) camp, (%d, %d) lobby ; attendu (2, 3) et (4, 5)",
			matchs[0].TeamSize, matchs[1].TeamSize, matchs[0].LobbySize, matchs[1].LobbySize)
	}
	prisesDuJoueur := 0
	for _, m := range matchs {
		for _, p := range m.Players {
			if p.XUID == "P" {
				prisesDuJoueur += p.PadPickups
			}
		}
	}
	if prisesDuJoueur != 5 {
		t.Errorf("prises de socle du joueur = %d, attendu 5 (1 + 4 : la passe A, 99, est supplantée)", prisesDuJoueur)
	}
}
