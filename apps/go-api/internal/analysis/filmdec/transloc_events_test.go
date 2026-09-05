package filmdec

// transloc_events_test.go — la forme du record d'événement 117, figée sur des octets
// synthétiques : la tête (type, slot) ET la charge (le va-et-vient quantifié). Le décodage
// sur film réel est couvert par les tests gatés (transloc_exemption_film_test.go,
// transloc_positions_film_test.go) et par la validation R6 (18/18 sur 5 films).

import (
	"math"
	"testing"
)

// translocWriter écrit des champs de bits MSB-first — l'ordre du flux du moteur.
type translocWriter struct {
	pay []byte
	at  int
}

func (w *translocWriter) put(n int, v uint64) {
	for need := (w.at + n + 7) / 8; len(w.pay) < need; {
		w.pay = append(w.pay, 0)
	}
	for i := 0; i < n; i++ {
		if v>>(uint(n-1-i))&1 == 1 {
			w.pay[(w.at+i)/8] |= 1 << (7 - uint((w.at+i)%8))
		}
	}
	w.at += n
}

// translocHeadBits / translocPayloadStartBits : les largeurs du layout, recomposées ICI pour
// que les points de coupe des tests de troncature se CALCULENT au lieu d'être choisis à la
// main. translocPayloadStartBits est le premier bit du DÉPART, mot d'effet présent.
const (
	translocHeadBits         = 1 + 1 + 7 + 1 + translocRefWidth + translocGenBits
	translocPayloadStartBits = translocHeadBits + 1 + 1 + 1 + translocEffectWordBits
)

// translocVecBits rend la largeur d'un vecteur quantifié sous la porte de région : la porte,
// l'index de région, puis les trois axes de la carte.
func translocVecBits(e MapQuantEntry) int {
	n := 1 + int(e.EffectiveRegionIndexBits())
	for ax := 0; ax < 3; ax++ {
		n += int(e.AxisWidths[ax])
	}
	return n
}

// translocHead écrit l'en-tête commun : [1 config][1 présence][7 type][1 porte ref0]
// [8 index][2 gen] — le type 117 (1110101) met le premier octet à 0xFA, son bit de poids
// faible étant le PREMIER bit du deuxième octet (rapport R1 §4.2).
func translocHead(idx uint32) *translocWriter {
	w := &translocWriter{}
	w.put(1, 1) // configuration
	w.put(1, 1) // présence : la liste porte un événement
	w.put(7, translocEventType)
	w.put(1, 1)           // porte de ref0
	w.put(8, uint64(idx)) // l'index de l'unité (domaine 2)
	w.put(2, 1)           // génération
	return w
}

// translocPacket fabrique un paquet dont la tête est un événement 117 désignant idx, SANS
// charge lisible (le tampon s'arrête après ref0).
func translocPacket(idx uint32) []byte {
	w := translocHead(idx)
	for len(w.pay) < 4 {
		w.pay = append(w.pay, 0)
	}
	return w.pay
}

// translocTestEntry est une entrée de catalogue aux bornes CHOISIES POUR ÊTRE CALCULABLES À
// LA MAIN : 10 bits par axe sur [0, 1024] donnent un pas de 1 m, donc une position de
// q + 0,5 exactement (déquantification à mi-quantum). Aucun chiffre du corpus ici : ce test
// verrouille l'ORDRE DES CHAMPS, pas la loi de quantification (map_quant_control_test.go).
func translocTestEntry() MapQuantEntry {
	return MapQuantEntry{
		Module:     "essai",
		Min:        [3]float32{0, 0, 0},
		Max:        [3]float32{1024, 1024, 1024},
		AxisWidths: [3]uint{10, 10, 10},
	}
}

// translocJumpPacket fabrique un paquet 117 COMPLET : en-tête, mot d'effet gardé, puis les
// deux positions quantifiées sous la porte de région (bit à 0 — la porte INVERSÉE).
func translocJumpPacket(idx uint32, e MapQuantEntry, region uint64, qa, qb [3]uint32) []byte {
	w := translocHead(idx)
	w.put(1, 0) // porte de ref1
	w.put(1, 0) // porte de ref2
	w.put(1, 1) // porte du mot d'effet
	w.put(translocEffectWordBits, 0xA1344FC2)
	for _, q := range [][3]uint32{qa, qb} {
		w.put(1, 0) // porte de région à 0 = bornes de la carte (piège n°1)
		w.put(int(e.EffectiveRegionIndexBits()), region)
		for ax := 0; ax < 3; ax++ {
			w.put(int(e.AxisWidths[ax]), uint64(q[ax]))
		}
	}
	w.put(1, 1) // bit de continuation de liste (un type 15 `Script` suit, R6 §3.2)
	return w.pay
}

func TestDecodeTranslocHead(t *testing.T) {
	pay := translocPacket(23)
	if pay[0] != translocFamilyByte {
		t.Fatalf("premier octet fabriqué %#x, attendu %#x — la trame d'essai contredit la forme"+
			" mesurée du record", pay[0], translocFamilyByte)
	}
	ev, ok := decodeTranslocHead(pay, 42_000, nil)
	if !ok || ev.Slot != 535 || ev.TimestampUS != 42_000 {
		t.Fatalf("décodage = %+v (ok=%v), attendu slot 535 (index 23 + base 512) @42000us", ev, ok)
	}
	if ev.HasPositions {
		t.Errorf("un paquet sans charge lisible a rendu des positions : %+v", ev)
	}
}

func TestDecodeTranslocHeadRefuse(t *testing.T) {
	t.Run("type voisin", func(t *testing.T) {
		pay := translocPacket(23)
		pay[1] &^= 0x80 // le bit de poids faible du type : 117 -> 116
		if _, ok := decodeTranslocHead(pay, 0, nil); ok {
			t.Fatal("un type 116 a été lu comme une téléportation")
		}
	})
	t.Run("liste vide", func(t *testing.T) {
		pay := translocPacket(23)
		pay[0] &^= 0x40 // bit de présence à 0
		if _, ok := decodeTranslocHead(pay, 0, nil); ok {
			t.Fatal("une liste vide a rendu un événement")
		}
	})
	t.Run("unite non designee", func(t *testing.T) {
		pay := translocPacket(23)
		pay[1] &^= 0x40 // porte de ref0 à 0
		if _, ok := decodeTranslocHead(pay, 0, nil); ok {
			t.Fatal("un événement sans référence d'unité a rendu un slot — un slot ne se devine pas")
		}
	})
}

// TestDecodeTranslocJump fige l'ORDRE DE LA CHARGE : position A = DÉPART, position B =
// ARRIVÉE (R6 §1.3, 18/18 dans cet ordre). Une inversion des deux lectures fait tomber ce test.
func TestDecodeTranslocJump(t *testing.T) {
	e := translocTestEntry()
	pay := translocJumpPacket(23, e, 0, [3]uint32{100, 200, 300}, [3]uint32{400, 500, 600})
	ev, ok := decodeTranslocHead(pay, 7_000, &e)
	if !ok || !ev.HasPositions {
		t.Fatalf("décodage = %+v (ok=%v) : la charge complète devait être lue", ev, ok)
	}
	want := [2][3]float32{{100.5, 200.5, 300.5}, {400.5, 500.5, 600.5}}
	got := [2][3]float32{ev.From, ev.To}
	for i, nom := range []string{"départ (A)", "arrivée (B)"} {
		for ax := 0; ax < 3; ax++ {
			if math.Abs(float64(got[i][ax]-want[i][ax])) > 1e-3 {
				t.Fatalf("%s axe %d = %.3f, attendu %.3f (déquantification à mi-quantum) — "+
					"l'ordre ou les largeurs de la charge ont bougé", nom, ax, got[i][ax], want[i][ax])
			}
		}
	}
}

// TestDecodeTranslocJumpPorteInversee prouve le PIÈGE n°1 sur pièces synthétiques : le bit de
// région à UN sélectionne les bornes par défaut du moteur (±20000, 22 bits), pas celles de la
// carte. Lire la porte dans le sens naïf donnerait des positions fausses silencieuses.
func TestDecodeTranslocJumpPorteInversee(t *testing.T) {
	e := translocTestEntry()
	w := translocHead(23)
	w.put(1, 0) // ref1
	w.put(1, 0) // ref2
	w.put(1, 0) // pas de mot d'effet : la charge enchaîne sur les positions
	for i := 0; i < 2; i++ {
		w.put(1, 1) // porte de région à 1 = bornes PAR DÉFAUT du moteur
		for ax := 0; ax < 3; ax++ {
			w.put(translocDefaultAxisBits, uint64(1<<21)) // le milieu de la plage
		}
	}
	w.put(1, 1)
	ev, ok := decodeTranslocHead(w.pay, 0, &e)
	if !ok || !ev.HasPositions {
		t.Fatalf("décodage = %+v (ok=%v) : la branche des bornes par défaut devait être lue", ev, ok)
	}
	// Milieu de [-20000, 20000] au quantum près : le demi-pas vaut 40000/2^23 ≈ 0,0048 m.
	for ax := 0; ax < 3; ax++ {
		if math.Abs(float64(ev.From[ax])) > 0.01 || math.Abs(float64(ev.To[ax])) > 0.01 {
			t.Fatalf("axe %d : from=%.4f to=%.4f, attendu ~0 (milieu des bornes par défaut) — "+
				"la porte de région a été lue dans le mauvais sens", ax, ev.From[ax], ev.To[ax])
		}
	}
}

// TestDecodeTranslocJumpDegradation vérifie la DÉGRADATION HONNÊTE : sans bornes utilisables,
// ou sur une région que le catalogue ne décrit pas, l'événement sort daté et attribué mais
// SANS positions — jamais une position fausse.
func TestDecodeTranslocJumpDegradation(t *testing.T) {
	e := translocTestEntry()
	pay := translocJumpPacket(23, e, 0, [3]uint32{100, 200, 300}, [3]uint32{400, 500, 600})
	t.Run("carte hors catalogue", func(t *testing.T) {
		ev, ok := decodeTranslocHead(pay, 0, nil)
		if !ok || ev.Slot != 535 {
			t.Fatalf("l'événement doit rester lu sans bornes : %+v (ok=%v)", ev, ok)
		}
		if ev.HasPositions {
			t.Fatal("des positions ont été rendues sans bornes de carte")
		}
	})
	t.Run("entree sans largeurs", func(t *testing.T) {
		vide := MapQuantEntry{Min: e.Min, Max: e.Max}
		if ev, _ := decodeTranslocHead(pay, 0, &vide); ev.HasPositions {
			t.Fatalf("une entrée sans largeurs d'axe a rendu des positions : %+v", ev)
		}
	})
	t.Run("autre region", func(t *testing.T) {
		// Le paquet porte l'index de région 1 ; le catalogue décrit la région 0.
		autre := translocJumpPacket(23, e, 1, [3]uint32{100, 200, 300}, [3]uint32{400, 500, 600})
		if ev, _ := decodeTranslocHead(autre, 0, &e); ev.HasPositions {
			t.Fatalf("une région étrangère au catalogue a été déquantifiée quand même : %+v", ev)
		}
	})
	// LA COUPE TOMBE ENTRE LES DEUX VECTEURS, et c'est ce qui rend le test DISCRIMINANT
	// (revue P1bis ronde 1, G4) : le DÉPART tient entièrement dans le tampon, l'ARRIVÉE en
	// est ENTIÈREMENT absente. Seule la garde de débordement de `readTranslocVec` peut alors
	// refuser — la retirer fait publier les zéros du padding de queue comme une arrivée. Le
	// point de coupe est CALCULÉ du layout, jamais choisi à la main.
	t.Run("troncature entre les deux vecteurs", func(t *testing.T) {
		finA := translocPayloadStartBits + translocVecBits(e)
		court := pay[:(finA+7)/8]
		if len(court) >= len(pay) {
			t.Fatalf("la coupe calculée (%d octets) ne tronque rien sur %d", len(court), len(pay))
		}
		ev, ok := decodeTranslocHead(court, 0, &e)
		if !ok || ev.Slot != 535 {
			t.Fatalf("l'événement doit rester lu sur une charge tronquée : %+v (ok=%v)", ev, ok)
		}
		if ev.HasPositions {
			t.Fatalf("l'ARRIVÉE est absente du tampon et a pourtant été publiée (%v) — le "+
				"padding de queue a été lu comme une position", ev.To)
		}
	})
	// PORTE DE REF1/REF2 : le seul chemin où l'ordre de lecture des deux portes pouvait se
	// perdre (le `||` court-circuitait la seconde). L'événement doit rester daté et attribué,
	// et sortir SANS positions — jamais la charge décalée d'un bit.
	t.Run("ref1 presente", func(t *testing.T) {
		w := translocHead(23)
		w.put(1, 1) // porte de ref1 PRÉSENTE : la charge n'est plus lisible
		w.put(1, 0) // porte de ref2
		w.put(1, 0)
		for i := 0; i < 2; i++ {
			w.put(1, 0)
			w.put(int(e.EffectiveRegionIndexBits()), 0)
			for ax := 0; ax < 3; ax++ {
				w.put(int(e.AxisWidths[ax]), 100)
			}
		}
		ev, ok := decodeTranslocHead(w.pay, 0, &e)
		if !ok || ev.Slot != 535 {
			t.Fatalf("l'événement doit rester lu malgré la ref1 : %+v (ok=%v)", ev, ok)
		}
		if ev.HasPositions {
			t.Fatalf("une charge décalée par la ref1 a été publiée comme des positions : %+v", ev)
		}
	})
	t.Run("troncature au milieu du depart", func(t *testing.T) {
		court := pay[:(translocPayloadStartBits+8)/8]
		if ev, _ := decodeTranslocHead(court, 0, &e); ev.HasPositions {
			t.Fatalf("un DÉPART tronqué a rendu des positions : %+v", ev)
		}
	})
}

// TestTranslocEntryUsable fige le refus des entrées de catalogue hors enveloppe : une largeur
// aberrante commande le nombre de bits consommés, la lire ferait décoder les bits du voisin.
func TestTranslocEntryUsable(t *testing.T) {
	ok := translocTestEntry()
	if !translocEntryUsable(&ok) {
		t.Fatal("l'entrée d'essai devrait être utilisable")
	}
	if translocEntryUsable(nil) {
		t.Error("une entrée absente a été jugée utilisable")
	}
	large := ok
	large.AxisWidths[1] = translocMaxAxisBits + 1
	if translocEntryUsable(&large) {
		t.Error("une largeur d'axe hors de la loi du moteur a été acceptée")
	}
	plates := ok
	plates.Max[2] = plates.Min[2]
	if translocEntryUsable(&plates) {
		t.Error("des bornes plates ont été acceptées : la déquantification y diviserait par zéro d'étendue")
	}
	regions := ok
	regions.RegionIndexBits = translocMaxRegionBits + 1
	if translocEntryUsable(&regions) {
		t.Error("une largeur d'index de région aberrante a été acceptée")
	}
}
