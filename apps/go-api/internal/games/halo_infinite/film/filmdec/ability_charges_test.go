package filmdec

// ability_charges_test.go — les pièces PURES du balayage des charges : la résolution du
// composant par son nom EXACT, les quartets, le masque, l'ordre total et les
// dénominateurs. Le balayage lui-même se juge sur pièces (golden d'assemblage du paquet
// `replay`, et test d'acceptation contre le relevé Theater — ability_charges_film_test.go).
//
// AUCUN ORACLE CIRCULAIRE (leçon H1 de la revue P3) : les valeurs attendues sont écrites
// EN DUR — jamais lues des constantes de production ni recalculées par ses formules.

import "testing"

func TestAbilityChargeResolutionParNomExact(t *testing.T) {
	// LA RÉSOLUTION EST EXACTE, jamais par préfixe : « biped-spartan-ability » (i57) est un
	// PRÉFIXE de « biped-spartan-ability-energy » (i56). Un appariement par préfixe, dans un
	// sens ou dans l'autre, lirait le mauvais composant — donc publierait des tags
	// d'impulsion comme des charges.
	comps := make([]string, 60)
	comps[56] = "biped-spartan-ability-energy"
	comps[57] = "biped-spartan-ability"
	arch := Archetype{Components: comps}
	if got := componentIndexOfAny(arch, abilityEnergyName, abilityEnergyNameAlt); got != 56 {
		t.Fatalf("i56 resolu a %d, attendu 56 (etiquette sans suffixe)", got)
	}
	// Le voisin-prefixe reste resolu chez LUI : la presence d'i56 ne le detourne pas.
	if got := componentIndexOfAny(arch, abilityPredictedName, abilityPredictedNameAlt); got != 57 {
		t.Fatalf("i57 resolu a %d, attendu 57 — la resolution a apparie un prefixe", got)
	}
	// L'etiquette AVEC suffixe se resout aussi, et au meme index.
	comps[56] = "biped-spartan-ability-energy-component"
	if got := componentIndexOfAny(Archetype{Components: comps},
		abilityEnergyName, abilityEnergyNameAlt); got != 56 {
		t.Fatalf("i56 (etiquette -component) resolu a %d, attendu 56", got)
	}
	if got := componentIndexOfAny(Archetype{Components: []string{"autre"}},
		abilityEnergyName, abilityEnergyNameAlt); got != -1 {
		t.Fatalf("composant absent resolu a %d, attendu -1", got)
	}
}

// TestAbilityChargePublieLesQuartetsEnDur — LA VALEUR 7 BITS SE COUPE EN DEUX QUARTETS, et
// les attendus sont ÉCRITS À LA MAIN : un test qui recalculerait `(v>>4)&0xF` depuis la
// même formule que la production passerait pour n'importe quel découpage.
func TestAbilityChargePublieLesQuartetsEnDur(t *testing.T) {
	cas := []struct {
		v            int
		charges, low int
		quoi         string
	}{
		{64, 4, 0, "la premiere lecture de la serie temoin de 1cd3848a (R11 §2)"},
		{48, 3, 0, "la deuxieme lecture de la serie temoin"},
		{0, 0, 0, "la derniere : plus aucune charge"},
		{127, 7, 15, "0x7F — la valeur PLEINE que le moteur pose, si un film la transmettait"},
		{47, 2, 15, "une recharge fractionnaire non nulle voyage dans le quartet bas"},
	}
	for _, c := range cas {
		var st AbilityChargeStats
		sc := &abilityChargeScanner{st: &st, mask: 0b001}
		sc.ch[0] = c.v
		sc.publish(512, 3, FilmPacket{Index: 7, TimestampUS: 1_234_000})
		if len(sc.out) != 1 {
			t.Fatalf("valeur %d (%s) : %d sortie(s), attendu 1", c.v, c.quoi, len(sc.out))
		}
		got := sc.out[0]
		if got.Charges != c.charges || got.Low != c.low {
			t.Fatalf("valeur %d (%s) : charges=%d bas=%d, attendu %d/%d",
				c.v, c.quoi, got.Charges, got.Low, c.charges, c.low)
		}
		if got.Slot != 512 || got.Chunk != 3 || got.PacketIndex != 7 || got.TimestampUS != 1_234_000 {
			t.Fatalf("lecture %+v : la localisation du paquet ne voyage pas", got)
		}
	}
}

// TestAbilityChargeMasqueNonArmeNePublieRien — LE PIÈGE (a) DE R11 : un bit de masque à 0
// signifie « le moteur pose 0x7F » (plein), ce N'EST PAS une lecture. Publier un
// emplacement non armé fabriquerait une charge pleine à chaque record — le film ne
// transmet rien au ramassage.
func TestAbilityChargeMasqueNonArmeNePublieRien(t *testing.T) {
	var st AbilityChargeStats
	sc := &abilityChargeScanner{st: &st, mask: 0b101}
	sc.ch = [AbilityEnergyCharges]int{32, AbilityEnergyUnarmed, 16}
	sc.publish(512, 3, FilmPacket{Index: 7, TimestampUS: 1_000})
	if len(sc.out) != 2 || st.Armed != 2 {
		t.Fatalf("%d sortie(s), armes=%d : attendu 2/2 — un emplacement non arme a ete publie, "+
			"ou un arme a ete perdu", len(sc.out), st.Armed)
	}
	if sc.out[0].Emplacement != 0 || sc.out[1].Emplacement != 2 {
		t.Fatalf("emplacements %d et %d, attendu 0 et 2 (e1 n'est pas arme)",
			sc.out[0].Emplacement, sc.out[1].Emplacement)
	}
	// Masque 000 : le composant a parlé — « aucun emplacement armé » — et rien ne sort.
	// C'est le zéro que R11 §4 mesure 485 fois sur les six films sans grappin ni propulseur.
	st, sc.out = AbilityChargeStats{}, nil
	sc.st, sc.mask = &st, 0
	sc.ch = [AbilityEnergyCharges]int{AbilityEnergyUnarmed, AbilityEnergyUnarmed, AbilityEnergyUnarmed}
	sc.publish(512, 3, FilmPacket{Index: 8})
	if len(sc.out) != 0 || st.Armed != 0 {
		t.Fatalf("masque 000 : %d sortie(s), armes=%d — une non-lecture a ete publiee",
			len(sc.out), st.Armed)
	}
}

// TestAbilityChargeDenominateurs — un composant annoncé et non atteint est une lecture
// PERDUE, comptée à part : sans ce dénominateur, une liste courte ne se distingue pas d'un
// film pauvre en changements de charge.
func TestAbilityChargeDenominateurs(t *testing.T) {
	var st AbilityChargeStats
	sc := &abilityChargeScanner{st: &st, idx: 56}
	// La marche n'atteint pas i56 (hook jamais déclenché) : Unread, rien de publié.
	sc.got = false
	sc.account(nil, 0, 0, []int{0, 56}, 512, 1, FilmPacket{})
	if st.WithI56 != 1 || st.Unread != 1 || st.Read != 0 || len(sc.out) != 0 {
		t.Fatalf("stats %+v, sorties=%d : attendu WithI56=1 Unread=1 Read=0 et rien de publie",
			st, len(sc.out))
	}
	// Un record dont le masque n'annonce PAS i56 ne compte nulle part ici.
	sc.account(nil, 0, 0, []int{0, 21}, 512, 1, FilmPacket{})
	if st.WithI56 != 1 {
		t.Fatalf("WithI56=%d : un record sans i56 a ete compte", st.WithI56)
	}
}

func TestSortAbilityChargesOrdreTotal(t *testing.T) {
	// L'ORDRE EST TOTAL — instant, puis slot, puis emplacement : sans le troisième critère,
	// deux emplacements armés du même record sortiraient dans l'ordre du parcours, et
	// l'artefact dépendrait de rien de mesurable.
	out := []AbilityCharge{
		{Slot: 20, TimestampUS: 5, Emplacement: 0},
		{Slot: 10, TimestampUS: 5, Emplacement: 2},
		{Slot: 10, TimestampUS: 5, Emplacement: 0},
		{Slot: 10, TimestampUS: 1, Emplacement: 1},
	}
	sortAbilityCharges(out)
	want := []AbilityCharge{
		{Slot: 10, TimestampUS: 1, Emplacement: 1},
		{Slot: 10, TimestampUS: 5, Emplacement: 0},
		{Slot: 10, TimestampUS: 5, Emplacement: 2},
		{Slot: 20, TimestampUS: 5, Emplacement: 0},
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("position %d = %+v, attendu %+v (ordre %+v)", i, out[i], want[i], out)
		}
	}
}
