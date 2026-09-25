package grammar

import "testing"

// composants_vue_b_m4b_test.go — LES LARGEURS DES PORTS DU LOT M4b, sur des flux synthetiques.
//
// Chaque cas ecrit la grammaire relue a l ecrivain (`composants_vue_b_m4b.go`) et exige que
// `consumeByName` la PORTE et consomme EXACTEMENT ses bits. Avant le lot, ces composants rendaient
// `ported = false` (la vue B s ouvrait) : les cas sont rouges sur la base `c9ef97ec6`.

// casDeLargeur est un flux synthetique d un composant et la largeur qu il doit consommer.
type casDeLargeur struct {
	nom     string
	comp    string
	ecrire  func(w *bitw)
	largeur int
}

func TestComposantsVueBM4bLargeurs(t *testing.T) {
	cas := []casDeLargeur{
		{"ti=47 i2 poignee absente", "personal-ai-data-component",
			func(w *bitw) { w.put(0xffffffff, 32) }, 32},
		{"ti=47 i2 poignee sans valeur", "personal-ai-data-component",
			func(w *bitw) { w.put(0x40000123, 32); w.put(0, 1) }, 33},
		{"ti=47 i2 poignee et valeur", "personal-ai-data-component",
			func(w *bitw) { w.put(0x40000123, 32); w.put(1, 1); w.put(0x5a5, 12) }, 45},
		{"ti=5 i22 porte fermee", "player-aim-assist-component",
			func(w *bitw) { w.put(0, 1); w.put(1, 1) }, 2},
		{"ti=5 i24 trois elements vides", "player-desired-frame-configuration-component",
			func(w *bitw) {
				w.put(0, 1) // FUN_14080d69c : pas de poignee
				for i := 0; i < 3; i++ {
					w.put(1, 1) // FUN_1406d1024 : porte inversee, pas de R(6)
					w.put(0, 1) // g1
					w.put(0, 1) // g2
				}
			}, 10},
		{"ti=10 i24 son en boucle", "managed-object-looping-sound-component",
			func(w *bitw) { w.put(0xdeadbeef, 32) }, 32},
		{"ti=40 i37 minuteur EMP", "vehicle-emp-timer-component",
			func(w *bitw) { w.put(0xa5, 8) }, 8},
		{"ti=40 i34 mode 2 (bruts)", "vehicle-type-physics-component",
			func(w *bitw) { w.put(1, 1); w.pad(192 + 96) }, 1 + 192 + 96},
		{"ti=40 i34 mode 0 (quantifies, vitesse absente)", "vehicle-type-physics-component",
			func(w *bitw) { w.put(0, 1); w.put(1, 1); w.put(0x7f, 8); w.put(1, 1) }, 1 + 9 + 1},
	}
	for _, c := range cas {
		w := &bitw{}
		c.ecrire(w)
		w.put(0x2a, 8) // un marqueur apres le composant : la lecture ne doit pas l entamer
		br := lecteurDInstrument(append(w.buf, make([]byte, 16)...))
		_, _, porte := consumeByName(br, c.comp, 0, 1)
		if !porte {
			t.Errorf("%s : composant NON porte (la vue B s ouvrirait)", c.nom)
			continue
		}
		if got := br.BitPos(); got != c.largeur {
			t.Errorf("%s : %d bits consommes, %d attendus", c.nom, got, c.largeur)
		}
		if marque := br.ReadBits(8); marque != 0x2a {
			t.Errorf("%s : le marqueur qui suit vaut %#x, 0x2a attendu", c.nom, marque)
		}
	}
}

// ecrireCorpsDeMort ecrit le corps lourd d `object-dead-state` d un bipede (`FUN_140c1dce0` puis
// `FUN_140c1dd44`), references absentes, avec l octet `comp+0x1c` donne ; `vitesse` ecrit le bloc
// de vitesse (R(2) + 3 x R(14)) que cet octet annonce.
func ecrireCorpsDeMort(w *bitw, octet1c uint64, vitesse bool) int {
	w.put(1, 1)       // Mort
	w.put(0, 1)       // FUN_14080d69c : pas de poignee d animation
	w.put(octet1c, 8) // FUN_140c1e3f0 -> comp+0x1c
	w.put(1, 1)       // enum A absent
	w.put(1, 1)       // enum B absent
	w.put(3, 4)       // R(4)
	w.put(2, 3)       // R(3)
	w.put(0, 1)       // bloc de reference absent
	w.put(5, 4)       // R(4) -> +0x38
	w.put(6, 4)       // FUN_1407f1f24 R(4)
	w.put(1, 1)       // FUN_1407f1e4c : porte posee, pas de R(10)
	w.put(0, 1)       // FUN_14076dc04 absent
	n := 1 + 1 + 8 + 1 + 1 + 4 + 3 + 1 + 4 + 4 + 1 + 1
	if vitesse {
		w.put(1, 2)
		w.pad(3 * 14)
		n += 2 + 3*14
	}
	w.put(9, 5)  // FUN_1424cd17c
	w.put(10, 5) // FUN_1424cd150
	w.put(0, 1)  // FUN_14080d69c de queue : absent
	w.put(1, 1)  // comp+0xc4 (bipede)
	return n + 5 + 5 + 1 + 1
}

// TestCorpsDeMortLitLaVitesseAnnonceeEtUnSeulScalaire eprouve la queue du corps de mort relue au
// lot M4b : l octet de tete `comp+0x1c` (bit 0x10) ANNONCE le bloc de vitesse, et
// `FUN_1424cd17c` n est appele qu UNE fois. La base lisait un R(5) de trop et jamais la vitesse :
// les deux cas y sont rouges (5 bits de trop, puis 39 de trop peu).
func TestCorpsDeMortLitLaVitesseAnnonceeEtUnSeulScalaire(t *testing.T) {
	for _, c := range []struct {
		nom     string
		octet   uint64
		vitesse bool
	}{
		{"octet sans le bit 0x10", 0x2f, false},
		{"octet avec le bit 0x10", 0x10, true},
	} {
		w := &bitw{}
		attendu := ecrireCorpsDeMort(w, c.octet, c.vitesse)
		w.put(0x2a, 8)
		br := lecteurDInstrument(append(w.buf, make([]byte, 16)...))
		ds := consumeObjectDeadStateBipedTI(br, BipedTypeIndex)
		if !ds.Mort {
			t.Errorf("%s : drapeau de mort perdu", c.nom)
		}
		if got := br.BitPos(); got != attendu {
			t.Errorf("%s : %d bits consommes, %d attendus", c.nom, got, attendu)
		}
		if marque := br.ReadBits(8); marque != 0x2a {
			t.Errorf("%s : le marqueur qui suit vaut %#x, 0x2a attendu", c.nom, marque)
		}
	}
}

// TestCorpsDActionDeMobiliteLitLesPositionsDeLaCarte eprouve les deux positions du corps d `i54`
// (`FUN_1408f02c8`, niveau 0x10 aux deux sites) : largeurs ABSOLUES de la carte. Flux nul : les
// deux portes du corps a 0 (position et avant/haut presents), index de region ecrit. La base lisait
// les deux positions aux largeurs du delta du bipede (6/6/6) : rouge.
func TestCorpsDActionDeMobiliteLitLesPositionsDeLaCarte(t *testing.T) {
	br := lecteurDInstrument(make([]byte, 256))
	consumeMobilityActionBody(br)
	p := br.worldObjectPrecision()
	position := 1 + int(p.IndexW) + int(p.AxisW[0]+p.AxisW[1]+p.AxisW[2])
	attendu := 1 + 1 + position + (1 + 19 + 8) + 96 + position + 3*36 + 2*24 + 36 + 2*10 + 1 + 7 + 2 + 1
	if got := br.BitPos(); got != attendu {
		t.Fatalf("%d bits consommes, %d attendus (position de %d bits)", got, attendu, position)
	}
}
