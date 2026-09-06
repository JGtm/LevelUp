//go:build integration

package replayartifacts

// positions_integration_test.go — L'EQUIPE DES POSITIONS PROJETEES, LUE EN BASE POUR DE VRAI
// (constat C5 de la revue A-R1).
//
// POURQUOI UN TEST D'INTEGRATION. La jointure passe par `port.ReplayFactsRepo`, dont la requete
// JOINT `match_participants` a `match_registry` et coalesce `team_id`. Un test a doubles
// prouverait la boucle `poserEquipes` (elle a le sien, unitaire) mais pas que la LECTURE rend
// quelque chose : une colonne renommee, un JOIN qui ne matche pas, et toutes les positions
// repartiraient a -1 — c'est-a-dire au defaut que ce constat ferme, en silence.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/replay"
)

// artefactDeuxCamps ecrit un artefact dont DEUX vies portent des xuids differents, aucune
// equipe (le film ne la porte pas), plus une vie anonyme qui doit rester non situee.
func artefactDeuxCamps(t *testing.T, dir, matchID string) string {
	t.Helper()
	doc := replay.ReplayDocument{
		SchemaVersion:   replay.SchemaVersion,
		MatchID:         matchID,
		FrameIntervalMS: 100,
		Tracks: []replay.Track{
			{Slot: 1, XUID: "111", Team: EquipeInconnue, Points: []replay.Point{{T: 0, X: 1, Y: 1}}},
			{Slot: 2, XUID: "222", Team: EquipeInconnue, Points: []replay.Point{{T: 0, X: 2, Y: 2}}},
			{Slot: 3, XUID: "", Team: EquipeInconnue, Points: []replay.Point{{T: 0, X: 3, Y: 3}}},
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal artefact: %v", err)
	}
	path := filepath.Join(dir, matchID+".json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write artefact: %v", err)
	}
	return path
}

// TestPersisterPositions_EquipeJointeDepuisLaBase — deux slots, deux camps : les lignes
// projetees ressortent a 0 et a 1, et la vie anonyme reste non situee.
//
// SANS CETTE JOINTURE, toutes les positions porteraient -1 et le filtre Global / Equipe A /
// Equipe B de la carte de chaleur (`MatchPositionsHeatmap.tsx`, qui ne s'affiche que si au
// moins une position porte une equipe) serait du code mort pour toute donnee projetee.
func TestPersisterPositions_EquipeJointeDepuisLaBase(t *testing.T) {
	db := baseRegistre(t)
	dir := t.TempDir()
	ctx := context.Background()
	const matchID = "mpositions1"

	inscrireAuRegistre(t, db, matchID, time.Now().UTC().Add(-time.Hour), 0)
	for _, p := range []struct {
		xuid string
		team int
	}{{"111", 0}, {"222", 1}} {
		if _, err := db.ExecContext(ctx, `INSERT INTO match_participants
			(match_id, xuid, team_id, kills, deaths, assists) VALUES (?, ?, ?, 0, 0, 0)`,
			matchID, p.xuid, p.team); err != nil {
			t.Fatalf("insert participant %s: %v", p.xuid, err)
		}
	}

	acquis, relaches := 0, 0
	d := depsUsage(t, db, "halo_infinite", &acquis, &relaches)
	persisterPositions(ctx, d, &bilanDerivations{}, lus(d,
		ArtefactRange{MatchID: matchID, Path: artefactDeuxCamps(t, dir, matchID)},
	))

	rows, err := db.QueryContext(ctx, `SELECT team, COUNT(*) FROM match_player_positions_latest
		WHERE match_id = ? GROUP BY team ORDER BY team`, matchID)
	if err != nil {
		t.Fatalf("lecture _latest: %v", err)
	}
	defer func() { _ = rows.Close() }()
	parEquipe := map[int]int{}
	for rows.Next() {
		var team, n int
		if err := rows.Scan(&team, &n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		parEquipe[team] = n
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	attendu := map[int]int{0: 1, 1: 1, EquipeInconnue: 1}
	if len(parEquipe) != len(attendu) {
		t.Fatalf("repartition = %v, attendue %v — l'equipe n'est pas jointe depuis la base "+
			"(constat C5)", parEquipe, attendu)
	}
	for team, n := range attendu {
		if parEquipe[team] != n {
			t.Errorf("equipe %d : %d ligne(s), attendu %d", team, parEquipe[team], n)
		}
	}
}
