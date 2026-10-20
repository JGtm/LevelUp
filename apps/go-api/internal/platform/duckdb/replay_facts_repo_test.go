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

// seedReplayFacts cree le schema minimal : les seules colonnes que le repo lit — dont `map_id`,
// la cle du catalogue d objectifs lue par `registryFacts` depuis b0fb3e10f, et les quatre champs
// de PRESENCE ajoutes par le lot sieges (2026-09-02).
//
// CETTE DDL EST UNE COPIE, ET UNE COPIE DERIVE. Elle l a fait : le lot sieges a ajoute
// `joined_in_progress` a la requete de `playerFacts` sans toucher a ce fichier, et les quatre
// tests sont partis en « Binder Error: Table p does not have a column named joined_in_progress »
// — rouge en CI pendant une journee. Deux consequences tirees ici :
//
//   - les INSERT NOMMENT leurs colonnes. Une DDL positionnelle casse tous les cas des qu une
//     colonne s ajoute, ce qui transforme un ajout de champ en chantier de test ;
//   - toute colonne lue par `replay_facts_repo.go` doit apparaitre ci-dessous. C est la seule
//     regle a tenir quand la requete de production bouge.
func seedReplayFacts(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open :memory: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(context.Background(), `
		CREATE TABLE match_registry (
			match_id VARCHAR, team_0_score INTEGER, team_1_score INTEGER, game_variant_name VARCHAR,
			map_id VARCHAR,
			-- Les deux faces du temps canonique (regle n 8) : SQLStartTimeCanonical lit la
			-- premiere et retombe sur la seconde.
			start_time_utc TIMESTAMP WITH TIME ZONE, start_time TIMESTAMP);
		CREATE TABLE match_participants (
			match_id VARCHAR, xuid VARCHAR, kills INTEGER, deaths INTEGER, assists INTEGER, team_id INTEGER,
			-- Presence (lot sieges) : les deux drapeaux et les deux instants dont playerFacts
			-- derive JoinMatchMS / LeaveMatchMS.
			joined_in_progress BOOLEAN, left_in_progress BOOLEAN,
			first_joined_time TIMESTAMP, last_leave_time TIMESTAMP);
	`); err != nil {
		t.Fatalf("ddl: %v", err)
	}
	return db
}

// TestReplayFactsForMatch — le cas nominal : les deux scores, la variante, la carte (blanchie), et
// les lignes de match avec leur camp.
func TestReplayFactsForMatch(t *testing.T) {
	db := seedReplayFacts(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO match_registry (match_id, team_0_score, team_1_score, game_variant_name, map_id, start_time_utc)
		 VALUES ('m1', 3, 0, 'CTF:Arena', ' asset-m1 ', TIMESTAMPTZ '2026-09-01 20:00:00+00')`); err != nil {
		t.Fatalf("insert registry: %v", err)
	}
	// Le second joueur ENTRE EN COURS, 90 s apres le debut, et ne sort pas : c est le cas que
	// le lot sieges a ajoute, et il doit remonter jusqu a JoinMatchMS.
	if _, err := db.ExecContext(ctx, `INSERT INTO match_participants
		(match_id, xuid, kills, deaths, assists, team_id, joined_in_progress, left_in_progress, first_joined_time)
		VALUES
		('m1', '2533274', 12, 7, 3, 0, FALSE, FALSE, NULL),
		('m1', '2533275', 9, 11, 5, 1, TRUE, FALSE, TIMESTAMP '2026-09-01 20:01:30')`); err != nil {
		t.Fatalf("insert participants: %v", err)
	}

	got, err := NewReplayFactsRepo(db).FactsForMatch(ctx, "m1")
	if err != nil {
		t.Fatalf("FactsForMatch: %v", err)
	}
	if got.GameVariantName != "CTF:Arena" {
		t.Errorf("variante = %q, attendu \"CTF:Arena\"", got.GameVariantName)
	}
	if got.MapID != "asset-m1" {
		t.Errorf("carte = %q, attendu \"asset-m1\" (map_id blanchi)", got.MapID)
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
	// LA PRESENCE, ET C EST CE QUI MANQUAIT : le premier joueur est la au coup d envoi, le
	// second entre a 90 s. Sans ces deux assertions, la DDL de test pourrait reperdre les
	// colonnes du lot sieges sans que rien ne le dise.
	if p.JoinedInProgress || p.JoinMatchMS != nil {
		t.Errorf("joueur present au depart : entree en cours=%t, instant=%v", p.JoinedInProgress, p.JoinMatchMS)
	}
	tard := got.Players[1]
	if !tard.JoinedInProgress {
		t.Error("le second joueur entre en cours de partie et n est pas signale comme tel")
	}
	if tard.JoinMatchMS == nil || *tard.JoinMatchMS != 90_000 {
		t.Errorf("instant d entree = %v, attendu 90000 ms", tard.JoinMatchMS)
	}
	if tard.LeftInProgress || tard.LeaveMatchMS != nil {
		t.Errorf("sortie inventee : en cours=%t, instant=%v", tard.LeftInProgress, tard.LeaveMatchMS)
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
		`INSERT INTO match_registry (match_id, team_0_score, team_1_score, game_variant_name, map_id)
		 VALUES ('m2', 5, NULL, 'Slayer:Arena', NULL)`); err != nil {
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
	if got.MapID != "" {
		t.Errorf("carte = %q pour un map_id NULL, attendu vide", got.MapID)
	}
}

// TestReplayFactsNullTeamIsMinusOne — un camp absent vaut -1 et non 0 : zero EST un camp.
func TestReplayFactsNullTeamIsMinusOne(t *testing.T) {
	db := seedReplayFacts(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO match_registry (match_id, team_0_score, team_1_score, game_variant_name, map_id)
		 VALUES ('m3', 50, 43, 'Slayer:Arena', 'asset-m3')`); err != nil {
		t.Fatalf("insert registry: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO match_participants (match_id, xuid, kills, deaths, assists, team_id)
		 VALUES ('m3', '2533276', 4, 5, 6, NULL)`); err != nil {
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
