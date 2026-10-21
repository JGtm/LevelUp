package duckdb

// resumes_parite_s0_test.go — PARITÉ 9.1 (transitoire) : les lecteurs de couverture basculés sur
// la tête S0 rendent exactement ce que rendait la lecture du journal.

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	titlepkg "levelup/go-api/internal/domain/title"
)

// lignesDe exécute une requête et rend ses lignes formatées, triées : le multiset fait foi.
func lignesDe(t *testing.T, db *sql.DB, requete string, args ...any) []string {
	t.Helper()
	rows, err := db.Query(requete, args...)
	if err != nil {
		t.Fatalf("requête : %v\n%s", err, requete)
	}
	defer func() { _ = rows.Close() }()
	cols, _ := rows.Columns()
	var out []string
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatalf("scan : %v", err)
		}
		out = append(out, fmt.Sprint(vals...))
	}
	sort.Strings(out)
	return out
}

func exigerMemesLignes(t *testing.T, nom string, ancien, nouveau []string) {
	t.Helper()
	if len(ancien) == 0 {
		t.Fatalf("%s : l'ancien lecteur ne rend rien, la parité ne mesure rien", nom)
	}
	if strings.Join(ancien, "\n") != strings.Join(nouveau, "\n") {
		t.Fatalf("%s : parité rompue\nancien  : %q\nnouveau : %q", nom, ancien, nouveau)
	}
	t.Logf("%s : %d ligne(s) identiques", nom, len(ancien))
}

func TestParite91_Univers(t *testing.T) {
	db := corpusParite(t)
	for _, slug := range []string{titlepkg.DefaultSlug, "halo_5"} {
		shared := newTestDB(db, ":memory:")
		pdb := &PlayerDB{Shared: shared, SharedReader: LegacySharedReader(shared), XUID: pJ1, TitleSlug: slug}
		repo := NewTacticalRepo(pdb)
		q := domain.TacticalQuery{PlayerXUID: pJ1, Matchs: domain.RestreindreAux(corpusPariteMatchs)}
		nouveau, args := repo.universSQL(q)
		ancien := strings.NewReplacer(
			"match_kill_heads_latest h", "match_kill_events_latest e",
			"h.match_id", "e.match_id",
			"h.publishable AND h.n_rows > 0", "e.publishable",
		).Replace(nouveau)
		if ancien == nouveau {
			t.Fatal("la requête de l'univers ne porte plus la tête : parité sans objet")
		}
		l := lignesDe(t, db, ancien, args...)
		exigerMemesLignes(t, "univers "+slug, l, lignesDe(t, db, nouveau, args...))
		if !strings.Contains(strings.Join(l, "|"), "true") || !strings.Contains(strings.Join(l, "|"), "false") {
			t.Fatalf("le corpus doit porter des matchs mesurés ET non mesurés : %v", l)
		}
	}
}

func TestParite91_JournalNonPubliable(t *testing.T) {
	db := corpusParite(t)
	clause, liste := clauseListeMatchs("e.match_id", corpusPariteMatchs)
	ancien := lignesDe(t, db, fmt.Sprintf(ancien_qSoloJournalNonPubliable, clause), liste)
	clauseH, listeH := clauseListeMatchs("h.match_id", corpusPariteMatchs)
	exigerMemesLignes(t, "journal non publiable", ancien, lignesDe(t, db, fmt.Sprintf(qSoloJournalNonPubliable, clauseH), listeH))
}

func TestParite91_DenominateurPortee(t *testing.T) {
	db := corpusParite(t)
	ids := corpusPariteMatchs
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	ancien := lignesDe(t, db, fmt.Sprintf(ancien_matchRangeKillsTotalSQL, Placeholders(len(ids))), args...)
	exigerMemesLignes(t, "dénominateur de portée", ancien, lignesDe(t, db, fmt.Sprintf(matchRangeKillsTotalSQL, Placeholders(len(ids))), args...))
	// Un sous-ensemble sans match publiable avec tueur nommé rend 0 des deux côtés.
	vide := []any{"pB", "pF"}
	a := lignesDe(t, db, fmt.Sprintf(ancien_matchRangeKillsTotalSQL, Placeholders(2)), vide...)
	n := lignesDe(t, db, fmt.Sprintf(matchRangeKillsTotalSQL, Placeholders(2)), vide...)
	exigerMemesLignes(t, "dénominateur nul", a, n)
}

func TestParite91_Q21dPortee(t *testing.T) {
	db := corpusParite(t)
	for _, m := range corpusPariteMatchs {
		a := lignesDe(t, db, ancien_Q21dAssistPairs, m, m)
		n := lignesDe(t, db, Q21dAssistPairs, m, m)
		exigerMemesLignes(t, "Q21d "+m, a, n)
	}
}

func TestParite91_Q32dCouverture(t *testing.T) {
	db := corpusParite(t)
	ids := corpusPariteMatchs
	squad := []string{pJ1, pJ2, pJ3}
	build := func(tpl string) (string, []any) {
		sqlText := fmt.Sprintf(tpl, Placeholders(len(ids)), Placeholders(len(ids)),
			Placeholders(len(squad)), Placeholders(len(squad)))
		var args []any
		for _, l := range [][]string{ids, ids, squad, squad} {
			for _, v := range l {
				args = append(args, v)
			}
		}
		return sqlText, args
	}
	sa, aa := build(ancien_Q32dSquadAssistPairsTemplate)
	sn, an := build(Q32dSquadAssistPairsTemplate)
	exigerMemesLignes(t, "Q32d", lignesDe(t, db, sa, aa...), lignesDe(t, db, sn, an...))
}

func TestParite91_Q28cPorteuses(t *testing.T) {
	db := corpusParite(t)
	ids := corpusPariteMatchs
	for _, scope := range [][]string{nil, ids[:3], ids} {
		q := assistExchangeQuery{me: pJ1, scope: scope, borne: ids}
		nouveau, nArgs := buildRelationAssistsQuery(q)
		// L'ancien texte : mêmes arguments sauf la liste des porteuses, qui n'existait pas.
		borneClause, borneArg := clauseListeMatchs("kv.match_id", ids)
		args := []any{q.me, borneArg}
		scopeClause := ""
		if scope != nil {
			scopeClause = " AND mp.match_id IN (" + Placeholders(len(scope)) + ")"
			args = append(args, ToAnySlice(scope)...)
		}
		low, mid := domain.AssistTierLowMaxPct, domain.AssistTierMidMaxPct
		ancien := fmt.Sprintf(ancien_Q28cRelationAssistsTpl, borneClause, analysis.SQLIsNotBotCol("p.xuid"), scopeClause, "",
			low, low, mid, mid, low, low, mid, mid)
		exigerMemesLignes(t, fmt.Sprintf("Q28c scope=%d", len(scope)),
			lignesDe(t, db, ancien, args...), lignesDe(t, db, nouveau, nArgs...))
	}
}

func TestParite91_Q26lPorteuses(t *testing.T) {
	db := corpusParite(t)
	nouveau, nArgs := buildHomeAssistedFragsQuery(pJ1, corpusPariteMatchs)
	nomme := sqlAssistantNomme("kv")
	lotCTE, argCTE := clauseListeMatchs("match_id", corpusPariteMatchs)
	lotKV, argKV := clauseListeMatchs("kv.match_id", corpusPariteMatchs)
	low, mid := domain.AssistTierLowMaxPct, domain.AssistTierMidMaxPct
	ancien := fmt.Sprintf(ancien_Q26lHomeAssistedFragsTpl, lotCTE, nomme, nomme, low, nomme, low, mid, nomme, mid, lotKV)
	exigerMemesLignes(t, "Q26l", lignesDe(t, db, ancien, argCTE, argKV, pJ1), lignesDe(t, db, nouveau, nArgs...))
}
