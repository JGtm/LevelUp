package grammar

// lecteur_position_exceptions_r3bis_test.go — LES DEUX EXCEPTIONS DU LOT R3-BIS (2026-09-30) LISENT
// LE FORMAT QUE LE FLUX PORTE.
//
// Chaque flux est construit selon la lecture qui FERME au bit pres, dans la marche des trames, les
// paquets de la reference J4.0.5 (`56299bdc3~1`) que la lecture du jeu (`FUN_14076e494(0x10)`) perd
// a la tete du plan — carte de fermeture paquet par paquet, vingt films (les dix-huit de la
// reference, `f75e7053` et `000d5950`) :
//
//	unit-actor-state      4f77afc1 chunk 38 paquet 410 (liste, 17 entrees de controle ; record NEW
//	(FUN_14058c058)       ti=40 slot 968, emplacement a = 0, b = 0), chunk 54 paquet 316 (liste, 5 ;
//	                      NEW ti=35 slot 5035, a = 0, b = 0) et chunk 54 paquet 1140 (liste, 11 ;
//	                      NEW ti=35 slot 569, a = 0, b = 1), et par elle le paquet 12:1118 dont le
//	                      slot 570 se lie a ti=4 au lieu de ti=35, d ou 12:1120..12:1128 : le
//	                      vecteur de l emplacement sur 16 bits plats ; le jeu en lit 49 a 67
//	tacmap-waypointstate  d9781168 chunk 34 paquet 336 (liste, 4 entrees ; ti=34 slot 1) : R(1),
//	(ti=34 i7)            R(32), la position aux largeurs du descripteur de TRAVERSEE (porte, index
//	                      1 bit, 6/6/6), et PAS le R(1) de `param_4 > 1` ; le jeu lit la garde,
//	                      `FUN_14076e524(0x10)` et ce R(1) (+30 bits)
//
// Ces cas ROUGISSENT si le site est rendu a la lecture du jeu sans que l exception tombe.

import "testing"

func TestLesExceptionsR3bisLisentLeFormatDuFlux(t *testing.T) {
	enTeteActeur := seul(fixe(32), fixe(32), fixe(8), fixe(4))
	absents := func(n int) []champDeFlux {
		var f []champDeFlux
		for range n {
			f = append(f, bit(false))
		}
		return f
	}
	cas := []struct {
		nom  string
		flux []champDeFlux
		lire func(br *Lecteur)
	}{
		{"unit-actor-state emplacement a=0 b=0 (4f77afc1 54:316)",
			concat(enTeteActeur,
				seul(bit(true), bit(false), bit(false), fixe(32), fixe(16), fixe(8), bit(false)),
				absents(4)),
			parNom("unit-actor-state-component", 35, 1)},
		{"unit-actor-state emplacement a=0 b=1 (4f77afc1 54:1140)",
			concat(enTeteActeur,
				seul(bit(true), bit(false), bit(true), fixe(32),
					bit(true), fixe(2), fixe(10), fixe(10), fixe(16), bit(false), fixe(32), bit(false),
					bit(false)),
				absents(4)),
			parNom("unit-actor-state-component", 35, 1)},
		{"tacmap-waypointstate niveau registre 2, porte a 0 (d9781168 34:336)",
			seul(bit(true), fixe(32), bit(false), fixe(1), fixe(6), fixe(6), fixe(6)),
			parNom("tacmap-waypointstate", 34, 2)},
		{"tacmap-waypointstate niveau registre 2, porte posee",
			seul(bit(true), fixe(32), bit(true), fixe(6), fixe(6), fixe(6)),
			parNom("tacmap-waypointstate", 34, 2)},
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
