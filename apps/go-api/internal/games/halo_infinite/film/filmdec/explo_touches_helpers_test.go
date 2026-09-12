package filmdec

// explo_touches_helpers_test.go — collecteurs, types et petits utilitaires de l'instrument
// explo_touches_research_test.go (scinde pour le seuil de 500 lignes). Voir l'en-tete de ce
// fichier-la pour le raisonnement, les seuils et les 6 mesures.

import (
	"sort"
	"testing"

	"levelup/go-api/internal/analysis"
)

// exploShot : un tir 0xD2 t36 LONG horodate (attaquant ref0 dom1 + WeaponID + lourd?).
type exploShot struct {
	ts    uint64
	att   uint64
	wid   uint64
	heavy bool
	has   bool
}

// exploDmg : un damage_aftermath 0xC0 t0 avec resolution de ses deux references d'en-tete.
type exploDmg struct {
	ts         uint64
	idx0, idx1 int // ref0 victime, ref1 responsable ; -1 absente
	src        uint64
	hasSrc     bool
	neg        bool
	mag        float64
	vicBiped   bool // (exploBase+idx0) lie a un bipede
	respBiped  bool // (exploBase+idx1) lie a un bipede
	respTI     int  // archetype de (exploBase+idx1) si lie, -1 sinon
}

// exploClass classe un degat selon sa responsabilite ref1.
type exploClass int

const (
	exploRespAbsent   exploClass = iota // ref1 absente
	exploRespBiped                      // ref1 -> bipede (tir direct, appariable au tireur)
	exploRespNonBiped                   // ref1 presente mais non-bipede (candidate projectile)
)

func exploClassify(e exploDmg) exploClass {
	if e.idx1 < 0 {
		return exploRespAbsent
	}
	if e.respBiped {
		return exploRespBiped
	}
	return exploRespNonBiped
}

// exploResolve rend (bipede?, ti) pour (base+idx) contre le monde ; ti=-1 si non lie.
func exploResolve(w *World, base, idx int) (bool, int) {
	if idx < 0 {
		return false, -1
	}
	slot := base + idx
	if slot < 0 || slot >= 8192 {
		return false, -1
	}
	ti, ok := w.ArchetypeForSlot(uint32(slot))
	if !ok {
		return false, -1
	}
	return ti == BipedTypeIndex, int(ti)
}

// exploWeaponName : nom d'arme par WeaponID (table statique) ou l'hexa.
func exploWeaponName(wid uint64) string {
	if nm, ok := analysis.WeaponIDToName[wid]; ok {
		return nm
	}
	return attribWeaponName(wid)
}

// exploCollectShots decode les tirs longs 0xD2 t36 : attaquant (lot1RefDom1) + WeaponID
// (decodeFireEvent) + drapeau lourd (lot1IsHeavy sur le nom d'arme).
func exploCollectShots(t *testing.T, dir string, n int) []exploShot {
	t.Helper()
	var out []exploShot
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 4 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xD2 {
				continue
			}
			br := NewBitReader(pay)
			br.Skip(2)
			if br.ReadBits(7) != 36 {
				continue
			}
			att, okA := lot1RefDom1(br)
			fe, okF := decodeFireEvent(pay)
			if okA && okF {
				out = append(out, exploShot{ts: pk.TimestampUS, att: att, wid: fe.WeaponID,
					heavy: lot1IsHeavy(exploWeaponName(fe.WeaponID)), has: true})
			}
		}
	}
	return out
}

// exploCollectDamage rejoue keyframe + tick-frames (comme sondeScanDamage) et rend les
// damage_aftermath avec resolution de ref0/ref1 contre le monde de fin de chunk (base 512).
func exploCollectDamage(t *testing.T, dir string, reg *Registry, n int) []exploDmg {
	t.Helper()
	cfg := DefaultFrameConfig()
	var evs []exploDmg
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		pks := WalkPackets(data)
		w := NewWorld(reg)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			if pay := pk.Payload(data); pay[0]&0x40 == 0 {
				br := NewBitReader(pay)
				_, _ = DecodeFrameRecords(br, w, cfg)
			}
		}
		evs = exploDamageInChunk(pks, data, w, evs)
	}
	return evs
}

// exploDamageInChunk decode les 0xC0 t0 d'un chunk contre le monde w (fin de chunk).
func exploDamageInChunk(pks []FilmPacket, data []byte, w *World, evs []exploDmg) []exploDmg {
	for _, pk := range pks {
		if pk.Type != PacketTypeDelta || pk.Size < 2 {
			continue
		}
		pay := pk.Payload(data)
		if pay[0] != 0xC0 {
			continue
		}
		br := NewBitReader(pay)
		br.Skip(2)
		if br.ReadBits(7) != 0 {
			continue
		}
		e := exploDmg{ts: pk.TimestampUS, idx0: -1, idx1: -1}
		if i0, ok := lot1RefDom1(br); ok {
			e.idx0 = int(i0)
		}
		if i1, ok := lot1RefDom1(br); ok {
			e.idx1 = int(i1)
		}
		lot1RefDom(br, 7)
		r := lot1DecodeDamageAftermath(br)
		e.src, e.hasSrc, e.neg, e.mag = r.sourceID, r.hasSource, r.negatif, r.dmgClear
		e.vicBiped, _ = exploResolve(w, exploBase, e.idx0)
		e.respBiped, e.respTI = exploResolve(w, exploBase, e.idx1)
		evs = append(evs, e)
	}
	return evs
}

// exploPrecede rend vrai si un ts trie tombe dans [T-W, T] (tir qui PRECEDE l'impact).
func exploPrecede(sorted []uint64, T, W uint64) bool {
	lo := uint64(0)
	if T > W {
		lo = T - W
	}
	i := sort.Search(len(sorted), func(i int) bool { return sorted[i] >= lo })
	return i < len(sorted) && sorted[i] <= T
}

// exploAvg : moyenne protegee.
func exploAvg(sum float64, n int) float64 {
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

// exploTopTI : les k archetypes les plus frequents, "ti=<v>:<n>".
func exploTopTI(m map[int]int, k int) string {
	type kv struct {
		ti, n int
	}
	var rows []kv
	for ti, n := range m {
		rows = append(rows, kv{ti, n})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].n > rows[j].n })
	out := ""
	for i, r := range rows {
		if i >= k {
			break
		}
		out += " ti=" + itoa(r.ti) + ":" + itoa(r.n)
	}
	if out == "" {
		return "(aucun)"
	}
	return out
}
