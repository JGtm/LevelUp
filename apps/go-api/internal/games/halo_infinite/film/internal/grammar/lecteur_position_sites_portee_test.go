package grammar

// lecteur_position_sites_portee_test.go — LES EXCEPTIONS DATEES DU PORTAGE, SOUS LA PORTEE DE
// L ETAT COMPLET (plan LK, LK.5).
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

// brut96 est le vecteur brut de `FUN_1411b259c` : trois flottants de 32 bits.
func brut96() []champDeFlux { return seul(fixe(32), fixe(32), fixe(32)) }

// casDesExceptionsSousLaPortee rend un flux par site, tel que l ecrit le jeu sous la portee.
func casDesExceptionsSousLaPortee() []casDeSite {
	enTeteActeur := seul(fixe(32), fixe(32), fixe(8), fixe(4))
	quatreAbsents := seul(bit(false), bit(false), bit(false), bit(false))
	return []casDeSite{
		// unit-actor-state (FUN_14058c058, CALLs 1422cddc1 et 1422cde0e) : emplacement present, a = 0,
		// b = 0 — R(32), le vecteur, R(8) de queue, FUN_14080d69c ferme ; puis quatre absents.
		{nom: "unit-actor-state emplacement a=0 b=0", indexW: 1,
			flux: concat(enTeteActeur, seul(bit(true), bit(false), bit(false), fixe(32)), brut96(),
				seul(fixe(8), bit(false)), quatreAbsents),
			lire: parNom("unit-actor-state-component", 35, 1)},
		// a = 0, b = 1 : c, R(2), deux R(10), le vecteur, FUN_14080d69c, R(32), FUN_1408f0ac4(0) ferme ;
		// pas de R(8) ; FUN_14080d69c ferme.
		{nom: "unit-actor-state emplacement a=0 b=1", indexW: 1,
			flux: concat(enTeteActeur, seul(bit(true), bit(false), bit(true), fixe(32),
				bit(true), fixe(2), fixe(10), fixe(10)), brut96(),
				seul(bit(false), fixe(32), bit(false), bit(false)), quatreAbsents),
			lire: parNom("unit-actor-state-component", 35, 1)},
	}
}

// TestChaqueExceptionLitLeFluxDuJeuSousLaPortee — LK.5 : sous la portee, chaque exception relue lit
// comme le jeu, et n est pas notee comme exception.
func TestChaqueExceptionLitLeFluxDuJeuSousLaPortee(t *testing.T) {
	for _, c := range casDesExceptionsSousLaPortee() {
		t.Run(c.nom, func(t *testing.T) {
			buf, total := ecrireFlux(c.flux)
			br := sousLaPortee(lecteurDeSite(buf, c.indexW))
			c.lire(br)
			if got := br.BitPos(); got != total {
				t.Fatalf("%s sous la portee : %d bits lus, l ecrivain du jeu en pose %d", c.nom, got, total)
			}
			if br.exceptionDatee {
				t.Errorf("%s sous la portee : le site lit comme le jeu, il ne doit pas noter d exception", c.nom)
			}
		})
	}
}
