//go:build integration

package duckdb

import (
	"context"
	"testing"
)

// seedRelations crée le schéma minimal pour Q28RelationsTpl : match_participants
// (avec team_id/outcome/kda), match_registry (start_time_utc + start_time),
// killer_victim_pairs et la vue v_gamertag_lookup (root-level, contrat
// SharedReader sans préfixe shared.).
func seedRelations(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	for _, ddl := range []string{
		`CREATE TABLE match_participants (
			match_id VARCHAR, xuid VARCHAR, team_id INTEGER, outcome INTEGER, kda DOUBLE)`,
		`CREATE TABLE match_registry (
			match_id VARCHAR, start_time_utc TIMESTAMPTZ, start_time TIMESTAMP, pair_name VARCHAR, map_name VARCHAR, map_name_fr VARCHAR)`,
		`CREATE TABLE killer_victim_pairs (
			match_id VARCHAR, killer_xuid VARCHAR, victim_xuid VARCHAR, kill_count INTEGER)`,
		`CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR)`,
		`CREATE VIEW v_gamertag_lookup AS SELECT xuid, gamertag FROM xuid_aliases`,
	} {
		if _, err := db.Exec(ctx, ddl); err != nil {
			t.Fatalf("seedRelations DDL: %v\nSQL: %s", err, ddl)
		}
	}

	// Joueur courant = xuidMe.
	// m1 : me + Ally (team 0, WIN=2)
	// m2 : me + Ally (team 0, WIN=2)
	// m3 : me (team 0) vs Foe (team 1, my outcome LOSS=3)
	// m4 : me (team 0) vs Foe (team 1, my outcome LOSS=3)
	// Once : me + Once (un seul match commun → exclu par HAVING >= 2)
	for _, ins := range []string{
		`INSERT INTO match_participants VALUES
			('m1','xuidMe',0,2,1.5), ('m1','xuidAlly',0,2,2.0),
			('m2','xuidMe',0,2,1.5), ('m2','xuidAlly',0,2,3.0),
			('m3','xuidMe',0,3,0.8), ('m3','xuidFoe',1,2,2.5),
			('m4','xuidMe',0,3,0.8), ('m4','xuidFoe',1,2,1.5),
			('m5','xuidMe',0,2,1.0), ('m5','xuidOnce',0,2,1.0)`,
		// Heures UTC distinctes pour exercer le bucketing day-parts du heatmap :
		// m1=02h (Nuit), m2=09h (Matin), m3=19h (Soir), m4=20h (Soir), m5=14h.
		`INSERT INTO match_registry VALUES
			('m1', TIMESTAMPTZ '2026-01-10 02:00:00+00', NULL, 'Slayer', 'Aquarius', NULL),
			('m2', TIMESTAMPTZ '2026-02-10 09:00:00+00', NULL, 'Slayer', 'Aquarius', NULL),
			('m3', TIMESTAMPTZ '2026-03-10 19:00:00+00', NULL, 'Capture the Flag', 'Bazaar', NULL),
			('m4', TIMESTAMPTZ '2026-04-10 20:00:00+00', NULL, 'Capture the Flag', 'Bazaar', NULL),
			('m5', TIMESTAMPTZ '2026-05-10 14:00:00+00', NULL, 'Oddball', 'Live Fire', NULL)`,
		`INSERT INTO killer_victim_pairs VALUES
			('m3','xuidMe','xuidFoe',2),
			('m3','xuidFoe','xuidMe',6),
			('m4','xuidFoe','xuidMe',4)`,
		`INSERT INTO xuid_aliases VALUES
			('xuidMe','MePlayer'),
			('xuidAlly','AllyPlayer'),
			('xuidFoe','FoePlayer'),
			('xuidOnce','OncePlayer')`,
	} {
		if _, err := db.Exec(ctx, ins); err != nil {
			t.Fatalf("seedRelations INSERT: %v\nSQL: %s", err, ins)
		}
	}
}

func TestCareerRepo_GetRelations(t *testing.T) {
	db := openMemDB(t)
	seedRelations(t, db)

	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidMe", Gamertag: "MePlayer"}
	repo := NewCareerRepo(pdb)

	rows, err := repo.GetRelations(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetRelations: %v", err)
	}

	byGT := map[string]bool{}
	for _, r := range rows {
		byGT[r.Gamertag] = true
	}
	// OncePlayer (1 match commun) doit être exclu par HAVING >= 2.
	if byGT["OncePlayer"] {
		t.Fatal("OncePlayer should be excluded (only 1 common match)")
	}
	if !byGT["AllyPlayer"] || !byGT["FoePlayer"] {
		t.Fatalf("expected AllyPlayer + FoePlayer, got %v", byGT)
	}
	if len(rows) != 2 {
		t.Fatalf("relations len=%d want 2", len(rows))
	}

	var ally, foe *RelationRowFixture
	for i := range rows {
		switch rows[i].Gamertag {
		case "AllyPlayer":
			ally = toFixture(rows[i].TotalMatches, rows[i].TeammateCount, rows[i].EnemyCount,
				rows[i].TeammateWins, rows[i].EnemyWins, rows[i].KillsDealt, rows[i].DeathsSuffered)
			if rows[i].AvgKDAWith == nil || *rows[i].AvgKDAWith != 2.5 {
				t.Fatalf("AllyPlayer avg_kda_with=%v want 2.5", rows[i].AvgKDAWith)
			}
			if rows[i].FirstSeen.IsZero() {
				t.Fatal("AllyPlayer first_seen must be set")
			}
		case "FoePlayer":
			foe = toFixture(rows[i].TotalMatches, rows[i].TeammateCount, rows[i].EnemyCount,
				rows[i].TeammateWins, rows[i].EnemyWins, rows[i].KillsDealt, rows[i].DeathsSuffered)
			// kills dealt to foe = 2 (m3) ; deaths suffered = 6+4 = 10
			if rows[i].KillsDealt != 2 {
				t.Fatalf("FoePlayer kills_dealt=%d want 2", rows[i].KillsDealt)
			}
			if rows[i].DeathsSuffered != 10 {
				t.Fatalf("FoePlayer deaths_suffered=%d want 10", rows[i].DeathsSuffered)
			}
		}
	}
	if ally == nil || foe == nil {
		t.Fatal("missing ally/foe rows")
	}
	// AllyPlayer : 2 matchs alliés, 2 victoires alliées, 0 ennemi.
	if ally.total != 2 || ally.teammate != 2 || ally.teammateWins != 2 || ally.enemy != 0 {
		t.Fatalf("AllyPlayer agg=%+v", ally)
	}
	// FoePlayer : 2 matchs ennemis, 0 victoire ennemie (les 2 sont des LOSS).
	if foe.total != 2 || foe.enemy != 2 || foe.enemyWins != 0 || foe.teammate != 0 {
		t.Fatalf("FoePlayer agg=%+v", foe)
	}
}

func TestCareerRepo_GetRelations_Empty(t *testing.T) {
	db := openMemDB(t)
	seedRelations(t, db)

	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidUnknown"}
	repo := NewCareerRepo(pdb)

	rows, err := repo.GetRelations(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetRelations: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 relations for unknown player, got %d", len(rows))
	}
}

// ─── Phase 2 : scope match_id (segmentation serveur) ────────────────────────

// Scope vide (non-nil) → court-circuit : aucune relation, aucune requête.
func TestCareerRepo_GetRelations_EmptyScope(t *testing.T) {
	db := openMemDB(t)
	seedRelations(t, db)
	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidMe", Gamertag: "MePlayer"}
	repo := NewCareerRepo(pdb)

	rows, err := repo.GetRelations(context.Background(), []string{})
	if err != nil {
		t.Fatalf("GetRelations empty scope: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("empty scope must yield 0 relations, got %d", len(rows))
	}
}

// Scope = {m1,m2} (les 2 matchs alliés) → seul Ally émerge (Foe hors scope) ;
// kills/deaths restreints au scope (les frags m3/m4 vs Foe disparaissent).
func TestCareerRepo_GetRelations_ScopedToAllyMatches(t *testing.T) {
	db := openMemDB(t)
	seedRelations(t, db)
	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidMe", Gamertag: "MePlayer"}
	repo := NewCareerRepo(pdb)

	rows, err := repo.GetRelations(context.Background(), []string{"m1", "m2"})
	if err != nil {
		t.Fatalf("GetRelations scoped: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("scoped relations len=%d want 1 (Ally only)", len(rows))
	}
	if rows[0].Gamertag != "AllyPlayer" {
		t.Fatalf("scoped relation=%q want AllyPlayer", rows[0].Gamertag)
	}
	if rows[0].TeammateCount != 2 {
		t.Fatalf("Ally teammate count=%d want 2", rows[0].TeammateCount)
	}
}

// Scope = {m3,m4} (les 2 matchs ennemis) → seul Foe émerge ; les frags/morts
// échangés (m3+m4) sont conservés (scope inclut ces matchs).
func TestCareerRepo_GetRelations_ScopedToFoeMatches(t *testing.T) {
	db := openMemDB(t)
	seedRelations(t, db)
	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidMe", Gamertag: "MePlayer"}
	repo := NewCareerRepo(pdb)

	rows, err := repo.GetRelations(context.Background(), []string{"m3", "m4"})
	if err != nil {
		t.Fatalf("GetRelations scoped foe: %v", err)
	}
	if len(rows) != 1 || rows[0].Gamertag != "FoePlayer" {
		t.Fatalf("scoped foe rows=%+v want [FoePlayer]", rows)
	}
	// kills_dealt = 2 (m3), deaths_suffered = 6 (m3) + 4 (m4) = 10.
	if rows[0].KillsDealt != 2 || rows[0].DeathsSuffered != 10 {
		t.Fatalf("Foe duel kills=%d deaths=%d want 2/10", rows[0].KillsDealt, rows[0].DeathsSuffered)
	}
}

// Scope qui restreint à un seul match commun avec Ally → tombe sous HAVING>=2
// → Ally disparaît (relation non significative sur ce scope).
func TestCareerRepo_GetRelations_ScopeDropsBelowHaving(t *testing.T) {
	db := openMemDB(t)
	seedRelations(t, db)
	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidMe", Gamertag: "MePlayer"}
	repo := NewCareerRepo(pdb)

	rows, err := repo.GetRelations(context.Background(), []string{"m1"})
	if err != nil {
		t.Fatalf("GetRelations scope single: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("single-match scope must drop all (HAVING>=2), got %d", len(rows))
	}
}

// ─── Phase 3a : Moments & Rivalités ─────────────────────────────────────────

// GetRelationsHeatmap : top-N relations × heure. Ally apparaît à 02h (m1) et
// 09h (m2) ; Foe à 19h (m3) et 20h (m4). Once exclu (1 match < HAVING 2).
func TestCareerRepo_GetRelationsHeatmap(t *testing.T) {
	db := openMemDB(t)
	seedRelations(t, db)
	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidMe", Gamertag: "MePlayer"}
	repo := NewCareerRepo(pdb)

	rows, err := repo.GetRelationsHeatmap(context.Background(), nil, 8)
	if err != nil {
		t.Fatalf("GetRelationsHeatmap: %v", err)
	}

	type cell struct {
		gt   string
		hour int
	}
	got := map[cell]int{}
	for _, r := range rows {
		got[cell{r.Gamertag, r.Hour}] = r.Count
	}
	if got[cell{"AllyPlayer", 2}] != 1 || got[cell{"AllyPlayer", 9}] != 1 {
		t.Fatalf("Ally hours wrong: %v", got)
	}
	if got[cell{"FoePlayer", 19}] != 1 || got[cell{"FoePlayer", 20}] != 1 {
		t.Fatalf("Foe hours wrong: %v", got)
	}
	// Once (1 match) absent.
	for c := range got {
		if c.gt == "OncePlayer" {
			t.Fatalf("OncePlayer should be excluded (HAVING>=2)")
		}
	}
}

// GetRelationsHeatmap : topN=1 ne garde que la relation la plus fréquente.
// Ally et Foe ont chacun 2 matchs → tiebreak xuid ASC → "xuidAlly" < "xuidFoe".
func TestCareerRepo_GetRelationsHeatmap_TopN(t *testing.T) {
	db := openMemDB(t)
	seedRelations(t, db)
	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidMe", Gamertag: "MePlayer"}
	repo := NewCareerRepo(pdb)

	rows, err := repo.GetRelationsHeatmap(context.Background(), nil, 1)
	if err != nil {
		t.Fatalf("GetRelationsHeatmap topN=1: %v", err)
	}
	for _, r := range rows {
		if r.Gamertag != "AllyPlayer" {
			t.Fatalf("topN=1 should keep only AllyPlayer, got %q", r.Gamertag)
		}
	}
	if len(rows) == 0 {
		t.Fatal("expected at least one cell for AllyPlayer")
	}
}

// GetRivalTimeline : Foe est un ennemi sur m3 (LOSS, me kills 2 / lui 6) et m4
// (LOSS, lui 4). Ordre ancien→récent (m3 puis m4).
func TestCareerRepo_GetRivalTimeline(t *testing.T) {
	db := openMemDB(t)
	seedRelations(t, db)
	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidMe", Gamertag: "MePlayer"}
	repo := NewCareerRepo(pdb)

	rows, err := repo.GetRivalTimeline(context.Background(), "xuidFoe", nil, 20)
	if err != nil {
		t.Fatalf("GetRivalTimeline: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("timeline len=%d want 2 (%+v)", len(rows), rows)
	}
	// m3 (mars) avant m4 (avril).
	if rows[0].MatchID != "m3" || rows[1].MatchID != "m4" {
		t.Fatalf("order=%s,%s want m3,m4", rows[0].MatchID, rows[1].MatchID)
	}
	// my outcome LOSS → result code 2.
	if rows[0].Result != 2 || rows[1].Result != 2 {
		t.Fatalf("results=%d,%d want 2,2", rows[0].Result, rows[1].Result)
	}
	// m3 : me→Foe 2, Foe→me 6.
	if rows[0].KillsOnRival != 2 || rows[0].DeathsByRival != 6 {
		t.Fatalf("m3 frags kills=%d deaths=%d want 2/6", rows[0].KillsOnRival, rows[0].DeathsByRival)
	}
	// m4 : me→Foe 0, Foe→me 4.
	if rows[1].KillsOnRival != 0 || rows[1].DeathsByRival != 4 {
		t.Fatalf("m4 frags kills=%d deaths=%d want 0/4", rows[1].KillsOnRival, rows[1].DeathsByRival)
	}
	// mode/map résolus depuis match_registry (pair_name + map_name).
	if rows[0].Mode != "Capture the Flag" || rows[0].MapName != "Bazaar" {
		t.Fatalf("m3 mode/map = %q/%q want \"Capture the Flag\"/\"Bazaar\"", rows[0].Mode, rows[0].MapName)
	}
}

// GetRivalTimeline : un allié (toujours même équipe) n'a aucun duel ennemi.
func TestCareerRepo_GetRivalTimeline_AllyNoDuel(t *testing.T) {
	db := openMemDB(t)
	seedRelations(t, db)
	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidMe", Gamertag: "MePlayer"}
	repo := NewCareerRepo(pdb)

	rows, err := repo.GetRivalTimeline(context.Background(), "xuidAlly", nil, 20)
	if err != nil {
		t.Fatalf("GetRivalTimeline ally: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("ally timeline len=%d want 0 (same team always)", len(rows))
	}
}

// GetRivalTimeline : scope restreint à m4 → un seul duel.
func TestCareerRepo_GetRivalTimeline_Scoped(t *testing.T) {
	db := openMemDB(t)
	seedRelations(t, db)
	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidMe", Gamertag: "MePlayer"}
	repo := NewCareerRepo(pdb)

	rows, err := repo.GetRivalTimeline(context.Background(), "xuidFoe", []string{"m4"}, 20)
	if err != nil {
		t.Fatalf("GetRivalTimeline scoped: %v", err)
	}
	if len(rows) != 1 || rows[0].MatchID != "m4" {
		t.Fatalf("scoped timeline=%+v want [m4]", rows)
	}
	if rows[0].DeathsByRival != 4 || rows[0].KillsOnRival != 0 {
		t.Fatalf("m4 scoped frags kills=%d deaths=%d want 0/4", rows[0].KillsOnRival, rows[0].DeathsByRival)
	}
}

// GetRivalTimeline : scope vide (non-nil) → court-circuit.
func TestCareerRepo_GetRivalTimeline_EmptyScope(t *testing.T) {
	db := openMemDB(t)
	seedRelations(t, db)
	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidMe", Gamertag: "MePlayer"}
	repo := NewCareerRepo(pdb)

	rows, err := repo.GetRivalTimeline(context.Background(), "xuidFoe", []string{}, 20)
	if err != nil {
		t.Fatalf("GetRivalTimeline empty scope: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("empty scope must yield 0 duels, got %d", len(rows))
	}
}

// GetCoreEngagement : WR HISTORIQUE (lift) + frise forme récente jouée À CÔTÉ
// d'un membre. me = 3 victoires (m1,m2,m5) / 2 défaites (m3,m4) → WR historique
// 0.6 (constant, non scopé). Avec xuidAlly : coéquipier sur m1,m2 (2 WIN) →
// frise ["win","win"].
func TestCareerRepo_GetCoreEngagement(t *testing.T) {
	db := openMemDB(t)
	seedRelations(t, db)
	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidMe", Gamertag: "MePlayer"}
	repo := NewCareerRepo(pdb)
	ctx := context.Background()

	eng, err := repo.GetCoreEngagement(ctx, []string{"xuidAlly"}, nil, 25)
	if err != nil {
		t.Fatalf("GetCoreEngagement: %v", err)
	}
	if eng.PlayerWinRate == nil || *eng.PlayerWinRate < 0.59 || *eng.PlayerWinRate > 0.61 {
		t.Fatalf("PlayerWinRate=%v want ~0.6", eng.PlayerWinRate)
	}
	if len(eng.RecentForm) != 2 || eng.RecentForm[0] != "win" || eng.RecentForm[1] != "win" {
		t.Fatalf("RecentForm=%v want [win win]", eng.RecentForm)
	}

	// Un rival (toujours équipe adverse) n'est jamais coéquipier → frise vide.
	// WR historique inchangé.
	engFoe, err := repo.GetCoreEngagement(ctx, []string{"xuidFoe"}, nil, 25)
	if err != nil {
		t.Fatalf("GetCoreEngagement foe: %v", err)
	}
	if len(engFoe.RecentForm) != 0 {
		t.Fatalf("RecentForm (foe as core)=%v want empty", engFoe.RecentForm)
	}
	if engFoe.PlayerWinRate == nil {
		t.Fatal("PlayerWinRate must still be computed with non-teammate core")
	}

	// Scope = {m1} : 1 match avec l'allié → frise ["win"]. Le WR reste HISTORIQUE
	// (0.6), pas scopé.
	engScoped, err := repo.GetCoreEngagement(ctx, []string{"xuidAlly"}, []string{"m1"}, 25)
	if err != nil {
		t.Fatalf("GetCoreEngagement scoped: %v", err)
	}
	if engScoped.PlayerWinRate == nil || *engScoped.PlayerWinRate < 0.59 || *engScoped.PlayerWinRate > 0.61 {
		t.Fatalf("scoped PlayerWinRate=%v want ~0.6 (historique, non scopé)", engScoped.PlayerWinRate)
	}
	if len(engScoped.RecentForm) != 1 || engScoped.RecentForm[0] != "win" {
		t.Fatalf("scoped RecentForm=%v want [win]", engScoped.RecentForm)
	}

	// Scope vide (non-nil) → court-circuit : aucun agrégat.
	engEmpty, err := repo.GetCoreEngagement(ctx, []string{"xuidAlly"}, []string{}, 25)
	if err != nil {
		t.Fatalf("GetCoreEngagement empty scope: %v", err)
	}
	if engEmpty.PlayerWinRate != nil || len(engEmpty.RecentForm) != 0 {
		t.Fatalf("empty scope must yield zero engagement, got %+v", engEmpty)
	}
}

// GetRelationRecentForm : forme récente jouée à côté d'un seul allié (binôme).
// xuidAlly coéquipier sur m1,m2 (2 WIN) → ["win","win"] ; rival jamais coéquipier
// → vide ; xuid vide → nil.
func TestCareerRepo_GetRelationRecentForm(t *testing.T) {
	db := openMemDB(t)
	seedRelations(t, db)
	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidMe", Gamertag: "MePlayer"}
	repo := NewCareerRepo(pdb)
	ctx := context.Background()

	form, err := repo.GetRelationRecentForm(ctx, "xuidAlly", nil, 25)
	if err != nil {
		t.Fatalf("GetRelationRecentForm: %v", err)
	}
	if len(form) != 2 || form[0] != "win" || form[1] != "win" {
		t.Fatalf("form=%v want [win win]", form)
	}

	foe, err := repo.GetRelationRecentForm(ctx, "xuidFoe", nil, 25)
	if err != nil {
		t.Fatalf("GetRelationRecentForm foe: %v", err)
	}
	if len(foe) != 0 {
		t.Fatalf("foe form=%v want empty (jamais coéquipier)", foe)
	}

	empty, err := repo.GetRelationRecentForm(ctx, "", nil, 25)
	if err != nil || empty != nil {
		t.Fatalf("empty xuid: form=%v err=%v want nil/nil", empty, err)
	}
}

// GetRelationEnemyRecentForm : miroir ennemi — forme récente jouée CONTRE un
// adversaire (bête noire). xuidFoe adverse sur m3,m4 (my outcome LOSS), ordonné
// ancien→récent → ["loss","loss"] ; l'allié (jamais adverse) → vide ; xuid vide → nil.
func TestCareerRepo_GetRelationEnemyRecentForm(t *testing.T) {
	db := openMemDB(t)
	seedRelations(t, db)
	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidMe", Gamertag: "MePlayer"}
	repo := NewCareerRepo(pdb)
	ctx := context.Background()

	form, err := repo.GetRelationEnemyRecentForm(ctx, "xuidFoe", nil, 25)
	if err != nil {
		t.Fatalf("GetRelationEnemyRecentForm: %v", err)
	}
	if len(form) != 2 || form[0] != "loss" || form[1] != "loss" {
		t.Fatalf("enemy form=%v want [loss loss]", form)
	}

	ally, err := repo.GetRelationEnemyRecentForm(ctx, "xuidAlly", nil, 25)
	if err != nil {
		t.Fatalf("GetRelationEnemyRecentForm ally: %v", err)
	}
	if len(ally) != 0 {
		t.Fatalf("ally enemy form=%v want empty (jamais adverse)", ally)
	}

	empty, err := repo.GetRelationEnemyRecentForm(ctx, "", nil, 25)
	if err != nil || empty != nil {
		t.Fatalf("empty xuid: form=%v err=%v want nil/nil", empty, err)
	}
}

// RelationRowFixture : agrégats clés pour assertions concises.
type RelationRowFixture struct {
	total, teammate, enemy, teammateWins, enemyWins, kills, deaths int
}

func toFixture(total, teammate, enemy, teammateWins, enemyWins, kills, deaths int) *RelationRowFixture {
	return &RelationRowFixture{total, teammate, enemy, teammateWins, enemyWins, kills, deaths}
}
