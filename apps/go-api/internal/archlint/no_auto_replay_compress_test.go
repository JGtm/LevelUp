// Package archlint — no_auto_replay_compress_test.go : la conversion du parc d'artefacts de rejeu en
// artefacts compressés ne se lance JAMAIS toute seule.
//
// POURQUOI. `levelup replay compress` réécrit tout le dossier d'artefacts. La règle produit est que
// conversion et recuisson d'un parc sont des COMMANDES que l'utilisateur lance lui-même. Un cron, une
// tâche de fond ou le démarrage du serveur qui appellerait `replaybuild.CompressDir` feraient de cette
// décision un comportement par défaut.
//
// RÈGLE. `replaybuild.CompressDir` n'a, en production, qu'un appelant : `cmd/levelup/cmd_replay.go`.
// Le garde-rail balaie `apps/go-api` ; les `_test.go` sont exclus.
package archlint

import (
	"go/ast"
	"testing"
)

// appelantsManuelsDeLaCompression : les seuls fichiers de production qui appellent CompressDir.
var appelantsManuelsDeLaCompression = map[string]bool{"cmd/levelup/cmd_replay.go": true}

func TestLaConversionDuParcDeRejeuxEstManuelle(t *testing.T) {
	appels := map[string]int{}
	for _, s := range goSourcesUnder(t, "cmd", "internal") {
		ast.Inspect(s.File, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "replaybuild" && sel.Sel.Name == "CompressDir" {
				appels[s.Rel]++
			}
			return true
		})
	}
	for rel, n := range appels {
		if !appelantsManuelsDeLaCompression[rel] {
			t.Errorf("%s appelle replaybuild.CompressDir (%d fois) : la conversion du parc ne se lance que par "+
				"`levelup replay compress`, jamais depuis un cron, une tâche de fond ou le démarrage", rel, n)
		}
	}
	for rel := range appelantsManuelsDeLaCompression {
		if appels[rel] == 0 {
			t.Errorf("%s n'appelle plus replaybuild.CompressDir : retirer l'entrée de ce garde-rail avec le code "+
				"(CLAUDE.md règle 7)", rel)
		}
	}
}
