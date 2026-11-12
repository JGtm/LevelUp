//go:build integration

package replayartifacts

// leadshare_integration_test.go — LA MAINMISE, DE L'ARTEFACT A LA VUE `_latest`, sur une VRAIE base
// migrée et un VRAI artefact rangé.

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
)

// lsLot : base migrée, registre garni (mode et durée), artefact rangé, Deps du fil de l'eau.
func lsLot(t *testing.T, matchID, pair string, duree int) (*sql.DB, Deps, string) {
	t.Helper()
	db := baseRegistre(t)
	repoRoot := racineAvecConfigDuTitre(t, titlePkg.DefaultSlug)
	quand := temoinT0()
	if _, err := db.Exec(`INSERT INTO match_registry
		(match_id, start_time, start_time_utc, backfill_completed, map_name, pair_name, duration_seconds)
		VALUES (?, ?, ?, 0, 'Recharge', ?, ?)`, matchID, quand, quand, pair, duree); err != nil {
		t.Fatalf("insert registre: %v", err)
	}
	doc := lsDocument()
	doc.MatchID = matchID
	chemin := vhPoser(t, repoRoot, matchID, *doc)
	d := Deps{
		RepoRoot: repoRoot, TitleSlug: titlePkg.DefaultSlug, Gamertag: "testeur",
		WithRead:      func(_ context.Context, _ string, fn func(*sql.DB)) { fn(db) },
		AcquireWriter: func(context.Context) (*sql.DB, func(), error) { return db, func() {}, nil },
	}
	return db, d, chemin
}

func lsFamille(t *testing.T, d Deps, b *bilanDerivations, chemin, matchID string) {
	t.Helper()
	ctx := context.Background()
	lus := lireArtefacts(ctx, d, []ArtefactRange{{MatchID: matchID, Path: chemin}})
	if len(lus) != 1 {
		t.Fatalf("%d artefact(s) lu(s), attendu 1", len(lus))
	}
	ecrireMainmise(ctx, d, b, preparerMainmise(ctx, d, b, lus))
}

func lsLire(t *testing.T, db *sql.DB, matchID string) (measured bool, reason string, measuredMS int64, changes, camps int) {
	t.Helper()
	if err := db.QueryRow(`SELECT measured, unmeasured_reason, measured_ms, lead_changes
		FROM match_lead_share_latest WHERE match_id = ? AND row_kind = 'match'`, matchID).
		Scan(&measured, &reason, &measuredMS, &changes); err != nil {
		t.Fatalf("ligne match: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_lead_share_latest WHERE match_id = ? AND row_kind = 'camp'`,
		matchID).Scan(&camps); err != nil {
		t.Fatalf("camps: %v", err)
	}
	return measured, reason, measuredMS, changes, camps
}

func TestMainmise_EcritLaMesureEtLesCamps(t *testing.T) {
	db, d, chemin := lsLot(t, "0000ee01", lsPairSlay, lsDuree)
	lsFamille(t, d, &bilanDerivations{}, chemin, "0000ee01")
	measured, reason, ms, changes, camps := lsLire(t, db, "0000ee01")
	if !measured || reason != "" || ms != 580000 || changes != 2 || camps != 2 {
		t.Errorf("mesuré=%v raison=%q durée=%d retournements=%d camps=%d", measured, reason, ms, changes, camps)
	}
}

func TestMainmise_UnModeAPeuDePointsSEcritNonMesure(t *testing.T) {
	db, d, chemin := lsLot(t, "0000ee02", "CTF:Arena on Recharge", lsDuree)
	lsFamille(t, d, &bilanDerivations{}, chemin, "0000ee02")
	measured, reason, _, _, camps := lsLire(t, db, "0000ee02")
	if measured || reason != "mode_events" || camps != 0 {
		t.Errorf("mesuré=%v raison=%q camps=%d, attendu non mesuré mode_events sans camp", measured, reason, camps)
	}
}

func TestMainmise_MatchAbsentDuRegistreNEstNiProjeteNiMarque(t *testing.T) {
	db, d, chemin := lsLot(t, "0000ee03", lsPairSlay, lsDuree)
	if _, err := db.Exec(`DELETE FROM match_registry WHERE match_id = '0000ee03'`); err != nil {
		t.Fatalf("delete: %v", err)
	}
	b := &bilanDerivations{}
	lsFamille(t, d, b, chemin, "0000ee03")
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_lead_share WHERE match_id = '0000ee03'`).Scan(&n); err != nil {
		t.Fatalf("comptage: %v", err)
	}
	if n != 0 || !b.aEchoue("0000ee03") {
		t.Errorf("%d ligne(s) écrite(s), échec au bilan = %v : attendu 0 et vrai", n, b.aEchoue("0000ee03"))
	}
}

func TestMainmise_SansLaCapabilityRienNEstEcrit(t *testing.T) {
	db, d, chemin := lsLot(t, "0000ee04", lsPairSlay, lsDuree)
	p := filepath.Join(d.RepoRoot, "config", "titles", titlePkg.DefaultSlug, "mappings", "capabilities.toml")
	blob, err := os.ReadFile(p) //nolint:gosec // racine de test
	if err != nil {
		t.Fatalf("lecture capabilities: %v", err)
	}
	var out []string
	for _, l := range strings.Split(string(blob), "\n") {
		if !strings.Contains(l, string(games.CapFilmLeadShare)) {
			out = append(out, l)
		}
	}
	if err := os.WriteFile(p, []byte(strings.Join(out, "\n")), 0o600); err != nil {
		t.Fatalf("écriture capabilities: %v", err)
	}
	lsFamille(t, d, &bilanDerivations{}, chemin, "0000ee04")
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_lead_share`).Scan(&n); err != nil {
		t.Fatalf("comptage: %v", err)
	}
	if n != 0 {
		t.Errorf("%d ligne(s) écrite(s) sur un titre sans la capability", n)
	}
}

func TestMainmise_UneSecondePasseRemplaceLaPremiere(t *testing.T) {
	db, d, chemin := lsLot(t, "0000ee05", lsPairSlay, lsDuree)
	lsFamille(t, d, &bilanDerivations{}, chemin, "0000ee05")
	lsFamille(t, d, &bilanDerivations{}, chemin, "0000ee05")
	var vue, brut int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_lead_share_latest WHERE match_id = '0000ee05'`).Scan(&vue)
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_lead_share WHERE match_id = '0000ee05'`).Scan(&brut)
	if vue != 3 || brut != 6 {
		t.Errorf("vue=%d (attendu 3) brut=%d (attendu 6, append-only)", vue, brut)
	}
}

// Un reglement illisible est une REFERENCE manquante, pas un fait du match : le lot est inscrit en
// echec (donc sans marque) et rien n'est ecrit. Ecrire des passes ici ferait de la panne un refus
// que la marque rendrait definitif.
func TestMainmise_ReglementIllisibleEchoueLeLotSansRienEcrire(t *testing.T) {
	db, d, chemin := lsLot(t, "0000ee06", lsPairSlay, lsDuree)
	reg := filepath.Join(d.RepoRoot, "config", "titles", titlePkg.DefaultSlug, "mappings", "regulation.toml")
	if err := os.Remove(reg); err != nil {
		t.Fatalf("retrait du reglement: %v", err)
	}
	b := &bilanDerivations{}
	lsFamille(t, d, b, chemin, "0000ee06")
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_lead_share WHERE match_id = '0000ee06'`).Scan(&n); err != nil {
		t.Fatalf("comptage: %v", err)
	}
	if n != 0 || !b.aEchoue("0000ee06") {
		t.Errorf("%d ligne(s) ecrite(s), echec au bilan = %v : attendu 0 et vrai (pas de marque)", n, b.aEchoue("0000ee06"))
	}
}

// Un match sans calque de score n'a rien de mesurable, et l'identite du match n'y change rien : refus
// MESURE ecrit avec sa raison, meme sans segment de lecture ni ligne de registre, donc marque.
func TestMainmise_SansCalqueDeScoreEstUnRefusMesureSansLireLIdentite(t *testing.T) {
	db, d, _ := lsLot(t, "0000ee07", lsPairSlay, lsDuree)
	poserArtefact(t, d.RepoRoot, "0000ee07") // document vide : aucun calque de score
	chemin := titlePkg.NewPathResolver(d.RepoRoot).ReplayArtifactPath(titlePkg.DefaultSlug, "0000ee07")
	d.WithRead = nil
	b := &bilanDerivations{}
	lsFamille(t, d, b, chemin, "0000ee07")
	measured, reason, _, _, camps := lsLire(t, db, "0000ee07")
	if measured || reason != "score_missing" || camps != 0 || b.aEchoue("0000ee07") {
		t.Errorf("mesure=%v raison=%q camps=%d echec=%v, attendu non mesure score_missing, sans echec",
			measured, reason, camps, b.aEchoue("0000ee07"))
	}
}

// Un camp muet que le roster designe est ecrit comme une ligne de camp a zero MESURE.
func TestMainmise_UnCampMuetPorteSaLigneAZero(t *testing.T) {
	db, d, chemin := lsLot(t, "0000ee08", lsPairSlay, lsDuree)
	doc := lsDocument()
	doc.MatchID = "0000ee08"
	doc.ScoreTimeline.Teams = doc.ScoreTimeline.Teams[:1]
	chemin = vhPoser(t, d.RepoRoot, "0000ee08", *doc)
	lsFamille(t, d, &bilanDerivations{}, chemin, "0000ee08")
	var leadMS int64
	if err := db.QueryRow(`SELECT lead_ms FROM match_lead_share_latest
		WHERE match_id = '0000ee08' AND row_kind = 'camp' AND camp = 1`).Scan(&leadMS); err != nil {
		t.Fatalf("ligne du camp muet: %v", err)
	}
	if leadMS != 0 {
		t.Errorf("camp muet : %d ms en tete, attendu 0", leadMS)
	}
}
