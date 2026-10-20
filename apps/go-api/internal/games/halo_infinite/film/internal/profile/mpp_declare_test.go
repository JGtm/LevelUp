package profile

import (
	"fmt"
	"slices"
	"testing"
)

// TestMPPPourTailleDeclaree joue la regle sur ses trois issues, et sur un archetype hors de la cle.
func TestMPPPourTailleDeclaree(t *testing.T) {
	cas := []struct {
		nom  string
		ti   uint32
		n1   int32
		w    MPPWidths
		prov ProvenanceMPP
		ok   bool
	}{
		{"ti37 taille courante", 37, 0x68, MPPWidths{Lead: 9, Index: 5}, MPPRelu, true},
		{"ti37 taille courante - 4", 37, 0x68 - 4, MPPWidths{Lead: 8, Index: 3}, MPPPresumeParMesure, true},
		{"ti35 taille courante - 4", 35, 0x98 - 4, MPPWidths{Lead: 8, Index: 3}, MPPPresumeParMesure, true},
		{"ti37 autre taille", 37, 0x68 + 4, MPPWidths{}, MPPNonDeclare, false},
		{"ti37 taille nulle", 37, 0, MPPWidths{}, MPPNonDeclare, false},
		{"ti36 taille relue, regle non mesuree", 36, 0x60, MPPWidths{}, MPPNonDeclare, false},
		{"ti3 sans bloc MPP", 3, 0x68, MPPWidths{}, MPPNonDeclare, false},
	}
	for _, c := range cas {
		w, prov, ok := MPPPourTailleDeclaree(c.ti, c.n1)
		if w != c.w || prov != c.prov || ok != c.ok {
			t.Errorf("%s : rendu (%v, %v, %v), attendu (%v, %v, %v)", c.nom, w, prov, ok, c.w, c.prov, c.ok)
		}
	}
}

// decoupagesPresumesParMesureGeles : CE QUE LA REGLE PRESUME PAR MESURE, archetype par archetype —
// la taille declaree qui la declenche et le decoupage rendu. Une ligne qui change est une decision
// (provenance dans l en-tete de `mpp_declare.go`), pas une mise a jour.
var decoupagesPresumesParMesureGeles = []string{
	"ti=35 n1=148 -> 8/3",
	"ti=37 n1=100 -> 8/3",
	"ti=38 n1=100 -> 8/3",
	"ti=40 n1=172 -> 8/3",
	"ti=41 n1=208 -> 8/3",
	"ti=42 n1=164 -> 8/3",
	"ti=43 n1=92 -> 8/3",
}

// TestDecoupagesPresumesParMesureGeles fige la liste des decoupages que la regle presume par mesure.
func TestDecoupagesPresumesParMesureGeles(t *testing.T) {
	var vus []string
	for ti, courante := range tailleEtatDeCreationCourante {
		n1 := courante - ecartDeTailleDuDecoupageAncien
		if w, prov, ok := MPPPourTailleDeclaree(ti, n1); ok && prov == MPPPresumeParMesure {
			vus = append(vus, fmt.Sprintf("ti=%d n1=%d -> %v", ti, n1, w))
		}
	}
	slices.Sort(vus)
	if !slices.Equal(vus, decoupagesPresumesParMesureGeles) {
		t.Fatalf("decoupages presumes par mesure :\n%v\nfiges :\n%v", vus, decoupagesPresumesParMesureGeles)
	}
}

// TestLeDecoupageReluEstCeluiDuFormatCourant : la taille courante designe le decoupage que la table
// par format donne au format de l executable desassemble.
func TestLeDecoupageReluEstCeluiDuFormatCourant(t *testing.T) {
	duFormat, ok := MPPPourFormat(formatHI1120Et1130)
	if !ok {
		t.Fatalf("format %d sans decoupage", formatHI1120Et1130)
	}
	for ti, courante := range tailleEtatDeCreationCourante {
		if !CleDuDecoupageMPP(ti) {
			continue
		}
		if w, prov, ok := MPPPourTailleDeclaree(ti, courante); !ok || prov != MPPRelu || w != duFormat {
			t.Errorf("ti=%d : (%v, %v, %v), attendu (%v, relu, vrai)", ti, w, prov, ok, duFormat)
		}
	}
}
