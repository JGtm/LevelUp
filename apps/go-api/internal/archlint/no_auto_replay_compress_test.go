// Package archlint — no_auto_replay_compress_test.go : la conversion du parc d'artefacts de rejeu en
// artefacts compressés ne se lance JAMAIS toute seule.
//
// POURQUOI. `levelup replay compress` réécrit tout le dossier d'artefacts. La règle produit est que
// conversion et recuisson d'un parc sont des COMMANDES que l'utilisateur lance lui-même. Un cron, une
// tâche de fond ou le démarrage du serveur qui appellerait `replaybuild.CompressDir` feraient de cette
// décision un comportement par défaut.
//
// RÈGLE. `replaybuild.CompressDir` et `replaybuild.CompressDirWith` n'ont, en production, que DEUX appelants, chacun
// l'exécution d'un geste de l'utilisateur :
//
//   - `cmd/levelup/cmd_replay.go` : la commande `levelup replay compress` ;
//   - `internal/replayregen/compress.go` : la file de régénération des rejeux de l'application de bureau, À LA FIN d'une
//     passe que l'utilisateur a lancée (ou reprise : drapeau `replay_regen_pending` posé par ce geste). Ajouté le
//     2026-11-11 (décision D-9 du plan « rejeu à 30 images/s ») : une application installée n'a pas de ligne de commande,
//     et les rejeux dont le film n'est plus archivé ne peuvent pas être refaits, donc resteraient bruts. Le second
//     appelant ne part ni au démarrage du serveur ni du décompte : `replayregen` est tenu par son propre test
//     (`TestCompression_JamaisSansLeGeste`).
//
// Aucun autre : ni cron, ni tâche de fond, ni démarrage. Le garde-rail balaie `apps/go-api` ; les `_test.go` sont exclus.
package archlint

import (
	"go/ast"
	"testing"
)

// appelantsDeLaCompression : les seuls fichiers de production qui appellent la conversion (cf. l'en-tête).
var appelantsDeLaCompression = map[string]bool{
	"cmd/levelup/cmd_replay.go":        true,
	"internal/replayregen/compress.go": true,
}

// fonctionsDeConversion : les points d'entrée de la conversion du parc.
var fonctionsDeConversion = map[string]bool{"CompressDir": true, "CompressDirWith": true}

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
			if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "replaybuild" && fonctionsDeConversion[sel.Sel.Name] {
				appels[s.Rel]++
			}
			return true
		})
	}
	for rel, n := range appels {
		if !appelantsDeLaCompression[rel] {
			t.Errorf("%s appelle la conversion du parc (%d fois) : elle ne se lance que par `levelup replay compress` "+
				"et par la fin d'une passe de la file de régénération lancée par l'utilisateur, jamais depuis un cron, "+
				"une tâche de fond ou le démarrage", rel, n)
		}
	}
	for rel := range appelantsDeLaCompression {
		if appels[rel] == 0 {
			t.Errorf("%s n'appelle plus la conversion du parc : retirer l'entrée de ce garde-rail avec le code "+
				"(CLAUDE.md règle 7)", rel)
		}
	}
}
