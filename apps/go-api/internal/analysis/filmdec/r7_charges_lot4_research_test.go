package filmdec

// r7_charges_lot4_research_test.go — suite : les types courants « petits » (armes, dialogues,
// vehicules, corps a corps), derives de l'executable pendant ce lot.
//
// PRIMITIVES NOUVELLES ETABLIES ICI (increments `+0x2c` verifies) :
//
//	0x1406ce648  boucle de 32 tours a 1 bit  -> R(32)
//	0x1406d0f20  R(3) puis valeur-1 · 0x140c1e31c R(3) puis valeur-1
//	0x140f2ea18  R(5) · 0x142ef1734 R(5) · 0x142b169f8 R(4) · 0x142b19a90 R(32)
//	0x141d0f344  R(32) · 0x1424d9b10 R(1) · 0x1424d9a30 R(3)
//	0x140f2e7a4  [R(1) g ; si g==0 : R(12)]   POLARITE INVERSEE
//	0x1424e1464  R(1) puis valeur+1 (compteur dans {1,2})
//	0x140cec0a0  [R(1) g ; si g==1 : R(8)]
//	0x14076dc04  R(n), n = 4e argument (R9D) · 0x14076d6dc R(n), n = 5e argument
//	0x14076d528  [R(1) g ; si g==0 : R(w7) puis R(w6)]   POLARITE INVERSEE
//	0x1411b259c  R(96) brut (vecteur non quantifie)
//
// DEUX THUNKS D'ADAPTATION reperes (le « lecteur » de la vtable n'est pas la fonction) :
// `0x142ef8eb8` -> `0x142ef4344` (type 52) et `0x142ef8a58` -> `0x142ef4c98` (type 102).
//
// DEUX GARDES, toutes deux argumentees :
//   - type 40, garde de build `FUN_141102ed0(0x28)` : la table `DAT_14474cd90 + 0x28*8`
//     contient 2 -> largeur 3 bits. CONTRE-EPREUVE : l'ECRIVAIN `0x1407ebc90` ecrit 3 bits
//     SANS garde, donc tout film produit par ce build est en 3 bits.
//   - type 39, garde runtime `*(int*)(DAT_144c1cfa8+4) == 2` : l'ECRIVAIN porte la garde
//     IDENTIQUE, donc le film est auto-coherent, et garde fausse ferait echouer le parse cote
//     jeu — tout film qui se lit a la garde vraie. On lit donc la branche complete.

// r7Porte2Inv consomme `R(1) g ; si g==0 : R(2)` (primitive 0x1406d00ec, POLARITE INVERSEE).
func r7Porte2Inv(br *BitReader) {
	if !br.ReadBit() {
		br.Skip(2)
	}
}

// r7VariantArme consomme le motif commun aux types 11, 37, 40, 45, 46, 47 :
// `[R(1) g ; si g : R(32)]` puis `R(32) variant-name`.
func r7VariantArme(br *BitReader) {
	r7Porte32(br)
	br.Skip(32)
}

// r7SkipChargeLot4 est la suite de r7SkipChargeLot3. Rend false pour tout type non ferme.
func r7SkipChargeLot4(br *BitReader, typ int, ctx r7Ctx) bool {
	switch typ {
	// --- 38 weapon_reload : 4 x R(1) puis [R(1);si 0:R(5)] ---
	case 38:
		br.Skip(4)
		r7Porte5(br)
		return true

	// --- 9 biped_pickup : R(3) puis [R(1);si 1:R(32)] ---
	case 9:
		br.Skip(3)
		r7Porte32(br)
		return true

	// --- 76 Dialogue2D ---
	case 76:
		r7Porte32(br)
		r7Porte32(br)
		r7Porte5(br)
		br.Skip(32) // masque (FUN_1406ce648 : 32 tours de 1 bit)
		return true

	// --- 75 AIDialog ---
	case 75:
		br.Skip(5)
		r7Porte32(br)
		if !br.ReadBit() { // FUN_140f2e7a4, POLARITE INVERSEE
			br.Skip(12)
		}
		r7Porte5(br)
		n := int(br.ReadBits(1)) + 1 // FUN_1424e1464 : compteur dans {1,2}
		br.Skip(n)
		return true

	// --- 39 biped_throw_initiate (garde runtime : voir en-tete de fichier) ---
	case 39:
		if !br.ReadBit() { // k == 0
			br.Skip(3)
		} else { // k == 1
			r7Porte32(br)
			br.Skip(4) // FUN_142ed0674 : R(4) puis valeur-1
		}
		r7Porte5(br)
		return true

	// --- 22 unit_exit_vehicle : fixe 10 bits ---
	case 22:
		br.Skip(6 + 1 + 3)
		return true

	// --- 11 weapon_empty_click ---
	case 11:
		r7Porte2Inv(br)
		r7VariantArme(br)
		br.Skip(1)
		r7Porte5(br)
		return true

	// --- 47 weapon_throw ---
	case 47:
		br.Skip(1)
		r7Porte2Inv(br)
		r7VariantArme(br)
		r7Porte5(br)
		return true

	// --- 58 projectile_supercombine_request ---
	case 58:
		r7Porte32(br)
		return true

	// --- 8 biped_board_vehicle et 53 unit_enter_vehicle : lecteur commun, fixe 6 bits ---
	case 8, 53:
		br.Skip(6)
		return true

	// --- 41 vehicle_trick ---
	case 41:
		br.Skip(3)
		r7Porte5(br)
		return true

	// --- 44 weapon_pickup ---
	case 44:
		br.Skip(3) // FUN_1406d0f20 : R(3) puis valeur-1
		r7Porte2Inv(br)
		r7Porte2Inv(br)
		br.Skip(3)
		return true

	// --- 45 weapon_put_away ---
	case 45:
		br.Skip(1)
		r7Porte2Inv(br)
		r7VariantArme(br)
		return true

	// --- 46 weapon_drop ---
	case 46:
		br.Skip(1)
		r7Porte2Inv(br)
		r7VariantArme(br)
		br.Skip(1)
		return true

	// --- 40 biped_melee_initiate (largeur de tete 3 bits : voir en-tete) ---
	case 40:
		if br.ReadBits(3) == 1 {
			br.Skip(2)
		}
		br.Skip(2)
		r7VariantArme(br)
		r7Porte5(br)
		return true

	// --- 52 biped_melee_damage (vrai lecteur 0x142ef4344) ---
	case 52:
		if !r7VecteurQuantifie(br, ctx, 12) {
			return false
		}
		br.Skip(19)
		if br.ReadBit() {
			br.Skip(19)
		}
		r7Porte32(br)
		if br.ReadBit() {
			r7Porte32(br)
		}
		br.Skip(9 + 6 + 8 + 32 + 1)
		return true

	// --- 63 biped_laser_designation : fixe 1 bit ---
	case 63:
		br.Skip(1)
		return true

	// --- 37 weapon_overheat ---
	case 37:
		r7VariantArme(br)
		r7Porte2Inv(br)
		return true

	// --- 32 unit_teleported : deux vecteurs quantifies classe 0x10 ---
	case 32:
		for i := 0; i < 2; i++ {
			if !r7VecteurQuantifie(br, ctx, 16) {
				return false
			}
		}
		r7Porte32(br)
		r7Porte32(br)
		return true

	// --- 86 EngineClientEvent : fixe 32 bits ---
	case 86:
		br.Skip(32)
		return true

	// --- 102 NetworkedActionRequest (vrai lecteur 0x142ef4c98) : fixe 5 bits ---
	case 102:
		br.Skip(5)
		return true

	// --- 110 ObjectDeterministicDamageAcceleration (-> FUN_143203158) ---
	case 110:
		br.Skip(4)
		if !r7VecteurQuantifie(br, ctx, 12) {
			return false
		}
		if !br.ReadBit() { // FUN_14076d528, POLARITE INVERSEE
			br.Skip(19 + 12)
		}
		if br.ReadBit() {
			br.Skip(10)
		}
		if br.ReadBit() {
			br.Skip(8)
		}
		return true
	}
	return r7SkipChargeLot5(br, typ, ctx)
}
