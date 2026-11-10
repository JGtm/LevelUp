// Package archlint — no_replay_json_literal_test.go : les extensions des fichiers de rejeu s'écrivent en
// UN endroit, `internal/domain/title/film_id.go`.
//
// POURQUOI. L'artefact est `{short8}.json.gz`, sa marque de dérivation `{short8}.derived.json`, son
// sidecar de raster `rasters/{short8}.json` ; `HasSuffix(name, ".json")` était recopié dans six
// balayages de dossier, chacun avec sa façon d'écarter la marque. Les constantes
// (`title.ReplayArtifactExt`, `ReplayArtifactLegacyExt`, `ReplayDerivationsMarkSuffix`,
// `ReplaySidecarExt`) et le prédicat `title.IsReplayArtifactFile` sont la seule écriture de la règle.
//
// RÈGLE. Une fonction de production de `apps/go-api` qui construit ou balaie des chemins de rejeu (son
// corps nomme `ReplayArtifactPath`, `ReplayArtifactsDir`, `ReplayArtifactFileName`, `TacticalRasterPath`,
// `SousDossierRasters` ou une fonction de la porte `replayartifact`, ou son NOM parle d'artefact ou de rejeu)
// ne porte aucun littéral `".json"`, `".json.gz"` ni `".derived.json"`. Seul `film_id.go` les déclare.
// L'allowlist est vide.
//
// LIMITE. Il n'existe pas de règle simple qui voie « un `HasSuffix(x, ".json")` appliqué au listage du dossier des
// rejeux » : le dossier arrive par un paramètre (`mesureFrequentation(dir)`), et `".json"` est aussi l'extension
// des jetons, des sessions et de dizaines de catalogues, dans les mêmes fichiers. La règle est donc lexicale, par
// fonction ; les balayages de dossier de rejeu hors de ce périmètre (un script `.mjs`, un outil dont la fonction ne
// parle pas de rejeu) se tiennent par la migration vers `replayartifact.List` et par la revue.
package archlint

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const fichierDesExtensionsDeRejeu = "internal/domain/title/film_id.go"

// allowlistLitteralRejeu : fichiers tolérés. VIDE ; ne jamais y ajouter sans justification datée.
var allowlistLitteralRejeu = map[string]bool{}

var motifSurfaceDeRejeu = regexp.MustCompile(
	`ReplayArtifactPath|ReplayArtifactsDir|ReplayArtifactFileName|TacticalRasterPath|SousDossierRasters|` +
		`replayartifact\.(Read|ReadBytes|List|Locate|Exists|Copy|Remove|Write)\(|` +
		`replaybuild\.(ArtifactDigest|ArtifactHeader|StoreArtifact)\b`)

var litteralsDeRejeu = map[string]bool{".json": true, ".json.gz": true, ".derived.json": true}

// litteralsDExtensionDeRejeu rend les littéraux d'extension de rejeu des fonctions d'une source qui touchent aux
// chemins de rejeu (le texte de la fonction nomme l'un des repères de motifSurfaceDeRejeu).
func litteralsDExtensionDeRejeu(s goSource) []string {
	var out []string
	for _, d := range s.File.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		if !motifSurfaceDeRejeu.MatchString(texteDeLExpression(s, fn.Body)) && !motifArtefact.MatchString(fn.Name.Name) {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			if v, err := strconv.Unquote(lit.Value); err == nil && litteralsDeRejeu[v] {
				out = append(out, fmt.Sprintf("%s:%d %s", s.Rel, s.Fset.Position(lit.Pos()).Line, lit.Value))
			}
			return true
		})
	}
	return out
}

func TestNoReplayJSONLiteral_ReconnaitLAncienneCopie(t *testing.T) {
	ancienne := parseGoSnippet(t, "internal/x/y.go", `package x
func f(dir string, n string) bool {
	_ = filepath.Join(dir, title.ReplayArtifactFileName("a"))
	return strings.HasSuffix(n, ".json")
}
`)
	if got := litteralsDExtensionDeRejeu(ancienne); len(got) != 1 {
		t.Errorf("le garde-rail doit voir le littéral, vu %v", got)
	}
	horsSujet := parseGoSnippet(t, "internal/x/y.go", `package x
func f(n string) bool { return strings.HasSuffix(n, ".json") }
`)
	if got := litteralsDExtensionDeRejeu(horsSujet); len(got) != 0 {
		t.Errorf("un fichier qui ne touche pas aux chemins de rejeu n'est pas concerné : %v", got)
	}
}

func TestNoReplayJSONLiteral(t *testing.T) {
	var violations []string
	for _, s := range goSourcesUnder(t, "cmd", "internal") {
		if s.Rel == fichierDesExtensionsDeRejeu || allowlistLitteralRejeu[s.Rel] {
			continue
		}
		violations = append(violations, litteralsDExtensionDeRejeu(s)...)
	}
	if len(violations) > 0 {
		t.Errorf("littéral d'extension de rejeu hors de %s — utiliser title.ReplayArtifactExt, "+
			"ReplayArtifactLegacyExt, ReplayDerivationsMarkSuffix, ReplaySidecarExt ou "+
			"title.IsReplayArtifactFile :\n  %s", fichierDesExtensionsDeRejeu, strings.Join(violations, "\n  "))
	}
}
