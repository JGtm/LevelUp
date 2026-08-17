package filmdec

// keyframe_record_walk.go — LE WALKER DETERMINISTE DE LA TABLE D'IMAGE-CLE : parser le
// corps des records au lieu de balayer leurs en-tetes.
//
// CE QUE CE FICHIER ATTAQUE. Deux lots mesures le 2026-08-17 (R3 `ti=37`, R4 `ti=11`) ont
// conclu au meme manque : « le deserialiseur du corps d'un record d'image-cle n'est resolu
// nulle part ». La lecture du code du jeu (journal RE, `WALK_PORT_NOTES.md` section
// « IMAGE-CLE ») a montre l'inverse de ce qui etait suppose : les DEUX lecteurs de record
// NEW du jeu (`FUN_141f86704` bufferise, `FUN_1408f1aa4` direct) portent la MEME grammaire,
// et c'est celle que `TraverseEntity` porte deja ; le chemin DELTA appelle la MEME boucle de
// composants (`FUN_14076cb60`). Il n'existe pas de variante « image-cle » du corps.
//
// CE QUI RESTAIT SUSPECT, ET C'EST L'ORACLE. `WalkKeyframeWorld` ne PARSE pas : il BALAIE
// (`kfScanNext`) et n'accepte une ancre QUE si le mot de 32 bits a `q+32` vaut moins de 50
// (`keyframe_world.go:70`), c'est-a-dire seulement si les 26 bits de `field` de l'en-tete
// `[id:32][field:26][ti:6]` sont TOUS NULS. Un record dont le `field` n'est pas nul est donc
// SAUTE, et le « record suivant » que le balayeur rend n'est alors pas le voisin mais le
// voisin du voisin. Une marche juste atterrirait dans ce cas TROP TOT d'exactement une
// longueur de record — ce qui est le sens, l'ordre de grandeur et la recurrence de l'ecart
// mesure par R3 (557 a 1 104 bits, les memes valeurs d'un film a l'autre).
//
// CE QUE CE FICHIER FAIT. Il lit l'en-tete de 64 bits SANS ce filtre, puis rejoue le lecteur
// de record NEW de PRODUCTION (`TraverseEntity`) sur le corps, et enchaine. Il ne recopie
// AUCUN deserialiseur : c'est le meme code que partout ailleurs, positionne autrement.
//
// HORS LIGNE — jamais depuis un chemin de requete.

// keyframeHeaderBits est la largeur de l'en-tete d'un record de la table d'image-cle :
// `[id:32][field:26][ti:6]`. Le corps commence donc a `BitStart + 64`, et les 6 bits de
// `typeIndex` par lesquels `TraverseEntity` commence sont a `BitStart + 58`.
const keyframeHeaderBits = 64

// keyframePrefixBits est le prefixe de 1 bit en tete du payload d'image-cle, avant le
// premier record (meme valeur que `WalkKeyframeWorld`, qui demarre a `pos = 1`).
const keyframePrefixBits = 1

// KeyframeHeader est l'en-tete de 64 bits d'un record de la table d'image-cle, lu SANS le
// filtre fort du balayeur : `Field26` est rendu tel quel au lieu d'etre exige nul.
type KeyframeHeader struct {
	// Slot et Gen identifient l'entite (`id = gen<<30 | slot`).
	Slot, Gen int
	// TI est le typeIndex de l'archetype, lu sur les 6 bits de queue de l'en-tete.
	TI int
	// Field26 porte les 26 bits centraux, dont la semantique n'est PAS etablie (cf. journal
	// RE, section « ce qui reste NON resolu »). Le balayeur du depot exige qu'ils soient
	// nuls ; ce lecteur les MESURE.
	Field26 uint32
}

// readKeyframeHeader lit l'en-tete de 64 bits a la position bit q. `ok` est faux si
// l'en-tete deborde du payload, si l'identifiant est la sentinelle, si la generation est
// nulle (handle null), si le slot sort de la table ou si le typeIndex depasse le cap objet
// du jeu (50). AUCUNE contrainte sur `Field26` — c'est toute la difference avec
// `kfValidAnchor`.
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
	h.Field26 = uint32(kfReadBits(pay, q+32, 26))
	h.TI = int(kfReadBits(pay, q+58, 6))
	return h, h.TI < kfArchMax
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
// Les bascules globales de grammaire (`filmComponentCorruptionCheck`, `newRecordTailBits`,
// `useArchDefaultStateDeser`) sont celles du process : l'appelant les regle et detient
// `LockProcessDecode`.
func WalkKeyframeRecords(pay []byte, reg *Registry) ([]KeyframeWalkRec, KeyframeWalkStop) {
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
		rec, stop, done := walkOneKeyframeRecord(pay, reg, pos, h)
		out = append(out, rec)
		if done {
			return out, stop
		}
		pos, prevSlot = rec.BitEnd, h.Slot
	}
}

// walkOneKeyframeRecord rejoue le corps d'UN record et rend le record, la cause d'arret
// eventuelle et un booleen d'arret. Extrait de `WalkKeyframeRecords` pour tenir le seuil de
// 80 lignes par fonction.
func walkOneKeyframeRecord(pay []byte, reg *Registry, pos int, h KeyframeHeader) (
	KeyframeWalkRec, KeyframeWalkStop, bool,
) {
	br := NewBitReader(pay)
	br.SetBitPos(pos + keyframeRecordTIBit)
	tr := TraverseEntity(br, reg, 0)
	rec := KeyframeWalkRec{
		KeyframeHeader: h, BitStart: pos, BitEnd: tr.EndBit,
		Mask: tr.Mask, Gate: tr.Gate, DesyncAt: tr.DesyncAt,
	}
	if tr.DesyncAt >= 0 {
		return rec, KeyframeStopDesync, true
	}
	return rec, KeyframeStopEnd, false
}

// KeyframeChainResult est le resultat d'un CHAINAGE : partant de la fin de marche d'un
// record, combien de records intercales faut-il traverser pour retomber sur une frontiere
// connue de l'oracle (`WalkKeyframeWorld`) ?
//
// C'est la mesure qui tranche l'hypothese H1 du plan R5 : si la grammaire est juste et que
// c'est le FILTRE du balayeur qui saute des records, le chainage retombe exactement sur la
// frontiere de l'oracle apres un petit nombre de records intercales, et ces intercales
// portent un `Field26` NON NUL.
type KeyframeChainResult struct {
	// Reached : le chainage a atteint EXACTEMENT la frontiere visee.
	Reached bool
	// Skipped est le nombre de records intercales traverses avant de l'atteindre (0 = la
	// marche du record lui-meme atterrissait deja juste).
	Skipped int
	// SkippedFieldNonZero compte, parmi ces intercales, ceux dont `Field26` n'est pas nul —
	// c'est-a-dire ceux que le filtre fort du balayeur ne pouvait PAS voir.
	SkippedFieldNonZero int
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
// L'appelant regle les bascules globales de grammaire et detient `LockProcessDecode`.
func ChainKeyframeRecords(pay []byte, reg *Registry, from, want, prevSlot int) KeyframeChainResult {
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
		rec, stop, done := walkOneKeyframeRecord(pay, reg, pos, h)
		if done {
			res.Stop = stop
			return res
		}
		res.Skipped++
		if h.Field26 != 0 {
			res.SkippedFieldNonZero++
		}
		pos, prevSlot = rec.BitEnd, h.Slot
	}
	res.Stop = KeyframeStopBudget
	return res
}

// ---------------------------------------------------------------------------------------
// LES VARIANTES DE CORPS — parce qu'aucune n'est supposee, elles sont toutes MESUREES.
//
// Le balayage du decalage (instrument `TestKFGramOffset`) a etabli deux faits : les 6 bits
// de `typeIndex` sont bien a `+58` (415/415 et 2008/2008 records relus corrects, aucun autre
// decalage de 0 a 127 ne fait mieux que le hasard), et AUCUN decalage ne rend une seule
// marche bit-exacte. Le corps d'un record de la table d'image-cle n'est donc PAS le corps
// d'un record NEW, quel que soit l'endroit ou on le pose.
//
// Reste la lecture (a) de la decouverte 3 du lot R3 : l'image-cle porterait un ETAT COMPLET
// — tous les composants de l'archetype, sans masque epars. Les longueurs reelles mesurees
// vont dans ce sens (ti=38 : 39 valeurs distinctes seulement sur 2 008 records, dominante
// 827 bits ; la marche de record NEW n'en consomme qu'environ 40 %). `KeyframeBodyVariant`
// expose les trois bascules qui separent ces lectures, et l'instrument les balaie toutes.
// ---------------------------------------------------------------------------------------

// KeyframeBodyVariant decrit UNE lecture possible du corps d'un record d'image-cle.
type KeyframeBodyVariant struct {
	// DefaultState : jouer le deserialiseur d'etat par defaut de l'archetype (vtable[0x60]).
	DefaultState bool
	// Gate : lire la porte `R(1)` qui precede le masque dans un record NEW.
	Gate bool
	// Mask : lire le masque de presence dans le flux (sinon : TOUS les composants presents).
	Mask bool
}

// String rend une etiquette lisible de la variante.
func (v KeyframeBodyVariant) String() string {
	f := func(b bool) string {
		if b {
			return "oui"
		}
		return "non"
	}
	return "etatParDefaut=" + f(v.DefaultState) + " porte=" + f(v.Gate) + " masque=" + f(v.Mask)
}

// KeyframeBodyVariants est la matrice des huit lectures probees. La premiere est celle du
// record NEW (celle que `TraverseEntity` joue), la derniere l'etat complet nu.
var KeyframeBodyVariants = []KeyframeBodyVariant{
	{DefaultState: true, Gate: true, Mask: true},
	{DefaultState: true, Gate: true, Mask: false},
	{DefaultState: true, Gate: false, Mask: true},
	{DefaultState: true, Gate: false, Mask: false},
	{DefaultState: false, Gate: true, Mask: true},
	{DefaultState: false, Gate: true, Mask: false},
	{DefaultState: false, Gate: false, Mask: true},
	{DefaultState: false, Gate: false, Mask: false},
}

// WalkKeyframeBody rejoue le corps d'un record d'image-cle sous la variante `v`, en partant
// du debut du record (`recBit`). Il REUTILISE la boucle de composants de production
// (`traverseComponentLoop`) et les deserialiseurs d'etat par defaut de production : rien
// n'est recopie, seule la faconde lire l'en-tete de corps change.
func WalkKeyframeBody(pay []byte, recBit int, reg *Registry, v KeyframeBodyVariant) EntityTrace {
	br := NewBitReader(pay)
	br.SetBitPos(recBit + keyframeRecordTIBit)
	t := EntityTrace{HeldWeapon: noVariant, DesyncAt: -1}
	t.TypeIndex = uint32(br.ReadBits(6))
	if t.TypeIndex >= objectArchetypeCount {
		t.DesyncAt, t.EndBit = 0, br.BitPos()
		return t
	}
	if v.DefaultState {
		consumeKeyframeDefaultState(br, t.TypeIndex)
	}
	if v.Gate {
		t.Gate = br.ReadBit()
	}
	if v.Mask {
		t.Mask = consumeMask(br)
	} else {
		t.Mask = ^uint64(0) // etat complet : tous les composants presents
	}
	arch, ok := reg.Archetype(int(t.TypeIndex))
	if !ok {
		t.DesyncAt, t.EndBit = 0, br.BitPos()
		return t
	}
	traverseComponentLoop(br, arch, &t)
	t.EndBit = br.BitPos()
	return t
}

// consumeKeyframeDefaultState joue l'etat par defaut de l'archetype, par le MEME routage que
// le lecteur de record NEW de production (biped a part, table `defaultStateDeserByTI`
// ensuite, stub 0 bit sinon).
func consumeKeyframeDefaultState(br *BitReader, ti uint32) {
	if ti == bipedDefaultStateTypeIndex {
		consumeBipedDefaultState(br)
		consumeBipedDefaultStateTail(br)
		return
	}
	if fn, ok := defaultStateDeserByTI[ti]; ok {
		fn(br)
	}
}
