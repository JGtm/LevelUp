package filmcorpus

// objets_objectif_titre_test.go — LE MANIFESTE DU TITRE NOMME SES OBJETS D'OBJECTIF.
//
// Garde deplacee du paquet `replay` de la brique (plan D2, item 2.11, regle R1) : elle lit le
// manifeste de LevelUp (`replay_labels.toml`) et n'exerce que la surface exportee de la brique.
// L'autre moitie de la garde du drapeau, `TestLeDrapeauDuManifesteNeFaitJamaisUnSocle`, reste dans
// la brique : elle lit l'instantane du catalogue, pas le manifeste.

import (
	"testing"

	"levelup/go-api/internal/games/mappings"
)

// TestLeManifesteNommeSesObjetsDObjectifDansLesDeuxLangues : le nom vit dans le TOML, jamais en
// Go — et CHAQUE famille portee doit avoir au moins un representant nomme.
//
// LES DEUX FAMILLES SONT EXIGEES SEPAREMENT (2026-08-27). Compter globalement laisserait passer
// la disparition d'une famille entiere tant que l'autre survit : le jour ou l'entree du crane
// serait retiree du manifeste, l'exclusion des socles d'armes redeviendrait ACCIDENTELLE sur
// Oddball sans qu'aucun test ne l'annonce — exactement le defaut que ce fichier existe pour
// empecher sur le drapeau.
func TestLeManifesteNommeSesObjetsDObjectifDansLesDeuxLangues(t *testing.T) {
	labels := manifesteDuTitre(t)
	attendues := []string{mappings.ObjectiveFamilyFlag, mappings.ObjectiveFamilyBall}
	nb := map[string]int{}
	for id, o := range labels.ObjectiveObjects() {
		nb[o.Family]++
		if o.Label.Text.Pick("en") == "" || o.Label.Text.Pick("fr") == "" {
			t.Errorf("objet d'objectif %08X (famille %q) nomme dans une seule langue (en=%q fr=%q)",
				id, o.Family, o.Label.Text.Pick("en"), o.Label.Text.Pick("fr"))
		}
	}
	for _, f := range attendues {
		if nb[f] == 0 {
			t.Errorf("aucun objet d'objectif de famille %q au manifeste du titre — la chaine des "+
				"socles ne reconnaitrait plus cet objet, elle l'ecarterait par accident "+
				"(cf. replay_labels.toml, [[objective_objects]])", f)
		}
	}
}
