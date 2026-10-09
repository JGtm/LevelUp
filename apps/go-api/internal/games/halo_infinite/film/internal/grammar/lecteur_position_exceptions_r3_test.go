package grammar

// lecteur_position_exceptions_r3_test.go — LES CINQ EXCEPTIONS DU LOT R3 (2026-09-29) LISENT LE
// FORMAT QUE LE FLUX PORTE.
//
// Chaque flux est construit selon la lecture qui FERME au bit pres, dans la marche des trames, les
// paquets que la lecture du jeu (`FUN_14076e494(0x10)` / `FUN_141f85880`) perd — carte de fermeture
// du parent du lot J6.3 (`56299bdc3~1`) contre la tete du plan, paquet par paquet :
//
//	tacmap-displayasset    51ebbc0f chunk 7 paquets 2336..2382 et chunk 8 paquets 70..118 (onze
//	                       paquets, 6 ou 7 entrees de controle chacun) : porte posee, trois axes de
//	                       6 bits ; le jeu en lit 22 (+48 bits)
//	tacmap-areaofinterest  51ebbc0f chunk 12 paquet 608 (6 entrees) : porte a 0, index 1 bit, trois
//	                       axes de 6 bits ; le jeu lit les largeurs de la carte (+29 bits)
//	tacmap-cooptetherarea  c75f33b8 chunk 21 paquet 1012 (liste d evenements) : porte posee, 3 x 6
//	                       (+48 bits)
//	crew-order             084a804d chunk 22 paquet 538 (liste, 14 entrees) : R(3), presence,
//	                       precHigh = 0, porte a 0, index 1 bit, trois axes a 6 + niveau (+26 bits)
//	i0 du bipede, precHigh = 1 (branche absolue de `FUN_1406cfe44`)
//	                       0797ce72 chunk 9 paquet 138 (6 entrees), 084a804d chunk 25 paquet 356
//	                       (21 entrees) et chunk 37 paquet 22 (10 entrees) : le bit precHigh, puis
//	                       RIEN ; le jeu lit `FUN_141f85880` (3 x 14) et le R(2) (+44 bits)
//
// Ces cas ROUGISSENT si le site est rendu a la lecture du jeu sans que l exception tombe.

import "testing"

func TestLesExceptionsR3LisentLeFormatDuFlux(t *testing.T) {
	sixSixSix := seul(fixe(6), fixe(6), fixe(6))
	cas := []struct {
		nom  string
		flux []champDeFlux
		lire func(br *Lecteur)
	}{
		{"tacmap-displayasset porte posee (51ebbc0f 7:2336)",
			concat(seul(fixe(32), fixe(32), fixe(2), bit(true)), sixSixSix,
				seul(fixe(64), fixe(32), fixe(64), fixe(32), bit(true))),
			consumeTacmapDisplayAsset},
		{"tacmap-areaofinterest porte a 0 (51ebbc0f 12:608)",
			concat(seul(fixe(32), fixe(3), bit(false), fixe(1)), sixSixSix, seul(fixe(12))),
			consumeTacmapAreaOfInterest},
		{"tacmap-cooptetherarea porte posee (c75f33b8 21:1012)",
			concat(seul(bit(true)), sixSixSix, seul(fixe(12), fixe(12))),
			consumeTacmapCoopTetherArea},
		{"crew-order niveau 0 (084a804d 22:538)",
			concat(seul(fixe(3), bit(true), bit(false), bit(false), fixe(1)), sixSixSix),
			func(br *Lecteur) { consumeCrewOrder(br, 0) }},
		{"i0 absolu du bipede precHigh=1 (0797ce72 9:138)", seul(bit(true)), consumeAbsoluteWithGate},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			buf, total := ecrireFlux(c.flux)
			br := lecteurDeSite(buf, 1)
			c.lire(br)
			if got := br.BitPos(); got != total {
				t.Fatalf("%s : %d bits lus, le flux en porte %d", c.nom, got, total)
			}
		})
	}
}
