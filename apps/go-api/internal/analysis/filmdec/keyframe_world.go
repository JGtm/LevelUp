package filmdec

// keyframe_world.go — PONT réutilisable : reconstruction OFFLINE du World
// (mapping slot -> typeIndex) depuis la table keyframe type-2 d'un film, SANS Cheat
// Engine. Le walker porté ci-dessous est la version DURCIE validée par cmd/tmp_kfworldpos
// (249/250 entités + 8/8 bipeds slots 512-519 contre l'oracle CE), copiée fidèlement.
//
// En-tête RÉEL d'un record keyframe type-2 (RE FUN_141f86704) :
//   [id:32][field:26][ti:6] = 64 bits.
//   ti  = readBits(q+58, 6)  (== mot 32-bit à q+32 quand field26==0, cas du spawn)
//   gen = id>>30 ∈ {1,2,3}   (respawns mid-match ; gen==0 = handle null rejeté)
//   slot = id & 0x3FFFFFFF
//
// WorldFromKeyframe binde chaque record via World.BindFull(fullID, ti). Aucun input CE :
// seul le payload de frame type-2 du chunk (pay) est lu.

const (
	kfSent     = 0xFFFFFFFF // id sentinelle
	kfTableCap = 8192       // borne haute de slot
	// objectArchetypeCount (.exe DAT_144e61d88 COUNT=0x32) : ti valide < 50. Borne SÉMANTIQUE
	// (pas len(reg.Archetypes)=118, qui est la borne trop large « morte ») car ti tient sur 6 bits
	// et le cap objet réel = 50. Constante Halo Infinite : un autre titre exigerait de re-sourcer
	// ce COUNT depuis son .exe (walker keyframe = RE Halo-Infinite-spécifique pour l'instant).
	kfArchMax = 50
	// keyframeRecordTIBit est la position, EN BITS DEPUIS LE DEBUT DU RECORD, du champ `ti`
	// de 6 bits : c'est le point exact ou TraverseEntity doit prendre la main (en-tete
	// [id:32][field:26][ti:6] documente en tete de fichier). Definition unique du paquet —
	// les marcheurs d'image-cle la reutilisent au lieu de re-ecrire le litteral 58.
	keyframeRecordTIBit = 58
)

// KeyframeRec est un record de la table keyframe reconstruit offline.
type KeyframeRec struct {
	Slot, TI, Gen, Bit int
}

// kfReadBits lit n bits big-endian à partir de la position bit pos (0-safe hors borne).
//
// Lecture par mot (cf. bits_word.go) sur le domaine ou elle coincide avec la boucle
// d'origine : `pos >= 0` et `0 <= n <= 64`. C'est la primitive la plus chaude de toute la
// cuisson (58 a 61 % du CPU au profil du 2026-09-02) : `kfScanNext` l'appelle pour CHAQUE
// position de bit du payload d'image-cle.
func kfReadBits(buf []byte, pos, n int) uint64 {
	if pos >= 0 && n >= 0 && n <= 64 {
		return wordBitsAt(buf, pos, uint(n))
	}
	return kfReadBitsLoop(buf, pos, n)
}

// kfReadBitsLoop est la lecture bit a bit d'origine, gardee pour les positions NEGATIVES
// (l'indexation y panique, comme avant) et les largeurs > 64 (seuls les 64 derniers bits
// lus sont rendus). Aucun appelant de production n'y passe : elle est la pour que la
// reecriture par mot ne CHANGE rien, pas pour servir.
func kfReadBitsLoop(buf []byte, pos, n int) uint64 {
	var r uint64
	for i := 0; i < n; i++ {
		p := pos + i
		var bit uint64
		if idx := p >> 3; idx < len(buf) {
			bit = uint64(buf[idx]>>(7-uint(p&7))) & 1
		}
		r = r<<1 | bit
	}
	return r
}

// kfBitAt lit le bit unique à la position p (0 hors borne).
func kfBitAt(buf []byte, p int) uint64 {
	if idx := p >> 3; idx < len(buf) {
		return uint64(buf[idx]>>(7-uint(p&7))) & 1
	}
	return 0
}

// kfValidAnchor : en-tête de record valide ? gen∈{1,2,3}, prev<slot<cap. FILTRE FORT
// anti-faux-positif : le mot 32-bit à q+32 doit être < archMax (vrai pour TOUT record au
// spawn, car field26==0 -> ce mot == ti < 50). ti/field26 EXTRAITS de façon durcie.
// field26 n'est PAS rendu : il ne sert qu'à expliquer pourquoi le filtre fort tient
// (field26==0 au spawn -> le mot 32-bit vaut ti), et aucun appelant ne l'a jamais lu.
func kfValidAnchor(buf []byte, q, prevSlot, total int) (slot, ti, gen int, ok bool) {
	// La garde est ICI AUSSI parce qu'elle protege la LECTURE qui suit : `kfReadBits` a une
	// position negative panique (convention preservee, cf. bits_word.go). [kfAnchorFromID] la
	// rejoue pour son autre appelant, qui lui a deja lu l'identifiant.
	if q < 0 || q+64 > total {
		return
	}
	return kfAnchorFromID(buf, q, kfReadBits(buf, q, 32), prevSlot, total)
}

// kfAnchorFromID est [kfValidAnchor] quand l'appelant a DEJA lu les 32 bits d'identifiant a
// la position q. `kfScanNext` les lit pour son test de sentinelle et les relisait ensuite :
// deux lectures de 32 bits par position de bit du payload, soit la moitie du temps de la
// primitive la plus chaude de la cuisson. Meme logique, memes gardes, meme resultat — `id`
// est une fonction pure de (buf, q).
func kfAnchorFromID(buf []byte, q int, id uint64, prevSlot, total int) (slot, ti, gen int, ok bool) {
	if q < 0 || q+64 > total {
		return
	}
	if id == kfSent {
		return
	}
	gen = int(id >> 30)
	if gen == 0 { // handle null / zone de données
		return
	}
	slot = int(id & 0x3FFFFFFF)
	if slot <= prevSlot || slot >= kfTableCap {
		return
	}
	if kfReadBits(buf, q+32, 32) >= kfArchMax { // filtre fort : field26==0 & ti<50
		return
	}
	ti = int(kfReadBits(buf, q+keyframeRecordTIBit, 6)) // extraction durcie (== mot 32-bit quand field26==0)
	ok = true
	return
}

// kfCand est une ancre candidate en cours d'évaluation. Les quatre champs sont exactement les
// quatre clés de tri de betterThan, dans l'ordre — les passer groupés évite la file de huit
// entiers nus où deux clés s'inversent sans que rien ne le signale.
type kfCand struct {
	consecutive int // 1 si slot == prevSlot+1
	gen         int
	slot        int
	bit         int // position en bits dans le payload
}

// betterThan : priorité de sélection d'ancre — consécutif (slot==prev+1) d'abord, puis gen
// BAS (gen==1 réel bat un faux gen 2/3 au même slot ; un vrai gen≥2 mid-match reste choisi
// s'il est seul candidat du slot), puis slot bas, puis bit bas.
func (c kfCand) betterThan(best kfCand) bool {
	if c.consecutive != best.consecutive {
		return c.consecutive > best.consecutive
	}
	if c.gen != best.gen {
		return c.gen < best.gen
	}
	if c.slot != best.slot {
		return c.slot < best.slot
	}
	return c.bit < best.bit
}

// kfScanNext : POSITION EN BITS de la prochaine ancre après `from` (-1 si aucune).
// Fast-path : slot==prev+1 ET gen==1 (cas dominant du keyframe de spawn, non ambigu) ->
// retour immédiat. Sinon score complet kfBetterCand (gère les faux ancres gen 2/3 de la
// zone d'état + les vrais gen≥2 mid-match).
//
// Seule la position est rendue : slot/ti/gen de l'ancre retenue se relisent en rejouant
// kfValidAnchor à cette position — c'est exactement ce que l'appelant fait déjà.
func kfScanNext(buf []byte, from, prevSlot, total, maxWin int) (at int) {
	at = -1
	best := kfCand{consecutive: -1, gen: 1 << 30, slot: 1 << 30, bit: 1 << 30}
	end := from + maxWin
	if end > total {
		end = total
	}
	sentStreak := 0
	for q := from; q+64 <= end; q++ {
		id := kfReadBits(buf, q, 32)
		if id == kfSent {
			if sentStreak++; sentStreak >= 2048 {
				break
			}
			continue
		}
		sentStreak = 0
		s, _, g, ok := kfAnchorFromID(buf, q, id, prevSlot, total)
		if !ok {
			continue
		}
		if s == prevSlot+1 && g == 1 {
			return q // consécutif gen-1 : non ambigu
		}
		cand := kfCand{gen: g, slot: s, bit: q}
		if s == prevSlot+1 {
			cand.consecutive = 1
		}
		if at < 0 || cand.betterThan(best) {
			at, best = q, cand
		}
	}
	return
}

// WalkKeyframeWorld porte EXACTEMENT la logique durcie de walkOffline (cmd/tmp_kfworldpos) :
// il parcourt la table keyframe type-2 (payload de frame `buf`) et retourne les records
// slot->typeIndex reconstruits. Sortie bit-à-bit équivalente à tmp_kfworldpos (249/250).
func WalkKeyframeWorld(buf []byte) []KeyframeRec {
	total := len(buf) * 8
	const maxWin = 120000
	width := map[int]int{}
	seen := map[int]int{}
	var out []KeyframeRec
	pos := 1 // préfixe 1 bit
	prev := -1
	if _, _, _, ok := kfValidAnchor(buf, pos, prev, total); !ok {
		pos = kfScanNext(buf, pos, prev, total, maxWin)
	}
	for pos >= 0 {
		slot, ti, gen, ok := kfValidAnchor(buf, pos, prev, total)
		if !ok {
			break
		}
		startState := pos + 64
		var nat int
		if w, has := width[ti]; has {
			// fast-path saut-de-largeur : n'accepte QUE gen==1 (non ambigu). Les faux ancres
			// gen 2/3 de la zone d'état peuvent tomber pile sur startState+w ; on ne saute pas
			// dessus. Un vrai record gen≥2 (mid-match) est résolu par kfScanNext (fallback).
			if _, _, jg, vok := kfValidAnchor(buf, startState+w, slot, total); vok && jg == 1 {
				nat = startState + w
			} else {
				nat = kfScanNext(buf, startState, slot, total, maxWin)
			}
		} else {
			nat = kfScanNext(buf, startState, slot, total, maxWin)
		}
		out = append(out, KeyframeRec{Slot: slot, TI: ti, Gen: gen, Bit: pos})
		prev = slot
		if nat < 0 {
			break
		}
		w := nat - startState
		if prevW, s := seen[ti]; s {
			if prevW == w {
				width[ti] = w
			} else {
				delete(width, ti)
			}
		} else {
			seen[ti] = w
		}
		pos = nat
	}
	return out
}

// WorldFromKeyframe construit un World OFFLINE en bindant chaque record keyframe reconstruit.
// pay = payload de frame type-2 du chunk (extrait comme cmd/tmp_kfworldpos le fait). Aucun
// input Cheat Engine : la source de vérité est uniquement le film.
func WorldFromKeyframe(reg *Registry, pay []byte) *World {
	w := NewWorld(reg)
	for _, r := range WalkKeyframeWorld(pay) {
		w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
	}
	return w
}
