package replay

// held_object_carry_test.go — Tests PURS de la reconstruction des périodes de portage.
// Aucun film : les scénarios couvrent les trois fermetures (lâcher, mort, fin de film) et
// l'agrégation du temps de portage. La validation SUR FILMS vit dans
// bombe_b2_chronologie_test.go (V1/V2/V3, garde ASSAUT_CACHE).

import "testing"

// hocEvents construit des événements de test.
func hocEvents(list ...HeldObjectEvent) []HeldObjectEvent { return list }

func TestBuildHeldObjectCarryLacher(t *testing.T) {
	// Prise à 1000 par le slot 5, lâcher à 4000 : une période fermée, 3 s de portage.
	c := BuildHeldObjectCarry(hocEvents(
		HeldObjectEvent{TimeMS: 1000, Slot: 5, Pickup: true},
		HeldObjectEvent{TimeMS: 4000, Slot: 5, Pickup: false},
	), map[uint32]uint64{5: 42}, nil)
	if len(c.Periods) != 1 {
		t.Fatalf("périodes : %d, attendu 1", len(c.Periods))
	}
	p := c.Periods[0]
	if p.Slot != 5 || p.XUID != 42 || p.DebutMS != 1000 || p.FinMS != 4000 || p.FinParMort || p.Ouverte {
		t.Errorf("période inattendue : %+v", p)
	}
	if c.CarryMSByXUID[42] != 3000 {
		t.Errorf("temps de portage : %d, attendu 3000", c.CarryMSByXUID[42])
	}
}

func TestBuildHeldObjectCarryMortFermeLaPeriode(t *testing.T) {
	// Le porteur meurt à 2500 sans lâcher ; le suivant prend à 6000. La première période
	// se ferme à la MORT (2500), pas à la prise suivante.
	c := BuildHeldObjectCarry(hocEvents(
		HeldObjectEvent{TimeMS: 1000, Slot: 5, Pickup: true},
		HeldObjectEvent{TimeMS: 6000, Slot: 9, Pickup: true},
	), map[uint32]uint64{5: 42, 9: 77}, []Death{{XUID: 42, TimeMS: 2500}})
	if len(c.Periods) != 2 {
		t.Fatalf("périodes : %d, attendu 2", len(c.Periods))
	}
	if p := c.Periods[0]; p.FinMS != 2500 || !p.FinParMort {
		t.Errorf("première période : %+v, attendu fin 2500 par mort", p)
	}
	if c.CarryMSByXUID[42] != 1500 {
		t.Errorf("portage du mort : %d, attendu 1500", c.CarryMSByXUID[42])
	}
}

func TestBuildHeldObjectCarryFinDeFilm(t *testing.T) {
	// Prise sans lâcher ni mort : période OUVERTE, exclue du temps de portage.
	c := BuildHeldObjectCarry(hocEvents(
		HeldObjectEvent{TimeMS: 1000, Slot: 5, Pickup: true},
	), map[uint32]uint64{5: 42}, nil)
	if len(c.Periods) != 1 || !c.Periods[0].Ouverte || c.Periods[0].FinMS != HeldObjectOpenEndMS {
		t.Fatalf("période ouverte attendue : %+v", c.Periods)
	}
	if c.CarryMSByXUID[42] != 0 {
		t.Errorf("une période ouverte ne compte pas : %d", c.CarryMSByXUID[42])
	}
}

func TestBuildHeldObjectCarryLacherOrphelin(t *testing.T) {
	// Un lâcher d'un slot qui ne porte pas (désynchronisation) n'invente pas de période.
	c := BuildHeldObjectCarry(hocEvents(
		HeldObjectEvent{TimeMS: 1000, Slot: 5, Pickup: true},
		HeldObjectEvent{TimeMS: 2000, Slot: 9, Pickup: false},
		HeldObjectEvent{TimeMS: 3000, Slot: 5, Pickup: false},
	), map[uint32]uint64{5: 42}, nil)
	if len(c.Periods) != 1 || c.Periods[0].FinMS != 3000 {
		t.Fatalf("le lâcher orphelin ne doit rien fermer : %+v", c.Periods)
	}
}

func TestBuildHeldObjectCarrySlotNonPonte(t *testing.T) {
	// Un slot sans xuid porte quand même (XUID 0), mais n'entre pas dans l'agrégat.
	c := BuildHeldObjectCarry(hocEvents(
		HeldObjectEvent{TimeMS: 1000, Slot: 5, Pickup: true},
		HeldObjectEvent{TimeMS: 2000, Slot: 5, Pickup: false},
	), nil, nil)
	if len(c.Periods) != 1 || c.Periods[0].XUID != 0 {
		t.Fatalf("période sans identité attendue : %+v", c.Periods)
	}
	if len(c.CarryMSByXUID) != 0 {
		t.Errorf("aucun agrégat attendu : %v", c.CarryMSByXUID)
	}
}
