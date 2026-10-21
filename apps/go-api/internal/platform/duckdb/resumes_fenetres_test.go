package duckdb

// resumes_fenetres_test.go — LES VUES DES RÉSUMÉS PAR MATCH SOUS UNE LISTE DE MATCHS (ADR 0039
// décision 7, ADR 0036 I2).
//
// Ce test rejoue, sous `EXPLAIN (ANALYZE, FORMAT JSON)`, les formes de lecture de
// `match_kill_heads_latest` (S0), `match_kill_pairs_latest` (S1) et `match_assist_pairs_latest`
// (S2) et lit le nombre de lignes vues par chaque fenêtre. S1 et S2 portent la fenêtre de la
// tête dans leur jointure.
//
// Ce qu'il tient (DuckDB embarqué) :
//   - S0 lue sous une liste `VARCHAR[]` liée en une constante : la fenêtre ne voit que les têtes
//     des matchs de la liste ;
//   - S1 et S2 lues pour UN match (`match_id = ?`) : l'égalité traverse la jointure, la fenêtre de
//     la tête ne voit que ce match ;
//   - S1 et S2 lues sous une liste : la liste ne traverse PAS la jointure, la fenêtre de la tête
//     voit toute la table des têtes (une ligne par match et génération), jamais les lignes des
//     filles. Cette fenêtre est acceptée (ADR 0039 décision 7, exception I2 de l'ADR 0036) ; le
//     test en tient la taille, il échoue si elle grossit.

import (
	"database/sql/driver"
	"fmt"
	"testing"
	"time"
)

// Corpus : dix matchs, deux générations de résumés chacun (la passe d'origine et une relecture),
// trois paires et deux groupes d'appui par génération.
const (
	resumesMatchs       = 10
	resumesGenerations  = 2
	resumesPairesParGen = 3
	resumesAppuisParGen = 2
)

func seedResumesFenetres(t *testing.T, b baseNotee) []string {
	t.Helper()
	base := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	ids := make([]string, 0, resumesMatchs)
	for i := 1; i <= resumesMatchs; i++ {
		id := fmt.Sprintf("r%02d", i)
		ids = append(ids, id)
		for g := 1; g <= resumesGenerations; g++ {
			gen := fmt.Sprintf("%s-g%d", id, g)
			at := base.Add(time.Duration(i)*time.Hour + time.Duration(g)*time.Minute)
			tacExec(t, b.pdb, `INSERT INTO match_kill_heads (match_id, summary_pass, summary_rev,
				src_decode_pass, written_at, publishable, n_rows, n_assist_known, n_killer_xuid, n_src_killer)
				VALUES (?, ?, 'ks1', ?, ?, TRUE, ?, 0, 0, 0)`, id, gen, id+"-pass", at, resumesPairesParGen)
			for k := 0; k < resumesPairesParGen; k++ {
				tacExec(t, b.pdb, `INSERT INTO match_kill_pairs (match_id, summary_pass, summary_rev,
					src_decode_pass, written_at, feed_killer_xuid, victim_xuid, source_tag, kills)
					VALUES (?, ?, 'ks1', ?, ?, ?, ?, NULL, 1)`, id, gen, id+"-pass", at,
					fmt.Sprintf("k%d", k), tacXUIDMoi)
			}
			for k := 0; k < resumesAppuisParGen; k++ {
				tacExec(t, b.pdb, `INSERT INTO match_assist_pairs (match_id, summary_pass, summary_rev,
					src_decode_pass, written_at, assist_xuid, assist_gamertag, feed_killer_xuid,
					feed_killer_gamertag, victim_xuid, n, n_stolen, sum_assist_pct, n_assist_pct,
					n_low, n_mid, n_high) VALUES (?, ?, 'ks1', ?, ?, ?, 'A', ?, 'K', NULL, 1, 0, 0, 0, 0, 0, 0)`,
					id, gen, id+"-pass", at, fmt.Sprintf("a%d", k), tacXUIDMoi)
			}
		}
	}
	return ids
}

// lectureResume : une forme de lecture d'une vue de résumé et ce que sa fenêtre peut voir.
type lectureResume struct {
	nom   string
	sql   string
	args  []driver.NamedValue
	borne int
}

// TestResumes_VuesSousUneListe_Fenetres : deux matchs demandés sur dix (un pour les lectures d'un
// seul match). Les bornes sont celles de l'en-tête.
func TestResumes_VuesSousUneListe_Fenetres(t *testing.T) {
	b := newBaseNotee(t)
	ids := seedResumesFenetres(t, b)
	clause, liste := clauseListeMatchs("match_id", ids[:2])
	parListe := []driver.NamedValue{{Ordinal: 1, Value: liste}}
	unMatch := []driver.NamedValue{{Ordinal: 1, Value: ids[0]}}
	toutesLesTetes := resumesMatchs * resumesGenerations
	for _, l := range []lectureResume{
		{"S0 sous une liste", `SELECT match_id, n_rows FROM match_kill_heads_latest WHERE ` + clause,
			parListe, 2 * resumesGenerations},
		{"S1 d'un match", `SELECT SUM(kills) FROM match_kill_pairs_latest WHERE match_id = ?`,
			unMatch, resumesGenerations},
		{"S2 d'un match", `SELECT SUM(n) FROM match_assist_pairs_latest WHERE match_id = ?`,
			unMatch, resumesGenerations},
		{"S1 sous une liste", `SELECT match_id, SUM(kills) FROM match_kill_pairs_latest WHERE ` + clause +
			` GROUP BY match_id`, parListe, toutesLesTetes},
		{"S2 sous une liste", `SELECT match_id, SUM(n) FROM match_assist_pairs_latest WHERE ` + clause +
			` GROUP BY match_id`, parListe, toutesLesTetes},
	} {
		fenetres := fenetresVues(t, b.brute, requeteNotee{sql: l.sql, args: l.args})
		t.Logf("%s : fenêtres vues %v (borne %d, table des têtes %d lignes)",
			l.nom, fenetres, l.borne, toutesLesTetes)
		if len(fenetres) == 0 {
			t.Fatalf("%s : aucune fenêtre dans le plan — la lecture ne passe plus par la tête", l.nom)
		}
		for _, n := range fenetres {
			if n > l.borne {
				t.Errorf("%s : la fenêtre de la tête a vu %d lignes, borne %d", l.nom, n, l.borne)
			}
		}
	}
}
