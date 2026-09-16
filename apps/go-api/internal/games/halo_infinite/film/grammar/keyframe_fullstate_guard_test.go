package grammar

// keyframe_fullstate_guard_test.go — LE CADRE D'IMAGE-CLE NE REDEVIENT PAS UNE OPTION (lot 1.4).
//
// # CE QUE CE RATCHET GARDE
//
// La decision D8 du PLAN_DECODEUR_FILM dit que le cadre d'etat complet (en-tete de 108 bits,
// deux mots de taille, etat par defaut) est LA lecture, pas un reglage. `WalkKeyframeFullState`
// ne prend donc plus d'option, et le seul bouton qui subsiste — `keyframeFullStateTemoin` — est
// NON EXPORTE et reserve aux deux temoins negatifs nommes (temoin de hasard a +1 bit, oracle
// `n2` sans etat par defaut).
//
// Rien n'empeche un lot futur de rouvrir ce bouton a du code de production DANS le paquet : le
// compilateur ne voit qu'un identifiant non exporte. Ce test le voit, lui. Il echoue si un
// fichier NON-test de `filmdec` cite `keyframeFullStateTemoin` ou `walkKeyframeFullState`
// ailleurs que dans le fichier qui les declare.
//
// LA PREUVE QU'IL MORD : ajouter un appel a `walkKeyframeFullState` dans n'importe quel
// `*.go` non-test du paquet autre que `keyframe_fullstate_loop.go` rougit ce test.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// temoinFichierDeclarant est le SEUL fichier non-test qui a le droit de citer le bouton.
const temoinFichierDeclarant = "keyframe_fullstate_loop.go"

// temoinIdentifiantsGardes : les identifiants qui ne doivent pas fuir hors des instruments.
func temoinIdentifiantsGardes() []string {
	return []string{"keyframeFullStateTemoin", "walkKeyframeFullState("}
}

// TestCadreImageCleSansOptionEnProduction : le bouton des temoins reste aux temoins.
func TestCadreImageCleSansOptionEnProduction(t *testing.T) {
	entrees, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	vus := 0
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		vus++
		if nom == temoinFichierDeclarant {
			continue
		}
		src, err := os.ReadFile(filepath.Clean(nom))
		if err != nil {
			t.Fatalf("lecture de %s : %v", nom, err)
		}
		for _, id := range temoinIdentifiantsGardes() {
			if strings.Contains(string(src), id) {
				t.Errorf("%s cite %q : le cadre d'image-cle est redevenu une OPTION de production.\n"+
					"La lecture de production est `WalkKeyframeFullState(pay, recBit, reg)` et elle "+
					"n'a pas de variante (decision D8). Le bouton `keyframeFullStateTemoin` est "+
					"reserve aux deux temoins negatifs nommes, dans des fichiers `_test.go`.",
					nom, id)
			}
		}
	}
	if vus < 50 {
		t.Fatalf("seulement %d fichiers non-test lus dans filmdec : le balayage ne porte sur rien", vus)
	}
}
