package filmdec

// components_managed_object_test.go — les trois desers du lot C phase 1b, contraints par les
// VECTEURS D'OCTETS REELS releves en phase 1a sur les films Strongholds `7344d24f` et `696a9d7c`
// (`.ai/V7.5/replay2d/registre_film/lotC/*_vecteurs.tsv`).
//
// POURQUOI CES VECTEURS SONT DES ORACLES ET PAS DES FIXTURES AUTO-VALIDANTES. Chacun vient d'un
// record delta dont le masque annonce EXACTEMENT le composant vise : la charge utile commence
// donc au premier bit apres le masque, sans qu'il faille connaitre la largeur d'un voisin. Deux
// des trois composants portent en plus leur propre controle interne : l'identifiant de 32 bits
// des canaux RTPC est CONSTANT par composant et identique sur les deux films (un cadrage faux ne
// rendrait pas une constante), et la valeur de 22 bits qui le suit croit d'un paquet au suivant.
//
// Ces tests ne lisent AUCUN film : les octets sont recopies ici. Ils tournent partout, CI
// comprise, sans garde d'environnement.

import "testing"

// zoneVecBits fabrique un lecteur sur une suite de bits donnee en hexadecimal gros-boutien,
// alignee sur l'octet. Les vecteurs de la phase 1a sont releves a des positions de bit
// arbitraires dans le film, mais leur CHARGE UTILE est recopiee ici cadree a zero — c'est la
// meme suite de bits, et c'est tout ce que le deser voit.
func zoneVecBits(t *testing.T, b ...byte) *BitReader {
	t.Helper()
	return NewBitReader(b)
}

// TestNavpointRadialProgressVecteurs : ti=12 i14, R(8) sur [-1, +1].
func TestNavpointRadialProgressVecteurs(t *testing.T) {
	release := LockProcessDecode()
	defer release()

	cas := []struct {
		nom     string
		octets  []byte
		quantum uint64
		valeur  float32
	}{
		// 7344d24f chunk 2 paquet 2066, slot 1624 : bruts 0x7F.
		{"7344d24f p2066 q127", []byte{0x7F}, 127, -0.00390625},
		// 7344d24f chunk 2 paquet 2068, slot 1624 : bruts 0x80 — le quantum 128 est le ZERO
		// de la plage, et c'est la valeur de repos mesuree sur 17 244 records.
		{"7344d24f p2068 q128", []byte{0x80}, 128, 0.00390625},
		// 696a9d7c chunk 3 paquet 236, slot 1624 : bruts 0x7F (meme slot, autre film).
		{"696a9d7c p236 q127", []byte{0x7F}, 127, -0.00390625},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			var got []uint64
			prev := navpointHook
			SetNavpointHook(func(f NavpointField, values []uint64) {
				if f != NavpointRadialProgress {
					t.Fatalf("champ publie %v, attendu %v", f, NavpointRadialProgress)
				}
				got = append([]uint64(nil), values...)
			})
			defer SetNavpointHook(prev)

			br := zoneVecBits(t, c.octets...)
			consumeNavpointRadialProgress(br)

			if br.BitPos() != 8 {
				t.Errorf("bits consommes = %d, attendu 8", br.BitPos())
			}
			if len(got) != 1 || got[0] != c.quantum {
				t.Fatalf("quantum publie %v, attendu [%d]", got, c.quantum)
			}
			if v := NavpointRadialProgressValue(got[0]); v != c.valeur {
				t.Errorf("valeur dequantifiee %v, attendu %v", v, c.valeur)
			}
		})
	}
}

// TestManagedObjectBoundaryColorVecteurs : ti=10 i1, 4 x R(8) sur [0, 1].
func TestManagedObjectBoundaryColorVecteurs(t *testing.T) {
	release := LockProcessDecode()
	defer release()

	cas := []struct {
		nom    string
		octets []byte
		quanta [4]uint64
	}{
		// 7344d24f chunk 2 paquet 1684, slot 2478 : bruts 0x358B4A2B.
		{"7344d24f p1684", []byte{0x35, 0x8B, 0x4A, 0x2B}, [4]uint64{53, 139, 74, 43}},
		// 7344d24f chunk 2 paquet 1686, slot 2478 : bruts 0x3541CACB — meme rouge (53), le
		// reste change : c'est bien quatre canaux independants et non un mot unique.
		{"7344d24f p1686", []byte{0x35, 0x41, 0xCA, 0xCB}, [4]uint64{53, 65, 202, 203}},
		// 696a9d7c chunk 3 paquet 506, slot 2060 : bruts 0x77E7C106 (rouge 119, l'un des
		// quatre niveaux dominants mesures : 55, 119, 183, 247).
		{"696a9d7c p506", []byte{0x77, 0xE7, 0xC1, 0x06}, [4]uint64{119, 231, 193, 6}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			var got []uint64
			prev := managedObjectHook
			SetManagedObjectHook(func(f ManagedObjectField, values []uint64) {
				if f != ManagedObjectBoundaryColor {
					t.Fatalf("champ publie %v, attendu %v", f, ManagedObjectBoundaryColor)
				}
				got = append([]uint64(nil), values...)
			})
			defer SetManagedObjectHook(prev)

			br := zoneVecBits(t, c.octets...)
			consumeManagedObjectBoundaryColor(br)

			if br.BitPos() != 32 {
				t.Errorf("bits consommes = %d, attendu 32", br.BitPos())
			}
			if len(got) != 4 {
				t.Fatalf("%d valeurs publiees, attendu 4 (RGBA) : %v", len(got), got)
			}
			for i := range c.quanta {
				if got[i] != c.quanta[i] {
					t.Errorf("canal %d : quantum %d, attendu %d", i, got[i], c.quanta[i])
				}
			}
		})
	}
}

// TestManagedObjectRTPCVecteurs : ti=10 i26..i29, R(32) id puis R(22) si l'id est non nul.
func TestManagedObjectRTPCVecteurs(t *testing.T) {
	release := LockProcessDecode()
	defer release()

	cas := []struct {
		nom     string
		octets  []byte
		id      uint64
		valeur  uint64
		aValeur bool
		bitsLus int
	}{
		// 7344d24f chunk 2 paquet 1656, slot 1544, i26 : bruts 0x0685454080000C40.
		{"i26 p1656", []byte{0x06, 0x85, 0x45, 0x40, 0x80, 0x00, 0x0C, 0x40},
			0x06854540, 0x200003, true, 54},
		// meme canal, deux paquets plus loin : l'identifiant NE BOUGE PAS, la valeur MONTE.
		{"i26 p1716", []byte{0x06, 0x85, 0x45, 0x40, 0x80, 0x00, 0x18, 0x40},
			0x06854540, 0x200006, true, 54},
		// 7344d24f chunk 2 paquet 2066, slot 1621, i27 : autre identifiant, meme forme.
		{"i27 p2066", []byte{0x7C, 0xBF, 0x00, 0x66, 0x80, 0x00, 0xC6, 0x65},
			0x7CBF0066, 0x200031, true, 54},
		// La branche « identifiant nul » : le jeu ecrit une sentinelle SANS lire un bit de
		// plus. Vecteur SYNTHETIQUE et signale comme tel — la branche existe dans le
		// deserialiseur (`FUN_140796d38`) mais aucun record singleton du corpus ne la porte.
		{"id nul (synthetique)", []byte{0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0xFF, 0xFF},
			0, 0, false, 32},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			var got []uint64
			prev := managedObjectHook
			SetManagedObjectHook(func(f ManagedObjectField, values []uint64) {
				if f != ManagedObjectRTPC {
					t.Fatalf("champ publie %v, attendu %v", f, ManagedObjectRTPC)
				}
				got = append([]uint64(nil), values...)
			})
			defer SetManagedObjectHook(prev)

			br := zoneVecBits(t, c.octets...)
			consumeManagedObjectRTPC(br)

			if br.BitPos() != c.bitsLus {
				t.Errorf("bits consommes = %d, attendu %d", br.BitPos(), c.bitsLus)
			}
			if len(got) == 0 || got[0] != c.id {
				t.Fatalf("identifiant publie %v, attendu 0x%X", got, c.id)
			}
			if !c.aValeur {
				if len(got) != 1 {
					t.Fatalf("identifiant nul : %d valeurs publiees, attendu 1 (l'id seul) : %v",
						len(got), got)
				}
				return
			}
			if len(got) != 2 {
				t.Fatalf("%d valeurs publiees, attendu 2 (id + valeur) : %v", len(got), got)
			}
			if got[1] != c.valeur {
				t.Errorf("valeur publiee 0x%X, attendu 0x%X", got[1], c.valeur)
			}
		})
	}
}

// TestManagedObjectRTPCIdentifiantConstant fige le controle qui a valide le cadrage en phase 1a :
// sur les records SINGLETON du meme composant, l'identifiant de 32 bits est le MEME d'un paquet a
// l'autre et d'un film a l'autre. Un decalage de cadrage, meme d'un bit, le ferait varier.
func TestManagedObjectRTPCIdentifiantConstant(t *testing.T) {
	release := LockProcessDecode()
	defer release()

	// i26 sur `7344d24f` (paquets 1656, 1716, 1776) puis sur `696a9d7c` (paquet 1680).
	vecteurs := [][]byte{
		{0x06, 0x85, 0x45, 0x40, 0x80, 0x00, 0x0C, 0x40},
		{0x06, 0x85, 0x45, 0x40, 0x80, 0x00, 0x18, 0x40},
		{0x06, 0x85, 0x45, 0x40, 0x80, 0x00, 0x28, 0x40},
		{0x06, 0x85, 0x45, 0x40, 0x80, 0x00, 0x0C, 0x40},
	}
	const idAttendu = uint64(0x06854540)

	var ids []uint64
	var valeurs []uint64
	prev := managedObjectHook
	SetManagedObjectHook(func(_ ManagedObjectField, values []uint64) {
		ids = append(ids, values[0])
		if len(values) > 1 {
			valeurs = append(valeurs, values[1])
		}
	})
	defer SetManagedObjectHook(prev)

	for _, v := range vecteurs {
		consumeManagedObjectRTPC(zoneVecBits(t, v...))
	}
	for i, id := range ids {
		if id != idAttendu {
			t.Errorf("vecteur %d : identifiant 0x%X, attendu 0x%X (constant par composant)",
				i, id, idAttendu)
		}
	}
	// Les trois premiers viennent de paquets successifs du meme film : la valeur CROIT.
	for i := 1; i < 3; i++ {
		if valeurs[i] <= valeurs[i-1] {
			t.Errorf("valeur %d (0x%X) ne croit pas apres %d (0x%X) : la rampe mesuree en"+
				" phase 1a n'est pas reproduite", i, valeurs[i], i-1, valeurs[i-1])
		}
	}
}

// TestZoneHooksConsommentLesMemesBitsSansHook est le garde-rail de non-regression du chemin de
// production : poser ou retirer un hook ne doit RIEN changer a la consommation de bits, sinon un
// artefact construit avec sonde et un artefact construit sans divergeraient en silence.
//
// Modele : `TestHooksConsumeSameBitsWithoutHook` (components_hooks_test.go). Un fichier a part
// parce que celui-la fait deja 600 lignes.
func TestZoneHooksConsommentLesMemesBitsSansHook(t *testing.T) {
	release := LockProcessDecode()
	defer release()

	octets := []byte{0x06, 0x85, 0x45, 0x40, 0x80, 0x00, 0x0C, 0x40, 0x35, 0x8B, 0x4A, 0x2B}
	cas := []struct {
		nom   string
		deser func(*BitReader)
	}{
		{compNavpointRadialProgress, consumeNavpointRadialProgress},
		{compManagedObjectBoundaryColor, consumeManagedObjectBoundaryColor},
		{compManagedObjectRTPC, consumeManagedObjectRTPC},
		{compManagedObjectBoundaryVisibility, consumeManagedObjectBoundaryVisibility},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			prevM, prevN := managedObjectHook, navpointHook
			SetManagedObjectHook(nil)
			SetNavpointHook(nil)
			brSans := zoneVecBits(t, octets...)
			c.deser(brSans)
			sans := brSans.BitPos()

			SetManagedObjectHook(func(ManagedObjectField, []uint64) {})
			SetNavpointHook(func(NavpointField, []uint64) {})
			brAvec := zoneVecBits(t, octets...)
			c.deser(brAvec)
			avec := brAvec.BitPos()

			SetManagedObjectHook(prevM)
			SetNavpointHook(prevN)

			if sans != avec {
				t.Fatalf("%s : %d bits sans hook contre %d avec — la sonde change la"+
					" consommation", c.nom, sans, avec)
			}
		})
	}
}

// TestZoneDequantification fige la convention RETENUE (milieu d'intervalle) sur ses trois plages,
// pour qu'un changement d'avis soit un changement VISIBLE et non une derive.
func TestZoneDequantification(t *testing.T) {
	cas := []struct {
		nom     string
		got     float32
		attendu float32
	}{
		{"progression radiale, quantum 0", NavpointRadialProgressValue(0), -0.99609375},
		{"progression radiale, quantum 128 (le zero de la plage)", NavpointRadialProgressValue(128), 0.00390625},
		{"progression radiale, quantum 255", NavpointRadialProgressValue(255), 0.99609375},
		{"couleur, quantum 0", ManagedObjectBoundaryColorValue(0), 0.001953125},
		{"couleur, quantum 255", ManagedObjectBoundaryColorValue(255), 0.998046875},
	}
	for _, c := range cas {
		if c.got != c.attendu {
			t.Errorf("%s : %v, attendu %v", c.nom, c.got, c.attendu)
		}
	}
}
