package filmdec

// r7_charges_lot5_research_test.go — LES TYPES CIBLES : equipement, repulseur, propulseur.
//
// TROISIEME CORRECTION AUX PRIMITIVES (celle-ci change des largeurs) : la reference var-int
// `0x1406d3140` N'A PAS de bit de porte en tete. Le desassemblage (0x1406d3140..0x1406d31a0)
// montre que le prologue teste `DAT_144706104`, lit (base, cardinal) dans
// `DAT_1451f98d0 + domaine*8`, et n'accede au flux qu'ensuite :
//
//	si domaine == 1 : R(1) sonde
//	w = ceil(log2(cardinal))   (FUN_1406d310c, 0 bit)
//	si w >= 1 : R(w) index
//	R(2) suffixe               (toujours)
//
// Le « R(1) porte » appartient a l'EN-TETE (« 3 references gardees »), pas a la primitive.
// Consequence directe : dans une BOUCLE de charge (type 119), une reference du domaine 0
// coute 13+2 = 15 bits, pas 16.
//
// LE REPULSEUR ET LE PROPULSEUR SONT LISIBLES. Leurs charges sont fermees et courtes :
//   - 104 EquipmentKnockbackPlayer : 1 bit si poussee nulle, sinon 30 bits — direction
//     unitaire quantifiee sur 19 bits (meme codec que 42/119) puis magnitude sur 10 bits,
//     echelle LOGARITHMIQUE entre 0,05 (DAT_143cd8648) et 20,0 (DAT_143cd8f60) ;
//   - 42 biped_dodge : 60 ou 65 bits, dont une direction unitaire 19 bits ;
//   - 119 EquipmentKnockbackRequest : direction globale puis n couples (victime, direction).
// Reste a savoir s'ils apparaissent dans le flux : c'est la mesure, pas la grammaire.

// r7RefCharge consomme une reference var-int A L'INTERIEUR d'une charge (sans bit de porte).
func r7RefCharge(br *BitReader, dom int) {
	w := r7DomWidth[dom]
	if dom == 1 && br.ReadBit() { // sonde
		w = 9
	}
	br.Skip(int(w) + 2)
}

// r7Direction19 consomme une direction unitaire quantifiee (FUN_14076dc04 R9D=0x13 puis
// FUN_1406d8288, 0 bit) — le codec commun au repulseur, au propulseur et a la requete.
func r7Direction19(br *BitReader) { br.Skip(19) }

// r7ORI consomme le bloc d'orientation par defaut (FUN_140c5fa84) : `R(1) g ; si g==0 :
// R(19)` puis `R(8)`. La variante `DAT_145121140 == 1` (R(30) au lieu de R(19)) n'est pas
// modelisee : elle rendrait le type concerne opaque, et aucun de ces types n'apparait.
func r7ORI(br *BitReader) {
	if !br.ReadBit() {
		br.Skip(19)
	}
	br.Skip(8)
}

// r7SkipChargeLot5 : les types d'equipement. Rend false pour tout type non ferme.
func r7SkipChargeLot5(br *BitReader, typ int, ctx r7Ctx) bool {
	switch typ {
	// --- 104 EquipmentKnockbackPlayer : LE REPULSEUR (lecteur 0x14116c344) ---
	// FUN_14076d528(flux, .., 0.05f, 20.0f, 10, 19) : R(1) ; si 0 : R(19) + R(10).
	case 104:
		if !br.ReadBit() {
			r7Direction19(br)
			br.Skip(10) // magnitude logarithmique
		}
		return true

	// --- 42 biped_dodge : LE PROPULSEUR (lecteur 0x142f169d0) ---
	case 42:
		br.Skip(32) // FUN_141015740 : R(32)
		r7Direction19(br)
		br.Skip(8)
		r7Porte5(br)
		return true

	// --- 43 initiate_mobility_action (lecteur 0x142ef8f04 -> FUN_1408f02c8) ---
	case 43:
		if !br.ReadBit() { // presence
			return true
		}
		if br.ReadBit() { // g1
			br.Skip(10)
		}
		if !br.ReadBit() { // g2 == 0 : vecteur quantifie + orientation
			if !r7VecteurQuantifie(br, ctx, 16) {
				return false
			}
			r7ORI(br)
		}
		br.Skip(96) // FUN_1406d676c(..., 0x60) : 3 float32 bruts
		if !r7VecteurQuantifie(br, ctx, 16) {
			return false
		}
		br.Skip(3 * 3 * 12) // 3 appels de 3 x R(12)
		br.Skip(24 + 24)    // deux directions quantifiees sur 24 bits
		br.Skip(3 * 12)     // un dernier triplet R(12)
		br.Skip(10 + 10)    // deux scalaires
		br.Skip(1 + 7 + 2 + 1)
		return true

	// --- 119 EquipmentKnockbackRequest : la requete du repulseur (lecteur 0x142eebcec) ---
	case 119:
		r7Direction19(br)
		n := int(br.ReadBits(4)) + 1 // FUN_142ed0abc : R(4) puis valeur+1
		for i := 0; i < n; i++ {
			r7RefCharge(br, 0) // domaine 0, SANS bit de porte
			r7Direction19(br)
		}
		return true

	// --- 31 equipment_teleport_request : R(4) + [R(1);si 0:R(5)] ---
	case 31:
		br.Skip(4)
		r7Porte5(br)
		return true

	// --- 105 EquipmentObjectKnockedBack : [R(1) g ; si g : R(32)] ---
	case 105:
		r7Porte32(br)
		return true

	// --- 98 Equipment : R(8) + R(1), fixe 9 bits ---
	case 98:
		br.Skip(9)
		return true

	// --- 48 weapon_tether_request (version 2 : le R(1) final est lu) ---
	case 48:
		r7Porte2Inv(br)
		r7VariantArme(br)
		br.Skip(1)
		return true

	// --- 28 biped_debug_teleport : un seul vecteur quantifie ---
	case 28:
		return r7VecteurQuantifie(br, ctx, 16)

	// --- 116 teleport_effects (lecteur 0x142ef93e0) ---
	case 116:
		if br.ReadBit() {
			r7ORI(br)
		}
		br.Skip(1)
		if br.ReadBit() { // la porte du R(32) commande aussi les deux vecteurs
			br.Skip(32)
			for i := 0; i < 2; i++ {
				if !r7VecteurQuantifie(br, ctx, 16) {
					return false
				}
			}
		}
		return true

	// --- 93 activate_spartan_ability : ferme SAUF la branche k == 2 ---
	case 93:
		br.Skip(15)
		if br.ReadBit() { // inhibe
			return true
		}
		switch br.ReadBits(2) { // FUN_141102ed0(0x5d) = 2 -> pas de « valeur-1 »
		case 0:
			br.Skip(2 + 24)
			return true
		case 1, 3:
			return true
		default:
			return false // k == 2 : bloc FUN_142f262d4, gouverne par un etat hors flux
		}
	}
	return r7SkipChargeLot6(br, typ, ctx)
}
