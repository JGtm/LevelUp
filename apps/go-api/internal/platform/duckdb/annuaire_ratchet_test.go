package duckdb

// annuaire_ratchet_test.go — RATCHET DE `v_gamertag_lookup` SOUS internal/ (lots perf L2, L7,
// L9-go).
//
// La vue canonique des noms n'accepte aucun filtre poussé (agrégats en FULL OUTER JOIN) : chaque
// lecture la matérialise EN ENTIER, 1,7 à 3 s sur la base de production. Les lectures Escouade
// (L2), Carrière (L7) et les amis des rencontres (L9-go) nomment leurs lignes sans elle
// (squad_repo_annuaire.go, career_repo_friends.go). Ce test fige, fichier par fichier, les
// occurrences de l'IDENTIFIANT NU `v_gamertag_lookup` dans les littéraux de chaîne de TOUS les
// paquets de internal/ (hors tests) : en ajouter une — lecture, gabarit construit par
// concaténation, constante nommant la vue — le fait échouer ; en retirer une aussi, pour que la
// table suive (elle ne fait que descendre).
//
// ÉLARGI LE 2026-09-23 (lot L9-go, revue adversariale D) : le ratchet de L7 ne lisait que ce
// paquet et que `JOIN|FROM v_gamertag_lookup` — deux lectures du sync (killcollector) lui
// échappaient, comme lui aurait échappé un gabarit assemblé (`"FROM " + vue`).
//
// Seuls les littéraux de chaîne comptent (analyse syntaxique) : un commentaire Go qui cite la
// vue n'est pas une occurrence ; un commentaire SQL DANS un littéral, si.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var identifiantDeLaVueDesNoms = regexp.MustCompile(`\bv_gamertag_lookup\b`)

// occurrencesDeLaVueRestantes : fichier (relatif à internal/) -> occurrences permises, et
// pourquoi elles restent. Datée du 2026-09-23 (lots L7 et L9-go).
var occurrencesDeLaVueRestantes = map[string]int{
	// ── DDL et migrations : la vue se crée, se répare et se vérifie quelque part.
	"analysis/identity.go":                                1, // CREATE OR REPLACE VIEW : la SOURCE UNIQUE du DDL
	"games/halo_infinite/migrations/steps_shared_core.go": 3, // descriptions de trois étapes de migration (leurs NOMS contiennent l'identifiant sans le nommer seul : non comptés)
	"migration/steps_shared.go":                           2, // messages d'erreur de la création de la vue
	"migration/steps_shared_kill_events_from_pairs.go":    1, // message d'erreur de sa recréation
	"ops/seed_demo.go":                                    1, // message d'erreur de sa création (démo)
	"sync/schema.go":                                      1, // commentaire SQL du schéma de base
	"validation/gate.go":                                  2, // le gate vérifie que la vue existe (libellé + nom)
	// ── Lectures consignées (lot L7, plan perf §9 ter, journal (a) à (f)) : coût mesuré, non
	// mécaniques ou hors pages, chacune à retirer avec sa lecture.
	"platform/duckdb/explorer_repo.go":             1, // Explorer : ResolveXUIDByGamertag, un joueur cherché par NOM — plus lue par la Carrière (amis : career_repo_friends.go, lot L9-go)
	"platform/duckdb/gamertag_repo.go":             1, // ResolveGamertags : des xuids sans matchs
	"platform/duckdb/leaderboard_world_repo.go":    1, // classement mondial
	"platform/duckdb/media_repo_filters.go":        1, // Médias : lobbies des matchs
	"platform/duckdb/queries_career_encounters.go": 2, // Relations : Q28 et Q28 scopé
	"platform/duckdb/queries_match.go":             3, // vue match : Q12 tableau de score (lecture + commentaire SQL), Q21 événements
	"platform/duckdb/queries_match_detail.go":      2, // vue match : Q23 et Q23b
	"platform/duckdb/queries_relations_moments.go": 1, // Relations : heatmap Q29
	// ── Lectures du sync, hors pages : consignées le 2026-09-23 (lot L9-go) — le ratchet de L7
	// ne les voyait pas (autre paquet).
	"sync/killcollector/credit_annuaire.go": 1, // annuaire d'une passe de crédit, chargé une fois par passe
	"sync/killcollector/roster.go":          1, // roster d'un match sans chargeur d'annuaire (repli)
}

// occurrencesDeLaVueDesNoms parcourt internal/ (hors tests) et compte, par fichier, les
// occurrences de l'identifiant de la vue dans les littéraux de chaîne.
func occurrencesDeLaVueDesNoms(t *testing.T) map[string]int {
	t.Helper()
	racine := filepath.Join("..", "..") // internal/, depuis internal/platform/duckdb
	fset := token.NewFileSet()
	trouvees := map[string]int{}
	err := filepath.WalkDir(racine, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(chemin, ".go") || strings.HasSuffix(chemin, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, chemin, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(racine, chemin)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			if s, err := strconv.Unquote(lit.Value); err == nil {
				if k := len(identifiantDeLaVueDesNoms.FindAllStringIndex(s, -1)); k > 0 {
					trouvees[filepath.ToSlash(rel)] += k
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de internal/ : %v", err)
	}
	return trouvees
}

func TestLecturesDeLaVueDesNoms_Ratchet(t *testing.T) {
	trouvees := occurrencesDeLaVueDesNoms(t)
	for name, n := range trouvees {
		if n > occurrencesDeLaVueRestantes[name] {
			t.Errorf("%s : %d occurrence(s) de v_gamertag_lookup dans ses littéraux, %d permise(s). Chaque "+
				"lecture matérialise la vue entière (1,7 à 3 s) : nommer les lignes par l'annuaire de la "+
				"lecture (squad_repo_annuaire.go, nommerLignes).", name, n, occurrencesDeLaVueRestantes[name])
		}
	}
	for name, permis := range occurrencesDeLaVueRestantes {
		if trouvees[name] < permis {
			t.Errorf("%s : %d occurrence(s) de v_gamertag_lookup, la table en fige %d — une occurrence est "+
				"partie : descendre la table (ratchet).", name, trouvees[name], permis)
		}
	}
}
