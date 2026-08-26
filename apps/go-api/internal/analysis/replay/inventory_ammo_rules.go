package replay

// inventory_ammo_rules.go — LE BLOC MUNITIONS (i30..i42, règles R3+R4) d'un record de biped
// aux images-clés. Extrait d'inventory_decode.go (seuil de taille du dépôt, CLAUDE.md n°5),
// exactement comme inventory_grenades_rules.go l'a été avant lui : R1/R2/R5 (capacité,
// grenades, sélection) restent dans inventory_decode.go et inventory_grenade_selection.go.
//
// R4, ici, est le repère dont dépend R2b (invGrenadesNearAmmo, inventory_grenades_rules.go) :
// le bloc se termine EXACTEMENT sur le bit de porte d'i43, juste avant la première famille
// d'arme. Aucune VALEUR n'entre dans ce critère — c'est un critère de LARGEUR, pas de contenu.

// invAmmoSearchSpan est la profondeur de recherche du début du bloc de munitions, en bits,
// avant la première famille d'arme. Le bloc mesure ~200 bits ; 300 laisse de la marge sans
// ouvrir la porte à des débuts absurdes.
const invAmmoSearchSpan = 300

// SlotAmmo est l'état de munitions d'UN emplacement d'arme.
//
// LES TROIS CAS NE SE CONFONDENT PAS, et c'est tout l'intérêt des pointeurs :
//   - Mag non nil    : arme à chargeur (chargeur + réserve) ;
//   - Gauge non nil  : arme à jauge de charge, fraction dans [0,1] sur 4096 niveaux ;
//   - les deux nil   : le film n'écrit RIEN pour cet emplacement. Pour une arme à charge, cela
//     veut dire PLEIN — le flux est différentiel et le plein est la valeur par défaut, donc il
//     n'est jamais transmis. Ce n'est PAS « zéro » : publier 0 affirmerait un chargeur vide.
type SlotAmmo struct {
	Mag   *uint32
	Res   *uint32
	Gauge *float64
	// Overheat et Flags sont lus mais non interprétés : ils bornent le parse (leur largeur
	// entre dans le critère d'atterrissage) sans qu'on prétende savoir ce qu'ils disent.
	Overheat uint32
	Flags    uint32
}

// readAmmo résout le bloc de munitions et pose son résultat sur l'inventaire. Rend le PREMIER
// BIT du bloc retenu, qui est aussi le repère de R2b (cf. inventory_grenades_rules.go) : le
// début du bloc est établi par un critère de largeur, sans aucune information de grenade.
func readAmmo(pay []byte, inv *KeyframeInventory, from, firstFamilyBit int) (start int, ok bool) {
	end := firstFamilyBit - 1
	lo := end - invAmmoSearchSpan
	if lo < from {
		lo = from
	}
	sols := invSolveAmmoBlock(pay, end, lo)
	inv.AmmoCandidates = len(sols)
	if len(sols) == 0 {
		return 0, false
	}
	// DÉPARTAGE : on retient le bloc le PLUS LONG (le début le plus petit). Une solution plus
	// courte s'obtient en réinterprétant des bits réels comme appartenant au composant
	// précédent ; le début réel est donc le premier qui parse. Le nombre de candidats reste
	// publié pour que ce choix soit visible.
	st, sel, _, _ := invParseAmmoBlock(pay, sols[0], end+1)
	inv.Ammo, inv.AmmoRead, inv.DrawnSlot = st, true, sel
	return sols[0], true
}

// invParseAmmoBlock parse le bloc i30..i42 depuis le bit s : quatre fois
// [union chargeur/jauge + réserve R(11) + drapeaux R(2) + surchauffe R(7)], puis i42.
// Rend l'état des quatre emplacements, le sélecteur, et le bit d'arrivée. ok=false si un champ
// déborde de `limit`.
func invParseAmmoBlock(
	pay []byte, s, limit int,
) (st [invGrenadeSlots]SlotAmmo, sel int, end int, ok bool) {
	p := s
	sel = -1
	rd := func(n int) uint32 {
		v := invBits(pay, p, n)
		p += n
		return v
	}
	fits := func(n int) bool { return p+n <= limit }

	for k := 0; k < invGrenadeSlots; k++ {
		if !fits(1) {
			return st, sel, p, false
		}
		if rd(1) == 0 { // branche chargeur
			if !fits(8) {
				return st, sel, p, false
			}
			m := rd(8)
			st[k].Mag = &m
		}
		if !fits(1) {
			return st, sel, p, false
		}
		if rd(1) == 0 { // branche jauge : dequant(R(12), 0, 1)
			if !fits(12) {
				return st, sel, p, false
			}
			g := float64(rd(12)) / 4095.0
			st[k].Gauge = &g
		}
		if !fits(11 + 9) {
			return st, sel, p, false
		}
		r := rd(11)
		st[k].Res = &r
		st[k].Flags = rd(2)
		st[k].Overheat = rd(7)
	}
	// i42 : R(3) puis deux fois [porte active-bas R(1) + valeur optionnelle R(2)].
	if !fits(3) {
		return st, sel, p, false
	}
	rd(3)
	for g := 0; g < 2; g++ {
		if !fits(1) {
			return st, sel, p, false
		}
		if rd(1) == 0 {
			if !fits(2) {
				return st, sel, p, false
			}
			sel = int(rd(2))
		}
	}
	return st, sel, p, true
}

// invSolveAmmoBlock cherche les débuts dont le parse atterrit EXACTEMENT sur `end`.
//
// DEUX CONTRAINTES, TOUTES DEUX STRUCTURELLES — aucune ne regarde une valeur des emplacements
// 0 et 1, ceux que la confrontation au terrain mesure :
//
//	(a) la largeur de l'union vaut 2 (vide), 10 (chargeur) ou 14 (jauge) ; la largeur 22, qui
//	    porterait les DEUX branches, n'existe pas dans la carte mémoire ;
//	(b) une union vide signifie qu'aucun état d'arme ne vit dans cet emplacement : ni réserve,
//	    ni drapeaux, ni surchauffe. Un Spartan ne portant que deux armes, les emplacements 2 et
//	    3 imposent à eux seuls 44 bits nuls.
func invSolveAmmoBlock(pay []byte, end, lo int) []int {
	var sols []int
	for s := lo; s < end; s++ {
		st, _, e, ok := invParseAmmoBlock(pay, s, end+1)
		if !ok || e != end {
			continue
		}
		valid := true
		for k := 0; k < invGrenadeSlots && valid; k++ {
			if st[k].Mag != nil && st[k].Gauge != nil {
				valid = false // (a)
				break
			}
			if st[k].Mag == nil && st[k].Gauge == nil { // (b)
				if st[k].Res == nil || *st[k].Res != 0 || st[k].Flags != 0 || st[k].Overheat != 0 {
					valid = false
				}
			}
		}
		if valid {
			sols = append(sols, s)
		}
	}
	return sols
}
