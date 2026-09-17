package grammar

// keyframe_closure_ti9_test.go — LA FERMETURE DE `ti=9` (LE JOUEUR GERE) SUR LES SEPT BOBINES,
// LOT 3.6.a.
//
// # POURQUOI UN TEST DEDIE A COTE DU RATCHET 0.A.3
//
// Le ratchet (`keyframe_closure_ratchet_test.go`) garde TOUS les archetypes contre une BAISSE, et
// accepte une hausse en la signalant. C est ce qu il doit faire : le lot 3.6 fait monter la
// couverture archetype par archetype, et rougir sur le progres aurait refuse exactement ce qu il
// garde. Mais un golden qu on refige n affirme rien sur la CIBLE du lot : il enregistre ce qui
// est, pas ce qui etait promis.
//
// Ce test-ci porte la cible de 3.6.a, chiffree, et il rougit DANS LES DEUX SENS — une fermeture
// qui descend comme une qui monte sans qu on ait dit pourquoi.
//
// # LA MESURE, AVANT ET APRES (collee, 2026-09-17)
//
//	avant 3.6.a       0 / 1 717   bloquant `i4 managed-player-forge-weather-effect-overrides-component`
//	apres `i4`        1 716 / 1 717   AUCUN bloquant sur aucune des sept bobines
//
// # LE 1 717e RECORD N EST PAS UN RECORD, ET C EST MESURE
//
// Le seul record qui n atterrit pas sur la frontiere est sur `111fa685` (build `HI_1_10_0`),
// chunk 1, bit 9145 : il marche jusqu au bout de ses composants (`DesyncAt = -1`, tous portes) et
// finit 3 415 bits AVANT la frontiere visee. Son premier mot de taille vaut `n1 = 2 154 823 696`
// quand les 1 716 autres valent tous `12` — c est une ANCRE FORTUITE, exactement la population
// que `default_state_n2_constant_test.go` ecarte deja par ce critere (releve B.1 de
// `NOTE_IMAGECLE_ETAT_COMPLET_2026-09-13.md`) : le balayeur d ancres retient un motif qui passe
// son filtre sans etre un record. Aucune largeur de `ti=9` ne peut le fermer, et en inventer une
// qui le ferait serait truquer l oracle.
//
// LA CIBLE HONNETE DE 3.6.a EST DONC 1 716 / 1 717 (99,94 %) SUR LES BOBINES, pas 1 717 : le plan
// ecrivait « 0 -> 1 717 » avant que cette ancre ne soit mesuree.
//
// # CE QUE LE TEST TIENT, ET QUI EST PLUS FORT QUE LE TAUX
//
// `Blocking` vide sur les sept bobines veut dire qu AUCUN record n a bute sur un composant sans
// lecteur : la couverture du dispatch pour `ti=9` est COMPLETE. C est cette ligne-la qui dit que
// l archetype est ferme ; le taux, lui, porte encore le bruit du balayeur d ancres.

import "testing"

// ti9TypeIndex : l archetype du joueur gere au registre du film.
const ti9TypeIndex = 9

// ti9RecordsAttendus : le nombre de records d image-cle `ti=9` BORNES que portent les sept
// bobines (337 + 393 + 240 + 262 + 144 + 261 + 80).
//
// C EST UNE MESURE, PAS UNE CIBLE, et elle est ecrite ici pour qu une bobine perdue ou tronquee
// rougisse au lieu de rendre « 100 % de pas grand-chose ».
const ti9RecordsAttendus = 1717

// ti9FermesAttendus : les records qui ATTERRISSENT sur la frontiere du record suivant.
//
// 1 716 / 1 717 AU 2026-09-17. Le reste est l ancre fortuite de `111fa685` decrite en tete de
// fichier — pas une largeur.
const ti9FermesAttendus = 1716

// ti9BloquantPorte : le composant que le golden nommait comme bloquant des SEPT bobines avant ce
// lot. Il ne doit plus bloquer aucune d elles.
const ti9BloquantPorte = "i4 managed-player-forge-weather-effect-overrides-component"

// TestTI9FermeSurLesSeptBobines : la fermeture de `ti=9` vaut ce que le lot a mesure, et plus
// aucun composant de l archetype n arrete une marche.
func TestTI9FermeSurLesSeptBobines(t *testing.T) {
	var fermes, total int
	for _, court := range closureMiniFilms() {
		s := fermetureDUneBobine(t, "../../replay/testdata/minifilm_"+court)[ti9TypeIndex]
		fermes, total = fermes+s.Closed, total+s.Total
		switch s.Blocking {
		case "":
		case ti9BloquantPorte:
			t.Errorf("%s ti=9 : %q bloque encore %d/%d records alors que le lot 3.6.a le porte",
				court, s.Blocking, s.Total-s.Closed, s.Total)
		default:
			t.Errorf("%s ti=9 : %q arrete %d/%d records — un composant de l archetype n a plus de "+
				"lecteur, la couverture du dispatch a regresse",
				court, s.Blocking, s.Total-s.Closed, s.Total)
		}
		t.Logf("%s ti=9 : %d/%d records fermes", court, s.Closed, s.Total)
	}
	if total != ti9RecordsAttendus {
		t.Fatalf("ti=9 : %d records bornes mesures sur les sept bobines, %d attendus — ce n est pas "+
			"la grammaire qui a bouge mais le corpus (bobine absente, tronquee ou regeneree)",
			total, ti9RecordsAttendus)
	}
	if fermes != ti9FermesAttendus {
		t.Fatalf("ti=9 : %d/%d records fermes, %d attendus.\nEn BAISSE : une largeur de `ti=9` a "+
			"bouge, la corriger. En HAUSSE : c est un gain a figer ici ET dans "+
			"`testdata/keyframe_closure.golden`, avec ce qui le produit.",
			fermes, total, ti9FermesAttendus)
	}
}

// TestTI9ForgeWeatherConsommeSoixanteQuatreBits : `i4` lit ses deux mots de 32 bits, et rien de plus.
//
// LE COMPOSANT SE LIT, ET LA MESURE EST INDEPENDANTE DU FILM : la grammaire relevee chez
// l ecrivain `FUN_142ed5bc8` est INCONDITIONNELLE, donc la largeur ne depend d aucun bit lu. Le
// test le verifie sur les deux polarites et sur l alternance — trois tampons, une seule largeur.
func TestTI9ForgeWeatherConsommeSoixanteQuatreBits(t *testing.T) {
	for _, motif := range ecsBitsPatterns {
		buf := make([]byte, 16)
		for i := range buf {
			buf[i] = motif
		}
		br := LecteurSur(buf)
		_, _, porte := consumeByName(br, compManagedPlayerForgeWeather, ti9TypeIndex, 1)
		if !porte {
			t.Fatalf("motif %#02x : le composant n est pas porte", motif)
		}
		if got := br.BitPos(); got != 64 {
			t.Errorf("motif %#02x : %d bits consommes, 64 attendus (R(32) + R(32) inconditionnels)",
				motif, got)
		}
	}
}
