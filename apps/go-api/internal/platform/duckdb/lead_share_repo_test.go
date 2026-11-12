package duckdb

// lead_share_repo_test.go — LA MAINMISE SE LIT BORNÉE, SUR LA VUE `_latest` (ADR 0036, ADR 0026).
//
// Base `:memory:` migrée (les vraies migrations shared, donc la vraie vue). Ce que chaque test
// verrouille :
//   - la lecture ne rend que la DERNIÈRE PASSE ENTIÈRE de chaque match, et la fenêtre `_latest` ne
//     voit que les lignes des matchs demandés ;
//   - un match non mesuré, ou dont le camp du joueur est inconnu, n'est pas rendu — jamais un zéro ;
//   - un camp muet (jamais marqué) est une part nulle MESURÉE quand la passe le nomme (sa ligne de camp à
//     zéro) ; un camp absent de la passe ne se lit pas, même quand la passe compte des camps muets ;
//   - la part est la fraction 0..1 du temps joué ; table absente : games.ErrCapabilityNotSupported.

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"

	"levelup/go-api/internal/games"
)

const (
	lsrAncien  = "2026-11-01 10:00:00"
	lsrCourant = "2026-11-02 10:00:00"
)

// poserPasseMainmise écrit UNE ligne de `match_lead_share` : kind `match` (camp -1) ou `camp`.
func poserPasseMainmise(t *testing.T, pdb *PlayerDB, match, pass, written, kind string, camp int, leadMS int64,
	measured bool, reason string, measuredMS, tieMS int64, changes, silent int) {
	t.Helper()
	tacExec(t, pdb, `INSERT INTO match_lead_share
		(match_id, decode_pass, written_at, row_kind, camp, lead_ms, measured, unmeasured_reason, doc_schema,
		 measured_ms, tie_ms, lead_changes, silent_camps)
		VALUES (?, ?, CAST(? AS TIMESTAMP), ?, ?, ?, ?, ?, 71, ?, ?, ?, ?)`,
		match, pass, written, kind, camp, leadMS, measured, reason, measuredMS, tieMS, changes, silent)
}

// passeMesuree : une passe mesurée de deux camps (0 et 1), sur 1000 s.
func passeMesuree(t *testing.T, pdb *PlayerDB, match, pass, written string, lead0, lead1 int64, changes int) {
	t.Helper()
	tie := int64(1_000_000) - lead0 - lead1
	poserPasseMainmise(t, pdb, match, pass, written, "match", -1, 0, true, "", 1_000_000, tie, changes, 0)
	poserPasseMainmise(t, pdb, match, pass, written, "camp", 0, lead0, true, "", 1_000_000, tie, changes, 0)
	poserPasseMainmise(t, pdb, match, pass, written, "camp", 1, lead1, true, "", 1_000_000, tie, changes, 0)
}

func corpusMainmise(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	// m1 : deux passes, la courante dit 60 % pour le camp 0 et 3 retournements.
	passeMesuree(t, pdb, "m1", "p1", lsrAncien, 100_000, 100_000, 0)
	passeMesuree(t, pdb, "m1", "p2", lsrCourant, 600_000, 250_000, 3)
	// m2 : non mesuré (mode à peu de points).
	poserPasseMainmise(t, pdb, "m2", "p2", lsrCourant, "match", -1, 0, false, "mode_events", 0, 0, 0, 0)
	// m3 : un camp muet NOMMÉ (le camp 0 du joueur n'a jamais marqué, sa ligne est à zéro), le camp 1 mène 700 s.
	poserPasseMainmise(t, pdb, "m3", "p2", lsrCourant, "match", -1, 0, true, "", 1_000_000, 300_000, 0, 1)
	poserPasseMainmise(t, pdb, "m3", "p2", lsrCourant, "camp", 0, 0, true, "", 1_000_000, 300_000, 0, 1)
	poserPasseMainmise(t, pdb, "m3", "p2", lsrCourant, "camp", 1, 700_000, true, "", 1_000_000, 300_000, 0, 1)
	// m5 : la passe compte un camp muet mais ne le nomme pas, et le camp du joueur (0) n'a pas de ligne :
	// ce camp n'est pas prouvé au match, sa part ne se lit pas.
	poserPasseMainmise(t, pdb, "m5", "p2", lsrCourant, "match", -1, 0, true, "", 1_000_000, 300_000, 0, 1)
	poserPasseMainmise(t, pdb, "m5", "p2", lsrCourant, "camp", 1, 700_000, true, "", 1_000_000, 300_000, 0, 1)
	// m4 : mesuré, mais le joueur n'a pas de camp connu.
	passeMesuree(t, pdb, "m4", "p2", lsrCourant, 500_000, 400_000, 1)
	// m9 : HORS de la liste demandée, avec autant de lignes que possible.
	for i := 0; i < 12; i++ {
		passeMesuree(t, pdb, "m9", fmt.Sprintf("p%d", i), lsrCourant, 1, 1, 0)
	}
	for _, m := range []string{"m1", "m2", "m3", "m5", "m9"} {
		poserParticipant(t, pdb, m, "P", 0)
	}
	poserParticipant(t, pdb, "m1", "E", 1)
}

func TestLeadShareRepo_DernierePasseEntiereEtBorne(t *testing.T) {
	b := newBaseNotee(t)
	corpusMainmise(t, b.pdb)
	b.carnet.vider()

	got, err := NewLeadShareRepo(b.pdb).LoadLeadShare(context.Background(), []string{"m1", "m2", "m3", "m4", "m5"}, "P")
	if err != nil {
		t.Fatalf("LoadLeadShare : %v", err)
	}
	// Fenêtre : les lignes des cinq matchs demandés seulement (m1 : 6, m2 : 1, m3 : 3, m4 : 3, m5 : 2),
	// que les 36 lignes de m9 ne doivent pas grossir.
	exigerFenetresBornees(t, b, "LoadLeadShare", 15, 1)

	if s, ok := got["m1"]; !ok || math.Abs(s.Share-0.6) > 1e-9 || s.Changes != 3 {
		t.Errorf("m1 = %+v (ok=%v), attendu 0,6 et 3 retournements — la passe ancienne a survécu ?", s, ok)
	}
	if _, ok := got["m2"]; ok {
		t.Error("m2 non mesuré est rendu : jamais un zéro")
	}
	if s, ok := got["m3"]; !ok || s.Share != 0 {
		t.Errorf("m3 = %+v (ok=%v), attendu une part nulle MESURÉE (camp muet)", s, ok)
	}
	if _, ok := got["m5"]; ok {
		t.Error("m5 : un camp absent de la passe est rendu comme un zéro mesuré, alors que rien ne le prouve au match")
	}
	if _, ok := got["m4"]; ok {
		t.Error("m4, sans camp connu pour le joueur, est rendu")
	}
	if _, ok := got["m9"]; ok {
		t.Error("m9, hors de la liste, est lu")
	}
}

func TestLeadShareRepo_RienAPasLire(t *testing.T) {
	b := newBaseNotee(t)
	repo := NewLeadShareRepo(b.pdb)
	if got, err := repo.LoadLeadShare(context.Background(), nil, "P"); err != nil || len(got) != 0 {
		t.Errorf("liste vide : %v, %v", got, err)
	}
	if got, err := repo.LoadLeadShare(context.Background(), []string{"m1"}, ""); err != nil || len(got) != 0 {
		t.Errorf("joueur inconnu : %v, %v", got, err)
	}
}

func TestLeadShareRepo_TableAbsente(t *testing.T) {
	b := newBaseNotee(t)
	tacExec(t, b.pdb, `DROP VIEW match_lead_share_latest`)
	_, err := NewLeadShareRepo(b.pdb).LoadLeadShare(context.Background(), []string{"m1"}, "P")
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("erreur = %v, attendu games.ErrCapabilityNotSupported", err)
	}
}
