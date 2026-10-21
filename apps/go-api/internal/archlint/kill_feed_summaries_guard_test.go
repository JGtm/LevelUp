package archlint

// kill_feed_summaries_guard_test.go — LE GARDE-RAIL DE L'ADR 0039 (décision 8) : un agrégat du
// journal des morts qui se décompose par match se lit sur son résumé (`match_kill_heads_latest`,
// `match_kill_pairs_latest`, `match_assist_pairs_latest`), jamais en recomptant le journal. Seuls
// les lecteurs de LIGNES (instants, coordonnées, médianes exactes, noms), listés ci-dessous par
// requête, lisent encore `match_kill_events_latest`.
//
// Niveau de la REQUÊTE (modèle ADR 0030 décision 4, comme le garde des lectures brutes des tables
// append-only) : on balaie la syntaxe, pas le texte, donc les commentaires ne comptent pas. Une
// déclaration de premier niveau (constante, variable ou fonction) qui porte l'identifiant
// `KillEventsCanonicalTable` ou un littéral de chaîne contenant `match_kill_events_latest` doit
// figurer dans l'allowlist, avec sa raison et sa date. Périmètre : platform/duckdb, api, service,
// analysis (comme TestNoRawAppendOnlyReads) ; sync, cmd, scheduler, ops, migration et persist, qui
// écrivent ou diagnostiquent, sont hors balayage.
//
// L'allowlist ne fait que DÉCROÎTRE : une entrée dont la déclaration ne lit plus le journal (ou
// n'existe plus) fait échouer le test, il faut alors la retirer.

import (
	"go/ast"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	identTableDuJournal   = "KillEventsCanonicalTable"
	litteralVueDuJournal  = "match_kill_events_latest"
	declarationDeLaTable  = "internal/platform/duckdb/kill_events_source.go:" + identTableDuJournal
	messageAllowlistVidee = "retirer l'entrée de l'allowlist de kill_feed_summaries_guard_test.go"
)

// lecteursDeLignesDuJournal : clé « fichier:déclaration » -> raison datée. Les lecteurs qui ont
// besoin de la mort elle-même (instant, position, source par instant, nom) et qu'un résumé par
// match ne peut pas servir.
var lecteursDeLignesDuJournal = map[string]string{
	// Le journal de la vue match et ses replis : une ligne par mort, instant à la milliseconde.
	"internal/platform/duckdb/queries_match.go:Q20KVPairs":                                                     "2026-10-21 : journal de la vue match, une ligne par mort avec son instant (Q20).",
	"internal/platform/duckdb/queries_match.go:Q21bKillSources":                                                "2026-10-21 : source de dégât par (tueur, instant) pour le kill feed (Q21b).",
	"internal/platform/duckdb/queries_match.go:Q21cKillAssists":                                                "2026-10-21 : assistant et parts par (tueur, instant) pour le kill feed (Q21c).",
	"internal/platform/duckdb/squad_repo_kill_log.go:Q32eSquadKillLogTemplate":                                 "2026-10-21 : journal des morts de l'escouade, badge « Voleur » par instant (Q32e).",
	"internal/platform/duckdb/queries_squad.go:Q32cSquadKVPairsTemplate":                                       "2026-10-21 : repli des events d'impact, une paire par mort avec son instant (Q32c).",
	"internal/platform/duckdb/highlight_events_repo.go:(*HighlightEventsRepo).loadSyntheticKillEvents":         "2026-10-21 : repli de highlight_events, events kill/death synthétisés par mort.",
	"internal/platform/duckdb/engagement_score_repo_queries.go:(*EngagementScoreRepo).loadSyntheticKillEvents": "2026-10-21 : même repli synthétique pour le score d'engagement, par mort.",
	"internal/platform/duckdb/solo_lives_repo.go:qSoloFrags":                                                   "2026-10-21 : frags de l'Isolement avec instant et camp de la victime.",
	// Les lectures qui joignent la mort à ses positions (distance, portée, Tactique).
	"internal/platform/duckdb/kill_measured.go:measuredKillsSQLTemplate":               "2026-10-21 : jointure unique mort x positions (distance, portée d'arme, portée du lobby) : lignes mesurées, médianes exactes.",
	"internal/platform/duckdb/tactical_repo.go:QTacticalPositions":                     "2026-10-21 : positions des morts, jointure mort x position (Tactique).",
	"internal/platform/duckdb/tactical_repo_isolement.go:QTacticalIsolement":           "2026-10-21 : morts situées de l'Isolement tactique, par instant.",
	"internal/platform/duckdb/tactical_repo_journal.go:QTacticalJournal":               "2026-10-21 : journal des kills de la Tactique, une ligne par mort.",
	"internal/platform/duckdb/tactical_repo_morts_par_carte.go:QTacticalMortsParCarte": "2026-10-21 : morts par carte de la Tactique, jointure aux positions.",
	// Autres titres et noms.
	"internal/platform/duckdb/halo5/halo5_match_events_source.go:h5KillFeedQuery": "2026-10-21 : source des events de match de Halo 5, une ligne par mort avec son instant.",
	"internal/analysis/identity.go:gamertagKillFeedLegs":                          "2026-10-21 : annuaire des noms, les gamertags du journal (noms seulement, pas un agrégat) ; à reprendre dans le chantier « annuaire » si mesuré coûteux.",
	"internal/analysis/weapon_evidence.go:WeaponEvidenceTable":                    "2026-10-21 : sonde d'existence (information_schema) qui NOMME la table de preuve des armes ; ne lit aucune ligne.",
}

func TestJournalDesMortsLuSeulementParLesLecteursDeLignes(t *testing.T) {
	lues := declarationsLisantLeJournal(t)
	var intrus []string
	for cle := range lues {
		if cle == declarationDeLaTable {
			continue
		}
		if _, permis := lecteursDeLignesDuJournal[cle]; !permis {
			intrus = append(intrus, cle)
		}
	}
	sort.Strings(intrus)
	if len(intrus) > 0 {
		t.Errorf("%d requête(s) lisent `match_kill_events_latest` hors allowlist (ADR 0039, décision 8). "+
			"Un agrégat par match se lit sur `match_kill_heads_latest` / `match_kill_pairs_latest` / "+
			"`match_assist_pairs_latest` ; un lecteur de lignes s'ajoute à l'allowlist avec sa raison "+
			"datée :\n  %s", len(intrus), strings.Join(intrus, "\n  "))
	}
	var perimees []string
	for cle := range lecteursDeLignesDuJournal {
		if !lues[cle] {
			perimees = append(perimees, cle)
		}
	}
	sort.Strings(perimees)
	if len(perimees) > 0 {
		t.Errorf("entrée(s) d'allowlist qui ne lisent plus le journal (%s) :\n  %s",
			messageAllowlistVidee, strings.Join(perimees, "\n  "))
	}
}

// declarationsLisantLeJournal rend l'ensemble des « fichier:déclaration » du périmètre dont le code
// (hors commentaires) nomme la table du journal.
func declarationsLisantLeJournal(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, src := range goSourcesUnder(t, "internal/platform/duckdb", "internal/api", "internal/service", "internal/analysis") {
		for _, decl := range src.File.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if nommeLeJournal(d.Body) || nommeLeJournal(d.Type) {
					out[src.Rel+":"+nomDeFonction(d)] = true
				}
			case *ast.GenDecl:
				if d.Tok != token.CONST && d.Tok != token.VAR {
					continue
				}
				for _, spec := range d.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok || !nommeLeJournal(vs) {
						continue
					}
					for _, n := range vs.Names {
						out[src.Rel+":"+n.Name] = true
					}
				}
			}
		}
	}
	return out
}

// nommeLeJournal : le sous-arbre porte l'identifiant de la table ou un littéral qui la nomme. Les
// commentaires ne sont pas des nœuds de l'arbre : ils ne comptent jamais.
func nommeLeJournal(n ast.Node) bool {
	if n == nil {
		return false
	}
	trouve := false
	ast.Inspect(n, func(x ast.Node) bool {
		switch v := x.(type) {
		case *ast.Ident:
			if v.Name == identTableDuJournal {
				trouve = true
			}
		case *ast.BasicLit:
			if v.Kind == token.STRING {
				if s, err := strconv.Unquote(v.Value); err == nil && strings.Contains(s, litteralVueDuJournal) {
					trouve = true
				}
			}
		}
		return !trouve
	})
	return trouve
}

// TestGardeJournal_DetecteurSurSource : auto-preuve du détecteur. Un commentaire ne compte pas ; un
// identifiant ou un littéral de requête compte, dans une constante comme dans une fonction.
func TestGardeJournal_DetecteurSurSource(t *testing.T) {
	src := parseGoSnippet(t, "snippet.go", `package p

// commentaire : match_kill_events_latest et KillEventsCanonicalTable ne comptent pas.
const parIdentifiant = "SELECT 1 FROM " + KillEventsCanonicalTable
const parLitteral = "SELECT 1 FROM match_kill_events_latest"
const resume = "SELECT 1 FROM match_kill_pairs_latest"

func lecteur() string { return "FROM match_kill_events_latest" }
func (r *R) methode() string { return KillEventsCanonicalTable }
func propre() string { return "FROM match_kill_heads_latest" }
`)
	trouves := map[string]bool{}
	for _, decl := range src.File.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if nommeLeJournal(d.Body) {
				trouves[nomDeFonction(d)] = true
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok && nommeLeJournal(vs) {
					trouves[vs.Names[0].Name] = true
				}
			}
		}
	}
	attendus := map[string]bool{"parIdentifiant": true, "parLitteral": true, "lecteur": true, "(*R).methode": true}
	if len(trouves) != len(attendus) {
		t.Fatalf("détectés %v, attendus %v", trouves, attendus)
	}
	for k := range attendus {
		if !trouves[k] {
			t.Errorf("%s non détecté dans %v", k, trouves)
		}
	}
}
