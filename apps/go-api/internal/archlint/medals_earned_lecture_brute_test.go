// Package archlint — medals_earned_lecture_brute_test.go : LA LECTURE DES MÉDAILLES PASSE PAR LA
// VUE `v_medals_earned`, la table brute reste réservée à une liste fermée et datée.
//
// Pourquoi. Les médailles « Vengeur » et « Cible difficile » ne s'écrivent pas dans
// `medals_earned` : elles se dérivent du film et vivent dans les colonnes de
// `match_usage_players_latest`. La vue `v_medals_earned` (migration `shared_medals_earned_view_v1`)
// fusionne la table et ces colonnes ; un lecteur d'AFFICHAGE qui lirait la table brute servirait
// des médailles sans les médailles dérivées, et un Vengeur hérité gonflé.
//
// Ce qui est compté. Dans les fichiers non-test de `internal/` et `cmd/` (hors migrations, qui
// définissent les tables et les vues), chaque littéral chaîne qui lit la table (`FROM` ou `JOIN
// medals_earned`) et chaque littéral qui n'est que le nom de la table (liste de tables copiées ou
// contrôlées). Une écriture (`INSERT INTO`, `CREATE`) n'est pas une lecture.
//
// Ratchet par fichier. Chaque entrée de [lecturesBrutesMedailles] porte une date, une raison et
// son compte exact : un fichier qui en lit davantage, un fichier hors liste qui lit la table, ou une
// entrée dont le compte a baissé (à abaisser dans le même commit) font échouer le test. Une
// entrée ne s'ajoute pas pour faire passer un lecteur d'affichage : il lit la vue.
package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// lectureBruteMedailles : ce qu'un fichier est autorisé à lire sur la table brute.
type lectureBruteMedailles struct {
	lectures int
	date     string
	raison   string
}

// lecturesBrutesMedailles : la liste fermée (mesurée le 2026-11-22 : 19 lectures dans 15 fichiers,
// plus la valeur d'exploit, tranchée brute). Chemins relatifs à `apps/go-api`.
var lecturesBrutesMedailles = map[string]lectureBruteMedailles{
	"internal/platform/duckdb/player_matches_repo.go":             {1, "2026-11-22", "somme filtrée sur les médailles d'élimination parfaite (identifiants natifs)"},
	"internal/platform/duckdb/queries_squad.go":                   {1, "2026-11-22", "éliminations parfaites (identifiants natifs)"},
	"internal/platform/duckdb/queries_match.go":                   {4, "2026-11-22", "éliminations parfaites (identifiants natifs), quatre requêtes"},
	"internal/platform/duckdb/queries_home_citations.go":          {1, "2026-11-22", "éliminations parfaites (identifiants natifs)"},
	"internal/platform/duckdb/compare_repo.go":                    {1, "2026-11-22", "éliminations parfaites (identifiants natifs)"},
	"internal/platform/duckdb/vehicle_commendation_stats_repo.go": {1, "2026-11-22", "somme filtrée sur les médailles de détournement (identifiants natifs)"},
	"internal/sync/comeback.go":                                   {1, "2026-11-22", "filtre sur une médaille native (Steaktacular)"},
	"internal/platform/duckdb/player_snapshot_store.go":           {1, "2026-11-22", "notification « première médaille » : ne compare que les médailles reçues de l'API"},
	"internal/sync/backfill.go":                                   {1, "2026-11-22", "ingestion : matchs dont les médailles restent à charger"},
	"internal/sync/invariants/invariants.go":                      {2, "2026-11-22", "validation : lignes orphelines et citations manquantes"},
	"internal/sync/medal_exploit_loader.go":                       {1, "2026-11-22", "score d'exploit persisté au sync, hors affichage"},
	"internal/validation/gate.go":                                 {1, "2026-11-22", "validation : liste des tables contrôlées"},
	"internal/transfer/format.go":                                 {1, "2026-11-22", "transfert : table copiée avec sa clé"},
	"internal/ops/snapshot_export.go":                             {1, "2026-11-22", "export : table copiée"},
	"internal/ops/seed_demo.go":                                   {1, "2026-11-22", "jeu de démonstration : table copiée"},
	"cmd/h5-backfill/main.go":                                     {1, "2026-11-22", "diagnostic : compte de lignes"},
}

var (
	lectureTableMedailles = regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+(?:\w+\.)?medals_earned\b`)
	dateIso               = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// dispenseDeScanMedailles : répertoires où la table se définit (schéma, migrations, doublure de test).
func dispenseDeScanMedailles(rel string) bool {
	return strings.HasPrefix(rel, "internal/migration/") ||
		strings.Contains(rel, "/migrations/") ||
		strings.HasPrefix(rel, "internal/sync/testutil/")
}

func TestMedalsEarnedLectureBruteSurListeFermee(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	comptes := map[string]int{}
	for _, racine := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(goAPIRoot, racine), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if dispenseDeScanMedailles(rel) {
				return nil
			}
			f, perr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if perr != nil {
				return perr
			}
			if n := lecturesMedailles(f); n > 0 {
				comptes[rel] = n
			}
			return nil
		})
		if err != nil {
			t.Fatalf("%s : %v", racine, err)
		}
	}

	var echecs []string
	for rel, n := range comptes {
		e, listé := lecturesBrutesMedailles[rel]
		switch {
		case !listé:
			echecs = append(echecs, rel+" : "+strconv.Itoa(n)+" lecture(s) de la table brute medals_earned hors liste ; "+
				"un lecteur d'affichage lit v_medals_earned")
		case n > e.lectures:
			echecs = append(echecs, rel+" : "+strconv.Itoa(n)+" lectures brutes, la liste en autorise "+strconv.Itoa(e.lectures))
		case n < e.lectures:
			echecs = append(echecs, rel+" : "+strconv.Itoa(n)+" lectures brutes pour "+strconv.Itoa(e.lectures)+
				" listées, abaisser l'entrée dans le même commit")
		}
	}
	for rel, e := range lecturesBrutesMedailles {
		if !dateIso.MatchString(e.date) || strings.TrimSpace(e.raison) == "" || e.lectures <= 0 {
			echecs = append(echecs, rel+" : entrée incomplète (date AAAA-MM-JJ, raison et compte positif obligatoires)")
		}
		if comptes[rel] == 0 {
			echecs = append(echecs, rel+" : entrée périmée, ce fichier ne lit plus la table brute, la retirer")
		}
	}
	sort.Strings(echecs)
	if len(echecs) > 0 {
		t.Errorf("lectures de medals_earned hors de la liste fermée :\n  %s", strings.Join(echecs, "\n  "))
	}
}

// lecturesMedailles compte, dans un fichier, les littéraux chaîne qui lisent la table brute.
func lecturesMedailles(f *ast.File) int {
	n := 0
	ast.Inspect(f, func(x ast.Node) bool {
		lit, ok := x.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		s, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		if strings.TrimSpace(s) == "medals_earned" {
			n++
			return true
		}
		n += len(lectureTableMedailles.FindAllStringIndex(s, -1))
		return true
	})
	return n
}
