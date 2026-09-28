package grammar

// lecteur_position_exceptions_j6bis_test.go — LES DEUX EXCEPTIONS DU LOT J6-bis (2026-09-28) LISENT
// LE FORMAT QUE LE FLUX DE `11de8353` PORTE.
//
// Chaque flux est construit selon la lecture qui FERME les listes d evenements de `11de8353`
// (HI_1_9_0) au bit pres : chunk 9 paquet 1146 (flock-destination sur 4 bits, 22 entrees de
// controle derriere), chunk 19 paquet 494 (flock-destination sur 29 bits, 16 entrees), chunk 21
// paquet 1032 (tacmap-poiicon). La lecture du jeu (`FUN_14076e494(0x10)`) y lit 70, 52 et +25 bits :
// ces cas ROUGISSENT si le site est rendu au portage unique sans que l exception tombe.

import "testing"

func TestLesExceptionsJ6bisLisentLeFormatDuFlux(t *testing.T) {
	cas := []struct {
		nom    string
		flux   []champDeFlux
		niveau uint32
		lire   func(br *Lecteur, niveau uint32)
	}{
		// 9:1146 : R(1), precHigh = 1 (le vecteur par defaut, 0 bit), R(2) (niveau 2).
		{"flock-destination precHigh=1 niveau 2", seul(bit(true), bit(true), fixe(2)), 2,
			func(br *Lecteur, n uint32) { consumeByName(br, "flock-destination-component", 21, n) }},
		// 19:494 : R(1), precHigh = 0, porte = 1, trois axes a 6 + 2 = 8 bits, R(2).
		{"flock-destination precHigh=0 porte=1 niveau 2",
			seul(bit(false), bit(false), bit(true), fixe(8), fixe(8), fixe(8), fixe(2)), 2,
			func(br *Lecteur, n uint32) { consumeByName(br, "flock-destination-component", 21, n) }},
		// tacmap-poiicon, niveau 0 : le bloc, precHigh = 0, porte = 0, index 1 bit, 3 x 6, la queue.
		{"tacmap-poiicon precHigh=0 porte=0 niveau 0",
			concat(seul(fixe(32), fixe(32), bit(true), fixe(3), fixe(32), fixe(32), fixe(9), fixe(9)),
				seul(bit(false), bit(false), fixe(1), fixe(6), fixe(6), fixe(6)),
				seul(fixe(32), bit(false), fixe(8), fixe(8), fixe(8), fixe(8))), 0,
			func(br *Lecteur, n uint32) { consumeByName(br, "tacmap-poiicon", 30, n) }},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			buf, total := ecrireFlux(c.flux)
			br := lecteurDeSite(buf, 1)
			c.lire(br, c.niveau)
			if got := br.BitPos(); got != total {
				t.Fatalf("%s : %d bits lus, le flux de 11de8353 en porte %d", c.nom, got, total)
			}
		})
	}
}
