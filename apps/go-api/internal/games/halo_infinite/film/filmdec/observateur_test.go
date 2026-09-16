package filmdec

import "testing"

// observateur_test.go — L OBSERVATEUR NE CHANGE AUCUNE CONSOMMATION DE BITS (lot 2.2.f).
//
// # POURQUOI CE TEST, ET PAS UNE MUTATION DE VALEUR
//
// Les autres familles du lot 2.2 se prouvent par MUTATION : fausser la valeur dans le profil
// rougit une lecture. L observateur ne se prouve PAS ainsi, et c est sa definition — aucun de
// ses champs n a le droit de changer un compte de bits. La mutation qui doit rougir est donc
// l INVERSE : poser un observateur qui publie tout, et verifier que le curseur finit EXACTEMENT
// au meme bit qu avec l observateur vide.
//
// Le jour ou ce test rougit, un champ de [Observation] est une valeur de PROFIL mal rangee.
func TestObservateurNeChangeAucunBit(t *testing.T) {
	release := LockProcessDecode()
	defer release()

	// Les quatre chemins que les crochets traversent le plus, sur des flux figes.
	cas := []struct {
		nom  string
		flux []byte
		lire func(br *BitReader)
	}{
		{"i0 position absolue", bitsDe("", 32), func(br *BitReader) {
			consumeObjectPositionDynamicPrecisionD(br, br.traversal())
		}},
		{"i56 energie de capacite", bitsDe("111", 32), consumeBipedSpartanAbilityEnergy},
		{"action de mobilite", bitsDe("10", 32), consumeBipedMobilityAction},
		{"vitesse de translation", bitsDe("0", 32), consumeObjectTranslationalVelocity},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			br := NewBitReader(c.flux)
			c.lire(br)
			muet := br.BitPos()

			prev := poserObservateur(observateurBavard())
			defer poserObservateur(prev)
			br2 := NewBitReader(c.flux)
			c.lire(br2)
			if br2.BitPos() != muet {
				t.Errorf("sous observation, %s consomme %d bits au lieu de %d — un champ de "+
					"l observateur change la consommation, donc ce n est pas un observateur",
					c.nom, br2.BitPos(), muet)
			}
		})
	}
}

// observateurBavard rend un observateur dont TOUS les crochets du chemin teste sont poses.
func observateurBavard() *Observation {
	o := NouvelleObservation()
	o.PosCaptureHook = func(PositionSample) {}
	o.AbilityEnergyHook = func(uint32, [AbilityEnergyCharges]int) {}
	o.MobilityActionHook = func(bool, bool) {}
	o.GrenadeSetHook = func(uint32, int) {}
	o.AbilitySetHook = func(uint64, int, int) {}
	o.SpartanAbilityHook = func(uint64, uint64, uint64, bool) {}
	o.AbilityNonPredictedHook = func(AbilityNonPredictedState) {}
	return o
}
