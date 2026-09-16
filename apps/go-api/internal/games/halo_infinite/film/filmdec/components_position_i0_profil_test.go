package filmdec

import "testing"

// components_position_i0_profil_test.go — LE DESERIALISEUR D i0 CONSULTE BIEN LE PROFIL QUE SON
// LECTEUR PORTE (lot 2.2.a du PLAN_DECODEUR_FILM).
//
// # POURQUOI CE FICHIER EXISTE
//
// [TestLecteurPorteLeProfilDeMouvement] prouve que les accesseurs rendent ce qu on leur pose.
// Il ne prouve PAS que le deserialiseur les appelle : un `consume*` qui aurait garde une
// constante en dur passerait ce test-la sans broncher.
//
// Ce que ce fichier mesure est la seule chose qui compte pour la famille « positions » : le
// NOMBRE DE BITS que `consumeObjectPositionDynamicPrecisionD` consomme sur un flux FIGE change
// quand, et seulement quand, la valeur du profil change. Chaque cas donne son compte attendu
// par sa DERIVATION, jamais par relevé — un compte recopie d une execution ne dirait rien le
// jour ou l execution a tort.
//
// # LE CAS QUI A MOTIVE LE FICHIER
//
// A la mutation obligatoire du lot, quatre des cinq valeurs avaient deja un temoin nomme
// (`TestScanObjectDeathsSurBobineReelle`, `TestKeyframeClosureRatchet`,
// `killsource.TestGoldenMiniBobine`). La cinquieme — la queue de poignee du chemin delta
// predit — n en avait AUCUN : la fausser ne rougissait que l empreinte de grammaire, c est-a-dire
// le fait que la source a change, pas le fait que le decodage a change. Elle en a un ici.

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
	br := NewBitReader(buf)
	br.poserMouvement(mv)
	consumeObjectPositionDynamicPrecisionD(br, br.traversal())
	return br.BitPos()
}

// TestProfilDePositionChangeLaConsommationDeBits — LE TEMOIN DE MUTATION DES CINQ VALEURS.
func TestProfilDePositionChangeLaConsommationDeBits(t *testing.T) {
	release := LockProcessDecode()
	defer release()
	defer AbsIndexHistogram() // le compteur d observation du chemin absolu ne fuit pas sur les autres tests

	// LE FLUX ABSOLU : bUsePred=0, bDelta=0, precHigh=0, selecteur d index=0, puis du zero.
	// Derivation du compte, avec IndexW=1 et AbsoluteAxisW=14 :
	//   2 (bUsePred + bDelta) + 1 (precHigh) + 1 (selecteur) + 1 (index) + 3x14 + 2 (fini) = 49.
	absolu := bitsDe("", 32)
	// LE FLUX DELTA PREDIT : bUsePred=0, bDelta=1, predFlag=0, present=0, masque=1, puis les
	// trois deltas de 8 bits. Derivation : 3 + 2 + 24 = 29 sans queue de poignee ; avec elle,
	// deux bits de plus (selecteur de poignee a 0, presence de region a 0).
	delta := bitsDe("01001", 32)

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
