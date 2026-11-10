package grammar

// lecteur_position_exceptions_j6bis_test.go — L EXCEPTION DU LOT J6-bis (2026-09-28) QUI TIENT ENCORE
// LIT LE FORMAT QUE LE FLUX DE `11de8353` PORTE.
//
// Le flux est construit selon la lecture qui FERME la liste d evenements de `11de8353` (HI_1_9_0) au
// bit pres : chunk 21 paquet 1032 (tacmap-poiicon). La lecture du jeu (`FUN_14076e494(0x10)`) y lit
// +25 bits : ce cas ROUGIT si le site est rendu au portage unique sans que l exception tombe.
// L autre site du lot, flock-destination, lit comme le jeu (portage unique) : sa fermeture ne baisse
// sur aucun film du corpus.

import "testing"

func TestLesExceptionsJ6bisLisentLeFormatDuFlux(t *testing.T) {
	cas := []struct {
		nom    string
		flux   []champDeFlux
		niveau uint32
		lire   func(br *Lecteur, niveau uint32)
	}{
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
