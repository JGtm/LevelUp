package filmdec

// ground_weapon_ammo.go — LES MUNITIONS EXACTES d'une ARME AU SOL, lues sur son record de
// CREATION (lot 6.10, 2026-09-11, rapport `.ai/V7.5/RAPPORT_MUNITIONS_EXACTES_2026-09-11.md`).
//
// OU ELLES SONT, ET POURQUOI LE LOT 6.6 NE LES AVAIT PAS TROUVEES. Le lot 6.6 avait fouille les
// trois feuilles candidates de l'ETAT PAR DEFAUT de `ti=42` (`default_state_ti42.go`, points 3,
// 4 et 5) et les avait toutes REFUTEES. Les munitions ne sont pas dans le default-state : elles
// sont dans un COMPOSANT du meme record, `weapon-ammo-component` (i20, deser `FUN_140fc3028`,
// grammaire `R(8) + R(11) + R(12)`), qui vit APRES le masque de presence. Le balayage des
// creations s'arretait au composant i0 (la position) et n'allait jamais jusque-la.
//
// LE RECORD DE CREATION EST DATE A L'INSTANT DU LACHER : la valeur lue ici n'est pas une lecture
// d'inventaire vieille de neuf secondes, c'est l'etat de l'arme au moment ou elle touche le sol.
//
// CE QUE LA MESURE DIT DES DEUX PREMIERS CHAMPS (64 films du parc, 6 756 objets laches apparies
// a leur record de creation, dont 939 sur la marche prouvee bit-exacte — cf. LA RESERVE DE
// LECTURE plus bas). L'oracle est le chargeur et la reserve du LACHEUR a sa derniere lecture
// d'inventaire avant le lacher ; cette reference VIEILLIT (9 s en mediane), et c'est ce
// vieillissement qui signe la semantique :
//
//	reference fraiche (<= 1 s), n=62 : A == chargeur 74,2 % · A <= chargeur 100,0 %
//	                                   B == reserve  96,8 %
//	1 a 2 s,  n= 55 : A 52,7 % · A<= 94,5 % · B 94,5 %
//	2 a 5 s,  n=174 : A 42,5 % · A<= 96,0 % · B 90,8 %
//	5 a 10 s, n=247 : A 36,4 % · A<= 92,7 % · B 82,2 %
//	10 a 20 s,n=377 : A 41,4 % · A<= 92,0 % · B 66,0 %
//
// Le gradient EST la preuve : A decroit vite quand la reference vieillit (le porteur a TIRE
// entre les deux), B decroit lentement (la reserve ne bouge qu'au rechargement). Temoins :
// la meme comparaison contre la reference d'un AUTRE objet donne 3,2 % (A) et 9,7 % (B) ; un
// champ VOISIN de meme largeur, pris 19 bits plus loin dans le meme composant, donne 0,0 %.
//
// LE TROISIEME CHAMP (C, R(12)) N'EST PAS PUBLIE, et c'est une decision : sa semantique n'est
// pas etablie. Le nommer reviendrait a ecrire une conclusion avant la mesure (meme regle que la
// table ECS, qui laisse les trois champs POSITIONNELS).
//
// LA RESERVE DE LECTURE — ET ELLE EST DATEE. Pour atteindre i20 il faut traverser les composants
// qui le precedent. Tous sont portes, sauf UN : `object-multiplayer-properties-component` (i9,
// `consumeObjectMultiplayerProperties`, components_batch7.go), dont le flux TLV ne consomme pas
// le bon nombre de bits sur cet archetype. Mesure du 2026-09-11 : sur les records dont le masque
// ne porte PAS i9, la position d'i20 trouvee par l'oracle est le decalage ZERO dans 80 cas sur
// 89 (89,9 %) ; sur les records qui le portent, elle s'eparpille sur 121 decalages distincts et
// la longueur VRAIE d'i9 vaut 300 a 470 bits (34 valeurs distinctes, aucune structure modulo 8,
// aucune correlation avec ses six premiers bits). La grammaire d'i9 ne se retablit donc pas par
// la mesure : il faut desassembler `FUN_1407d4c94`.
//
// D'OU LA REGLE CI-DESSOUS : on ne lit les munitions que lorsque la marche est PROUVEE
// bit-exacte, c'est-a-dire quand le masque du record ne porte pas i9. Ce n'est pas une
// precaution de confort, c'est la difference entre un chiffre et un bruit — et c'est ce qui
// borne la couverture. CRITERE DE LEVEE : le portage d'i9 valide par un oracle (le meme que
// celui de ce lot : la position d'i20 doit retomber au decalage zero sur > 95 % des records qui
// portent i9). Tant qu'il n'est pas leve, l'infobulle garde son repli date pour les autres
// objets (`apps/web/.../model/groundWeaponAmmo.ts`).

// groundWeaponAmmoIndex est l'index de composant de `weapon-ammo-component` dans l'archetype
// ARME AU SOL, et groundWeaponMPPIndex celui d'`object-multiplayer-properties-component`. Les
// deux sont des index d'archetype du registre du film, pas des constantes du format : le
// balayage les VERIFIE par le nom du composant avant de s'en servir.
const (
	groundWeaponAmmoIndex = 20
	groundWeaponMPPIndex  = 9
)

// compWeaponAmmo est l'etiquette de registre du composant des munitions. UNE SEULE definition
// pour le dispatch (traverse.go) et pour le lecteur ci-dessous : deux litteraux auraient
// diverge au premier renommage.
const compWeaponAmmo = "weapon-ammo-component"

// compObjectMultiplayerProperties est l'etiquette du composant dont le portage TLV n'est pas
// bit-exact sur cet archetype (cf. l'en-tete). Meme raison d'etre.
const compObjectMultiplayerProperties = "object-multiplayer-properties-component"

// GroundWeaponAmmo porte les deux champs PROUVES du composant `weapon-ammo-component`.
type GroundWeaponAmmo struct {
	// Mag est le CHARGEUR de l'arme au moment ou elle a touche le sol (champ A, R(8)).
	Mag uint32
	// Res est la RESERVE a ce meme instant (champ B, R(11)).
	Res uint32
}

// readGroundWeaponAmmo rejoue la boucle de composants de PRODUCTION sur le record de creation,
// a partir du premier composant (i0, dont `compStart` est le premier bit), et rend le contenu
// d'i20. PUR : il lit `pay` avec son PROPRE curseur, sans toucher celui du balayage — aucun bit
// du chemin existant n'est lu autrement.
//
// ok=false quand la lecture n'est pas prouvee bit-exacte : masque sans i20 (rien a lire),
// masque avec i9 (marche non fiable, cf. l'en-tete), archetype dont les index ne portent pas les
// composants attendus, ou i20 non atteint.
func readGroundWeaponAmmo(pay []byte, compStart int, mask []int, arch Archetype) (GroundWeaponAmmo, bool) {
	if arch.component(groundWeaponAmmoIndex) != compWeaponAmmo ||
		arch.component(groundWeaponMPPIndex) != compObjectMultiplayerProperties {
		return GroundWeaponAmmo{}, false
	}
	var bits uint64
	hasAmmo := false
	for _, i := range mask {
		if i == groundWeaponMPPIndex {
			return GroundWeaponAmmo{}, false
		}
		if i == groundWeaponAmmoIndex {
			hasAmmo = true
		}
		bits |= uint64(1) << uint(i&63)
	}
	if !hasAmmo {
		return GroundWeaponAmmo{}, false
	}
	br := NewBitReader(pay)
	br.SetBitPos(compStart)
	tr := EntityTrace{TypeIndex: GroundWeaponTypeIndex, Mask: bits, DesyncAt: -1}
	traverseComponentLoop(br, arch, &tr)
	for _, c := range tr.Comps {
		if c.Index != groundWeaponAmmoIndex {
			continue
		}
		r := NewBitReader(pay)
		r.SetBitPos(c.StartBit)
		return GroundWeaponAmmo{Mag: uint32(r.ReadBits(8)), Res: uint32(r.ReadBits(11))}, true
	}
	return GroundWeaponAmmo{}, false
}
