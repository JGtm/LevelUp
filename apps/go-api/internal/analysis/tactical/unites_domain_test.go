package tactical

// unites_domain_test.go — LES UNITES NE DOIVENT PAS DIVERGER.
//
// `domain.SidecarRasterCourant` verifie qu'un sidecar porte la bonne grille et le bon pas
// d'echantillonnage. Il ne peut pas lire les constantes de CE paquet : `domain` est la
// couche basse, et c'est `tactical` qui l'importe — l'inverse ferait un cycle. Les deux
// valeurs y sont donc figees en dur.
//
// Ce test est le prix de cette duplication, et il est paye ICI, chez le PROPRIETAIRE des
// constantes : changer le pas de la grille ou celui de l'echantillonnage sans mettre a
// jour `domain` ferait ecarter TOUT le parc de sidecars en silence — la page deviendrait
// vide sans qu'aucune erreur ne soit levee.

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func TestUnitesDuSidecarNonDivergentes(t *testing.T) {
	if domain.TacticalRasterPasM != PasParDefautM {
		t.Fatalf("domain.TacticalRasterPasM = %v, tactical.PasParDefautM = %v : les deux "+
			"unites ont diverge, tout le parc de sidecars serait ecarte en silence",
			domain.TacticalRasterPasM, PasParDefautM)
	}
	if domain.TacticalRasterPasEchantillonMs != PasOccupationMs {
		t.Fatalf("domain.TacticalRasterPasEchantillonMs = %v, tactical.PasOccupationMs = %v : "+
			"les deux unites ont diverge, tout le parc de sidecars serait ecarte en silence",
			domain.TacticalRasterPasEchantillonMs, PasOccupationMs)
	}
}
