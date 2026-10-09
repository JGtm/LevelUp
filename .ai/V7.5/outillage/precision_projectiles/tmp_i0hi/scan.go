package main

import (
	"levelup/go-api/internal/analysis/filmdec"
)

// scan.go — LA GRAMMAIRE DE RECORD, reprise MOT POUR MOT de `filmdec/projectiles.go`
// (celle qui porte les 70 lancers de grenade sur 70), a une difference pres : la branche
// haute n'est pas rejetee, elle est etiquetee.

// scanFilm balaye un film et rend ses records i0 de projectile. Un chunk a la fois : le
// corpus est une bombe RAM documentee, on ne remonte jamais un film entier en memoire.
func scanFilm(dir string, wr *filmdec.Vec3Range) ([]sample, int, int) {
	n := filmdec.CountFilmChunks(dir)
	if n == 0 {
		return nil, 0, 0
	}
	band := slotBand(dir, n)
	if len(band) == 0 {
		return nil, 0, 0
	}
	var out []sample
	var tot, hi int
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta {
				continue
			}
			for _, s := range scanRecords(p.Payload(chunk), band, wr) {
				s.timestampUS = p.TimestampUS
				tot++
				if s.hi {
					hi++
				}
				out = append(out, s)
			}
		}
	}
	return out, tot, hi
}

// slotBand reprend `worldObjectSlotBand` : combler la plage de l'archetype, puis retirer tout
// slot vu porter un AUTRE archetype.
func slotBand(dir string, n int) map[uint32]bool {
	seen := map[uint32]bool{}
	others := map[uint32]bool{}
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if pk.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			for _, r := range filmdec.WalkKeyframeWorld(pk.Payload(data)) {
				if r.TI == projectileTI {
					seen[uint32(r.Slot)] = true
				} else {
					others[uint32(r.Slot)] = true
				}
			}
		}
	}
	band := fillBand(seen)
	for s := range others {
		delete(band, s)
	}
	return band
}

func fillBand(s map[uint32]bool) map[uint32]bool {
	if len(s) == 0 {
		return nil
	}
	lo, hi := ^uint32(0), uint32(0)
	for k := range s {
		if k < lo {
			lo = k
		}
		if k > hi {
			hi = k
		}
	}
	out := make(map[uint32]bool, hi-lo+1)
	for k := lo; k <= hi; k++ {
		out[k] = true
	}
	return out
}

// scanRecords reprend MOT POUR MOT la grammaire de `scanProjectileRecords`, a une difference
// pres : la branche haute n'est pas rejetee, elle est etiquetee.
func scanRecords(pay []byte, band map[uint32]bool, wr *filmdec.Vec3Range) []sample {
	var out []sample
	limit := len(pay)*8 - (21 + 6 + hiPosBits)
	for p := 0; p <= limit; p++ {
		if filmdec.PeekBits(pay, p, 1) != 1 { // prefixe de record DELTA
			continue
		}
		slot := uint32(filmdec.PeekBits(pay, p+1, 13))
		if !band[slot] {
			continue
		}
		gen := uint32(filmdec.PeekBits(pay, p+14, 2))
		if filmdec.PeekBits(pay, p+16, 2) != 0 { // porte de masque = 0 -> branche eparse
			continue
		}
		mc := int(filmdec.PeekBits(pay, p+18, 3))
		if mc < 1 || mc > 7 {
			continue
		}
		idx, ok := ascending(pay, p+21, mc)
		if !ok || idx[0] != 0 { // i0 doit etre present : c'est la position
			continue
		}
		at := p + 21 + 6*mc
		s := sample{slot: slot, gen: gen, bits: filmdec.PeekBits(pay, at, 64)}
		if filmdec.PeekBits(pay, at, 1) == 1 {
			s.hi = true
			out = append(out, s)
			p += hiPosBits
			continue
		}
		v, ok := decodeLow(pay, at, wr)
		if !ok {
			continue
		}
		s.pos = v
		for a := 0; a < 3; a++ {
			s.norm[a] = (v[a] - wr[a].Min) / (wr[a].Max - wr[a].Min)
		}
		out = append(out, s)
		p += lowPosBits
	}
	return out
}

func ascending(pay []byte, at, mc int) ([]int, bool) {
	idx := make([]int, mc)
	prev := -1
	for k := 0; k < mc; k++ {
		v := int(filmdec.PeekBits(pay, at+6*k, 6))
		if v <= prev {
			return nil, false
		}
		idx[k], prev = v, v
	}
	return idx, true
}

// decodeLow reprend `decodeWorldObjectPos` : la branche VALIDEE, qui sert d'oracle.
func decodeLow(pay []byte, at int, wr *filmdec.Vec3Range) ([3]float32, bool) {
	var v [3]float32
	if filmdec.PeekBits(pay, at, 3) != 0 {
		return v, false
	}
	off := at + 3
	for a := 0; a < 3; a++ {
		w := filmdec.WorldObjectPrecision.AxisW[a]
		q := filmdec.PeekBits(pay, off, int(w))
		if q == 0 || q == (uint64(1)<<w)-1 {
			return v, false
		}
		lo, hi := wr[a].Min, wr[a].Max
		v[a] = lo + (float32(q)+0.5)*(hi-lo)/float32(uint64(1)<<w)
		off += int(w)
	}
	return v, true
}

// reportLives publie la structure des vies : combien de vies melangent les deux branches, et
