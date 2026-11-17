package archlint

// lead_share_lecture_unique_test.go — LA MAINMISE SE LIT ET SE POSE EN UN SEUL ENDROIT (règle 6 de
// CLAUDE.md).
//
// Quatre services servent la Mainmise (vue du match, page Sessions, Séries temporelles, Escouade). Le
// bloc « charger, journaliser sauf capability absente, poser les deux pointeurs » avait été recopié
// dans chacun. La règle vit dans `service/leadshareload` (`Load`, `Pointers`) ; ce garde-rail interdit
// d'en réécrire la pose : copier `part, retournements := v.Share, v.Changes` pour en prendre
// l'adresse, c'est refaire une décision de dégradation que les autres services ne partageraient plus.

import (
	"regexp"
	"strings"
	"testing"
)

// proprietaireDeLaMainmise : le seul fichier de production qui pose les deux pointeurs.
const proprietaireDeLaMainmise = "internal/service/leadshareload/leadshareload.go"

// motifPoseMainmise : la copie des deux champs d'une part mesurée dans deux variables locales, sous
// n'importe quels noms (`a, b := v.Share, v.Changes`).
var motifPoseMainmise = regexp.MustCompile(`:=\s*\w+\.Share,\s*\w+\.Changes\b`)

// poseursDeMainmise rend les lignes de code (hors commentaire) du fichier qui copient une part
// mesurée dans deux variables locales.
func poseursDeMainmise(src goSource) []string {
	var out []string
	for i, line := range strings.Split(string(src.Src), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		if motifPoseMainmise.MatchString(line) {
			out = append(out, src.Rel+":"+itoa(i+1)+"  "+strings.TrimSpace(line))
		}
	}
	return out
}

func TestLaMainmiseSePoseEnUnSeulEndroit(t *testing.T) {
	var violations []string
	vuChezLeProprietaire := false
	for _, src := range goSourcesUnder(t, "internal") {
		trouves := poseursDeMainmise(src)
		if src.Rel == proprietaireDeLaMainmise {
			vuChezLeProprietaire = len(trouves) > 0
			continue
		}
		violations = append(violations, trouves...)
	}
	if !vuChezLeProprietaire {
		t.Fatalf("%s ne pose plus les pointeurs de la Mainmise : un helper a-t-il été renommé ou "+
			"déplacé ? Le garde-rail ne vérifie plus rien", proprietaireDeLaMainmise)
	}
	if len(violations) > 0 {
		t.Errorf("la pose de `lead_share` / `lead_changes` est recopiée hors de %s : appeler "+
			"leadshareload.Pointers (et leadshareload.Load pour la lecture) :\n  %s",
			proprietaireDeLaMainmise, strings.Join(violations, "\n  "))
	}
}

// Auto-preuve : le motif reconnaît la copie sous des noms différents et laisse en paix un commentaire.
func TestPoseursDeMainmise_AutoPreuve(t *testing.T) {
	src := parseGoSnippet(t, "internal/service/x.go", "package service\n\nfunc f() {\n"+
		"\tpart, retours := v.Share, v.Changes\n\t// a, b := v.Share, v.Changes\n}\n")
	if got := poseursDeMainmise(src); len(got) != 1 {
		t.Fatalf("une seule copie attendue, obtenu %d : %v", len(got), got)
	}
}
