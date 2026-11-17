// Package archlint — no_tactical_file_bytes_outside_gate_test.go : les octets d'un fichier tactique (le raster
// d'occupation d'un match, `rasters/{short8}.json.gz`) ne se lisent ni ne s'écrivent que par la porte
// `internal/replayartifact`.
//
// POURQUOI. Le fichier tactique est stocké en gzip, comme l'artefact dont il dérive : `os.ReadFile` sur son
// chemin rend du gzip à un décodeur JSON, `os.WriteFile` en dépose du JSON brut que la porte ne lit pas
// (aucune lecture des anciens `.json`). La porte (`replayartifact.Read`, `ReadBytes`, `Write`) fixe seule le
// format.
//
// RÈGLE. Hors `internal/replayartifact/`, aucun fichier Go de `apps/go-api`, test compris, n'appelle
// aucune des fonctions de [appelsFichierTactiqueInterdits] (`os.ReadFile`, `Open`, `OpenFile`, `Create`,
// `WriteFile`, `atomicfile.WriteFile`) sur un chemin de fichier tactique. Un appel est tenu pour tel quand le texte
// de son argument de chemin, ou un identifiant affecté depuis une expression qui nomme le fichier tactique
// ([motifFichierTactique], même passe de contamination que `no_artefact_bytes_outside_gate_test.go`), en
// désigne un. L'allowlist est vide. Le chemin du fichier d'une version ancienne (`TacticalRasterLegacyPath`,
// jamais lu) n'est pas un chemin de fichier tactique : un test le dépose en JSON brut pour prouver qu'on ne le lit
// pas.
//
// LIMITE : la règle est lexicale, dans la seule fonction. Un chemin reçu par un paramètre, un champ ou un retour
// de fonction qui ne le nomme pas passe ; elle attrape la dérive probable (un chemin obtenu de
// `TacticalRasterPath` puis lu à la main), pas une obfuscation.
package archlint

import (
	"fmt"
	"go/ast"
	"regexp"
	"strings"
	"testing"
)

// racinePorteDesFichiersTactiques : le paquet qui possède le format du fichier tactique.
const racinePorteDesFichiersTactiques = "apps/go-api/internal/replayartifact/"

// allowlistOctetsFichierTactique : appels tolérés hors de la porte. VIDE ; ne jamais y ajouter sans
// justification datée.
var allowlistOctetsFichierTactique = map[string]bool{}

// motifFichierTactique : ce qui désigne un chemin de fichier tactique dans une expression.
var motifFichierTactique = regexp.MustCompile(
	`TacticalRasterPath|TacticalRasterDir|TacticalRasterFileIn|SousDossierRasters|EcrireSidecarRaster`)

// appelsFichierTactiqueInterdits : par paquet, les fonctions qui lisent, écrivent, ouvrent ou créent les octets
// d'un fichier : celles de `no_artefact_bytes_outside_gate_test.go` (`appelsFichierInterdits`) moins `Stat` et
// `Lstat`, plus `os.Create` et l'écrivain atomique de la maison (`atomicfile`), forme que l'écrivain du fichier
// tactique utilisait avant la porte. `Stat` et `Lstat` testent une présence, pas des octets, et ne sont pas
// visés : la règle porte sur le format du fichier, que seule la porte fixe.
var appelsFichierTactiqueInterdits = map[string]map[string]bool{
	"os": {
		"ReadFile": true, "Open": true, "OpenFile": true, "WriteFile": true, "Create": true,
	},
	"atomicfile": {"WriteFile": true, "WriteFileStrict": true},
}

// indexCheminAppelFichierTactique : position de l'argument qui porte le chemin, quand ce n'est pas le premier
// (`atomicfile.WriteFile(ctx, chemin, ...)`).
var indexCheminAppelFichierTactique = map[string]int{"atomicfile.WriteFile": 1}

// appelsSurUnFichierTactique rend les appels de [appelsFichierTactiqueInterdits] d'une source qui portent sur un
// fichier tactique, sous la forme « fichier:ligne paquet.X ».
func appelsSurUnFichierTactique(s goSource) []string {
	var out []string
	for _, d := range s.File.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		tainted := identifiantsContaminesPar(s, fn, motifFichierTactique)
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || !appelsFichierTactiqueInterdits[pkg.Name][sel.Sel.Name] {
				return true
			}
			idx := indexCheminAppelFichierTactique[pkg.Name+"."+sel.Sel.Name]
			if idx >= len(call.Args) {
				return true
			}
			arg := texteDeLExpression(s, call.Args[idx])
			vise := motifFichierTactique.MatchString(arg)
			for id := range tainted {
				vise = vise || contientIdentifiant(arg, id)
			}
			if vise {
				out = append(out, fmt.Sprintf("%s:%d %s.%s", s.Rel, s.Fset.Position(call.Pos()).Line, pkg.Name, sel.Sel.Name))
			}
			return true
		})
	}
	return out
}

func TestNoTacticalFileBytesOutsideGate_ReconnaitLesCopies(t *testing.T) {
	copie := parseGoSnippet(t, "cmd/x/main.go", `package main
import "os"
func lire(pr *title.PathResolver) ([]byte, error) {
	return os.ReadFile(pr.TacticalRasterPath("s", "m"))
}
func ecrire(pr *title.PathResolver, raw []byte) error {
	chemin := pr.TacticalRasterPath("s", "m")
	cible := chemin
	return os.WriteFile(cible, raw, 0o644)
}
func parcourir(pr *title.PathResolver) {
	for _, f := range []string{pr.TacticalRasterPath("s", "m")} {
		_, _ = os.ReadFile(f)
	}
}
`)
	if got := appelsSurUnFichierTactique(copie); len(got) != 3 {
		t.Errorf("le garde-rail doit voir les trois appels, vu %v", got)
	}
	// Chaque autre forme d'accès aux octets d'un fichier tactique fait rougir le garde-rail.
	autresFormes := map[string]string{
		"os.Open":                    `f, err := os.Open(pr.TacticalRasterPath("s", "m")); _ = f; return err`,
		"os.OpenFile":                `f, err := os.OpenFile(pr.TacticalRasterPath("s", "m"), os.O_RDONLY, 0); _ = f; return err`,
		"os.Create":                  `f, err := os.Create(pr.TacticalRasterPath("s", "m")); _ = f; return err`,
		"atomicfile.WriteFile":       `return atomicfile.WriteFile(ctx, pr.TacticalRasterPath("s", "m"), raw, 0o644)`,
		"atomicfile.WriteFileStrict": `return atomicfile.WriteFileStrict(pr.TacticalRasterPath("s", "m"), raw, 0o644)`,
	}
	for forme, corps := range autresFormes {
		src := parseGoSnippet(t, "cmd/x/main.go", "package main\nimport (\"context\"; \"os\")\n"+
			"func f(ctx context.Context, pr *title.PathResolver, raw []byte) error {\n"+corps+"\n}\n")
		if got := appelsSurUnFichierTactique(src); len(got) != 1 {
			t.Errorf("%s sur un fichier tactique doit rougir le garde-rail, vu %v", forme, got)
		}
	}
	// Une forme voisine mais sur un chemin qui n'est pas tactique ne rougit pas.
	neutre := parseGoSnippet(t, "cmd/x/main.go", `package main
import "os"
func f(ctx context.Context, p string, raw []byte) error {
	f, err := os.Open(p)
	_ = f
	if err != nil {
		return err
	}
	return atomicfile.WriteFile(ctx, p, raw, 0o644)
}
`)
	if got := appelsSurUnFichierTactique(neutre); len(got) != 0 {
		t.Errorf("faux positif : %v", got)
	}
	innocent := parseGoSnippet(t, "cmd/x/main.go", `package main
import "os"
func porte(pr *title.PathResolver) ([]byte, error) {
	return replayartifact.ReadBytes(pr.TacticalRasterPath("s", "m"))
}
func presence(pr *title.PathResolver) error {
	_, err := os.Stat(pr.TacticalRasterPath("s", "m"))
	return err
}
func ancien(pr *title.PathResolver, raw []byte) error {
	return os.WriteFile(pr.TacticalRasterLegacyPath("s", "m"), raw, 0o644)
}
func config(path string) ([]byte, error) { return os.ReadFile(path) }
`)
	if got := appelsSurUnFichierTactique(innocent); len(got) != 0 {
		t.Errorf("faux positif : %v", got)
	}
}

func TestNoTacticalFileBytesOutsideGate(t *testing.T) {
	var violations []string
	vus := 0
	balayerGoTexte(t, func(rel, texte string) {
		if !strings.HasPrefix(rel, "apps/go-api/") || strings.HasPrefix(rel, racinePorteDesFichiersTactiques) {
			return
		}
		vus++
		for _, v := range appelsSurUnFichierTactique(parseGoSnippet(t, rel, texte)) {
			if !allowlistOctetsFichierTactique[v] {
				violations = append(violations, v)
			}
		}
	})
	if vus == 0 {
		t.Fatal("aucun fichier Go de apps/go-api balayé : le garde-rail ne garde rien")
	}
	if len(violations) > 0 {
		t.Errorf("lecture ou écriture d'un fichier tactique hors de la porte — passer par replayartifact.ReadBytes / "+
			"Write (internal/replayartifact/gate.go) :\n  %s", strings.Join(violations, "\n  "))
	}
}
