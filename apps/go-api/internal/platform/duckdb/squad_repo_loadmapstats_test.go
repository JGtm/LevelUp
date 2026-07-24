//go:build integration

package duckdb

import (
	"context"
	"testing"
)

// TestSquadRepo_LoadMapStatsForSquad_Empty : squadXUIDs vide → nil sans erreur.
func TestSquadRepo_LoadMapStatsForSquad_Empty(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewSquadRepo(pdb)

	got, err := repo.LoadMapStatsForSquad(context.Background(), pTestXUID, nil, nil)
	if err != nil {
		t.Fatalf("LoadMapStatsForSquad nil squad: %v", err)
	}
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}

	got, err = repo.LoadMapStatsForSquad(context.Background(), "", []string{"x"}, nil)
	if err != nil {
		t.Fatalf("LoadMapStatsForSquad empty main: %v", err)
	}
	if got != nil {
		t.Errorf("empty main: want nil, got %v", got)
	}
}

// TestSquadRepo_LoadMapStatsForSquad_StrictIntersection : seuls les matchs où
// TOUS les xuids du squad sont participants doivent être comptés. Un match avec
// un coéquipier manquant doit être exclu — le but du fix.
//
// Setup : 3 matchs sur la même map_id "aquarius".
//   - m1 (déjà seedé) : main + mate1 sur la même team — squad complet
//   - m2 : main + mate1 + mate2 sur la même team — squad complet
//   - m3 : main + mate1 seuls (pas mate2) — squad partiel → DOIT ÊTRE EXCLU
//
// Si le filtre "squad strict" fonctionne, on obtient pour aquarius wins=2 (m1,m2)
// et total=2. Si le filtre fuit (squad partiel inclus), on aurait wins=3 total=3.
func TestSquadRepo_LoadMapStatsForSquad_StrictIntersection(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	// m1 (existant) a déjà : main=win, map_id=aquarius, perf=85.5.
	// On ajoute mate1 ET mate2 sur m1 → squad complet présent.
	mate1, mate2 := "xuid_mate_001", "xuid_mate_002"
	for _, p := range []struct{ x, gt string }{{mate1, "Mate1"}, {mate2, "Mate2"}} {
		execOnSharedDBs(t, pdb, ctx,
			`INSERT INTO shared.match_participants
			 (match_id,xuid,gamertag,outcome,team_id) VALUES (?,?,?,?,?)`,
			"m1", p.x, p.gt, 2, 1)
	}

	// m2 : new match, map_id=aquarius, main+mate1+mate2 win
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_registry (match_id, map_id) VALUES ('m2', 'aquarius')`)
	for _, p := range []struct{ x, gt string }{
		{pTestXUID, pTestGamertag}, {mate1, "Mate1"}, {mate2, "Mate2"},
	} {
		execOnSharedDBs(t, pdb, ctx,
			`INSERT INTO shared.match_participants (match_id,xuid,gamertag,outcome,team_id) VALUES (?,?,?,?,?)`,
			"m2", p.x, p.gt, 2, 1)
	}
	// PME = player-only.
	if _, err := pdb.Player.Exec(ctx,
		`INSERT INTO player_match_enrichment (match_id, performance_score) VALUES ('m2', 75.5)`); err != nil {
		t.Fatal(err)
	}

	// m3 : main+mate1 seuls (pas mate2) — squad PARTIEL → exclu si filtre OK
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_registry (match_id, map_id) VALUES ('m3', 'aquarius')`)
	for _, p := range []struct {
		x       string
		outcome int
	}{
		{pTestXUID, 3}, {mate1, 3}, // loss tous les deux, mate2 absent
	} {
		execOnSharedDBs(t, pdb, ctx,
			`INSERT INTO shared.match_participants (match_id,xuid,outcome,team_id) VALUES (?,?,?,?)`,
			"m3", p.x, p.outcome, 1)
	}

	repo := NewSquadRepo(pdb)
	stats, err := repo.LoadMapStatsForSquad(ctx, pTestXUID, []string{mate1, mate2}, nil)
	if err != nil {
		t.Fatalf("LoadMapStatsForSquad: %v", err)
	}

	got, ok := stats["aquarius"]
	if !ok {
		t.Fatalf("aquarius absent du résultat ; stats=%v", stats)
	}
	if got.Total != 2 {
		t.Errorf("aquarius Total: want 2 (m1+m2), got %d — squad partiel m3 a fui ?", got.Total)
	}
	if got.Wins != 2 {
		t.Errorf("aquarius Wins: want 2, got %d", got.Wins)
	}
	if got.PerfAvg == nil {
		t.Errorf("aquarius PerfAvg: want non-nil (m1=85.5, m2=75.5)")
	} else if v := *got.PerfAvg; v < 80.4 || v > 80.6 {
		t.Errorf("aquarius PerfAvg: want ~80.5, got %.2f", v)
	}
}

// TestSquadRepo_LoadMapStatsForSquad_NeverPlayedTogether : si le squad n'a
// jamais joué ensemble, le map retourné ne contient aucune entrée pour les
// cartes du main — fallback côté front : "—".
func TestSquadRepo_LoadMapStatsForSquad_NeverPlayedTogether(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewSquadRepo(pdb)
	stats, err := repo.LoadMapStatsForSquad(context.Background(), pTestXUID, []string{"xuid_unknown_42"}, nil)
	if err != nil {
		t.Fatalf("LoadMapStatsForSquad: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("want empty map, got %v", stats)
	}
}

// TestSquadRepo_LoadMapStatsForSquad_ExactCompositionExclusion : composition
// {mate1} avec un coéquipier connu HORS sélection (mate2) sur l'équipe du main.
// Setup 2 matchs map_id=aquarius :
//   - m2 : main + mate1 + mate2 sur la même team → mate2 est un "extra connu"
//   - m4 : main + mate1 seuls (mate2 absent) → composition EXACTE {mate1}
//
// Sans exclusion, {mate1} compte m2 ET m4 (Total=2). Avec excludeXUIDs={mate2},
// l'anti-join écarte m2 (mate2 sur l'équipe du main) → Total=1 (m4 seul).
func TestSquadRepo_LoadMapStatsForSquad_ExactCompositionExclusion(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	mate1, mate2 := "xuid_mate_001", "xuid_mate_002"

	// m2 : main + mate1 + mate2 (win), même team_id=1.
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_registry (match_id, map_id) VALUES ('m2', 'aquarius')`)
	for _, x := range []string{pTestXUID, mate1, mate2} {
		execOnSharedDBs(t, pdb, ctx,
			`INSERT INTO shared.match_participants (match_id,xuid,outcome,team_id) VALUES (?,?,?,?)`,
			"m2", x, 2, 1)
	}

	// m4 : main + mate1 seuls (win), team_id=1 — composition exacte {mate1}.
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_registry (match_id, map_id) VALUES ('m4', 'aquarius')`)
	for _, x := range []string{pTestXUID, mate1} {
		execOnSharedDBs(t, pdb, ctx,
			`INSERT INTO shared.match_participants (match_id,xuid,outcome,team_id) VALUES (?,?,?,?)`,
			"m4", x, 2, 1)
	}

	repo := NewSquadRepo(pdb)

	// Sans exclusion : {mate1} compte m2 + m4.
	statsNoExcl, err := repo.LoadMapStatsForSquad(ctx, pTestXUID, []string{mate1}, nil)
	if err != nil {
		t.Fatalf("LoadMapStatsForSquad (no excl): %v", err)
	}
	if got := statsNoExcl["aquarius"].Total; got != 2 {
		t.Errorf("sans exclusion, aquarius Total: want 2 (m2+m4), got %d", got)
	}

	// Avec exclusion de mate2 : m2 écarté (mate2 sur l'équipe du main) → m4 seul.
	statsExcl, err := repo.LoadMapStatsForSquad(ctx, pTestXUID, []string{mate1}, []string{mate2})
	if err != nil {
		t.Fatalf("LoadMapStatsForSquad (excl mate2): %v", err)
	}
	if got := statsExcl["aquarius"].Total; got != 1 {
		t.Errorf("avec exclusion mate2, aquarius Total: want 1 (m4 seul), got %d — anti-join a fui ?", got)
	}
}
