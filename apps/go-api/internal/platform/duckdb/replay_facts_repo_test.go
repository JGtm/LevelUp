//go:build integration

// replay_facts_repo_test.go — les faits d'un match, tels que le constructeur d'artefact de rejeu
// a besoin de les connaitre.
//
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -run TestReplayFacts

package duckdb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// seedReplayFacts cree le schema minimal (les seules colonnes que le repo lit).
func seedReplayFacts(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open :memory: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(context.Background(), `
		CREATE TABLE match_registry (
			match_id VARCHAR, team_0_score INTEGER, team_1_score INTEGER, game_variant_name VARCHAR);
		CREATE TABLE match_participants (
			match_id VARCHAR, xuid VARCHAR, kills INTEGER, deaths INTEGER, assists INTEGER, team_id INTEGER);
	`); err != nil {
		t.Fatalf("ddl: %v", err)
	}
	return db
}

// TestReplayFactsForMatch — le cas nominal : les deux scores, la variante, et les lignes de
// match avec leur camp.
func TestReplayFactsForMatch(t *testing.T) {
	db := seedReplayFacts(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO match_registry VALUES ('m1', 3, 0, 'CTF:Arena')`); err != nil {
		t.Fatalf("insert registry: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO match_participants VALUES
		('m1', '2533274', 12, 7, 3, 0), ('m1', '2533275', 9, 11, 5, 1)`); err != nil {
		t.Fatalf("insert participants: %v", err)
	}

	got, err := NewReplayFactsRepo(db).FactsForMatch(ctx, "m1")
	if err != nil {
		t.Fatalf("FactsForMatch: %v", err)
	}
	if got.GameVariantName != "CTF:Arena" {
		t.Errorf("variante = %q, attendu \"CTF:Arena\"", got.GameVariantName)
	}
	if got.TeamScores == nil || *got.TeamScores != [2]int{3, 0} {
		t.Fatalf("scores = %v, attendu [3 0]", got.TeamScores)
	}
	if len(got.Players) != 2 {
		t.Fatalf("%d ligne(s) de match, attendu 2", len(got.Players))
	}
	p := got.Players[0]
	if p.XUID != "2533274" || p.Kills != 12 || p.Deaths != 7 || p.Assists != 3 || p.TeamID != 0 {
		t.Errorf("premiere ligne inattendue : %+v", p)
	}
	if got.Empty() {
		t.Error("des faits complets se declarent vides")
	}
}

// TestReplayFactsUnknownMatchIsNotAnError — UN MATCH ABSENT DU REGISTRE N'EST PAS UNE ERREUR.
//
// Le cas est reel (un film du cache dont le match n'a jamais ete synchronise) : il doit degrader
// l'artefact, pas faire echouer la passe de construction.
func TestReplayFactsUnknownMatchIsNotAnError(t *testing.T) {
	got, err := NewReplayFactsRepo(seedReplayFacts(t)).FactsForMatch(context.Background(), "inconnu")
	if err != nil {
		t.Fatalf("un match inconnu doit degrader, pas echouer : %v", err)
	}
	if !got.Empty() {
		t.Errorf("faits non vides pour un match inconnu : %+v", got)
	}
}

// TestReplayFactsNullScoresAreAbsent — LES DEUX SCORES OU AUCUN.
//
// Un seul des deux ne permet aucune comparaison, et publier l'autre a zero inventerait un score
// que la base ne porte pas.
func TestReplayFactsNullScoresAreAbsent(t *testing.T) {
	db := seedReplayFacts(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO match_registry VALUES ('m2', 5, NULL, 'Slayer:Arena')`); err != nil {
		t.Fatalf("insert registry: %v", err)
	}
	got, err := NewReplayFactsRepo(db).FactsForMatch(ctx, "m2")
	if err != nil {
		t.Fatalf("FactsForMatch: %v", err)
	}
	if got.TeamScores != nil {
		t.Errorf("scores publies alors qu'un seul est connu : %v", got.TeamScores)
	}
	if got.GameVariantName != "Slayer:Arena" {
		t.Errorf("variante = %q", got.GameVariantName)
	}
}

// TestReplayFactsNullTeamIsMinusOne — un camp absent vaut -1 et non 0 : zero EST un camp.
func TestReplayFactsNullTeamIsMinusOne(t *testing.T) {
	db := seedReplayFacts(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `INSERT INTO match_registry VALUES ('m3', 50, 43, 'Slayer:Arena')`); err != nil {
		t.Fatalf("insert registry: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO match_participants VALUES ('m3', '2533276', 4, 5, 6, NULL)`); err != nil {
		t.Fatalf("insert participants: %v", err)
	}
	got, err := NewReplayFactsRepo(db).FactsForMatch(ctx, "m3")
	if err != nil {
		t.Fatalf("FactsForMatch: %v", err)
	}
	if len(got.Players) != 1 || got.Players[0].TeamID != -1 {
		t.Fatalf("camp inconnu mal rendu : %+v", got.Players)
	}
}
