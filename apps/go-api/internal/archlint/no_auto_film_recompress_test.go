// Package archlint — no_auto_film_recompress_test.go : la recompression et le déplacement de
// l'archive des films ne se lancent JAMAIS tout seuls (lot L14, décision D-18 du 2026-11-01).
//
// POURQUOI. Une recompression ou un déplacement réécrit des dizaines de Go de films
// irremplaçables (un film expire côté serveur Halo et ne se retélécharge jamais), et le plafond
// mémoire d'un cache entier est une bombe connue. La règle produit de l'utilisateur est que
// purge et recompression sont des COMMANDES qu'il lance lui-même : `levelup film recompress` et
// `levelup film archive-move`. Un cron, une tâche de fond ou le démarrage du serveur qui les
// appellerait feraient de cette décision un comportement par défaut.
//
// RÈGLE. `filmarchive.Recompresser` et `filmarchive.Deplacer` n'ont, en production, qu'un appelant :
// `cmd/levelup/cmd_film.go`. Le garde-rail balaie tout le module `apps/go-api` ; l'allowlist est
// un fichier unique et n'a pas d'autre entrée à garder. Les `_test.go` sont exclus.
package archlint

import (
	"go/ast"
	"testing"
)

// gestesManuelsDeLArchive : les fonctions de `filmarchive` réservées à l'utilisateur.
var gestesManuelsDeLArchive = map[string]bool{"Recompresser": true, "Deplacer": true}

// appelantsManuelsDeLArchive : les seuls fichiers de production qui les appellent.
var appelantsManuelsDeLArchive = map[string]bool{"cmd/levelup/cmd_film.go": true}

func TestRecompressionEtDeplacementDeLArchiveSontManuels(t *testing.T) {
	sources := goSourcesUnder(t, "cmd", "internal")
	appels := map[string]int{}
	for _, s := range sources {
		ast.Inspect(s.File, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if ok && pkg.Name == "filmarchive" && gestesManuelsDeLArchive[sel.Sel.Name] {
				appels[s.Rel]++
			}
			return true
		})
	}
	for rel, n := range appels {
		if !appelantsManuelsDeLArchive[rel] {
			t.Errorf("%s appelle filmarchive.Recompresser/Deplacer (%d fois) : ces gestes ne se lancent "+
				"que par `levelup film ...`, jamais depuis un cron, une tâche de fond ou le démarrage "+
				"(décision D-18, 2026-11-01)", rel, n)
		}
	}
	for rel := range appelantsManuelsDeLArchive {
		if appels[rel] == 0 {
			t.Errorf("%s n'appelle plus filmarchive.Recompresser/Deplacer : retirer l'entrée de ce "+
				"garde-rail avec le code (CLAUDE.md règle 7)", rel)
		}
	}
}
