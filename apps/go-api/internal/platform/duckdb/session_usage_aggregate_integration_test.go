//go:build integration

// Package duckdb_test — session_usage_aggregate_integration_test.go : l'agrégat
// de session S2 DE BOUT EN BOUT sur une DB migrée par les VRAIES migrations et
// peuplée par le VRAI persister (persist.UsageSummaryPersister) — jamais de DDL
// recopiée, jamais d'INSERT direct dans les tables d'usage.
//
// En package duckdb_test (modèle csr_pipeline_e2e_integration_test.go) : le
// package duckdb ne peut pas importer persist (cycle persist→duckdb).
//
// Le scénario est le témoin miniature du plan S2 : 3 matchs dont 2 mesurés
// (couverture partielle), deux camps, effectifs INÉGAUX (2v2 puis 3v2 → parités
// 40 % et 22,2 %), une re-passe sur m1 (la vue _latest doit servir la DERNIÈRE
// passe), un coéquipier suivi présent partout (A) et un ami d'un seul match (B).
package duckdb_test

import (
	"context"
	"math"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
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

	// Assemblage (même chemin que le service) + agrégat.
	tc := sessionusage.BuildTeamContext("P", participants)
	playersByMatch := map[string][]sessionusage.PlayerRow{}
	for _, p := range players {
		playersByMatch[p.MatchID] = append(playersByMatch[p.MatchID], p)
	}
	in := sessionusage.Input{PlayerXUID: "P"}
	for _, id := range ids {
		film, measured := films[id]
		m := sessionusage.MatchInput{
			MatchID: id, Measured: measured,
			TeamOf: tc.TeamOf[id], TeamSize: tc.TeamSize[id], LobbySize: tc.LobbySize[id],
			Players: playersByMatch[id],
		}
		if team, ok := tc.PlayerTeam[id]; ok {
			tt := team
			m.PlayerTeam = &tt
		}
		if measured {
			m.DurationSeconds = float64(film.DurationMS) / 1000
			m.PadUnnamed = film.PadUnnamed
			m.PowerupPickups = film.PowerupPickups
		}
		in.Matches = append(in.Matches, m)
	}
	squad := sessionusage.ResolveTrackedSquad("P", ids, participants, nil)
	for _, member := range squad {
		in.SquadXUIDs = append(in.SquadXUIDs, member.XUID)
	}
	out := sessionusage.ComputeUsage(in)
	out.SquadPlayers = squad

	// ── Couverture partielle 2/3 et durée mesurée seule ─────────────────────────
	if out.MatchesMeasured != 2 || out.MatchesTotal != 3 {
		t.Errorf("couverture = %d/%d, attendu 2/3", out.MatchesMeasured, out.MatchesTotal)
	}
	if out.MeasuredDurationSeconds != 900 {
		t.Errorf("durée mesurée = %v, attendu 900 s", out.MeasuredDurationSeconds)
	}
	// La passe A de m1 (pad_unnamed 99) doit avoir été supplantée : 3 + 1 = 4.
	if out.PadUnnamedTotal != 4 {
		t.Errorf("pad_unnamed_total = %d, attendu 4 (la vue _latest sert la DERNIÈRE passe)", out.PadUnnamedTotal)
	}

	// ── Parités sur effectifs inégaux (2 puis 3 ; 4 puis 5) ─────────────────────
	if !proche(out.TeamParityPct, 40) || !proche(out.LobbyParityPct, 100/4.5) {
		t.Errorf("parités = (%v, %v), attendu (40, 22.22)", out.TeamParityPct, out.LobbyParityPct)
	}

	var pad *domain.SessionUsageMetric
	for i := range out.Metrics {
		if out.Metrics[i].Key == sessionusage.MetricPadPickups {
			pad = &out.Metrics[i]
		}
	}
	if pad == nil {
		t.Fatalf("métrique pad_pickups absente : %+v", out.Metrics)
	}
	// ── Deux camps : sommes joueur/camp/lobby (passe A 99 exclue) ───────────────
	if pad.PlayerTotal != 5 || !proche(pad.TeamTotal, 8) || pad.LobbyTotal != 12 {
		t.Errorf("(joueur, camp, lobby) = (%v, %v, %v), attendu (5, 8, 12)",
			pad.PlayerTotal, pad.TeamTotal, pad.LobbyTotal)
	}
	// ── Cadence sur la durée mesurée seule ──────────────────────────────────────
	if !proche(pad.PlayerPer10Min, 5*600.0/900) {
		t.Errorf("cadence joueur = %v, attendu 3.333", pad.PlayerPer10Min)
	}
	// ── Étendue + matchs au-dessus de la parité DU match ────────────────────────
	if !proche(pad.PlayerShareOfTeamMinPct, 100.0/3) || !proche(pad.PlayerShareOfTeamMaxPct, 80) {
		t.Errorf("étendue = (%v, %v), attendu (33.33, 80)",
			pad.PlayerShareOfTeamMinPct, pad.PlayerShareOfTeamMaxPct)
	}
	if pad.MatchesAboveTeamParity == nil || *pad.MatchesAboveTeamParity != 1 || pad.MatchesAboveLobbyParity != 1 {
		t.Errorf("au-dessus parité = (%v, %d), attendu (1, 1)",
			pad.MatchesAboveTeamParity, pad.MatchesAboveLobbyParity)
	}

	// ── Lignes d'escouade : A partout (2 prises), B exclu (un seul match) ───────
	if len(out.SquadPlayers) != 1 || out.SquadPlayers[0].Gamertag != "Alpha" {
		t.Fatalf("squad_players = %+v, attendu [Alpha]", out.SquadPlayers)
	}
	if len(pad.Squad) != 1 || pad.Squad[0].XUID != "A" || pad.Squad[0].Total != 2 {
		t.Fatalf("ligne squad = %+v, attendu A total 2", pad.Squad)
	}
	if !proche(pad.Squad[0].ShareOfTeamPct, 25) || !proche(pad.Squad[0].Per10Min, 2*600.0/900) {
		t.Errorf("ligne squad A = %+v, attendu part équipe 25 %%, cadence 1.333", pad.Squad[0])
	}

	// ── Ventilations : familles normalisées + bonus anonymes ────────────────────
	if len(out.PadFamilies) != 2 || out.PadFamilies[0].FamilyKey != "aabbccdd" || out.PadFamilies[0].LobbyTotal != 3 {
		t.Errorf("pad_families = %+v, attendu aabbccdd (3) en tête", out.PadFamilies)
	}
	if len(out.PowerupPickups) != 2 || out.PowerupPickups[0].FamilyKey != "powerup_camo" || out.PowerupPickups[0].Occupations != 3 {
		t.Errorf("powerup_pickups = %+v, attendu camo 3 puis overshield 1", out.PowerupPickups)
	}
}
