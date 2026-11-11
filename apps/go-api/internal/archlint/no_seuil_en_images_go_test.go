// Package archlint — no_seuil_en_images_go_test.go : un seuil temporel du rejeu s'écrit en
// MILLISECONDES, jamais en nombre d'images.
//
// POURQUOI. Un seuil de « 2 images » vaut 200 ms à 100 ms par image et 66 ms à 33 ms par image :
// changer la cadence du rejeu raccourcirait en silence chaque tolérance, et les corrections du
// décodeur rencontreraient des seuils cachés. Une durée en millisecondes, convertie avec le pas
// publié par le document (`frameIntervalMs`), laisse la cadence redevenir un réglage.
//
// RÈGLE. Aucune constante ni variable à valeur numérique dont le nom finit par `Frames` ou
// `Images` dans `internal/analysis/tactical`, `internal/replayverite` et
// `internal/sync/replayartifacts` (fichiers de production). L'allowlist est vide : ne jamais y
// ajouter sans justification datée.
//
// LIMITE. Règle lexicale sur le NOM : une constante en images nommée autrement (`tolerance`, `n`)
// passe ; c'est la revue et les tests « même durée à 100 et à 33 ms » qui la tiennent. Le garde-rail
// ne balaie PAS `libs/film/replay` : ce périmètre est tenu par la règle du module du décodeur.
// Les outils de recherche sous `cmd/` ne sont pas balayés.
package archlint

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"
)

// racinesSeuilsEnMillisecondes : les périmètres Go balayés (chemins de garde-rail).
var racinesSeuilsEnMillisecondes = []string{
	"internal/analysis/tactical",
	"internal/replayverite",
	"internal/sync/replayartifacts",
}

// allowlistSeuilEnImages : « fichier:nom » toléré. VIDE.
var allowlistSeuilEnImages = map[string]string{}

// nomEnImages dit qu'un identifiant se lit comme un nombre d'images.
func nomEnImages(nom string) bool {
	return strings.HasSuffix(nom, "Frames") || strings.HasSuffix(nom, "Images")
}

// valeurNumerique dit qu'une expression ne contient aucun littéral numérique hors d'un appel : un
// nombre posé en dur (`2`, `1.5`, `2 * 3`, `-1`).
func valeurNumerique(e ast.Expr) bool {
	numerique := false
	ast.Inspect(e, func(n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); ok && (lit.Kind == token.INT || lit.Kind == token.FLOAT) {
			numerique = true
		}
		return true
	})
	return numerique
}

// seuilsEnImages rend « fichier:nom » de chaque constante ou variable numérique suffixée
// `Frames` / `Images` d'une source.
func seuilsEnImages(s goSource) []string {
	var out []string
	for _, d := range s.File.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok || (gen.Tok != token.CONST && gen.Tok != token.VAR) {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, nom := range vs.Names {
				if !nomEnImages(nom.Name) || i >= len(vs.Values) || !valeurNumerique(vs.Values[i]) {
					continue
				}
				out = append(out, s.Rel+":"+nom.Name)
			}
		}
	}
	return out
}

func TestAucunSeuilEnImagesCoteGo(t *testing.T) {
	for _, s := range goSourcesUnder(t, racinesSeuilsEnMillisecondes...) {
		for _, trouve := range seuilsEnImages(s) {
			if _, ok := allowlistSeuilEnImages[trouve]; ok {
				continue
			}
			t.Errorf("%s : un seuil temporel s'écrit en millisecondes (suffixe `MS`), converti avec le pas du "+
				"document ; un nombre d'images changerait de durée avec la cadence", trouve)
		}
	}
}

// TestSeuilsEnImages_AutoPreuve : le détecteur voit les formes visées et laisse passer les autres.
func TestSeuilsEnImages_AutoPreuve(t *testing.T) {
	src := `package p
const (
	toleranceImages = 2
	margeFrames float64 = 1.5
	doubleFrames = 2 * 3
	negImages = -1
	toleranceMS = 200
	copieImages = toleranceMS
	nbMaxImagesParCle = 4
)
var cadenceFrames = 3
`
	got := seuilsEnImages(parseGoSnippet(t, "p.go", src))
	veut := map[string]bool{
		"p.go:toleranceImages": true, "p.go:margeFrames": true, "p.go:doubleFrames": true,
		"p.go:negImages": true, "p.go:cadenceFrames": true,
	}
	if len(got) != len(veut) {
		t.Fatalf("détecté %v, attendu %v", got, veut)
	}
	for _, g := range got {
		if !veut[g] {
			t.Errorf("faux positif %s", g)
		}
	}
}
