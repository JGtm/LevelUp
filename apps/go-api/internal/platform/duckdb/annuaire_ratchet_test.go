package duckdb

// annuaire_ratchet_test.go — RATCHET DES LECTURES DE `v_gamertag_lookup` (lots perf L2 et L7).
//
// La vue canonique des noms n'accepte aucun filtre poussé (agrégats en FULL OUTER JOIN) : chaque
// lecture la matérialise EN ENTIER, 1,7 à 3 s sur la base de production. Les lectures Escouade
// (L2) et Carrière (L7) nomment leurs lignes par l'annuaire de la lecture (squad_repo_annuaire.go). Ce
// test fige, fichier par fichier, les lectures SQL de la vue qui restent dans la couche de
// lecture : en ajouter une — ou la réintroduire dans une lecture passée à l'annuaire — le fait
// échouer ; en retirer une aussi, pour que la table suive (elle ne fait que descendre).
//
// Seuls les littéraux de chaîne comptent (analyse syntaxique) : un commentaire qui cite la
// jointure retirée n'est pas une lecture.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var lectureDeLaVueDesNoms = regexp.MustCompile(`(?i)\b(JOIN|FROM)\s+v_gamertag_lookup\b`)

// lecturesDeLaVueRestantes : fichier -> nombre de lectures SQL de la vue, et pourquoi elles
// restent (lot perf L7, 2026-09-23 ; le détail et le coût de chacune : plan perf §9 ter).
var lecturesDeLaVueRestantes = map[string]int{
	"compare_repo.go":              1, // Comparer : GetLocalStats
	"explorer_repo.go":             1, // Explorer : ResolveXUIDByGamertag, recherche par NOM
	"gamertag_repo.go":             1, // ResolveGamertags : des xuids sans matchs
	"leaderboard_world_repo.go":    1, // classement mondial
	"media_repo_filters.go":        1, // Médias : lobbies des matchs
	"queries_career_encounters.go": 3, // Carrière : Q27 rivaux ; Relations : Q28 et Q28 scopé
	"queries_match.go":             3, // Q10 (/career/encounters), Q12 tableau de score, Q21 événements
	"queries_match_detail.go":      2, // vue match : Q23 et Q23b
	"queries_relations_moments.go": 1, // Relations : heatmap Q29
}

func TestLecturesDeLaVueDesNoms_Ratchet(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	fset := token.NewFileSet()
	trouvees := map[string]int{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("analyse de %s : %v", name, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			if s, err := strconv.Unquote(lit.Value); err == nil {
				trouvees[name] += len(lectureDeLaVueDesNoms.FindAllStringIndex(s, -1))
			}
			return true
		})
	}
	for name, n := range trouvees {
		if n > lecturesDeLaVueRestantes[name] {
			t.Errorf("%s : %d lecture(s) de v_gamertag_lookup, %d permise(s). Chacune matérialise la "+
				"vue entière (1,7 à 3 s) : nommer les lignes par l'annuaire de la lecture "+
				"(squad_repo_annuaire.go, nommerLignes).", name, n, lecturesDeLaVueRestantes[name])
		}
	}
	for name, permis := range lecturesDeLaVueRestantes {
		if trouvees[name] < permis {
			t.Errorf("%s : %d lecture(s) de v_gamertag_lookup, la table en fige %d — une lecture est "+
				"partie : descendre la table (ratchet).", name, trouvees[name], permis)
		}
	}
}
