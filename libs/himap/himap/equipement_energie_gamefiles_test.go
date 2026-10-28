//go:build gamefiles

package himap

import "testing"

// TestEnergiesDEquipementDuJeu relit les tags du jeu installe et confronte les constantes aux baisses
// d energie mesurees au parc (238 films, 2026-10-14) : le repulseur coute 0,2 par usage et se ramasse
// a 0,6 (3 usages, releve Theater de 72b0a25e) ; capteur, ecran et traqueur 0,25 ; mur 0,33 ; champ de
// reparation 0,5 ; translocateur 1.
func TestEnergiesDEquipementDuJeu(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	paires, err := LireLesEnergiesDEquipement(root)
	if err != nil {
		t.Fatal(err)
	}
	parRamassable := map[uint32]EnergieDEquipement{}
	for _, p := range paires {
		parRamassable[p.Ramassable] = p
	}
	attendus := []struct {
		nom                string
		ramassable         uint32
		energie, variation float32
	}{
		{"repulseur", 0x7ca85adc, 0.6, -0.2},
		{"capteur", 0x72199cba, 0.5, -0.25},
		{"ecran", 0x4396db42, 0.5, -0.25},
		{"traqueur", 0x4744d742, 0.5, -0.25},
		{"mur", 0x2974c233, 0.66, -0.33},
		{"champ", 0x32d97758, 0.5, -0.5},
		{"translocateur", 0x730dc70f, 1, -1},
	}
	for _, a := range attendus {
		p, ok := parRamassable[a.ramassable]
		switch {
		case !ok:
			t.Errorf("%s (0x%08x) : aucune fiche ne le relie a un objet porte", a.nom, a.ramassable)
		case p.EnergieRamassable != a.energie || p.Variation != a.variation || p.Seuil <= 0:
			t.Errorf("%s : energie %v, variation %v, seuil %v ; attendu %v, %v, seuil > 0", a.nom,
				p.EnergieRamassable, p.Variation, p.Seuil, a.energie, a.variation)
		}
	}
}
