package grammar

// lecteur_position_sites_portee_test.go — LES EXCEPTIONS DATEES DU PORTAGE, SOUS LA PORTEE DE
// L ETAT COMPLET (plan LK, LK.5.4).
//
// Sous la portee `DAT_144e61ea0` ([Lecteur.portee]), la garde de pleine precision du jeu
// (`FUN_14076f91c`) est vraie : `FUN_14076e494` lit le vecteur BRUT (`FUN_1411b259c`, R(96)) a la
// place de la position quantifiee. Un site en exception datee (`lecteur_position_exceptions.go`) dont
// la lecture sous la garde est relue chez le jeu la porte en tete de sa fonction : chaque cas ci-dessous
// fabrique le flux qu ECRIT le jeu sous la portee et exige que le site le lise au bit pres, SANS noter
// d exception. Hors de la portee, les cas de `lecteur_position_sites_test.go` et des fichiers
// `lecteur_position_exceptions_*_test.go` tiennent l ancien lecteur.
//
// Mutation jouee pour chaque site (retirer le bloc de tete) : son cas rougit.

import "testing"

// brut96 est le vecteur brut de `FUN_1411b259c` : trois flottants de 32 bits, finis (motif alterne).
func brut96() []champDeFlux { return seul(fixe(32), fixe(32), fixe(32)) }

// casDesExceptionsSousLaPortee rend un flux par site, tel que l ecrit le jeu sous la portee.
func casDesExceptionsSousLaPortee() []casDeSite {
	wo := parNom(compObjectPosition, 38, 0)
	return []casDeSite{
		// world-object i0 (FUN_14076e29c) : precHigh, R(96), queue FUN_14076e3e4(precHigh), R(2).
		{nom: "world-object i0 precHigh=0", indexW: 1,
			flux: concat(seul(bit(false)), brut96(), seul(fixe(2))), lire: wo},
		// precHigh = 1 : la queue — FUN_1408f0ac4 ferme, porte du mot de region posee, R(11).
		{nom: "world-object i0 precHigh=1", indexW: 1,
			flux: concat(seul(bit(true)), brut96(), seul(bit(false), bit(true), fixe(11), fixe(2))), lire: wo},
	}
}

// TestChaqueExceptionLitLeFluxDuJeuSousLaPortee — LK.5.4 : sous la portee, chaque exception relue lit
// comme le jeu, et n est pas notee comme exception.
func TestChaqueExceptionLitLeFluxDuJeuSousLaPortee(t *testing.T) {
	for _, c := range casDesExceptionsSousLaPortee() {
		t.Run(c.nom, func(t *testing.T) {
			buf, total := ecrireFlux(c.flux)
			br := sousLaPortee(lecteurDeSite(buf, c.indexW))
			c.lire(br)
			if got := br.BitPos(); got != total || br.arret != ArretAucun {
				t.Fatalf("%s sous la portee : %d bits lus, arret %v ; l ecrivain du jeu en pose %d", c.nom, got,
					br.arret, total)
			}
			if br.exceptionDatee {
				t.Errorf("%s sous la portee : le site lit comme le jeu, il ne doit pas noter d exception", c.nom)
			}
		})
	}
}

// TestLaPositionDObjetDuMondeNonFinieArreteLaLecture : sous la garde, `FUN_14076e29c` rend faux sur un
// flottant non fini (14076e2fe), sans lire le R(2) : la lecture s arrete, cause nommee.
func TestLaPositionDObjetDuMondeNonFinieArreteLaLecture(t *testing.T) {
	buf, _ := ecrireFlux(seul(bit(false), fixe(32), champDeFlux{motInfini, 32}, fixe(32), fixe(2)))
	br := sousLaPortee(lecteurDeSite(buf, 1))
	parNom(compObjectPosition, 38, 0)(br)
	if br.arret != ArretPositionNonFinie || br.BitPos() != 97 {
		t.Fatalf("world-object i0, flottant infini : arret %v apres %d bits ; attendu %v apres 97", br.arret,
			br.BitPos(), ArretPositionNonFinie)
	}
}
