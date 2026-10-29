package grammar

// keyframe_entrees.go — LA TABLE D'IMAGE-CLE EST UNE TABLE D'ENTREES, UNE PAR SLOT.
//
// # CE QUE LE JEU LIT (Ghidra, lecture seule, HaloInfinite.exe)
//
//	FUN_1408be074   le tableau compte 0x1fff entrees (`FUN_1411b2400(param_1, 0x1fff)`), indexe par
//	                slot : l entree d un slot vivant est `(eid & 0x3fffffff) * 200 + base`
//	                (`FUN_1405d5dbc`, `FUN_1406cb5f0`)
//	FUN_142e2bfd0   le lecteur relit TOUTES les entrees du tableau, dans l ordre, sans compte dans le
//	                flux : par entree R(32) -> +0, R(32) -> +4 (l archetype), R(32) -> +0xc,
//	                R(4) -> +8 (`FUN_142e29cf8`), R(8) -> +9, soit 108 bits d en-tete ; le corps
//	                (n1, etat par defaut, controle, n2, composants) seulement si
//	                `puVar12[1] != 0xffffffff` ; puis l entree suivante (`puVar12 + 0x32`, 200 octets)
//	FUN_1408f1948   un slot LIBERE porte `+0 = +4 = +0xc = 0xffffffff`
//
// Une entree SANS ARCHETYPE (mot de 32 bits a `q+32` egal a `0xffffffff`) fait donc 108 bits, et
// rien d autre ne la suit. C est la forme d un slot libere chez l ecrivain (`FUN_142e2d08c` ecrit
// toutes les entrees avec le meme en-tete) : identifiant, archetype et mot +0xc a `0xffffffff`, R(4)
// et R(8) nuls — forme mesuree sur 263 224 entrees (rapport R-VEH-4, 2026-10-09). Le jeu ne teste que
// l archetype : cette lecture aussi. L identifiant `0xffffffff` n est donc PAS une sentinelle de fin :
// la table finit apres l entree 0x1ffe.
//
// # LA LECTURE ENTRE DEUX ANCRES (regle Rslot)
//
// Entre deux records de slots s et t, la table porte exactement t - s - 1 entrees : la lecture du jeu
// les traverse toutes. [lireLesEntrees] les lit comme le jeu ; la preuve d une frontiere est
// l atterrissage EXACT sur l ancre suivante. Le slot d une entree intercalee n est pas verifie : le
// lecteur du jeu ne le verifie pas non plus.
//
// HORS LIGNE — jamais depuis un chemin de requete.

import "github.com/JGtm/LevelUp/libs/film/internal/source"

// keyframeEntreesDeTable est le nombre d entrees de la table d image-cle (`FUN_1408be074`) : les slots
// 0 a 0x1ffe.
const keyframeEntreesDeTable = 0x1fff

// genreDEntree dit ce que porte une entree de la table d image-cle.
type genreDEntree uint8

const (
	// entreeIllisible : l entree deborde du payload, ou son mot d archetype n est ni
	// `0xffffffff` ni un archetype de la table du jeu (< 50) — le jeu deriverait sur un descripteur
	// hors table ; une lecture qui y arrive n est pas sur une frontiere d entree.
	entreeIllisible genreDEntree = iota
	// entreeSansCorps : archetype `0xffffffff`, 108 bits d en-tete et rien d autre (slot libere).
	entreeSansCorps
	// entreeRecord : un archetype de la table ; le corps suit l en-tete.
	entreeRecord
)

// lireEntreeDImageCle classe l entree qui commence au bit `q` d un payload de `total` bits, sous un
// en-tete de `enTete` bits (le cadre du profil, 108 chez le jeu).
func lireEntreeDImageCle(pay []byte, q, total, enTete int) genreDEntree {
	if q < 0 || q+enTete > total {
		return entreeIllisible
	}
	switch arch := source.BitsBourres(pay, q+32, 32); {
	case arch == keyframeArchetypeNone:
		return entreeSansCorps
	case arch < kfArchMax:
		return entreeRecord
	}
	return entreeIllisible
}

// lireLesEntrees lit `n` entrees de la table depuis le bit `pos`, comme `FUN_142e2bfd0` : 108 bits
// pour une entree sans archetype, l etat complet ([WalkKeyframeFullState]) pour un record. Elle rend
// la position atteinte, et faux quand une entree est illisible, qu un record intercale desynchronise
// ou que la lecture deborde du payload.
//
// AUCUNE OBSERVATION : les entrees intercalees ne sont pas des records du paquet ; leurs crochets ne
// s executent pas.
func lireLesEntrees(pay []byte, pos, n int, reg *Registry, ctx ContexteDeLecture) (int, bool) {
	ctx.Obs = nil
	total := len(pay) * 8
	enTete := ctx.Profil.Cadre.EnTeteBits
	for range n {
		switch lireEntreeDImageCle(pay, pos, total, enTete) {
		case entreeSansCorps:
			pos += enTete
		case entreeRecord:
			tr := WalkKeyframeFullState(pay, pos, reg, ctx)
			if tr.DesyncAt >= 0 || tr.EndBit > total {
				return pos, false
			}
			pos = tr.EndBit
		default:
			return pos, false
		}
	}
	return pos, true
}

// fermeSurLAncreSuivante dit si le record borne `b` FERME : la traversee de son etat complet (`tr`),
// puis la lecture des `slotSuivant - slot - 1` entrees que la table porte entre lui et l ancre
// suivante, atterrissent EXACTEMENT sur le premier bit de cette ancre. C est la seule preuve qu on ait
// que toutes ses largeurs sont justes (`keyframe_closure.go`), et LA definition de la fermeture : la
// preuve d etat complet de la phase des images-cles et les compteurs `KeyClosed` des balayages la
// partagent (garde-rail `keyframe_fermeture_unique_test.go`).
//
// Un record sans frontiere (dernier de son paquet), arrete, ou dont l ancre suivante n a pas un slot
// superieur ne ferme pas. Chaque entree fait au moins un en-tete : une traversee qui finit a moins de
// `n` en-tetes de l ancre ne peut pas l atteindre, et ne se lit pas plus loin.
func fermeSurLAncreSuivante(pay []byte, tr EntityTrace, b keyframeBorne, reg *Registry, ctx ContexteDeLecture) bool {
	if b.Want < 0 || tr.DesyncAt >= 0 {
		return false
	}
	n := b.SlotSuivant - b.Slot - 1
	switch {
	case n < 0:
		return false
	case n == 0:
		return tr.EndBit == b.Want
	case tr.EndBit+n*ctx.Profil.Cadre.EnTeteBits > b.Want:
		return false
	}
	pos, ok := lireLesEntrees(pay, tr.EndBit, n, reg, ctx)
	return ok && pos == b.Want
}

// sauterLesEntreesSansCorps avance, depuis le bit `pos`, sur les entrees sans archetype qui s y
// suivent (108 bits chacune, [entreeSansCorps]) et rend la position atteinte et leur nombre.
func sauterLesEntreesSansCorps(pay []byte, pos, total, enTete int) (int, int) {
	k := 0
	for lireEntreeDImageCle(pay, pos, total, enTete) == entreeSansCorps {
		pos += enTete
		k++
	}
	return pos, k
}

// finitSurUneEntree dit si la position `pos` porte une entree de la table : un en-tete de record
// valide ([readKeyframeHeader]) ou un slot libere sous la forme EXACTE de l ecrivain
// ([estUnSlotLibere]). C est le temoin FAIBLE de chainage des balayages (`KeyChained`) : une traversee
// qui finit devant un slot libere finit sur une frontiere d entree.
//
// LA FORME EXACTE, ET NON LE SEUL ARCHETYPE : un temoin juge une position qui n est peut-etre pas une
// frontiere. Les 96 bits a 1 d un slot libere portent un mot d archetype `0xffffffff` a 65 positions
// (de 32 bits avant l entree a 32 bits apres) ; seule sa premiere porte les trois mots.
func finitSurUneEntree(pay []byte, pos int, ctx ContexteDeLecture) bool {
	total := len(pay) * 8
	if estUnSlotLibere(pay, pos, total, ctx.Profil.Cadre.EnTeteBits) {
		return true
	}
	_, ok := readKeyframeHeader(pay, pos, total)
	return ok
}

// estUnSlotLibere dit si l entree du bit `pos` a la forme qu ecrit `FUN_1408f1948` pour un slot
// libere : identifiant, archetype et mot +0xc a `0xffffffff`.
func estUnSlotLibere(pay []byte, pos, total, enTete int) bool {
	if pos < 0 || pos+enTete > total {
		return false
	}
	for _, o := range [3]int{0, 32, 64} {
		if source.BitsBourres(pay, pos+o, 32) != keyframeArchetypeNone {
			return false
		}
	}
	return true
}

// fermeSurLaFinDeTable dit si le DERNIER record d un paquet (`b.Want < 0` : aucune ancre ne le suit)
// FERME sur la fin de la table : la traversee de son etat complet (`tr`), puis la lecture des
// `0x1ffe - slot` entrees qui le suivent jusqu a la derniere de la table, finissent dans le dernier
// octet du payload — la table remplit le payload, a l octet de bourrage pres. C est la meme preuve que
// [fermeSurLAncreSuivante], la fin de la table tenant lieu d ancre suivante.
func fermeSurLaFinDeTable(pay []byte, tr EntityTrace, b keyframeBorne, reg *Registry, ctx ContexteDeLecture) bool {
	if b.Want >= 0 || tr.DesyncAt >= 0 || b.Slot < 0 || b.Slot >= keyframeEntreesDeTable {
		return false
	}
	total, n := len(pay)*8, keyframeEntreesDeTable-1-b.Slot
	if tr.EndBit+n*ctx.Profil.Cadre.EnTeteBits > total {
		return false
	}
	pos, ok := lireLesEntrees(pay, tr.EndBit, n, reg, ctx)
	return ok && pos <= total && total-pos < 8
}
