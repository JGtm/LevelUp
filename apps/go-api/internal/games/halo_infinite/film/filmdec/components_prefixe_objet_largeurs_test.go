package filmdec

// components_prefixe_objet_largeurs_test.go — LES LARGEURS DU PREFIXE OBJET SOUS GARDE-RAIL
// (lot 1.9.1 bis, pas 2, 2026-09-15).
//
// # POURQUOI CE FICHIER EXISTE
//
// Le pas 1 a nomme six composants du prefixe « objet du monde » comme ceux qui font franchir la
// frontiere du record sur `ti=37` : i15, i6, i14, i9, i17, i7. Le pas 2 les a relus CHEZ
// L ECRIVAIN, un par un, adresse resolue par le registre ECS du binaire (chaine
// `nom -> thunk de nom -> vtable+0x08 -> vtable_base+0x30`, la meme que celle du lot 1.3) :
//
//	i6  `object-region-state-component`        FUN_140e1bfa0  vtable 143d0b8b0
//	i7  `object-damage-sections-component`     FUN_142f03c80  vtable 143d0b860
//	i9  `object-multiplayer-properties-comp.`  FUN_140f53308 -> FUN_1407d4c94  vtable 143d0b9a0
//	i14 `object-dissolver-component`           FUN_140dd9f9c  vtable 143d0bf78
//	i15 `object-low-frequency-component`       FUN_1407ef088  vtable 143d0bf28
//	i17 `object-frame-configuration-comp.`     FUN_1407f0534 -> FUN_1407f0550  vtable 143d0be80
//
// AUCUNE LARGEUR N A CHANGE : les six portages etaient deja bit-exacts. C est exactement ce
// qu il faut verrouiller — une relecture qui ne corrige rien ne laisse aucune trace, et la
// suivante repartira de zero. i14 avait deja son garde-rail (`components_object_state_test.go`,
// lot 6.10 bis) ; i9 a le sien par la table du flux TLV. Ce fichier pose celui des quatre
// autres, ECRIT EN COMPTES EN CLAIR : un test qui reutilise la constante qu il verifie ne
// verifie rien.
//
// CHAQUE CAS EST UNE MUTATION QUI ROUGIT : changer une largeur du deserialiseur (R(6) du compte
// d i6 en R(5), R(7) de la section d i7 en R(8), R(12) de l image-cle d i15, R(6) du compte
// d i17) fait echouer au moins une ligne de ce fichier.

import "testing"

// TestConsumeObjectRegionStateLargeurs fige les deux boucles d i6 (FUN_140e1bfa0).
//
// GRAMMAIRE RELUE : R(1) present -> [0xc5] ; R(6) compte -> [0xc6] ; compte x R(3) -> [0x148+i] ;
// si present : compte x R(10) -> [0xc8+]. La seconde boucle est gardee par le R(1) DE TETE,
// pas par une valeur lue dans la boucle.
func TestConsumeObjectRegionStateLargeurs(t *testing.T) {
	cas := []struct {
		nom            string
		present        uint64
		compte         uint64
		bits           int
		commentaireFin string
	}{
		// 7 = 1 (present) + 6 (compte) + 0 region.
		{"compte nul, absent : la tete seule", 0, 0, 7, "aucune region"},
		{"compte nul, present : la tete seule", 1, 0, 7, "aucune region"},
		// 7 + 3 = 10 : une region, sans la boucle des dix bits.
		{"une region, absent", 0, 1, 10, "R(3) seul"},
		// 7 + 3 + 10 = 20 : une region, les deux boucles.
		{"une region, present", 1, 1, 20, "R(3) puis R(10)"},
		// 7 + 63*3 + 63*10 = 826 : le MAXIMUM lisible, celui que la mesure du pas 1 observe.
		{"63 regions, present : le maximum", 1, 63, 826, "les deux boucles pleines"},
	}
	for _, c := range cas {
		w := &bitw{}
		w.put(c.present, 1)
		w.put(c.compte, 6)
		br := LecteurSur(append(w.buf, make([]byte, 256)...))
		consumeObjectRegionState(br)
		if br.BitPos() != c.bits {
			t.Errorf("i6 %s : %d bits consommes, %d attendus (%s)", c.nom, br.BitPos(), c.bits, c.commentaireFin)
		}
	}
}

// TestConsumeObjectDamageSectionsLargeurs fige la boucle d i7 (FUN_142f03c80).
//
// GRAMMAIRE RELUE : R(6) compte -> [0x168] ; compte x { R(1) ; si 1 : R(7) dequantifie
// (142f03dac : MOV dword ptr [RSP + 0x20],0x7) + R(16) }.
func TestConsumeObjectDamageSectionsLargeurs(t *testing.T) {
	cas := []struct {
		nom     string
		compte  uint64
		drapeau uint64
		bits    int
	}{
		{"compte nul", 0, 0, 6},
		{"une section, drapeau a zero", 1, 0, 7},
		{"une section, drapeau a un", 1, 1, 30},
		{"quatre sections, drapeaux a zero", 4, 0, 10},
		{"quatre sections, drapeaux a un", 4, 1, 102},
	}
	for _, c := range cas {
		w := &bitw{}
		w.put(c.compte, 6)
		for i := uint64(0); i < c.compte; i++ {
			w.put(c.drapeau, 1)
			if c.drapeau != 0 {
				w.put(0, 7)
				w.put(0, 16)
			}
		}
		br := LecteurSur(append(w.buf, make([]byte, 64)...))
		consumeObjectDamageSections(br)
		if br.BitPos() != c.bits {
			t.Errorf("i7 %s : %d bits consommes, %d attendus", c.nom, br.BitPos(), c.bits)
		}
	}
}

// TestConsumeObjectLowFrequencyLargeurs fige le SQUELETTE d i15 (FUN_1407ef088).
//
// GRAMMAIRE RELUE INSTRUCTION PAR INSTRUCTION (image base 140000000) :
//
//	1407ef0ba: R(2) tete -> [0x3c4]
//	si tete < 2 : 1424cd058 -> 140e9fadc = R(7) -> [0x3c5] ; 1407ef106: R(8) -> [0x3c6]
//	1407ef12e: FUN_1407ef804 = R(4) ; 1407ef13b: FUN_1407ef724 = R(6)
//	1407ef171: R(6) compte d images-cles -> [0x3cc]
//	boucle : R(1) ; si 1 -> R(1) ; si 0 -> 2 x R(12) (1407ef2db / 1407ef2f8 : [RSP+0x20],0xc)
//	7 x R(1) ; 1409684dc = R(1)[si 0 : R(4)] ; 1407ef6d4 = R(1)[si 1 : R(1)+b1ac0+R(1)] ;
//	1407ef520 = R(3)[si bits 0 et 1 : R(2)+R(5) [si bit 2 : 3 x R(8)]] ;
//	1407ef4c8 = R(1)[si 1 : R(3)+R(32)+R(1)[si 1 : R(32)]+R(14) (142325e41 : [RSP+0x20],0xe)]
//
// Le cas MINIMAL est ecrit en clair : tete = 2 (pas de R(7)+R(8)), compte = 0, tous les
// drapeaux de queue a la valeur qui coupe leur corps.
func TestConsumeObjectLowFrequencyLargeurs(t *testing.T) {
	// 2 + 4 + 6 + 6 = 18 de tete ; 7 de drapeaux ; 1 (1409684dc, bit a 1 = rien de plus) ;
	// 1 (1407ef6d4, bit a 0) ; 3 (1407ef520, bits 0 et 1 pas tous deux mis) ;
	// 1 (1407ef4c8, bit a 0) = 31 bits.
	w := &bitw{}
	w.put(2, 2) // tete >= 2 : ni R(7) ni R(8)
	w.put(0, 4) // FUN_1407ef804
	w.put(0, 6) // FUN_1407ef724
	w.put(0, 6) // compte d images-cles = 0
	w.put(0, 7) // les 7 drapeaux
	w.put(1, 1) // 1409684dc : bit a 1 -> pas de R(4)
	w.put(0, 1) // 1407ef6d4 : bit a 0 -> rien
	w.put(0, 3) // 1407ef520 : mot de drapeaux nul
	w.put(0, 1) // 1407ef4c8 : bit a 0 -> rien
	br := LecteurSur(append(w.buf, make([]byte, 64)...))
	consumeObjectLowFrequency(br)
	if br.BitPos() != 31 {
		t.Errorf("i15 cas minimal : %d bits consommes, 31 attendus", br.BitPos())
	}
	// Le cas a UNE image-cle quantifiee : 31 + 1 (drapeau) + 24 (2 x R(12)) = 56.
	w2 := &bitw{}
	w2.put(2, 2)
	w2.put(0, 4)
	w2.put(0, 6)
	w2.put(1, 6) // une image-cle
	w2.put(0, 1) // drapeau a 0 -> 2 x R(12)
	w2.put(0, 24)
	w2.put(0, 7)
	w2.put(1, 1)
	w2.put(0, 1)
	w2.put(0, 3)
	w2.put(0, 1)
	br2 := LecteurSur(append(w2.buf, make([]byte, 64)...))
	consumeObjectLowFrequency(br2)
	if br2.BitPos() != 56 {
		t.Errorf("i15 une image-cle quantifiee : %d bits consommes, 56 attendus", br2.BitPos())
	}
}

// TestConsumeObjectLowFrequencyTeteBasse fige la branche `tete < 2` d i15 : elle AJOUTE
// R(7) + R(8), donc 15 bits, au cas minimal.
func TestConsumeObjectLowFrequencyTeteBasse(t *testing.T) {
	w := &bitw{}
	w.put(0, 2) // tete < 2 : R(7) + R(8) suivent
	w.put(0, 15)
	w.put(0, 4)
	w.put(0, 6)
	w.put(0, 6)
	w.put(0, 7)
	w.put(1, 1)
	w.put(0, 1)
	w.put(0, 3)
	w.put(0, 1)
	br := LecteurSur(append(w.buf, make([]byte, 64)...))
	consumeObjectLowFrequency(br)
	if br.BitPos() != 46 {
		t.Errorf("i15 tete basse : %d bits consommes, 46 attendus (31 + 7 + 8)", br.BitPos())
	}
}

// TestConsumeObjectFrameConfigurationLargeurs fige i17 (FUN_1407f0534 -> FUN_1407f0550).
//
// GRAMMAIRE RELUE : FUN_1404d343c (initialisation, ZERO bit) ; FUN_14080d69c = R(1)[si 1 :
// R(32)] ; si ce bit est mis : FUN_1407f061c = FUN_1424cd07c (R(6), qui rend la valeur PLUS UN)
// puis ce nombre de R(1) ; puis TROIS iterations de { FUN_1406d1024 = R(1)[si 0 : R(6)] ;
// R(1)[si 1 : R(12)] ; R(1)[si 1 : R(12)] }.
//
// Le nombre d iterations est 3, et c est la BORNE DU TABLEAU qui le dit : la boucle de
// FUN_1407f0550 part de `param_1 + 0x10` par pas de 12 octets et s arrete quand `puVar4 + 2`
// atteint `param_1 + 0x30` — 0x10, 0x1c, 0x28, fin.
func TestConsumeObjectFrameConfigurationLargeurs(t *testing.T) {
	// Cas minimal : porte a 0 (aucun R(32), aucune boucle de drapeaux) puis 3 x (1 + 1 + 1),
	// avec le premier bit de chaque triplet a 1 (FUN_1406d1024 est a porte INVERSEE).
	w := &bitw{}
	w.put(0, 1) // FUN_14080d69c : porte fermee
	for i := 0; i < 3; i++ {
		w.put(1, 1) // FUN_1406d1024 : bit a 1 -> pas de R(6)
		w.put(0, 1) // pas de R(12)
		w.put(0, 1) // pas de R(12)
	}
	br := LecteurSur(append(w.buf, make([]byte, 64)...))
	consume1407f0550(br)
	if br.BitPos() != 10 {
		t.Errorf("i17 cas minimal : %d bits consommes, 10 attendus", br.BitPos())
	}
	// Cas porte OUVERTE, compte lu a 0 (donc UNE iteration de drapeau, la valeur rendue etant
	// v+1) : 1 + 32 + 6 + 1 + 3 x 3 = 49.
	w2 := &bitw{}
	w2.put(1, 1)
	w2.put(0, 32)
	w2.put(0, 6) // compte lu = 0 -> 1 drapeau
	w2.put(0, 1)
	for i := 0; i < 3; i++ {
		w2.put(1, 1)
		w2.put(0, 1)
		w2.put(0, 1)
	}
	br2 := LecteurSur(append(w2.buf, make([]byte, 64)...))
	consume1407f0550(br2)
	if br2.BitPos() != 49 {
		t.Errorf("i17 porte ouverte : %d bits consommes, 49 attendus", br2.BitPos())
	}
}
