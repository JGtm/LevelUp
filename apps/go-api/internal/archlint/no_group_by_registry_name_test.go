// Package archlint — no_group_by_registry_name_test.go : interdit un GROUP BY sur un NOM du
// registre des matchs (`map_name`, `pair_name`, `playlist_name`, `game_variant_name`, et leurs
// variantes `_fr`).
//
// POURQUOI. Le nom d'un asset varie d'un match a l'autre pour un meme identifiant : vrai nom,
// NULL, ou l'identifiant recopie quand la sync n'avait pas encore la traduction. Grouper sur lui
// scinde une carte (un mode, une playlist) en plusieurs rangees, chacune portant une part du
// compte — la grille de l'onglet Tactique a ainsi affiche une meme carte en deux ou trois
// vignettes, toutes sous le plancher de matchs. La cle d'un agregat est l'IDENTIFIANT
// (`map_id`, `pair_id`, `playlist_id`, `game_variant_id`) ; le nom se choisit par un agregat
// (modele : platform/duckdb/tactical_repo.go, nomDeCarteRetenuSQL) ou se resout par les
// traductions d'asset.
//
// PORTEE. Fichiers .go non-test de internal/ et cmd/ (migrations/ sautees : DDL gelee). Le motif
// lit le texte qui suit `GROUP BY` jusqu'a la fin de la clause (backtick, point-virgule ou
// parenthese fermante, 160 caracteres au plus), qualifie ou non par un alias. Un GROUP BY
// POSITIONNEL (`GROUP BY 1, 2`) sur une projection de nom lui echappe : la revue le couvre.
//
// EXCEPTIONS : par fichier, datees et justifiees. Le test echoue aussi sur une exception qui ne
// designe plus aucun GROUP BY de nom (exception perimee a retirer).
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// groupByRegistryNameRE : un GROUP BY dont la liste de cles contient une colonne de nom.
var groupByRegistryNameRE = regexp.MustCompile(
	"(?is)GROUP\\s+BY[^`;)]{0,160}?\\b(?:\\w+\\.)?(?:map|pair|playlist|game_variant)_name(?:_fr)?\\b")

// groupByRegistryNameExceptions : fichier (depuis apps/go-api) -> justification datee.
var groupByRegistryNameExceptions = map[string]string{
	"internal/platform/duckdb/replay_facts_repo.go": "2026-10-08 — `GROUP BY mr.match_id, mr.map_name` : " +
		"cle PAR MATCH. match_id est unique au registre, map_name en depend fonctionnellement : une rangee " +
		"par match, aucune scission possible.",
	"internal/ops/data_quality.go": "2026-10-08 — diagnostic des pair_name BRUTS non traduits : le nom " +
		"brut est l'objet mesure (filtre `pair_name <> pair_id`), regroupe ensuite par mode normalise.",
	"cmd/diag_expected_assists/main.go": "2026-10-08 — outil de diagnostic ponctuel, hors serveur, lecture " +
		"seule ; regroupement exploratoire par libelle de variante, aucun affichage produit.",
}

func TestNoGroupByRegistryName(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	var violations []string
	vus := map[string]bool{}
	for _, sub := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(goAPIRoot, sub), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "migrations" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			for _, loc := range groupByRegistryNameRE.FindAllIndex(data, -1) {
				if _, exempt := groupByRegistryNameExceptions[rel]; exempt {
					vus[rel] = true
					continue
				}
				line := 1 + strings.Count(string(data[:loc[0]]), "\n")
				snippet := strings.Join(strings.Fields(string(data[loc[0]:loc[1]])), " ")
				violations = append(violations, rel+":"+strconv.Itoa(line)+"  "+snippet)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", sub, err)
		}
	}
	sort.Strings(violations)
	for _, v := range violations {
		t.Errorf("GROUP BY sur un nom du registre (grouper sur l'identifiant, choisir le nom par agregat) : %s", v)
	}
	for f := range groupByRegistryNameExceptions {
		if !vus[f] {
			t.Errorf("exception perimee : %s ne contient plus de GROUP BY sur un nom — la retirer", f)
		}
	}
}
