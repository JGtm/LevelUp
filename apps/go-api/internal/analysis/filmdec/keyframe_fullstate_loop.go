package filmdec

// keyframe_fullstate_loop.go — LA BOUCLE D'ETAT COMPLET DU JEU, PORTEE.
//
// D'OU ELLE VIENT. Le lot R7-d a trouve, en deroulant la piste du lot R7-c, la chaine que le
// jeu emprunte pour lire un ETAT COMPLET (et non un delta) :
//
//	FUN_142e2bfd0   en-tete PAR ENTITE, puis l'etat par defaut et la boucle
//	  -> FUN_1428e2b68   recupere le descripteur d'archetype et la TABLE de 64 entrees
//	    -> FUN_142e2c690  LA BOUCLE : 64 entrees nommees, AUCUN masque de presence
//
// Detail dans `.ai/V7.5/killweapon/WALK_PORT_NOTES.md`, section 6. Ce fichier porte cette
// marche, chaque ecart au walker historique (`WalkKeyframeBody`) etant une OPTION mesurable :
// R7-d avait etabli la forme sans la porter, et son atterrissage plafonnait a 0,85 %.
//
// CE QUI EST PORTE, ET SA PREUVE (adresses relues le 2026-08-17, Ghidra lecture seule) :
//
//	FUN_142e2bfd0 : R(32) id · R(32) typeIndex · R(32) · R(4) (FUN_142e29cf8) · R(8)
//	                = 108 bits d'en-tete, puis R(32) n1 (> 0 -> etat par defaut vtable[0x60],
//	                suivi d'un R(32) de controle si le drapeau film est mis), puis R(32) n2
//	                (> 0 -> vtable[0x88], 0 bit, puis la boucle de composants).
//	FUN_142e2c690 : pour k de 0 a 63, si l'entree k de la table porte un NOM, deserialiser
//	                via vtable[0x28] avec le niveau lu en `entree + 0x100`, puis, si le
//	                drapeau film est mis, R(1) et, si ce bit est mis, R(32).
//
// HORS LIGNE — jamais depuis un chemin de requete. Aucune ecriture, aucun schema.

// keyframeFullStateHeaderBits est l'en-tete PAR ENTITE d'un etat complet tel que
// `FUN_142e2bfd0` le lit : `R(32)` id, `R(32)` typeIndex, `R(32)`, `R(4)`, `R(8)`.
//
// Il ne CONTREDIT pas l'en-tete de 64 bits `[id:32][field:26][ti:6]` valide par R3/R5 : le
// balayeur oracle (`kfValidAnchor`) n'accepte une ancre que si le mot de 32 bits a `q+32`
// vaut moins de 50, ce qui veut dire `field26 == 0` sous une lecture et `typeIndex < 50`
// sous l'autre. Les deux sont INDISCERNABLES sur les ancres acceptees, et les 6 bits de
// `typeIndex` lus en `+58` valent la meme chose dans les deux cas. C'est donc une VARIABLE
// a mesurer, pas un fait acquis d'un cote ou de l'autre.
const keyframeFullStateHeaderBits = 108

// keyframeFullStateSizeBits est la largeur des deux mots de taille que `FUN_142e2bfd0` lit
// autour de l'etat par defaut (`n1` avant, `n2` apres) : ce sont des comptes testes `> 0`,
// pas des longueurs de saut.
const keyframeFullStateSizeBits = 32

// KeyframeFullStateOpt decrit UNE lecture du corps d'un record d'image-cle par la boucle
// d'etat complet. Chaque champ est une des variables du plan R7-e, allumee SEULE.
type KeyframeFullStateOpt struct {
	// HeaderBits : largeur de l'en-tete par entite. `keyframeHeaderBits` (64) = la lecture
	// historique ; `keyframeFullStateHeaderBits` (108) = celle de `FUN_142e2bfd0`.
	HeaderBits int
	// SizeWords : lire les deux `R(32)` de taille qui encadrent l'etat par defaut.
	SizeWords bool
	// DefaultState : jouer le deserialiseur d'etat par defaut de l'archetype (vtable[0x60]).
	DefaultState bool
	// LevelShift : passer a chaque composant le niveau que le JEU lui passe
	// (`u32 @ entree + 0x100`, soit le `Flags[i+1]` de `registry.go`) au lieu du
	// `Flags[i]` que le decodeur sert aujourd'hui — les deux layouts de l'entree de
	// `0x104` octets ne placent pas ce champ au meme endroit.
	LevelShift bool
}

// WalkKeyframeFullState rejoue le corps d'un record d'image-cle par la boucle d'ETAT COMPLET
// du jeu, en partant du premier bit du record (`recBit`). Il REUTILISE la boucle de
// composants de production (`traverseComponentLoop`) et les deserialiseurs d'etat par defaut :
// rien n'est recopie, seul le CADRE change.
//
// Les bascules globales de grammaire (`filmComponentCorruptionCheck`, `simStateComplete`,
// `keyframeWriterI0Grammar`, ...) sont celles du process : l'appelant les regle et detient
// `LockProcessDecode`.
func WalkKeyframeFullState(pay []byte, recBit int, reg *Registry, o KeyframeFullStateOpt) EntityTrace {
	hdr := o.HeaderBits
	if hdr <= 0 {
		hdr = keyframeHeaderBits
	}
	t := EntityTrace{HeldWeapon: noVariant, DesyncAt: -1}
	// Le typeIndex se lit aux 6 bits de queue du deuxieme mot de 32 bits, position
	// commune aux deux lectures d'en-tete (cf. `keyframeFullStateHeaderBits`).
	t.TypeIndex = uint32(kfReadBits(pay, recBit+keyframeRecordTIBit, 6))
	br := NewBitReader(pay)
	br.SetBitPos(recBit + hdr)
	if t.TypeIndex >= objectArchetypeCount {
		t.DesyncAt, t.EndBit = 0, br.BitPos()
		return t
	}
	arch, ok := reg.Archetype(int(t.TypeIndex))
	if !ok {
		t.DesyncAt, t.EndBit = 0, br.BitPos()
		return t
	}
	consumeFullStateDefaultBlock(br, t.TypeIndex, o)
	t.Mask = ^uint64(0) // etat complet : aucun masque de presence, tous les composants presents
	if o.LevelShift {
		arch = shiftArchetypeLevels(arch)
	}
	traverseComponentLoop(br, arch, &t)
	t.EndBit = br.BitPos()
	return t
}

// consumeFullStateDefaultBlock joue ce que `FUN_142e2bfd0` lit ENTRE l'en-tete par entite et
// la boucle de composants : `R(32) n1`, l'etat par defaut, le mot de controle du mode film,
// puis `R(32) n2`. Chaque morceau est optionnel — c'est la variable (b) du plan.
func consumeFullStateDefaultBlock(br *BitReader, ti uint32, o KeyframeFullStateOpt) {
	if o.SizeWords {
		br.ReadBits(keyframeFullStateSizeBits) // n1 : > 0 => l'etat par defaut suit
	}
	if o.DefaultState {
		consumeKeyframeDefaultState(br, ti)
		if filmComponentCorruptionCheck {
			// FUN_142e2bfd0 : mot de controle INCONDITIONNEL (pas de R(1) de garde ici,
			// contrairement au controle PAR COMPOSANT de FUN_142e2c690).
			br.ReadBits(keyframeFullStateSizeBits)
		}
	}
	if o.SizeWords {
		br.ReadBits(keyframeFullStateSizeBits) // n2 : > 0 => vtable[0x88] puis la boucle
	}
}

// shiftArchetypeLevels rend une COPIE de l'archetype dont le niveau du composant `i` est le
// `Flags[i+1]` du registre — la lecture qu'impose le layout d'entree du jeu
// (`[nom @ +0x00][u32 niveau @ +0x100]`) face a celui que `registry.go` suppose
// (`[u32 kind][u32 flags][nom @ +8]`). Les noms, eux, tombent au MEME octet dans les deux
// lectures : seul le niveau se decale.
func shiftArchetypeLevels(a Archetype) Archetype {
	out := Archetype{Index: a.Index, Components: a.Components, Flags: make([]uint32, len(a.Components))}
	for i := range out.Flags {
		out.Flags[i] = a.Level(i + 1)
	}
	return out
}
