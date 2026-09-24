package grammar

// keyframe_world_preuve.go — L'ÉLECTION DE REPLI NE CONTREDIT PLUS UN RECORD PROUVÉ (lot D-fix
// de la campagne « retours rejeu », 2026-09-24).
//
// # CE QUE L'ÉLECTION PERDAIT, MESURÉ
//
// Quand aucun voisin immédiat ne suit et qu'aucun en-tête exact de bipède ne précède, la marche
// d'ancres ÉLIT le candidat de SLOT le plus BAS de sa fenêtre ([kfCand.betterThan], repli nommé
// `repli_ancre_d_image_cle_par_election`). Une fausse ancre de slot bas, prise dans le corps d'un
// vrai record, bat alors les vrais records qui la précèdent : ils sont perdus d'un coup, et la
// marche repart de la fausse ancre. Mesuré sur les sept bobines par build (instrument du lot,
// 2026-09-24) — l'image-clé d'AVANT-MATCH que la marche glissante du lot M3.1 atteint désormais :
//
//	bcb6d393 morceau 1, paquet 0   après le slot 122 (`ti 45`, un record de 125 270 bits), la
//	                               fenêtre porte les vrais records 1280..1298 puis, DANS le corps
//	                               du record 1298, la fausse ancre `0x400000C0 00000001` (slot
//	                               192, `ti 1`) : élue, elle efface 1280..1298 — dont le joueur
//	                               géré de l'index 0 (slot 1297, `ti 9`) ;
//	fb1a1a72 morceau 1, paquets 0 et 1   la même fausse ancre, les mêmes records perdus ;
//	bcb6d393 morceau 1, paquet 3   après le slot 1531, la fausse ancre slot 1536 `ti 0`
//	                               (`n1` 524 288) efface 1537..1601 (équipements, armes au sol).
//
// Le lot M2 lit alors le joueur géré perdu comme ARRIVÉ à l'image-clé suivante : un joueur absent
// 16,3 s au départ de `000d5950` (pré-intégration de la vague D, 2026-09-24).
//
// # LA RÈGLE, ET ELLE VIENT DE LA GRAMMAIRE
//
// La table est à slots CROISSANTS (l'écrivain `FUN_142e2bfd0` enchaîne les entités dans l'ordre
// de leurs slots) : deux vrais records n'ont jamais l'ordre de leurs bits et l'ordre de leurs
// slots inversés. Un candidat que la grammaire du film PROUVE être un record interdit donc tout
// candidat qui contredit cet ordre avec lui — placé AVANT lui avec un slot supérieur ou égal, ou
// APRÈS lui avec un slot inférieur ou égal. L'élection écarte l'élu contredit par un record
// prouvé, et élit parmi les candidats restants, par la même règle.
//
// UN RECORD EST PROUVÉ quand sa marche d'ÉTAT COMPLET ([WalkKeyframeFullState], le cadre de
// l'écrivain) lit un état par défaut (`n1 > 0`), traverse une boucle de composants sans
// désynchroniser, et s'arrête EXACTEMENT sur un en-tête valide de slot supérieur : c'est la
// fermeture, « la seule preuve qu'on ait que toutes ses largeurs sont justes »
// (`keyframe_closure.go`). AUCUN SEUIL : ni distance, ni longueur de chaîne, ni valeur de `n1`.
//
// LE CONTENU EST EXIGÉ, ET C'EST MESURÉ : un record VIDE (`n1 <= 0` ou `n2 <= 0`, 172 bits
// d'en-tête et de mots de taille) ferme aussi LU UN BIT À CÔTÉ — la lecture décalée d'une chaîne
// de records vides est une chaîne de records vides. Sur les sept bobines, les seules « preuves »
// de candidats sûrement faux (entre deux voisins de slots consécutifs) étaient ces chaînes
// décalées (`ti 41`, `ti 21`, `ti 25`) : 61 sur 163 579 candidats faux sans l'exigence, 0 avec.
//
// # CE QUI NE CHANGE PAS
//
// Le voisin immédiat, le saut de largeur et le recalage sur l'en-tête exact d'un bipède décident
// comme avant ; l'élection reste le repli nommé et compté (chaque élu, réfuté ou non, compte une
// élection). Sans preuve (bobine sans registre, instruments de [WalkKeyframeWorld]), l'élection
// est celle d'avant. Une réfutation se compte ([KeyframeWalkStats.Refutations]).
//
// # POURQUOI LE PROFIL INVARIANT DU FILM
//
// La preuve se joue au profil PAR DÉFAUT, complété de ce que le FILM déclare (le contrôle de
// corruption par composant, le découpage MPP de sa version de format) — jamais des largeurs de la
// carte ni d'une calibration. Toutes les marches d'un même payload, quelle que soit l'étape qui
// marche (cuisson, kill-feed, instrument), rendent donc les MÊMES records : une largeur de carte
// manquante ne peut que faire échouer une fermeture (moins de preuves), jamais en fabriquer une.

import "levelup/go-api/internal/games/halo_infinite/film/internal/profile"

// PreuveDImageCle est la grammaire qu'une marche d'ancres consulte pour PROUVER un candidat : le
// registre du film et son contexte de lecture invariant. Le pointeur nil ne prouve rien.
type PreuveDImageCle struct {
	reg *Registry
	ctx ContexteDeLecture
}

// PreuveDImageCle rend la preuve de ce film, ou nil quand son registre est illisible (bobine
// partielle, fixture sans `chunk_00`) — la marche élit alors comme avant.
func (c *FilmContext) PreuveDImageCle() *PreuveDImageCle {
	if c == nil {
		return nil
	}
	reg, err := c.Registry()
	if err != nil || reg == nil {
		return nil
	}
	bal := ProfilDeBalayageParDefaut()
	bal.Grammaire.ControleDeCorruption = c.controleDeCorruptionDuFilm()
	if format, ok := FilmFormatVersion(c.Film()); ok {
		if w, ok := profile.MPPPourFormat(format); ok && w.Valid() {
			bal.MPP = w
		}
	}
	return &PreuveDImageCle{reg: reg, ctx: ContexteDeLecture{Profil: bal}}
}

// MarcheDImageCle rend la marche d'ancres de ce film : elle refuse une élection contredite par un
// record prouvé. Sans registre, c'est la marche sans preuve.
func (c *FilmContext) MarcheDImageCle() MarcheDImageCle {
	return MarcheDImageCle{preuve: c.PreuveDImageCle()}
}

// prouve dit si le candidat de `bit` et de `slot` est un record PROUVÉ par la grammaire du film
// (cf. l'en-tête du fichier).
func (p *PreuveDImageCle) prouve(pay []byte, bit, slot int) bool {
	if p == nil {
		return false
	}
	total := len(pay) * 8
	cadre := p.ctx.Profil.Cadre
	if bit < 0 || bit+cadre.EnTeteBits+cadre.MotDeTailleBits > total {
		return false
	}
	//nolint:gosec // mot de taille de 32 bits, compare SIGNE comme chez l'ecrivain
	if n1 := int32(kfReadBits(pay, bit+cadre.EnTeteBits, cadre.MotDeTailleBits)); n1 <= 0 {
		return false
	}
	tr := WalkKeyframeFullState(pay, bit, p.reg, p.ctx)
	if tr.DesyncAt >= 0 || len(tr.Comps) == 0 {
		return false
	}
	h, ok := readKeyframeHeader(pay, tr.EndBit, total)
	return ok && h.Slot > slot
}

// kfCandidat est un candidat d'une fenêtre de recherche, avec son archétype.
type kfCandidat struct {
	gen, slot, bit, ti int
}

// contredit dit si deux candidats ont l'ordre de leurs bits et celui de leurs slots inversés : ils
// ne peuvent pas être deux records de la table.
func (a kfCandidat) contredit(b kfCandidat) bool {
	return (a.bit < b.bit && a.slot >= b.slot) || (a.bit > b.bit && a.slot <= b.slot)
}

// elire rend le bit de l'élu de la fenêtre courante : le meilleur candidat ([kfCand.betterThan])
// qu'aucun record prouvé ne contredit, et le nombre d'élus refusés. -1 quand tous le sont — la
// recherche retombe alors sur l'élection sans preuve.
func (r *kfRecherche) elire(prevSlot int) (at, refutes int) {
	exclus := make([]bool, len(r.cands))
	for {
		i := r.meilleurCandidat(prevSlot, exclus)
		if i < 0 {
			return -1, refutes
		}
		if !r.contreditParUnRecordProuve(i) {
			return r.cands[i].bit, refutes
		}
		exclus[i] = true
		refutes++
	}
}

// meilleurCandidat rend l'indice du meilleur candidat non exclu selon [kfCand.betterThan], -1 s'il
// n'en reste aucun.
func (r *kfRecherche) meilleurCandidat(prevSlot int, exclus []bool) int {
	best, bi := kfCand{}, -1
	for i, c := range r.cands {
		if exclus[i] {
			continue
		}
		k := kfCand{gen: c.gen, slot: c.slot, bit: c.bit}
		if c.slot == prevSlot+1 {
			k.consecutive = 1
		}
		if bi < 0 || k.betterThan(best) {
			best, bi = k, i
		}
	}
	return bi
}

// contreditParUnRecordProuve dit si un candidat PROUVÉ de la fenêtre contredit le candidat `i`.
// Les preuves se calculent à la demande et se mémorisent par position pour tout le payload.
func (r *kfRecherche) contreditParUnRecordProuve(i int) bool {
	e := r.cands[i]
	for j, a := range r.cands {
		if j == i || !a.contredit(e) {
			continue
		}
		if r.estProuve(a) {
			return true
		}
	}
	return false
}

// estProuve mémorise [PreuveDImageCle.prouve] par position.
func (r *kfRecherche) estProuve(a kfCandidat) bool {
	if v, ok := r.prouves[a.bit]; ok {
		return v
	}
	if r.prouves == nil {
		r.prouves = map[int]bool{}
	}
	v := r.preuve.prouve(r.buf, a.bit, a.slot)
	r.prouves[a.bit] = v
	return v
}
