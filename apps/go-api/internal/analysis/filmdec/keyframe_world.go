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
)

// KeyframeRec est un record de la table keyframe reconstruit offline.
type KeyframeRec struct {
	Slot, TI, Gen, Bit int
}

// kfReadBits lit n bits big-endian à partir de la position bit pos (0-safe hors borne).
func kfReadBits(buf []byte, pos, n int) uint64 {
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
func kfValidAnchor(buf []byte, q, prevSlot, total int) (slot, ti, gen, field26 int, ok bool) {
	if q < 0 || q+64 > total {
		return
	}
	id := kfReadBits(buf, q, 32)
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
	ti = int(kfReadBits(buf, q+58, 6)) // extraction durcie (== mot 32-bit quand field26==0)
	field26 = int(kfReadBits(buf, q+32, 26))
	ok = true
	return
}

// kfBetterCand : priorité de sélection d'ancre — consécutif (slot==prev+1) d'abord, puis gen
// BAS (gen==1 réel bat un faux gen 2/3 au même slot ; un vrai gen≥2 mid-match reste choisi
// s'il est seul candidat du slot), puis slot bas, puis bit bas.
func kfBetterCand(cons, g, s, q, bc, bg, bs, bb int) bool {
	if cons != bc {
		return cons > bc
	}
	if g != bg {
		return g < bg
	}
	if s != bs {
		return s < bs
	}
	return q < bb
}

// kfScanNext : prochaine ancre après `from`. Fast-path : slot==prev+1 ET gen==1 (cas dominant
// du keyframe de spawn, non ambigu) -> retour immédiat. Sinon score complet kfBetterCand
// (gère les faux ancres gen 2/3 de la zone d'état + les vrais gen≥2 mid-match).
func kfScanNext(buf []byte, from, prevSlot, total, maxWin int) (slot, ti, gen, at int) {
	at = -1
	bc, bg, bs, bb := -1, 1<<30, 1<<30, 1<<30
	end := from + maxWin
	if end > total {
		end = total
	}
	sentStreak := 0
	for q := from; q+64 <= end; q++ {
		if kfReadBits(buf, q, 32) == kfSent {
			if sentStreak++; sentStreak >= 2048 {
				break
			}
			continue
		}
		sentStreak = 0
		s, t, g, _, ok := kfValidAnchor(buf, q, prevSlot, total)
		if !ok {
			continue
		}
		if s == prevSlot+1 && g == 1 {
			return s, t, g, q // consécutif gen-1 : non ambigu
		}
		cons := 0
		if s == prevSlot+1 {
			cons = 1
		}
		if at < 0 || kfBetterCand(cons, g, s, q, bc, bg, bs, bb) {
			slot, ti, gen, at = s, t, g, q
			bc, bg, bs, bb = cons, g, s, q
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
	if _, _, _, _, ok := kfValidAnchor(buf, pos, prev, total); !ok {
		_, _, _, pos = kfScanNext(buf, pos, prev, total, maxWin)
	}
	for pos >= 0 {
		slot, ti, gen, _, ok := kfValidAnchor(buf, pos, prev, total)
		if !ok {
			break
		}
		startState := pos + 64
		var nat int
		if w, has := width[ti]; has {
			// fast-path saut-de-largeur : n'accepte QUE gen==1 (non ambigu). Les faux ancres
			// gen 2/3 de la zone d'état peuvent tomber pile sur startState+w ; on ne saute pas
			// dessus. Un vrai record gen≥2 (mid-match) est résolu par kfScanNext (fallback).
			if _, _, jg, _, vok := kfValidAnchor(buf, startState+w, slot, total); vok && jg == 1 {
				nat = startState + w
			} else {
				_, _, _, nat = kfScanNext(buf, startState, slot, total, maxWin)
			}
		} else {
			_, _, _, nat = kfScanNext(buf, startState, slot, total, maxWin)
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
