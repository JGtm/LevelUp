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
// LA RESERVE DE LECTURE A ETE LEVEE LE 2026-09-11 (lot 6.10 bis). Pour atteindre i20 il faut
// traverser les composants qui le precedent ; un seul ne l'etait pas bit-exact,
// `object-multiplayer-properties-component` (i9, `consumeObjectMultiplayerProperties`,
// components_batch7.go + tlv_mode2.go). Sa grammaire a ete RETABLIE SUR LE DESASSEMBLAGE de
// `FUN_1407d4c94` et de sa chaine de flux, pas par la mesure — le lot 6.10 avait refute sept
// lectures candidates et conclu que la mesure seule n'y suffirait pas.
//
// CE QUE LA LEVEE A COUTE ET RAPPORTE, EN CHIFFRES (64 films, meme oracle que le lot 6.10) :
//
//	                                      AVANT (refus d'i9)   APRES
//	creations ti=42 acceptees             27 155               27 155
//	dont le masque porte i20              16 218 (59,7 %)      16 218 (59,7 %)
//	dont les munitions sont LUES           4 283 (15,8 %)      16 218 (59,7 %)
//	decalage ZERO, records AVEC i9        eparpille sur 121    94,3 % (2 738 / 2 903)
//	decalage ZERO, records SANS i9        89,9 % (80 / 89)     91,8 % (391 / 426)
//	longueur d'i9, reste modulo 8         sans structure       6 dans 100,0 % des cas
//
// LE CONTROLE QUI TRANCHE N'EST PAS LE SEUIL, C'EST LA POPULATION TEMOIN. Le lot 6.10 avait
// ecrit « > 95 % » comme critere de levee ; la mesure montre que les records SANS i9 — dont la
// marche etait DEJA prouvee bit-exacte — plafonnent eux-memes a 91,8 % sur ce corpus, parce que
// l'oracle d'inventaire vieillit. Le residu n'est donc pas imputable a i9 : les records qui le
// portent font MIEUX que la population de reference.
//
// L'infobulle garde son repli date pour les objets dont le masque ne porte pas i20 — 40,3 % des
// creations (`apps/web/.../model/groundWeaponAmmo.ts`).

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

// compObjectMultiplayerProperties est l'etiquette du composant i9, que la marche TRAVERSE pour
// atteindre i20. Le lecteur verifie que l'index 9 le porte bien dans le registre du film : un
// index de composant est un numero de BUILD, et lire i20 derriere un i9 qui n'en serait pas un
// reviendrait a faire confiance a un decalage inconnu.
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
// archetype dont les index ne portent pas les composants attendus, ou i20 non atteint. Le refus
// des masques portant i9 a ete RETIRE le 2026-09-11 (cf. l'en-tete) : la marche les traverse.
func readGroundWeaponAmmo(pay []byte, compStart int, mask []int, arch Archetype) (GroundWeaponAmmo, bool) {
	if arch.component(groundWeaponAmmoIndex) != compWeaponAmmo ||
		arch.component(groundWeaponMPPIndex) != compObjectMultiplayerProperties {
		return GroundWeaponAmmo{}, false
	}
	var bits uint64
	hasAmmo := false
	for _, i := range mask {
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
