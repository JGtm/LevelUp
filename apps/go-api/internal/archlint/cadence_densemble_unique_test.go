package archlint

// cadence_densemble_unique_test.go — LA CADENCE D'UN ENSEMBLE DE MATCHS SE CALCULE EN UN SEUL ENDROIT
// (règle 6 de CLAUDE.md).
//
// « Somme des comptes / somme des minutes jouées » avait deux écritures Go : le groupe « Rythme » de
// Tendances et les dégâts par minute de l'escouade, chacun avec son propre filtre des matchs sans durée
// ou sans compte. La règle vit dans `analysis.PooledPerMinute` (jeu de cas partagé avec le web :
// analysis/testdata/par_minute_cas.json). Ce garde-rail exige que les deux appelants l'utilisent et
// interdit d'y réécrire le filtre : tester une durée contre zéro dans ces deux paquets, c'est refaire la
// règle d'exclusion que l'autre n'aurait plus.

import (
	"regexp"
	"strings"
	"testing"
)

// appelantsDeLaCadence : les fichiers qui rapportent un ensemble de matchs à la minute.
var appelantsDeLaCadence = []string{
	"internal/analysis/trends/indicators_pace.go",
	"internal/service/teammates/teammates_squad_per_minute.go",
}

// motifFiltreDeDuree : un test de durée contre zéro (`m.Seconds <= 0`, `secs <= 0`, `seconds < 1`...).
var motifFiltreDeDuree = regexp.MustCompile(`\b\w*[Ss]ec(s|onds)\s*(<=|<|==)\s*[01]\b`)

// filtresDeDuree rend les lignes de code (hors commentaire) qui testent une durée contre zéro.
func filtresDeDuree(src goSource) []string {
	var out []string
	for i, line := range strings.Split(string(src.Src), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		if motifFiltreDeDuree.MatchString(line) {
			out = append(out, src.Rel+":"+itoa(i+1)+"  "+strings.TrimSpace(line))
		}
	}
	return out
}

func TestLaCadenceDEnsembleSeCalculeEnUnSeulEndroit(t *testing.T) {
	var violations []string
	appelants := map[string]bool{}
	for _, src := range goSourcesUnder(t, "internal/analysis/trends", "internal/service/teammates") {
		violations = append(violations, filtresDeDuree(src)...)
		for _, a := range appelantsDeLaCadence {
			if src.Rel == a && strings.Contains(string(src.Src), "PooledPerMinute(") {
				appelants[a] = true
			}
		}
	}
	for _, a := range appelantsDeLaCadence {
		if !appelants[a] {
			t.Errorf("%s n'appelle plus analysis.PooledPerMinute : la règle « somme sur somme » a-t-elle été "+
				"réécrite ou le fichier déplacé ? Le garde-rail ne vérifie plus rien", a)
		}
	}
	if len(violations) > 0 {
		t.Errorf("la durée d'un match est testée contre zéro dans un paquet qui appelle analysis.PooledPerMinute "+
			"(%d) : le filtre des matchs sans durée vit dans cette fonction :\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
}

// Auto-preuve : le motif reconnaît l'ancien filtre sous ses deux formes et laisse en paix un commentaire.
func TestFiltresDeDuree_AutoPreuve(t *testing.T) {
	src := parseGoSnippet(t, "internal/analysis/trends/x.go", "package trends\n\nfunc f(m Match, secs int) {\n"+
		"\tif m.Seconds <= 0 {\n\t\treturn\n\t}\n\tif secs <= 0 {\n\t\treturn\n\t}\n\t// secs <= 0\n}\n")
	if got := filtresDeDuree(src); len(got) != 2 {
		t.Fatalf("deux filtres attendus, obtenu %d : %v", len(got), got)
	}
}
