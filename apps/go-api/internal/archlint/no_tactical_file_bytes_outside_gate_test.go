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
// `os.ReadFile` ni `os.WriteFile` sur un chemin de fichier tactique. Un appel est tenu pour tel quand le texte de
// son premier argument, ou un identifiant affecté depuis une expression qui nomme le fichier tactique
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

// appelsFichierTactiqueInterdits : les fonctions de `os` qui lisent ou écrivent les octets d'un fichier.
var appelsFichierTactiqueInterdits = map[string]bool{"ReadFile": true, "WriteFile": true}

// appelsSurUnFichierTactique rend les appels `os.ReadFile` / `os.WriteFile` d'une source qui portent sur un
// fichier tactique, sous la forme « fichier:ligne os.X ».
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
			if !ok || pkg.Name != "os" || !appelsFichierTactiqueInterdits[sel.Sel.Name] {
				return true
			}
			arg := texteDeLExpression(s, call.Args[0])
			vise := motifFichierTactique.MatchString(arg)
			for id := range tainted {
				vise = vise || contientIdentifiant(arg, id)
			}
			if vise {
				out = append(out, fmt.Sprintf("%s:%d os.%s", s.Rel, s.Fset.Position(call.Pos()).Line, sel.Sel.Name))
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
