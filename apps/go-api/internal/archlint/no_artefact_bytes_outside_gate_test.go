// Package archlint — no_artefact_bytes_outside_gate_test.go : les octets d'un artefact de rejeu ne se
// lisent ni ne se testent que par la porte `internal/replayartifact`.
//
// POURQUOI. Un artefact est stocké `{short8}.json.gz` (gzip), et un parc ancien porte encore des
// `{short8}.json` bruts : lire le chemin canonique avec `os.ReadFile` rend du gzip à un décodeur JSON,
// et un `os.Stat` du chemin canonique répond « absent » sur un parc brut. La porte
// (`replayartifact.Read`, `ReadBytes`, `Locate`, `Exists`, `List`, `Copy`, `Remove`, `Write`) détecte la
// forme par les octets magiques et résout les deux noms. Un changement de stockage ne touche alors qu'un
// fichier.
//
// RÈGLE. Hors `internal/replayartifact/` (la porte) et `internal/replaybuild/` (qui possède en plus l'en-tête
// `.meta`, la marque de dérivation et la purge d'un match),
// aucun fichier de production de `apps/go-api` n'appelle `os.ReadFile`, `os.Open`, `os.OpenFile`,
// `os.Stat`, `os.Lstat` ni `os.WriteFile` sur un chemin d'artefact. Un appel est tenu pour un appel sur un
// artefact quand le texte de son argument ou le nom de la fonction qui le porte contient « artefact » ou
// « artifact ». L'allowlist est vide. Les `_test.go` sont exclus (un témoin fabrique ses fichiers) ; les
// briques de `libs/` n'importent pas le serveur et lisent leurs témoins localement.
package archlint

import (
	"fmt"
	"go/ast"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// racinesPorteDesArtefacts : les paquets qui possèdent les fichiers d'artefact.
var racinesPorteDesArtefacts = []string{"internal/replayartifact/", "internal/replaybuild/"}

// allowlistOctetsArtefact : appels tolérés hors de la porte. VIDE ; ne jamais y ajouter sans justification datée.
var allowlistOctetsArtefact = map[string]bool{}

// appelsFichierInterdits : les fonctions de `os` qui touchent aux octets ou à la présence d'un fichier.
var appelsFichierInterdits = map[string]bool{
	"ReadFile": true, "Open": true, "OpenFile": true, "Stat": true, "Lstat": true, "WriteFile": true,
}

var motifArtefact = regexp.MustCompile(`(?i)art[ei]fact`)

// appelsSurUnArtefact rend les appels `os.X(...)` d'une source dont l'argument ou la fonction englobante
// désigne un artefact, sous la forme « fichier:ligne os.X ».
func appelsSurUnArtefact(s goSource) []string {
	var out []string
	for _, d := range s.File.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
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
			if !ok || pkg.Name != "os" || !appelsFichierInterdits[sel.Sel.Name] {
				return true
			}
			arg := texteDeLExpression(s, call.Args[0])
			if motifArtefact.MatchString(arg) || motifArtefact.MatchString(fn.Name.Name) {
				out = append(out, fmt.Sprintf("%s:%d os.%s", s.Rel, s.Fset.Position(call.Pos()).Line, sel.Sel.Name))
			}
			return true
		})
	}
	return out
}

func texteDeLExpression(s goSource, e ast.Node) string {
	a, b := s.Fset.Position(e.Pos()).Offset, s.Fset.Position(e.End()).Offset
	if a < 0 || b > len(s.Src) || a >= b {
		return ""
	}
	return string(s.Src[a:b])
}

func TestNoArtefactBytesOutsideGate_ReconnaitLesCopies(t *testing.T) {
	copie := parseGoSnippet(t, "cmd/x/main.go", `package main
import "os"
func lireUnArtefact(path string) ([]byte, error) { return os.ReadFile(path) }
func autre(p string) { _, _ = os.Stat(replayArtifactPath(p)) }
`)
	if got := appelsSurUnArtefact(copie); len(got) != 2 {
		t.Errorf("le garde-rail doit voir les deux appels, vu %v", got)
	}
	innocent := parseGoSnippet(t, "cmd/x/main.go", `package main
import "os"
func lireConfig(path string) ([]byte, error) { return os.ReadFile(path) }
func appelDeLaPorte(p string) ([]byte, error) { return replayartifact.ReadBytes(p) }
`)
	if got := appelsSurUnArtefact(innocent); len(got) != 0 {
		t.Errorf("faux positif : %v", got)
	}
}

func TestNoArtefactBytesOutsideGate(t *testing.T) {
	var violations []string
	for _, s := range goSourcesUnder(t, "cmd", "internal") {
		if slices.ContainsFunc(racinesPorteDesArtefacts, func(r string) bool { return strings.HasPrefix(s.Rel, r) }) {
			continue
		}
		for _, v := range appelsSurUnArtefact(s) {
			if !allowlistOctetsArtefact[v] {
				violations = append(violations, v)
			}
		}
	}
	if len(violations) > 0 {
		t.Errorf("lecture, test de présence ou écriture d'un artefact de rejeu hors de la porte — passer par "+
			"replayartifact.ReadBytes / Locate / Exists / List / Copy "+
			"(internal/replayartifact/gate.go) :\n  %s", strings.Join(violations, "\n  "))
	}
}
