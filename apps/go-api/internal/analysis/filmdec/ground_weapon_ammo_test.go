package filmdec

// ground_weapon_ammo_test.go — LE LECTEUR DES MUNITIONS, sur octets FABRIQUES.
//
// CE QUE CE TEST PROUVE, ET CE QU'IL NE PROUVE PAS. Il prouve le CABLAGE : que la boucle de
// composants de production, lancee au premier bit du composant i0, atteint `weapon-ammo-component`
// et y lit R(8) puis R(11) dans cet ordre ; et que les trois refus tiennent. Il ne prouve PAS la
// SEMANTIQUE des deux champs — celle-la se mesure sur le parc et son verdict est au rapport
// `.ai/V7.5/RAPPORT_MUNITIONS_EXACTES_2026-09-11.md`, resume en tete de ground_weapon_ammo.go.
//
// LA BOBINE VERSIONNEE NE POUVAIT PAS SERVIR ICI, et c'est mesure : ses 28 records de creation
// d'arme au sol portent TOUS le composant i9 (masques `[0 1 2 9 12 14 15 17 20]` et voisins), que
// la reserve de lecture ecarte. Elle rendrait donc 28 refus et zero lecture.

import "testing"

// munAmmoArch fabrique l'archetype ARME AU SOL avec ses 21 composants dans l'ordre du registre.
// Les seuls qui comptent ici sont ceux que le masque annonce ; les autres doivent simplement
// porter le bon NOM, puisque le lecteur verifie les index par leur etiquette.
func munAmmoArch() Archetype {
	comps := []string{
		"object-position-component", "object-translational-velocity-component",
		"object-forward-and-up-component", "object-angular-velocity-component",
		compObjectBodyVitality, "object-shield-vitality-component",
		"object-region-state-component", "object-damage-sections-component",
		"object-constraint-component", compObjectMultiplayerProperties,
		"object-parent-state-component", "object-dead-state-component",
		"object-scale-component", "object-maximum-vitalities-component",
		"object-dissolver-component", "object-low-frequency-component",
		"object-physics-flags-component", "object-frame-configuration-component",
		"item-at-rest-component", "item-ignore-player-component", compWeaponAmmo,
	}
	return Archetype{Index: GroundWeaponTypeIndex, Components: comps}
}

// munAmmoPayload ecrit un corps de record : i0 (chemin objet du monde, porte a zero),
// i18 `item-at-rest` R(1), puis i20 R(8)+R(11)+R(12).
func munAmmoPayload(mag, res, troisieme uint64) []byte {
	w := &bitWriter{}
	w.bit(0)                                    // i0 precHigh = 0
	w.bit(0)                                    // i0 index-sel = 0 -> lit l'index de region
	w.bits(0, int(WorldObjectPrecision.IndexW)) // index de region
	for a := 0; a < 3; a++ {                    // les trois axes
		w.bits(0, int(WorldObjectPrecision.AxisW[a]))
	}
	w.bits(0, 2)    // i0 queue R(2)
	w.bit(1)        // i18 item-at-rest R(1)
	w.bits(mag, 8)  // i20 champ A
	w.bits(res, 11) // i20 champ B
	w.bits(troisieme, 12)
	w.bits(0, 64) // marge, pour que la lecture ne deborde jamais du tampon
	return w.buf
}

func TestReadGroundWeaponAmmoLitLesDeuxChamps(t *testing.T) {
	release := LockProcessDecode()
	defer release()
	pay := munAmmoPayload(27, 114, 4095)
	got, ok := readGroundWeaponAmmo(pay, 0, []int{0, 18, 20}, munAmmoArch())
	if !ok {
		t.Fatal("lecture refusee alors que le masque porte i20 et pas i9")
	}
	if got.Mag != 27 || got.Res != 114 {
		t.Fatalf("munitions lues %+v, attendu {Mag:27 Res:114}", got)
	}
}

func TestReadGroundWeaponAmmoRefuse(t *testing.T) {
	release := LockProcessDecode()
	defer release()
	arch := munAmmoArch()
	pay := munAmmoPayload(27, 114, 0)
	cas := []struct {
		nom  string
		mask []int
		arch Archetype
	}{
		// i9 AU MASQUE : la marche n'est pas prouvee bit-exacte (cf. ground_weapon_ammo.go).
		{"masque avec i9", []int{0, 9, 18, 20}, arch},
		// SANS i20 : il n'y a rien a lire, et lire quand meme rendrait la valeur du voisin.
		{"masque sans i20", []int{0, 18}, arch},
		// ARCHETYPE ETRANGER : les index 9 et 20 n'y portent pas les composants attendus, donc
		// le lecteur ne peut pas savoir ce qu'il lit.
		{"archetype sans le composant", []int{0, 18, 20},
			Archetype{Index: GroundWeaponTypeIndex, Components: []string{"object-position-component"}}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if _, ok := readGroundWeaponAmmo(pay, 0, c.mask, c.arch); ok {
				t.Fatal("lecture acceptee alors qu'elle devait etre refusee")
			}
		})
	}
}
