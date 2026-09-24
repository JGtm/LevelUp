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
//	apres `i9`        1 716 / 1 717   INCHANGE, et c est la mesure, pas une deception
//
// # POURQUOI `i9` NE FAIT MONTER AUCUN COMPTE ICI, ET CE QU IL FERME QUAND MEME
//
// Mesure du 2026-09-17 sur les sept registres : `ti=9` porte NEUF composants sur cinq bobines
// (`a521164d`, `60ae07c4`, `11de8353`, `111fa685`, `e5adf7b2`) et DIX sur les deux plus recentes
// (`bcb6d393` = `HI_1_12_0`, `fb1a1a72` = `HI_1_13_0`). `i9` n existe donc pas au registre de
// cinq bobines sur sept ; sur les deux autres, la mesure d avant le port ne comptait AUCUNE
// desynchronisation (`d0` a l instrument `imagecle_fermeture`), c est-a-dire qu aucun record n y
// atteignait la branche `compteur > 0` que le depot refusait de lire.
//
// Ce que `i9` ferme est donc un cas que CE corpus ne porte pas : un sac texte non vide. Le
// mesurer ici serait impossible, et c est pour cela que sa grammaire est tenue par un test de
// largeur sur tampon synthetique ([TestTI9InputPromptLargeursDuSacTexte]) — la seule forme qui
// puisse juger une branche que le corpus ne visite pas.
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
// bobines (337 + 393 + 240 + 262 + 151 + 261 + 94).
//
// C EST UNE MESURE, PAS UNE CIBLE, et elle est ecrite ici pour qu une bobine perdue ou tronquee
// rougisse au lieu de rendre « 100 % de pas grand-chose ».
//
// 1 717 -> 1 738 AU LOT M3.1 (2026-09-23, campagne « retours rejeu ») : LA MARCHE D IMAGE-CLE
// NE S ARRETE PLUS SUR UNE FENETRE VIDE. `bcb6d393` passe de 144 a 151 records et `fb1a1a72` de
// 80 a 94 : ce sont les records ti=9 des images-cles que la fenetre de 120 000 bits coupait (la
// premiere image-cle de `bcb6d393` rendait 123 records et s arretait avant sa table de joueurs).
// Les 21 records gagnes FERMENT tous. Ce n est pas le corpus qui a bouge, c est la marche.
const ti9RecordsAttendus = 1738

// ti9FermesAttendus : les records qui ATTERRISSENT sur la frontiere du record suivant.
//
// 1 716 / 1 717 AU 2026-09-17 ; 1 737 / 1 738 AU LOT M3.1 (2026-09-23 : les 21 records gagnes
// ferment tous). Le reste est l ancre fortuite de `111fa685` decrite en tete de fichier — pas
// une largeur.
const ti9FermesAttendus = 1737

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

// TestTI9InputPromptLargeursDuSacTexte fige la largeur de CHAQUE chemin d `i9
// managed-player-custom-input-prompt-widget` (`FUN_141fcf160` -> `FUN_14080b034` ->
// `FUN_1407f0ebc`).
//
// # POURQUOI UN TAMPON SYNTHETIQUE, ET PAS UN FILM
//
// Les sept bobines du ratchet ne portent AUCUN sac texte non vide (cf. l en-tete de ce fichier) :
// la boucle a etiquette que ce lot porte n y est jamais visitee. Un oracle de fermeture ne peut
// donc rien en dire, ni en bien ni en mal. Le tampon synthetique pose chaque branche a la main et
// compte les bits consommes — c est la seule forme qui juge une branche que le corpus ne visite
// pas, et elle est directement confrontable au desassemblage cite dans
// `components_managed_player.go`.
//
// Largeurs attendues, toutes relevees chez l ecrivain :
//
//	present = 0                       1 bit
//	present = 1, texte = 0            1 + 2 + 1              = 4
//	present = 1, texte = 1, n = 0     4 + 32 + 3             = 39
//	un corps, `k` compris             k=0 : 3 · k=1 : 4 ou 9 · k=2 : 28 ou 36 · k=3 : 35 · k>=4 : 35
func TestTI9InputPromptLargeursDuSacTexte(t *testing.T) {
	cas := []struct {
		nom     string
		ecrire  func(w *bitWriterMSB)
		attendu int
	}{
		{
			nom:     "absent : la porte de presence a 0 arrete tout",
			ecrire:  func(w *bitWriterMSB) { w.put(0, 1) },
			attendu: 1,
		},
		{
			nom: "present sans texte : porte, mode, porte du sac",
			ecrire: func(w *bitWriterMSB) {
				w.put(1, 1) // present
				w.put(3, 2) // mode
				w.put(0, 1) // texte = 0
			},
			attendu: 4,
		},
		{
			nom: "sac vide : nom lu, compteur a zero",
			ecrire: func(w *bitWriterMSB) {
				ti9SacTexteEnTete(w, 0)
			},
			attendu: 39,
		},
		{
			nom: "corps k = 0 : etiquette seule, zero bit de charge",
			ecrire: func(w *bitWriterMSB) {
				ti9SacTexteEnTete(w, 1)
				w.put(0, 3)
			},
			attendu: 39 + 3,
		},
		{
			nom: "corps k = 1, porte posee : la reference de participant est absente",
			ecrire: func(w *bitWriterMSB) {
				ti9SacTexteEnTete(w, 1)
				w.put(1, 3)
				w.put(1, 1) // FUN_1407f2058 : polarite INVERSEE, 1 = absent
			},
			attendu: 39 + 4,
		},
		{
			nom: "corps k = 1, porte a zero : R(5) d index de participant",
			ecrire: func(w *bitWriterMSB) {
				ti9SacTexteEnTete(w, 1)
				w.put(1, 3)
				w.put(0, 1)
				w.put(29, 5)
			},
			attendu: 39 + 9,
		},
		{
			nom: "corps k = 2, porte a 1 : R(24) quantifie",
			ecrire: func(w *bitWriterMSB) {
				ti9SacTexteEnTete(w, 1)
				w.put(2, 3)
				w.put(1, 1)
				w.put(0x123456, 24)
			},
			attendu: 39 + 28,
		},
		{
			nom: "corps k = 2, porte a 0 : R(32) brut",
			ecrire: func(w *bitWriterMSB) {
				ti9SacTexteEnTete(w, 1)
				w.put(2, 3)
				w.put(0, 1)
				w.put(0xdeadbeef, 32)
			},
			attendu: 39 + 36,
		},
		{
			nom: "corps k = 3 : string_id",
			ecrire: func(w *bitWriterMSB) {
				ti9SacTexteEnTete(w, 1)
				w.put(3, 3)
				w.put(0x01020304, 32)
			},
			attendu: 39 + 35,
		},
		{
			nom: "corps k = 7 : la branche 4..7 lit un R(32) comme les autres",
			ecrire: func(w *bitWriterMSB) {
				ti9SacTexteEnTete(w, 1)
				w.put(7, 3)
				w.put(0x0a0b0c0d, 32)
			},
			attendu: 39 + 35,
		},
		{
			nom: "quatre corps : le compteur mene la boucle, sans borne a quatre",
			ecrire: func(w *bitWriterMSB) {
				ti9SacTexteEnTete(w, 4)
				for i := 0; i < 4; i++ {
					w.put(0, 3)
				}
			},
			attendu: 39 + 4*3,
		},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			w := &bitWriterMSB{}
			c.ecrire(w)
			// Queue de zeros : le lecteur ne doit JAMAIS y mordre, et s il y mord le compte le dit.
			w.put(0, 64)
			br := LecteurSur(w.buf)
			_, _, porte := consumeByName(br, compManagedPlayerInputPrompt, ti9TypeIndex, 1)
			if !porte {
				t.Fatalf("le composant n est plus porte — la boucle a etiquette est portee depuis 3.6.a")
			}
			if got := br.BitPos(); got != c.attendu {
				t.Errorf("%d bits consommes, %d attendus", got, c.attendu)
			}
		})
	}
}

// ti9SacTexteEnTete ecrit la tete commune d un `i9` a sac texte PRESENT : present, mode, porte du
// sac, nom de 32 bits, puis le compteur de 3 bits. 39 bits.
func ti9SacTexteEnTete(w *bitWriterMSB, n uint64) {
	w.put(1, 1)           // present
	w.put(2, 2)           // mode
	w.put(1, 1)           // texte
	w.put(0x4a4b4c4d, 32) // nom « text »
	w.put(n, 3)           // compteur d emplacements
}
