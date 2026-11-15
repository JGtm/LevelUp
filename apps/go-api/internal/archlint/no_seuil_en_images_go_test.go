// Package archlint — no_seuil_en_images_go_test.go : un seuil temporel du rejeu s'écrit en
// MILLISECONDES, jamais en nombre d'images.
//
// POURQUOI. Un seuil de « 2 images » vaut 200 ms à 100 ms par image et 66 ms à 33 ms par image :
// changer la cadence du rejeu raccourcirait en silence chaque tolérance, et les corrections du
// décodeur rencontreraient des seuils cachés. Une durée en millisecondes, convertie avec le pas
// publié par le document (`frameIntervalMs`), laisse la cadence redevenir un réglage.
//
// RÈGLE. Aucune constante ni variable à valeur numérique dont le nom finit par `Frames` ou
// `Images` dans `internal/analysis/tactical`, `internal/replayverite`,
// `internal/sync/replayartifacts` et `libs/film/replay` (fichiers de production). Seule exception
// admise : un nombre d'images qui est la GRILLE elle-même, donc le même à toute cadence (un pas
// d'échantillonnage de 1, une marge de 0). Elle s'inscrit dans l'allowlist, avec sa date et sa
// raison ; tout autre nombre d'images est une durée, qui s'écrit en millisecondes.
//
// LIMITE. Règle lexicale sur le NOM : une constante en images nommée autrement (`tolerance`, `n`)
// passe ; c'est la revue et les tests « même durée à 100 et à 33 ms » qui la tiennent. Une
// constante DÉRIVÉE d'une durée (`DefaultMaxGapFrames = DefaultMaxGapMS / DefaultFrameIntervalMS`)
// n'a aucun littéral numérique et passe : elle suit la cadence. Les outils de recherche sous
// `cmd/` ne sont pas balayés.
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
	"libs/film/replay",
}

// allowlistSeuilEnImages : « fichier:nom » toléré, avec la date et la raison. Deux entrées, toutes deux la
// grille elle-même : aucune ne change de durée quand la cadence change.
var allowlistSeuilEnImages = map[string]string{
	"libs/film/replay/vehicle_tracks.go:vehicleSampleStrideFrames": "2026-11-11 : pas d'échantillonnage des " +
		"véhicules, « un point par vie et par image » (valeur 1) : c'est la grille, pas une durée",
	"libs/film/replay/vehicle_tracks.go:vehicleRelayMarginFrames": "2026-11-11 : marge nulle au-delà de T1Max " +
		"(valeur 0) : zéro image est zéro milliseconde à toute cadence",
}

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
	vus := map[string]bool{}
	for _, s := range goSourcesUnder(t, racinesSeuilsEnMillisecondes...) {
		for _, trouve := range seuilsEnImages(s) {
			if _, ok := allowlistSeuilEnImages[trouve]; ok {
				vus[trouve] = true
				continue
			}
			t.Errorf("%s : un seuil temporel s'écrit en millisecondes (suffixe `MS`), converti avec le pas du "+
				"document ; un nombre d'images changerait de durée avec la cadence", trouve)
		}
	}
	for entree := range allowlistSeuilEnImages {
		if !vus[entree] {
			t.Errorf("%s : entrée d'allowlist sans objet, la retirer avec le code (CLAUDE.md règle 7)", entree)
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
