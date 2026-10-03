package filmproc

// runner_raison_test.go — LA RAISON D'UN REFUS TRAVERSE LE TUBE (lot J2.12, constat OPS-5,
// decision DT-5, 2026-09-26).
//
// Le code de sortie ne dit que « ecarte » : il ne dit pas POURQUOI. Le parent rangeait donc tout
// refus sous « carte hors catalogue ». La raison voyage desormais par une ligne de protocole, sur
// le modele du pic memoire : un JETON OPAQUE pour ce paquet, dont le vocabulaire vit chez
// l'appelant (`replaychild`).

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// TestRunRendLaRaisonDUnEnfantReel — de bout en bout, sur un vrai processus : le jeton emis par
// l'enfant arrive dans `Result.Raison`, et la ligne de protocole ne fuit pas dans le journal.
func TestRunRendLaRaisonDUnEnfantReel(t *testing.T) {
	const jeton = "raison_de_test"
	var journal strings.Builder
	r, err := NewRunner(t.TempDir(), &journal)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	args := append(argsEnfantDeTest(CodeSkipped), fmt.Sprintf("-filmproc-aide-raison=%s", jeton))
	res := r.Run(context.Background(), args)
	if res.Issue != IssueSkipped || res.Raison != jeton {
		t.Fatalf("Result{Issue:%v, Raison:%q}, attendu {%v, %q}", res.Issue, res.Raison, IssueSkipped, jeton)
	}
	if strings.Contains(journal.String(), raisonMarker) {
		t.Errorf("la ligne de protocole de la raison a fuit dans le journal :\n%s", journal.String())
	}
	sans := r.Run(context.Background(), argsEnfantDeTest(CodeSkipped))
	if sans.Raison != "" {
		t.Errorf("un enfant sans raison rend Raison=%q, attendu vide", sans.Raison)
	}
}

// TestParseRaisonFormesLimites — reconnu malgre les espaces de bord ; jamais precede de texte ;
// jamais vide ni fait de plusieurs mots (un jeton est UN mot).
func TestParseRaisonFormesLimites(t *testing.T) {
	if j, ok := parseRaison("  " + raisonMarker + "carte_hors_catalogue "); !ok || j != "carte_hors_catalogue" {
		t.Errorf("marqueur borde d'espaces : %q, %v", j, ok)
	}
	for _, ligne := range []string{
		"prefixe " + raisonMarker + "x",
		raisonMarker,
		raisonMarker + "deux mots",
	} {
		if j, ok := parseRaison(ligne); ok {
			t.Errorf("%q pris pour du protocole (jeton %q)", ligne, j)
		}
	}
}
