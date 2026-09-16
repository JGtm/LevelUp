package filmdec

import "testing"

// components_position_i0_profil_test.go — LE DESERIALISEUR D i0 CONSULTE BIEN LE PROFIL QUE SON
// LECTEUR PORTE (lots 2.2.a et 2.2.b du PLAN_DECODEUR_FILM).
//
// # POURQUOI CE FICHIER EXISTE
//
// [TestLecteurPorteLeProfilDeMouvement] prouve que les accesseurs rendent ce qu on leur pose.
// Il ne prouve PAS que le deserialiseur les appelle : un `consume*` qui aurait garde une
// constante en dur passerait ce test-la sans broncher.
//
// Ce que ce fichier mesure est la seule chose qui compte pour les familles « positions » et
// « objets du monde » : le
// NOMBRE DE BITS que `consumeObjectPositionDynamicPrecisionD` consomme sur un flux FIGE change
// quand, et seulement quand, la valeur du profil change. Chaque cas donne son compte attendu
// par sa DERIVATION, jamais par relevé — un compte recopie d une execution ne dirait rien le
// jour ou l execution a tort.
//
// # LE CAS QUI A MOTIVE LE FICHIER
//
// A la mutation obligatoire du lot 2.2.a, quatre des cinq valeurs avaient deja un temoin nomme
// (`TestScanObjectDeathsSurBobineReelle`, `TestKeyframeClosureRatchet`,
// `killsource.TestGoldenMiniBobine`). La cinquieme — la queue de poignee du chemin delta
// predit — n en avait AUCUN : la fausser ne rougissait que l empreinte de grammaire, c est-a-dire
// le fait que la source a change, pas le fait que le decodage a change. Elle en a un ici.
//
// MEME CONSTAT AU LOT 2.2.b sur TROIS des quatre valeurs des objets du monde : seul le
// descripteur world-object avait un temoin (`TestG5MesuresCiteesParLaTable`,
// `TestGoldenMiniBobineFamilles`). La largeur d axe du chemin delta axis-width entre dans la
// table de comptes ci-dessous ; la range de dequantification et le quantum de delta, qui ne
// deplacent AUCUN curseur, ont leur propre temoin en fin de fichier.

// bitsDe fabrique un tampon a partir d une chaine de bits MSB-first ; les bits manquants pour
// completer `octets` valent zero. Un tampon large evite qu une lecture morde le rembourrage.
func bitsDe(bits string, octets int) []byte {
	buf := make([]byte, octets)
	for i, c := range bits {
		if c == '1' {
			buf[i/8] |= 1 << (7 - uint(i%8))
		}
	}
	return buf
}

// consommationI0 rend le nombre de bits que le deserialiseur d i0 consomme sur `buf`, sous le
// profil `mv`.
func consommationI0(mv MovementProfile, buf []byte) int {
	br := lecteurDInstrument(buf)
	br.poserMouvement(mv)
	consumeObjectPositionDynamicPrecisionD(br, br.traversal())
	return br.BitPos()
}

// TestProfilDePositionChangeLaConsommationDeBits — LE TEMOIN DE MUTATION DES VALEURS QUI
// CHANGENT UN COMPTE DE BITS.
func TestProfilDePositionChangeLaConsommationDeBits(t *testing.T) {
	defer AbsIndexHistogram() // le compteur d observation du chemin absolu ne fuit pas sur les autres tests

	// LE FLUX ABSOLU : bUsePred=0, bDelta=0, precHigh=0, selecteur d index=0, puis du zero.
	// Derivation du compte, avec IndexW=1 et AbsoluteAxisW=14 :
	//   2 (bUsePred + bDelta) + 1 (precHigh) + 1 (selecteur) + 1 (index) + 3x14 + 2 (fini) = 49.
	absolu := bitsDe("", 32)
	// LE FLUX DELTA PREDIT : bUsePred=0, bDelta=1, predFlag=0, present=0, masque=1, puis les
	// trois deltas de 8 bits. Derivation : 3 + 2 + 24 = 29 sans queue de poignee ; avec elle,
	// deux bits de plus (selecteur de poignee a 0, presence de region a 0).
	delta := bitsDe("01001", 32)
	// LE FLUX DELTA AXIS-WIDTH : meme en-tete, masque a 0 — trois axes de `DeltaAxisWidth`
	// bits au lieu des trois deltas de 8 bits. Derivation : 3 + 2 + 3x14 = 47.
	deltaAxe := bitsDe("01000", 32)

	profil := ResolveProfile(nil, nil).Movement()
	cas := []struct {
		nom              string
		flux             []byte
		attendu, mutante int
		fausser          func(m MovementProfile) MovementProfile
	}{
		{"Traversal.IndexW", absolu, 49, 51, func(m MovementProfile) MovementProfile {
			m.Traversal.IndexW = 3
			return m
		}},
		{"AbsoluteAxisW", absolu, 49, 67, func(m MovementProfile) MovementProfile {
			m.AbsoluteAxisW = 20 // 3x20 au lieu de 3x14 : +18 bits
			return m
		}},
		{"FullPrecision", absolu, 49, 99, func(m MovementProfile) MovementProfile {
			m.FullPrecision = true // 2 + 1 + 96 bits bruts, la charge quantifiee n est pas lue
			return m
		}},
		{"CalibratedSkip", absolu, 49, 47, func(m MovementProfile) MovementProfile {
			m.CalibratedSkip = true // le banc saute au total mesure : 47 sur bUsePred=0
			return m
		}},
		{"DeltaHasHandleTail", delta, 29, 31, func(m MovementProfile) MovementProfile {
			m.DeltaHasHandleTail = true
			return m
		}},
		{"WorldObject.AxisW", absolu, 49, 49, func(m MovementProfile) MovementProfile {
			// LE CHEMIN ABSOLU NE LIT LES LARGEURS DE CARTE QUE PAR LE REPLI d `absAxisW`,
			// eteint tant que la largeur uniforme est posee. Le compte ne bouge donc PAS
			// ici — et c est la mesure, pas une lacune : le temoin du descripteur
			// world-object est `TestG5MesuresCiteesParLaTable`, qui mesure le chemin
			// `object-position-component` du dispatch. La ligne reste pour que la
			// prochaine main qui allume ce repli voie le compte changer.
			m.WorldObject.AxisW = [3]uint{17, 17, 16}
			return m
		}},
		{"DeltaAxisWidth", deltaAxe, 47, 41, func(m MovementProfile) MovementProfile {
			m.DeltaAxisWidth = 12 // 3x12 au lieu de 3x14 : -6 bits
			return m
		}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := consommationI0(profil, c.flux); got != c.attendu {
				t.Fatalf("sous le profil, i0 consomme %d bits, derivation %d — si la derivation "+
					"de l en-tete du cas est encore juste, c est la grammaire qui a bouge",
					got, c.attendu)
			}
			if got := consommationI0(c.fausser(profil), c.flux); got != c.mutante {
				t.Fatalf("la valeur %s faussee dans le profil fait consommer %d bits, attendu "+
					"%d — le deserialiseur ne lit donc PAS cette valeur au profil", c.nom,
					got, c.mutante)
			}
		})
	}
}

// TestProfilDeQuantificationChangeLaValeurRendue — LE TEMOIN DES DEUX VALEURS QUI NE CHANGENT
// AUCUN COMPTE DE BITS.
//
// La range de dequantification et le quantum du chemin delta ne deplacent pas le curseur : ils
// changent la COORDONNEE que le deserialiseur rend. Un temoin en bits ne peut donc rien en
// dire, et c est la raison d etre de ce second test — mesurer la valeur, pas la longueur.
//
// CE QU IL REVELE AU PASSAGE, et qui est consigne au plan : ces deux valeurs n atteignent la
// production que par le crochet de capture de position, nul partout sauf dans le balayage de
// resynchronisation. Le rejeu 2D, lui, dequantifie ailleurs (`ScanBipedPositions`, sur le
// decoupage d i0 de la carte). Leur mutation ne pouvait donc rougir aucun test de bout en bout.
func TestProfilDeQuantificationChangeLaValeurRendue(t *testing.T) {
	defer AbsIndexHistogram()

	profil := ResolveProfile(nil, nil).Movement()
	// Flux ABSOLU (trois axes quantifies a zero) et flux DELTA-8 (trois crans de +1).
	absolu := bitsDe("", 32)
	delta8 := bitsDe("01001"+"00000001"+"00000001"+"00000001", 32)

	lire := func(mv MovementProfile, buf []byte) [3]float32 {
		var vu [3]float32
		precedent := observateur.PosCaptureHook
		observateur.PosCaptureHook = func(s PositionSample) { vu = s.Vec }
		defer func() { observateur.PosCaptureHook = precedent }()
		br := lecteurDInstrument(buf)
		br.poserMouvement(mv)
		consumeObjectPositionDynamicPrecisionD(br, br.traversal())
		return vu
	}

	faussee := profil
	faussee.Range = QuantRangeWorld100
	if avant, apres := lire(profil, absolu), lire(faussee, absolu); avant == apres {
		t.Errorf("la range faussee dans le profil rend la MEME coordonnee absolue %v — le "+
			"deserialiseur ne lit donc pas la range au profil", avant)
	}
	faussee = profil
	faussee.DeltaQuantum = profil.DeltaQuantum * 2
	avant, apres := lire(profil, delta8), lire(faussee, delta8)
	// LA VALEUR ATTENDUE EST EPINGLEE, PAS RELATIVE, et c est ce qui fait de ce test un temoin
	// de MUTATION : un delta d un cran vaut EXACTEMENT le quantum, et le quantum du profil est
	// `0.01383` (ligne `Movement.DeltaQuantum` de [TableProfil], provenance MESUREE). Une
	// comparaison « apres = 2 x avant » ne dirait rien si quelqu un faussait la table : les deux
	// cotes bougeraient ensemble.
	const quantumDeLaTable = float32(0.01383)
	if avant[0] != quantumDeLaTable {
		t.Errorf("un cran de delta rend %v, la table du profil annonce %v", avant[0],
			quantumDeLaTable)
	}
	if apres[0] != avant[0]*2 {
		t.Errorf("quantum double : delta %v, attendu le double de %v", apres, avant)
	}
}

// TestProfilDeMobiliteChangeLaConsommationDeBits — LE TEMOIN DES BITS SUPPLEMENTAIRES D UNE
// ACTION DE MOBILITE.
//
// CE QUE LA MUTATION DU LOT 2.2.e A REVELE, ET QUI EST CONSIGNE AU PLAN : fausser
// `Movement.MobilityActionExtraBits` ne rougissait AUCUN test, et la cause n est pas un manque
// de couverture — c est que la valeur est INATTEIGNABLE en production. Elle ne se lit que sur
// la branche `else` du corps porte, et le corps EST porte par defaut
// (`MobilityActionBodyPorted`, vrai, remis a vrai par `killsource.resetGlobals`). Le chemin
// n existe donc que sous le harnais qui eteint le portage, pour rejouer la ligne de base
// d avant le portage du corps.
//
// CE TEST JOUE CE HARNAIS, et c est la seule facon honnete de donner un temoin a cette valeur :
// mesurer le chemin ou elle sert, en le nommant comme un harnais.
func TestProfilDeMobiliteChangeLaConsommationDeBits(t *testing.T) {

	// flag1=1 (le corps suit), flag2=0, puis la queue de poignee `FUN_1408f0ac4(...,0)` sur des
	// bits nuls. Le compte exact importe peu : ce qui compte est l ECART entre deux profils.
	flux := bitsDe("10", 32)
	consommation := func(extra int) int {
		mv := ResolveProfile(nil, nil).Movement()
		mv.MobilityActionExtraBits = extra
		br := lecteurDInstrument(flux)
		// LE HARNAIS EST DANS LE PROFIL DU LECTEUR (lot 2.3) : eteindre le portage du corps
		// d i54 ne touche que CE balayage.
		p := br.Profil()
		p.Grammaire.CorpsActionMobilite = false
		br.PoserProfil(p)
		br.poserMouvement(mv)
		consumeBipedMobilityAction(br)
		return br.BitPos()
	}
	// LA VALEUR DU PROFIL EST EPINGLEE, comme au temoin du quantum de delta : sans cela, fausser
	// la table ferait bouger les deux cotes de la comparaison et le temoin ne mordrait pas.
	// `0` est la ligne `Movement.MobilityActionExtraBits` de [TableProfil], provenance PRESUMEE.
	const extraDeLaTable = 0
	if got := ResolveProfile(nil, nil).Movement().MobilityActionExtraBits; got != extraDeLaTable {
		t.Errorf("le profil annonce %d bits supplementaires, la table %d", got, extraDeLaTable)
	}
	sansExtra := consommation(extraDeLaTable)
	if got := consommation(extraDeLaTable + 3); got != sansExtra+3 {
		t.Errorf("trois bits supplementaires au profil font consommer %d bits au lieu de %d — le "+
			"deserialiseur ne lit donc PAS cette valeur au profil", got, sansExtra+3)
	}
}
