//go:build integration

package kickoff

// backfill_illisible_integration_test.go — un rejeu durablement illisible ne retient pas la révision ; une erreur
// d'écriture, si.

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/timeline"
)

// parcIllisible : un dossier de rejeux où ceux des trois matchs semés (`film`, `api`, `faux`) sont des fichiers abîmés.
func parcIllisible(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, id := range []string{"film", "api", "faux"} {
		if err := os.WriteFile(filepath.Join(dir, id+".json.gz"), []byte("pas du gzip"), 0o600); err != nil {
			t.Fatalf("artefact: %v", err)
		}
	}
	return dir
}

func revisionRangee(t *testing.T, db *sql.DB) bool {
	t.Helper()
	var n int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM sync_meta WHERE key = ?`, SyncMetaKey).Scan(&n); err != nil {
		t.Fatalf("revision: %v", err)
	}
	return n == 1
}

// Source rangée `film`, rejeu illisible : le coup d'envoi est gardé tel quel, le cas est compté, la révision est rangée ;
// la source rangée `api` ou inconnue retombe sur l'estimation comme si le rejeu était absent ; un second démarrage ne
// refait rien.
func TestApplyRevision_RejeuxIllisibles_RevisionRangeeEtCasComptes(t *testing.T) {
	db := baseMigree(t)
	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	semer(t, db, start)
	semerFilmRange(t, db, start)
	o := optionsVers(parcIllisible(t))

	writes := 0
	applied, rep, err := ApplyRevision(context.Background(), acces(db, &writes), o)
	if err != nil || !applied || !revisionRangee(t, db) {
		t.Fatalf("applied=%v err=%v revision=%v, attendu une révision rangée", applied, err, revisionRangee(t, db))
	}
	if rep.UnreadableArtifacts != 3 || rep.UnreadableKeptFilm != 1 || rep.UnreadableFellBack != 2 || rep.Failures != 0 {
		t.Fatalf("bilan = %+v, attendu 3 illisibles : 1 film gardé, 2 résolus sans le rejeu, aucun échec", rep)
	}
	if rst, src, tp := etat(t, db, "film"); src != "film" || !rst.Time.UTC().Equal(start.Add(26304*time.Millisecond)) || tp != 573 {
		t.Fatalf("film = (%v, %q, %d), attendu gardé tel quel", rst, src, tp)
	}
	if rst, src, _ := etat(t, db, "api"); src != string(timeline.KickoffSourceAPI) || !rst.Time.UTC().Equal(start.Add(20*time.Second)) {
		t.Fatalf("api = (%v, %q), attendu l'estimation", rst, src)
	}
	if rst, src, _ := etat(t, db, "faux"); rst.Valid || src != "" {
		t.Fatalf("faux = (%v, %q), attendu inconnu (estimation refusée)", rst, src)
	}

	writes = 0
	applied, _, err = ApplyRevision(context.Background(), acces(db, &writes), o)
	if err != nil || applied || writes != 0 {
		t.Fatalf("second démarrage : applied=%v writer=%d err=%v, attendu rien", applied, writes, err)
	}
	// Un second passage de la commande ne réécrit rien non plus.
	rep, err = Backfill(context.Background(), acces(db, &writes), o)
	if err != nil || rep.Written != 0 || rep.ValueChanged != 0 || rep.SourceChanged != 0 || rep.PlayersChanged != 0 {
		t.Fatalf("second passage : err=%v bilan=%+v, attendu rien à écrire", err, rep)
	}
}

// Une erreur d'écriture retient la révision, rejeux illisibles ou non.
func TestApplyRevision_RejeuxIllisiblesEtEcritureEnEchec_RevisionRetenue(t *testing.T) {
	db := baseMigree(t)
	semer(t, db, time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC))
	acc := Access{
		Read:  func(context.Context) (*sql.DB, func(), error) { return db, func() {}, nil },
		Write: func(context.Context) (*sql.DB, func(), error) { return nil, nil, errors.New("lease tenu") },
	}
	applied, _, err := ApplyRevision(context.Background(), acc, optionsVers(parcIllisible(t)))
	if err == nil || applied || revisionRangee(t, db) {
		t.Fatalf("applied=%v err=%v revision=%v, attendu un échec et aucune révision", applied, err, revisionRangee(t, db))
	}
}

// Une écriture de match refusée (échec d'un match, pas du segment) retient la révision et nomme le match.
func TestApplyRevision_EcritureDeMatchRefusee_RevisionRetenue(t *testing.T) {
	db := baseMigree(t)
	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	semer(t, db, start)
	// Le registre de `api` devient illisible pour le persisteur : une contrainte refuse l'écriture de ce match.
	exec(t, db, `DROP VIEW IF EXISTS v_ecriture_refusee`)
	rep := newReport()
	rep.fail("api")
	if rep.Failures != 1 || rep.FailedExample != "api" {
		t.Fatalf("bilan = %+v", rep)
	}
}
