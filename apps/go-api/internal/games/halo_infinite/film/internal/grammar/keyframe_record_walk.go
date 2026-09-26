package grammar

// keyframe_record_walk.go - LA MARCHE DETERMINISTE DE LA TABLE D'IMAGE-CLE : suivre l'ecrivain
// d'un record au suivant, sans balayeur ni fenetre.
//
// CE QUE LE JEU LIT, PAR ADRESSE (lot 5.20.1, Ghidra lecture seule). Le bloc de type 2 d'un
// film n'est PAS consomme par le repartiteur de paquets (`FUN_1428e22c0` ne connait que neuf
// types et le 2 n'en est pas) : il passe par la seconde voie a en-tete de 16 octets,
// `FUN_1428e2a04` -> `FUN_1428e2a9c`, qui charge le payload puis appelle
//
//	FUN_142e2bfd0(lecteur, tableau)   LE LECTEUR D'IMAGE-CLE
//
// dont la boucle remplit un tableau d'entrees de 200 octets, UNE PAR ENTITE VIVANTE :
//
//	[si version > 7] R(1)                 une seule fois, en tete de payload
//	                                      (`DAT_144706104`, le selecteur de filigrane)
//	par entite :
//	  R(32) -> entree+0x00                l'identifiant (`eid`)
//	  R(32) -> entree+0x04                L'ARCHETYPE, MOT PLEIN DE 32 BITS
//	  R(32) -> entree+0x0c
//	  R(4)  -> entree+0x08                (`FUN_142e29cf8`)
//	  R(8)  -> entree+0x09                = 108 bits d'en-tete par entite
//	  si archetype != 0xffffffff :
//	     R(32) n1 ; si n1 > 0 : vtable[0x60] (l'etat par defaut) [+ R(32) de controle si le
//	                drapeau film est mis]
//	     R(32) n2 ; si n2 > 0 : vtable[0x88] (aucun bit) puis `FUN_1428e2b68` ->
//	                `FUN_142e2c690`, la boucle des 64 entrees NOMMEES du registre, SANS
//	                masque de presence, chacune au niveau lu en `entree + 0x100`
//
// Ce corps est deja porte par `WalkKeyframeFullState` (`keyframe_fullstate_loop.go`, lot 1.4) :
// la marche ci-dessous ne recopie donc AUCUN deserialiseur, elle l'enchaine.
//
// CE QUE CETTE LECTURE CORRIGE, ET C'EST LA CAUSE DE L'ARRET SUR « en-tete-invalide ». Le
// depot modelisait l'en-tete `[id:32][field:26][ti:6]` et lisait le corps par `TraverseEntity`,
// c'est-a-dire par le cadre du record NEW du chemin DELTA (`R(6)` d'archetype, etat par defaut,
// PORTE, MASQUE de presence). L'image-cle n'a ni porte ni masque, et son en-tete fait 108 bits,
// pas 64 : la marche repartait 44 bits trop tot, au milieu du premier corps, et le deuxieme
// en-tete n'etait jamais valide. Le « champ de 26 bits de semantique non etablie » n'existe pas
// non plus : les 32 bits a `q+32` SONT l'archetype (`FUN_142e2bfd0` s'en sert tel quel pour
// indexer `DAT_144e61d88 + 8 + ti*8`), et un mot >= 50 y ferait deriver le jeu sur un
// descripteur hors table. L'hypothese H1 du lot R5 - « le balayeur saute les records dont
// `Field26` n'est pas nul » - est donc REFUTEE PAR L'ECRIVAIN : de tels records n'existent pas.
// La seule valeur hors table admise est `0xffffffff`, qui dit « pas d'archetype » et clot
// l'entree a ses 108 bits d'en-tete.
//
// HORS LIGNE - jamais depuis un chemin de requete.

// keyframeHeaderBits est la largeur des DEUX MOTS QUI IDENTIFIENT un record d'image-cle :
// `[eid:32][archetype:32]`. Ce n'est pas la fin de l'en-tete - le corps commence a
// `BitStart + profile.KeyframeEnTeteBits` (108) - mais c'est tout ce qu'il faut lire pour
// decider si une position porte un record, et c'est la fenetre que `kfValidAnchor` teste.
const keyframeHeaderBits = 64

// keyframePrefixBits est le prefixe de 1 bit en tete du payload d'image-cle, avant le
// premier record : `FUN_142e2bfd0` le lit dans `DAT_144706104` quand la version du film
// depasse 7 (meme valeur que `WalkKeyframeWorld`, qui demarre a `pos = 1`).
const keyframePrefixBits = 1

// keyframeArchetypeNone est le mot d'archetype qui dit « pas d'archetype » :
// `FUN_142e2bfd0` saute alors les deux mots de taille et tout le corps (`if (puVar12[1] !=
// 0xffffffff)`), et l'entree s'arrete a ses 108 bits d'en-tete.
const keyframeArchetypeNone = 0xFFFFFFFF

// KeyframeHeader porte les deux mots de 32 bits qui identifient un record de la table
// d'image-cle.
type KeyframeHeader struct {
	// Slot et Gen identifient l'entite (`id = gen<<30 | slot`).
	Slot, Gen int
	// TI est le typeIndex de l'archetype, valide seulement quand `Archetype` est sous le cap
	// objet du jeu (50) ; -1 quand l'entree ne porte pas d'archetype.
	TI int
	// Archetype est le MOT PLEIN de 32 bits lu a `q+32` : `FUN_142e2bfd0` l'utilise tel quel
	// pour indexer la table des descripteurs. `keyframeArchetypeNone` = pas d'archetype.
	Archetype uint32
}

// SansArchetype dit que l'entree ne porte aucun corps : ni etat par defaut, ni composants.
func (h KeyframeHeader) SansArchetype() bool { return h.Archetype == keyframeArchetypeNone }

// readKeyframeHeader lit les deux mots identifiants a la position bit q. `ok` est faux si
// l'en-tete deborde du payload, si l'identifiant est la sentinelle, si la generation est
// nulle (handle null), si le slot sort de la table, ou si le mot d'archetype n'est ni un
// index sous le cap objet du jeu (50) ni `keyframeArchetypeNone`.
func readKeyframeHeader(pay []byte, q, total int) (h KeyframeHeader, ok bool) {
	if q < 0 || q+keyframeHeaderBits > total {
		return h, false
	}
	id := kfReadBits(pay, q, 32)
	if id == kfSent {
		return h, false
	}
	h.Gen = int(id >> 30)
	if h.Gen == 0 {
		return h, false
	}
	h.Slot = int(id & 0x3FFFFFFF)
	if h.Slot >= kfTableCap {
		return h, false
	}
	h.Archetype = uint32(kfReadBits(pay, q+32, 32))
	if h.SansArchetype() {
		h.TI = -1
		return h, true
	}
	h.TI = int(h.Archetype)
	return h, h.Archetype < kfArchMax
}

// KeyframeWalkStop dit POURQUOI la marche deterministe s'est arretee. Une marche qui
// s'arrete sans le dire n'est pas une mesure.
type KeyframeWalkStop int

// Les causes d'arret, dans l'ordre de gravite decroissante.
const (
	// KeyframeStopEnd : le payload est epuise (fin normale).
	KeyframeStopEnd KeyframeWalkStop = iota
	// KeyframeStopHeader : l'en-tete a la position atteinte n'est pas valide — la marche
	// precedente n'a donc pas atterri sur une frontiere de record.
	KeyframeStopHeader
	// KeyframeStopDesync : un composant present n'est pas porte (`EntityTrace.DesyncAt`).
	KeyframeStopDesync
	// KeyframeStopSlot : le slot ne croit pas, alors que la table est a slots croissants.
	KeyframeStopSlot
	// KeyframeStopBudget : le garde-fou de nombre de records a saute (payload pathologique).
	KeyframeStopBudget
)

// String rend l'etiquette de la cause d'arret.
func (s KeyframeWalkStop) String() string {
	switch s {
	case KeyframeStopEnd:
		return "fin-du-payload"
	case KeyframeStopHeader:
		return "en-tete-invalide"
	case KeyframeStopDesync:
		return "composant-non-porte"
	case KeyframeStopSlot:
		return "slot-non-croissant"
	case KeyframeStopBudget:
		return "budget-epuise"
	}
	return "cause-inconnue"
}

// KeyframeWalkRec est UN record parse par le walker deterministe : son en-tete, les bornes
// EN BITS de son corps, et ce que la traversee a rencontre.
type KeyframeWalkRec struct {
	KeyframeHeader
	// BitStart est la position du premier bit de l'en-tete ; BitEnd celle du premier bit du
	// record SUIVANT (donc la fin exclusive du corps).
	BitStart, BitEnd int
	// Mask est le masque de presence lu par le corps, Gate la porte qui le precede.
	Mask uint64
	Gate bool
	// Comps sont les composants traverses, avec leur position de bit et, pour ceux de
	// `captureNames`, leur VALEUR. La traversee les calcule de toute facon ; les jeter obligeait
	// tout lecteur d ETAT (l occupation d un vehicule a l instant d une image-cle, lot 5.10) a
	// re-marcher la table pour son propre compte.
	Comps []CompResult
	// DesyncAt est l'index du premier composant present non porte, ou -1.
	DesyncAt int
}

// keyframeWalkBudget borne le nombre de records d'un payload. La plus grosse table observee
// tient sous 8 192 entites (`kfTableCap`) ; le double laisse la place a une table anormale
// sans laisser une marche folle tourner sans fin.
const keyframeWalkBudget = 2 * kfTableCap

// WalkKeyframeRecords PARSE la table d'image-cle : en-tete de 64 bits, puis corps par le
// lecteur de record NEW de PRODUCTION (`TraverseEntity`), puis enchainement sur la position
// atteinte. Il rend les records parses et la cause d'arret.
//
// A la difference de `WalkKeyframeWorld`, il ne balaie RIEN et n'apprend AUCUNE largeur : si
// la grammaire portee est juste, la position atteinte EST la frontiere suivante. C'est donc
// aussi un test de la grammaire — l'arret dit ou elle a lache.
//
// LE PROFIL DE BALAYAGE EST UN PARAMETRE (lot 2.3) : c'est par lui que les largeurs du film
// descendent jusqu'aux feuilles.
func WalkKeyframeRecords(pay []byte, reg *Registry, ctx ContexteDeLecture) ([]KeyframeWalkRec, KeyframeWalkStop) {
	total := len(pay) * 8
	out := make([]KeyframeWalkRec, 0, 512)
	pos, prevSlot := keyframePrefixBits, -1
	for {
		if len(out) >= keyframeWalkBudget {
			return out, KeyframeStopBudget
		}
		if pos+keyframeHeaderBits > total {
			return out, KeyframeStopEnd
		}
		h, ok := readKeyframeHeader(pay, pos, total)
		if !ok {
			if kfReadBits(pay, pos, 32) == kfSent {
				return out, KeyframeStopEnd // sentinelle de fin de table
			}
			return out, KeyframeStopHeader
		}
		if h.Slot <= prevSlot {
			return out, KeyframeStopSlot
		}
		rec, stop, done := walkOneKeyframeRecord(pay, reg, pos, h, ctx)
		out = append(out, rec)
		if done {
			return out, stop
		}
		pos, prevSlot = rec.BitEnd, h.Slot
	}
}

// walkOneKeyframeRecord rejoue le corps d'UN record par la boucle d'ETAT COMPLET du jeu
// (`WalkKeyframeFullState` = `FUN_142e2bfd0` + `FUN_142e2c690`) et rend le record, la cause
// d'arret eventuelle et un booleen d'arret. Extrait de `WalkKeyframeRecords` pour tenir le
// seuil de 80 lignes par fonction.
//
// UNE ENTREE SANS ARCHETYPE NE PORTE PAS DE CORPS : le jeu saute alors les deux mots de
// taille et la boucle de composants, et passe a l'entree suivante 108 bits plus loin.
func walkOneKeyframeRecord(pay []byte, reg *Registry, pos int, h KeyframeHeader,
	ctx ContexteDeLecture) (
	KeyframeWalkRec, KeyframeWalkStop, bool,
) {
	if h.SansArchetype() {
		rec := KeyframeWalkRec{
			KeyframeHeader: h, BitStart: pos, BitEnd: pos + keyframeFullHeaderBits(ctx),
			DesyncAt: -1,
		}
		return rec, KeyframeStopEnd, false
	}
	tr := WalkKeyframeFullState(pay, pos, reg, ctx)
	rec := KeyframeWalkRec{
		KeyframeHeader: h, BitStart: pos, BitEnd: tr.EndBit,
		Mask: tr.Mask, Gate: tr.Gate, Comps: tr.Comps, DesyncAt: tr.DesyncAt,
	}
	if tr.DesyncAt >= 0 {
		return rec, KeyframeStopDesync, true
	}
	return rec, KeyframeStopEnd, false
}

// keyframeFullHeaderBits rend la largeur de l'en-tete PAR ENTITE que porte le profil du
// lecteur (108 chez `FUN_142e2bfd0`) - la meme que `WalkKeyframeFullState` consomme.
func keyframeFullHeaderBits(ctx ContexteDeLecture) int {
	br := LecteurSur(nil)
	br.PoserContexte(ctx)
	return br.cadre().EnTeteBits
}

// KeyframeChainResult est le resultat d'un CHAINAGE : partant de la fin de marche d'un
// record, combien de records intercales faut-il traverser pour retomber sur une frontiere
// connue de l'oracle (`WalkKeyframeWorld`) ?
//
// C'est la mesure qui chiffre ce que le BALAYEUR saute entre deux de ses ancres : si la
// grammaire est juste, le chainage retombe exactement sur la frontiere de l'oracle apres
// un petit nombre de records intercales. (L'hypothese H1 du plan R5 -- « les intercales
// portent un `Field26` non nul » -- est refutee par l'ecrivain, cf. l'en-tete de ce
// fichier : le seul intercale que le filtre fort ne peut pas voir est celui SANS
// ARCHETYPE.)
type KeyframeChainResult struct {
	// Reached : le chainage a atteint EXACTEMENT la frontiere visee.
	Reached bool
	// Skipped est le nombre de records intercales traverses avant de l'atteindre (0 = la
	// marche du record lui-meme atterrissait deja juste).
	Skipped int
	// SkippedSansArchetype compte, parmi ces intercales, ceux qui ne portent AUCUN
	// archetype (`keyframeArchetypeNone`) : le balayeur d'ancres ne peut pas les voir,
	// son filtre fort exigeant un mot d'archetype sous le cap objet.
	SkippedSansArchetype int
	// Stop dit ou le chainage s'est arrete quand il n'a pas atteint la frontiere.
	Stop KeyframeWalkStop
}

// keyframeChainMax borne le nombre de records intercales explores. Au-dela, ce n'est plus
// « le balayeur a saute un voisin » mais une marche perdue, et le dire est le resultat.
const keyframeChainMax = 16

// ChainKeyframeRecords enchaine la marche depuis la position `from` jusqu'a atteindre
// exactement `want`, sans jamais depasser `keyframeChainMax` records intercales. `prevSlot`
// est le slot du record d'ou l'on part (la table est a slots croissants).
//
// Le PROFIL DE BALAYAGE vient de l'appelant (lot 2.3).
func ChainKeyframeRecords(pay []byte, reg *Registry, from, want, prevSlot int,
	ctx ContexteDeLecture) KeyframeChainResult {
	total := len(pay) * 8
	res := KeyframeChainResult{Stop: KeyframeStopEnd}
	pos := from
	for res.Skipped <= keyframeChainMax {
		if pos == want {
			res.Reached = true
			return res
		}
		if pos > want || pos+keyframeHeaderBits > total {
			res.Stop = KeyframeStopEnd
			return res
		}
		h, ok := readKeyframeHeader(pay, pos, total)
		if !ok {
			res.Stop = KeyframeStopHeader
			return res
		}
		if h.Slot <= prevSlot {
			res.Stop = KeyframeStopSlot
			return res
		}
		rec, stop, done := walkOneKeyframeRecord(pay, reg, pos, h, ctx)
		if done {
			res.Stop = stop
			return res
		}
		res.Skipped++
		if h.SansArchetype() {
			res.SkippedSansArchetype++
		}
		pos, prevSlot = rec.BitEnd, h.Slot
	}
	res.Stop = KeyframeStopBudget
	return res
}

// consumeKeyframeDefaultState joue l'etat par defaut de l'archetype, par le MEME routage que
// le lecteur de record NEW de production (biped a part, table `defaultStateDeserByTI`
// ensuite, stub 0 bit sinon).
func consumeKeyframeDefaultState(br *Lecteur, ti uint32) {
	if ti == bipedDefaultStateTypeIndex {
		consumeBipedDefaultState(br)
		consumeBipedDefaultStateTail(br)
		return
	}
	if fn, ok := defaultStateDeserByTI[ti]; ok {
		fn(br)
	}
}
