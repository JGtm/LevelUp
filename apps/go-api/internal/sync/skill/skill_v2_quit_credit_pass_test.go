//go:build integration

package skill

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/domain"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/persist"
)

// TestLoadQuitTimeline_CreditOnlyPass : la pénalité de départ du LUSR v2 lit
// match_kill_events_latest. Au premier écrit d'un match, SEULE la passe crédit-seul existe
// (persistKillerVictim l'écrit avec killer_victim_pairs ; aucune passe film, qui n'arrive
// qu'à l'étape 1.57 du post-sync). Garde de l'ordre du post-sync : le LUSR peut tourner avant
// 1.57 parce que cette passe suffit — la chronologie est lue (available=true) et le delta suit
// la situation au moment du départ, pas le repli sur l'issue finale.
func TestLoadQuitTimeline_CreditOnlyPass(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	// Provider de steps title-owned (le schéma de base de shared en fait partie), comme au boot.
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migrate shared: %v", err)
	}
	for _, col := range []string{"first_joined_time TIMESTAMPTZ", "last_leave_time TIMESTAMPTZ"} {
		_, _ = db.Exec("ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS " + col)
	}

	const matchID = "quit-credit-1"
	start := time.Date(2026, 1, 10, 20, 0, 0, 0, time.UTC)
	team := func(v int) *int { return &v }
	names := map[string]string{"a1": "A1", "a2": "A2", "b1": "B1", "b2": "B2"}

	b := persist.NewBatchBuilder("halo_infinite", "A1", "a1", "test")
	b.SetMatch(&domain.MatchRegistryRow{MatchID: matchID, StartTime: start})
	var parts []domain.MatchParticipantRow
	for x, tm := range map[string]int{"a1": 0, "a2": 0, "b1": 1, "b2": 1} {
		gt := names[x]
		parts = append(parts, domain.MatchParticipantRow{MatchID: matchID, XUID: x, Gamertag: &gt, TeamID: team(tm)})
	}
	b.AddParticipants(parts)
	// L'équipe A mène 4-0 jusqu'à 40 s, puis l'équipe B remonte (6 frags) et gagne.
	var pairs []persist.KillerVictimInsert
	add := func(killer, victim string, sec int64) {
		pairs = append(pairs, persist.KillerVictimInsert{
			MatchID: matchID, KillerXUID: killer, KillerGamertag: names[killer],
			VictimXUID: victim, VictimGamertag: names[victim], Count: 1, TimeMS: sec * 1000,
		})
	}
	for _, s := range []int64{10, 20, 30, 40} {
		add("a1", "b1", s)
	}
	for _, s := range []int64{50, 60, 70, 80, 90, 100} {
		add("b1", "a1", s)
	}
	b.AddKillerVictim(pairs)
	if err := persist.NewSharedPersister(db).Persist(context.Background(), b.Build()); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	// Aucune passe film: la seule passe en base est la passe crédit-seul.
	var passes int
	if err := db.QueryRow(`SELECT COUNT(DISTINCT decode_pass) FROM match_kill_events WHERE match_id = ?`, matchID).Scan(&passes); err != nil {
		t.Fatalf("passes: %v", err)
	}
	if passes != 1 {
		t.Fatalf("passes en base = %d, want 1 (crédit-seul seule)", passes)
	}

	teamA := []rosterMember{{xuid: "a1"}, {xuid: "a2", lastLeaveTime: sql.NullTime{Time: start.Add(45 * time.Second), Valid: true}}}
	teamB := []rosterMember{{xuid: "b1"}, {xuid: "b2"}}
	qt := loadQuitTimeline(context.Background(), db, matchID, start, teamA, teamB)
	if !qt.available {
		t.Fatal("loadQuitTimeline: available=false avec la seule passe crédit-seul — le LUSR ne peut pas précéder 1.57")
	}
	if len(qt.frags) != 10 {
		t.Errorf("frags de la timeline = %d, want 10", len(qt.frags))
	}

	// A2 part à 45 s alors que son équipe mène 4-0 ; elle perd pourtant le match. Sur la timeline
	// le départ est « sans lien » (delta fort) ; le repli sur l'issue finale (défaite) rendrait
	// « lié » (delta modéré).
	got := quitBaseDelta(context.Background(), teamA[1], skillv2.TeamLoss, 0, qt)
	if got != skillv2.DefaultQuitDeltaUnrelated {
		t.Errorf("delta = %v, want %v (suit la timeline: départ en tête)", got, skillv2.DefaultQuitDeltaUnrelated)
	}
	if fallback := quitDeltaForTeam(skillv2.TeamLoss); got == fallback {
		t.Errorf("delta = repli sur l'issue finale (%v): la timeline n'a pas été utilisée", fallback)
	}
}
