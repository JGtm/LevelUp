// Package archlint — no_error_text_classification_test.go : aucun classement d'une erreur par son
// TEXTE sur le chemin de la cuisson hors processus (lot J2.12, constat OPS-5, decision DT-5,
// 2026-09-26).
//
// `replaychild` et `sync/replayartifacts` classaient les refus de l'enfant de cuisson par
// `strings.Contains(err.Error(), ErrX.Error())`. Le texte d'une erreur n'est pas un contrat : un
// refus non reconnu (`ErrFilmNonFinalise` dans l'enfant) tombait en echec, et le parent rangeait
// tout refus sous « carte hors catalogue ». La raison traverse desormais le tube en JETON
// (`filmproc.EmitRaison`), et chaque cote classe par `errors.Is`. Ce ratchet interdit le retour
// du motif dans les deux paquets.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// paquetsSansClassementParTexte : les paquets (relatifs a apps/go-api) tenus par ce ratchet.
var paquetsSansClassementParTexte = []string{
	"internal/replaychild",
	"internal/sync/replayartifacts",
}

// classementParTexteRE : une fonction de `strings` appliquee au texte d'une erreur.
var classementParTexteRE = regexp.MustCompile(
	`strings\.(Contains|HasPrefix|HasSuffix|Index|EqualFold)\([^)]*\.Error\(\)`)

func TestAucunClassementDErreurParSonTexteDansLaCuisson(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	lus := 0
	for _, paquet := range paquetsSansClassementParTexte {
		fichiers, err := filepath.Glob(filepath.Join(goAPIRoot, filepath.FromSlash(paquet), "*.go"))
		if err != nil {
			t.Fatalf("glob %s : %v", paquet, err)
		}
		for _, f := range fichiers {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			src, err := os.ReadFile(f) //nolint:gosec // fichier source du module
			if err != nil {
				t.Fatalf("%s : %v", f, err)
			}
			lus++
			for i, ligne := range strings.Split(string(src), "\n") {
				// Un commentaire peut CITER l ancien motif : seul le code est tenu.
				if !strings.HasPrefix(strings.TrimSpace(ligne), "//") && classementParTexteRE.MatchString(ligne) {
					t.Errorf("%s:%d : erreur classee par son TEXTE — classer par errors.Is "+
						"(raison du refus : jeton filmproc.EmitRaison, cf. replaychild/raison.go)",
						filepath.ToSlash(f), i+1)
				}
			}
		}
	}
	if lus < 10 {
		t.Fatalf("ratchet vacant : %d fichier(s) lu(s) dans %v", lus, paquetsSansClassementParTexte)
	}
}
