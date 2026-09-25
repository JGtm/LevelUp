package grammar

// debut_de_liste.go — LA LISTE D UN PAQUET A EVENEMENTS COMMENCE A SES RECORDS NEW DE TETE (lot
// M4b de la campagne « retours rejeu », 2026-09-25).
//
// # LE DEFAUT, MESURE
//
// Dans un paquet delta a liste d evenements, le localisateur de production (`marchLocateStrict`,
// signature du slot 123) demarre la marche sur le premier record qu il sait ancrer. Or les
// CREATIONS d objets de l instant sont les premiers records de la liste — la naissance d un
// bipede (lot M3.2 : 40/41, 91/99 et 123/125 cas, `birth_loadouts.go`), les armes et
// l equipement laches a sa mort, les projectiles : le localisateur les SAUTE. Le monde ne lie
// donc jamais l objet ne en milieu de chunk qui disparait avant l image-cle suivante (la table
// anticipee du lot 5.23 ne le voit pas), et CHAQUE delta suivant de cet objet clot la vue B sur
// le rejet de son en-tete : la vue C — le tir continu — n est plus lue. Mesure sur `81c02726` :
// le bipede 521 (index 4, ne a la trame 495) rejette 438 paquets entre 495 et 575, puis les armes
// et l equipement qu il lache (1748, 1750) ceux de 575 a 658 — le trou de lecture 500-658 du gate
// G1, deux frags au Ghost sans rafale.
//
// # LA LECTURE, ET SA PREUVE
//
// Un CANDIDAT est une position qui porte un en-tete de record NEW (prefixe `0` puis `01`) dont le
// slot est dans la BANDE de l archetype annonce, lue dans les images-cles du film par la regle des
// objets du monde ([TableAnticipee.SlotDeLArchetype]). Un candidat n est qu un candidat. Il n est
// retenu que si la CHAINE des records qui en partent — chacun lu par son en-tete comme la boucle
// de records le lit : NEW traverse sans desynchronisation, DEL, delta qui se decode sur un slot que
// le monde connait — finit EXACTEMENT sur le debut que le localisateur a trouve : deux lectures
// independantes, l en-tete en tete et la signature du slot 123 en queue, qui s accordent au bit
// pres. Quand le localisateur ne trouve rien, la preuve est la FERMETURE de la vue C ([vueCFermee])
// par la marche complete qui part du candidat. Sinon le debut du localisateur est garde, et rien
// ne change d un bit. Ce n est pas un repli : aucun bit n est devine, les records sont LUS ; les
// listes ainsi etendues sont comptees ([types.MovementStateStats.EventPacketsNewRecordStart]).

// debutDeLaListe rend le debut de la marche d un paquet a evenements : le premier record NEW de
// tete prouve ([debutParChaine], [debutParFermeture]), sinon le localisateur strict. `-1` : liste
// non localisee. Le booleen dit que la liste commence a un record NEW que le localisateur sautait.
func debutDeLaListe(pay []byte, w *World, cfg FrameConfig) (int, bool) {
	debut := marchLocateStrict(pay, w, cfg)
	if debut < 0 {
		return debutParFermeture(pay, candidatsDeTete(pay, len(pay)*8, w), w, cfg)
	}
	return debutParChaine(pay, debut, candidatsDeTete(pay, debut, w), w, cfg)
}

// candidatsDeTete rend, tries, les positions `p` (en-tete complet avant `fin`) qui portent un
// en-tete de record NEW dont le slot est dans la bande de l archetype annonce.
func candidatsDeTete(pay []byte, fin int, w *World) []int {
	var out []int
	for p := 0; p+woNewHeaderBits <= fin; p++ {
		slot, ti, ok := enteteNeufEn(pay, p)
		if ok && w.anticipee.SlotDeLArchetype(slot, ti) {
			out = append(out, p)
		}
	}
	return out
}

// enteteNeufEn lit, sans lecteur, l en-tete d un record NEW a `p` : prefixe `0` puis `01`, slot,
// generation, archetype borne par le registre des objets ([objectArchetypeCount]).
func enteteNeufEn(pay []byte, p int) (slot, ti uint32, ok bool) {
	if PeekBits(pay, p, 1) != 0 || PeekBits(pay, p+1, 2) != 1 {
		return 0, 0, false
	}
	ti = uint32(PeekBits(pay, p+woNewTypeBits+woNewSlotBits+woNewGenBits, woNewTIBits)) //nolint:gosec // 6 bits
	if ti >= objectArchetypeCount {
		return 0, 0, false
	}
	return uint32(PeekBits(pay, p+woNewTypeBits, woNewSlotBits)), ti, true //nolint:gosec // 13 bits
}

// motFacultatifDEnTete rend la largeur du mot facultatif qui precede le type de chaque record
// (`FUN_14080a9d4` : `HasExtraFields`), 0 sans lui.
func motFacultatifDEnTete(cfg FrameConfig) int {
	if cfg.HasExtraFields {
		return 32
	}
	return 0
}

// debutParChaine rend le premier candidat `p` < `debut` depuis lequel la chaine des records finit
// EXACTEMENT sur `debut` ; `false` : aucune chaine ne tient, le debut du localisateur est garde.
func debutParChaine(pay []byte, debut int, candidats []int, w *World, cfg FrameConfig) (int, bool) {
	extra := motFacultatifDEnTete(cfg)
	for _, p := range candidats {
		if p-extra < 0 || p >= debut {
			continue
		}
		if chaineJusqua(pay, p-extra, debut, extra, w, cfg) {
			return p - extra, true
		}
	}
	return debut, false
}

// plafondChaineDeTete borne la chaine d essai : ce n est pas une largeur de grammaire, c est la
// garde d une boucle hors ligne. Les chaines mesurees comptent de trois a quatorze records
// (`81c02726`, `8a485699`).
const plafondChaineDeTete = 64

// chaineJusqua marche les records qui partent de `pos`, chacun lu par son en-tete comme la boucle
// de records le lit ([pasDEssai]). Vrai si la chaine tombe EXACTEMENT sur `debut`.
func chaineJusqua(pay []byte, pos, debut, extra int, w *World, cfg FrameConfig) bool {
	essai := cfg
	essai.Obs = nil // traversees d ESSAI : rien n est publie, la marche relira les records
	for n := 0; n < plafondChaineDeTete && pos < debut; n++ {
		fin, ok := pasDEssai(pay, pos, extra, w, essai)
		if !ok || fin <= pos {
			return false
		}
		pos = fin
	}
	return pos == debut
}

// pasDEssai lit UN record a `pos` et rend la position qui le suit : un NEW traverse sans
// desynchronisation, un DEL (en-tete et mot de 32 bits), un delta qui se decode sur un slot que le
// monde connait. Un terminateur ou un record illisible refusent le pas.
func pasDEssai(pay []byte, pos, extra int, w *World, essai FrameConfig) (int, bool) {
	br := LecteurSur(pay)
	br.poserCadre(essai)
	br.SetBitPos(pos + extra)
	if br.Remaining() < woNewHeaderBits {
		return pos, false
	}
	switch readRecordType(br) {
	case recNew:
		readRecordID(br, essai.IDLowBits, essai.IDBase)
		tr := TraverseEntity(br, w.Reg, essai.NewDefaultStateBits)
		return tr.EndBit, tr.DesyncAt == -1
	case recDel:
		readRecordID(br, essai.IDLowBits, essai.IDBase)
		br.Skip(32)
		return br.BitPos(), true
	case recDelta:
		_, fin, ok := TryDeltaAt(pay, pos, w, essai)
		return fin, ok
	}
	return pos, false // terminateur : la liste finirait avant le debut localise
}

// debutParFermeture rend, pour un paquet dont le localisateur ne trouve pas la liste, le PREMIER
// candidat d ou la marche complete (monde restaure apres chaque essai) FERME la vue C au bit pres
// ([vueCFermee]) ; -1 sinon.
func debutParFermeture(pay []byte, candidats []int, w *World, cfg FrameConfig) (int, bool) {
	extra := motFacultatifDEnTete(cfg)
	for _, p := range candidats {
		if p-extra < 0 {
			continue
		}
		essai := cfg
		obs := NouvelleObservation()
		ferme := false
		obs.VueControleHook = func(l LectureVueC) { ferme = l.Fermee }
		essai.Obs = obs
		snap := w.Snapshot()
		DecodeFrameViewsCurseur(pay, w, essai, MovementStateViews, p-extra)
		w.Restore(snap)
		if ferme {
			return p - extra, true
		}
	}
	return -1, false
}
