//go:build integration

package replayartifacts

// derivations_writer_integration_test.go — LE SEGMENT UNIQUE, SUR UNE VRAIE BASE
// (constat C7 de la revue A-R1).
//
// POURQUOI EN PLUS DU TEST UNITAIRE. Le test unitaire compte les acquisitions sur une source
// EN ERREUR : il prouve qu'on n'appelle la source qu'une fois, pas que les quatre familles
// ecrivent bien toutes dans le MEME segment et qu'il est relache UNE fois. Un `release`
// appele par la premiere famille relacherait le lease sous les pieds des suivantes ; un
// `release` jamais appele le tiendrait jusqu'a la fin du process. Les deux sont invisibles
// sans base reelle.

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/replaybuild"
)

// TestDeriver_UnSegmentAcquisEtRelacheUneFois : un artefact qui donne du travail au report du
// coup d'envoi, au resume d'usage ET aux positions — une acquisition, un retrait, et les
// ecritures des trois familles sont en base.
func TestDeriver_UnSegmentAcquisEtRelacheUneFois(t *testing.T) {
	db := baseRegistre(t)
	dir := t.TempDir()
	ctx := context.Background()
	const matchID = "mwriterseg"

	inscrireAuRegistre(t, db, matchID, time.Now().UTC().Add(-time.Hour), 0)
	acquis, relaches := 0, 0
	d := depsUsage(t, db, "halo_infinite", &acquis, &relaches)
	chemin := artefactADeriver(t, dir, matchID)

	Deriver(ctx, DerivationsDeps{
		RepoRoot: d.RepoRoot, TitleSlug: d.TitleSlug, Gamertag: d.Gamertag,
		AcquireWriter: d.AcquireWriter,
	}, []ArtefactRange{{MatchID: matchID, Path: chemin}})

	if acquis != 1 || relaches != 1 {
		t.Fatalf("writer acquis %d fois et relache %d fois pour UNE passe, attendu 1 et 1 "+
			"(constat C7)", acquis, relaches)
	}
	// Les trois familles ont bien ecrit DANS ce segment — un `release` premature les aurait
	// fait echouer en silence.
	var n int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_usage_films_latest WHERE match_id = ?`, matchID).Scan(&n); err != nil || n != 1 {
		t.Errorf("resume d'usage : %d ligne(s) (err=%v), attendu 1", n, err)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_player_positions_latest WHERE match_id = ?`, matchID).Scan(&n); err != nil || n != 1 {
		t.Errorf("positions : %d ligne(s) (err=%v), attendu 1", n, err)
	}
	var qualite string
	if err := db.QueryRowContext(ctx,
		`SELECT COALESCE(t0_quality, '') FROM match_registry WHERE match_id = ?`, matchID).Scan(&qualite); err != nil {
		t.Fatalf("lecture du T0 : %v", err)
	}
	if qualite != "film_movement" {
		t.Errorf("t0_quality = %q, attendu film_movement — le report du coup d'envoi n'a pas "+
			"ecrit dans le segment partage", qualite)
	}
	// Et la passe est marquee : rien n'a echoue (constat C1).
	if !replaybuild.DerivationsUpToDate(chemin) {
		t.Error("marque absente alors que les quatre familles ont pu s'executer")
	}
}
