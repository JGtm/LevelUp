package filmdec

// projectile_owner_helpers_test.go — collecteurs et types de l'instrument
// projectile_owner_research_test.go (scinde pour le seuil de 500 lignes). Voir l'en-tete de
// ce fichier-la pour le raisonnement, la verite terrain et les mesures.

import (
	"sort"
	"testing"
)

// projOwnerRead : une lecture d'i10 (object-parent-state) sur un record de PROJECTILE (ti=41),
// estampillee du slot du record courant (accumSlot) et de l'instant du paquet.
type projOwnerRead struct {
	slot      uint32
	ts        uint64
	attached  bool   // branche "attache" (transform relatif) vs branche libre (en vol)
	hasFreeID bool   // la porte du bloc 1408f0ac4 s'est ouverte et a transmis un id
	freeID    uint64 // index R(13) de cet id (espace de handle dom1, comme les bipedes)
	word16    uint32 // R(16) inconditionnel de la branche attachee (candidat handle alternatif)
}

// projOwnerKill : un dead-state de bipede MORT, oracle de verite terrain. killer est l'index
// de participant absolu du TUEUR (EnumB = +0x08), le champ-lien deja prouve a 97,6 %.
type projOwnerKill struct {
	victimSlot uint32
	killer     int32
	ts         uint64
}

// projOwnerColl : le fruit d'une passe unique sur le film.
type projOwnerColl struct {
	reads       []projOwnerRead
	kills       []projOwnerKill
	dmgExplo    int // damage_aftermath (0xC0 t0) a ref1 presente NON-bipede (candidat explosif)
	dmgToTi41   int // ... dont ref1 (base 512) resout a un slot ti=41 vivant a l'instant du degat
	ti41Records int // records ti=41 vus cleanement decodes (pour situer le rendement)
}

// projOwnerCollect fait UNE passe : bind des keyframes, decodage de trame (qui declenche la
// sonde i10 et capture les dead-states), scan des damage_aftermath pour le pont M3.
func projOwnerCollect(t *testing.T, dir string, reg *Registry, n int) projOwnerColl {
	t.Helper()
	cfg := DefaultFrameConfig()
	var out projOwnerColl
	var curTS uint64
	prev := objectParentStateHook
	SetObjectParentStateHook(func(st ObjectParentState) {
		if st.TypeIndex != ProjectileTypeIndex {
			return
		}
		out.ti41Records++
		out.reads = append(out.reads, projOwnerRead{
			slot: accumSlot, ts: curTS, attached: st.Attached,
			hasFreeID: st.HasFreeID, freeID: st.FreeID, word16: st.Word16,
		})
	})
	defer SetObjectParentStateHook(prev)

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
			pay := pk.Payload(data)
			curTS = pk.TimestampUS
			switch {
			case pay[0]&0x40 == 0:
				br := NewBitReader(pay)
				recs, _ := DecodeFrameRecords(br, w, cfg)
				projOwnerHarvestKills(recs, pk.TimestampUS, &out)
			case pk.Size >= 2 && pay[0] == 0xC0:
				projOwnerHarvestDamage(pay, w, &out)
			}
		}
	}
	return out
}

// projOwnerHarvestKills recolte les dead-states de bipede mort (oracle tueur) des records
// cleanement decodes de la trame.
func projOwnerHarvestKills(recs []FrameRecord, ts uint64, out *projOwnerColl) {
	for i := range recs {
		rec := &recs[i]
		if rec.TypeIndex != BipedTypeIndex || rec.Trace.Dead == nil {
			continue
		}
		d := rec.Trace.Dead
		if !d.Mort || d.EnumB < 0 {
			continue
		}
		out.kills = append(out.kills, projOwnerKill{victimSlot: rec.Slot, killer: d.EnumB, ts: ts})
	}
}

// projOwnerHarvestDamage compte les damage_aftermath a responsable non-bipede (candidat
// explosif) et, parmi eux, ceux dont ref1 resout a un slot ti=41 vivant (pont M3).
func projOwnerHarvestDamage(pay []byte, w *World, out *projOwnerColl) {
	br := NewBitReader(pay)
	br.Skip(2)
	if br.ReadBits(7) != 0 {
		return
	}
	lot1RefDom1(br) // ref0 victime
	i1, ok1 := lot1RefDom1(br)
	if !ok1 {
		return
	}
	bip, _ := projOwnerResolve(w, lot1chReferenceBase, int(i1))
	if bip {
		return // tir direct (ref1 bipede)
	}
	out.dmgExplo++
	if ti, ok := projOwnerArchetype(w, lot1chReferenceBase, int(i1)); ok && ti == ProjectileTypeIndex {
		out.dmgToTi41++
	}
}

// projOwnerResolve rend (bipede?, ti) pour (base+idx) contre le monde w.
func projOwnerResolve(w *World, base, idx int) (bool, int) {
	ti, ok := projOwnerArchetype(w, base, idx)
	if !ok {
		return false, -1
	}
	return ti == BipedTypeIndex, ti
}

// projOwnerArchetype rend l'archetype de (base+idx) s'il est lie.
func projOwnerArchetype(w *World, base, idx int) (int, bool) {
	if idx < 0 {
		return 0, false
	}
	slot := base + idx
	if slot < 0 || slot >= 8192 {
		return 0, false
	}
	ti, ok := w.ArchetypeForSlot(uint32(slot))
	return int(ti), ok
}

// projOwnerBestBaseBiped balaye le jeu de bases et rend celle qui resout le PLUS d'ids en
// bipede, avec le compte. Sert a tester si freeID (ou word16) pointe un tireur.
func projOwnerBestBaseBiped(w *World, ids []uint64) (bestBase, bestHits int) {
	bestBase, bestHits = lot1chBases[0], -1
	for _, b := range lot1chBases {
		hits := 0
		for _, id := range ids {
			if bip, _ := projOwnerResolve(w, b, int(id)); bip {
				hits++
			}
		}
		if hits > bestHits {
			bestBase, bestHits = b, hits
		}
	}
	return bestBase, bestHits
}

// projOwnerDistinct rend le nombre de valeurs distinctes d'une tranche d'ids.
func projOwnerDistinct(ids []uint64) int {
	seen := map[uint64]bool{}
	for _, id := range ids {
		seen[id] = true
	}
	return len(seen)
}

// projOwnerTopVals rend les k valeurs les plus frequentes, "v:n", pour lire la cardinalite.
func projOwnerTopVals(ids []uint64, k int) string {
	m := map[uint64]int{}
	for _, id := range ids {
		m[id]++
	}
	type kv struct {
		v uint64
		n int
	}
	rows := make([]kv, 0, len(m))
	for v, n := range m {
		rows = append(rows, kv{v, n})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].n > rows[j].n })
	out := ""
	for i, r := range rows {
		if i >= k {
			break
		}
		out += " " + utoa(r.v) + ":" + itoa(r.n)
	}
	if out == "" {
		return " (aucune)"
	}
	return out
}

// projOwnerPerLifeStable mesure la stabilite de freeID par vie de projectile (slot) : une vraie
// reference d'owner est CONSTANTE sur la vie. Rend (vies avec >=2 lectures a id, vies stables).
func projOwnerPerLifeStable(reads []projOwnerRead) (lives, stable int) {
	bySlot := map[uint32]map[uint64]int{}
	for _, r := range reads {
		if !r.hasFreeID {
			continue
		}
		if bySlot[r.slot] == nil {
			bySlot[r.slot] = map[uint64]int{}
		}
		bySlot[r.slot][r.freeID]++
	}
	for _, vals := range bySlot {
		total := 0
		for _, n := range vals {
			total += n
		}
		if total < 2 {
			continue
		}
		lives++
		if len(vals) == 1 {
			stable++
		}
	}
	return lives, stable
}

// projOwnerMaskCensus balaye TOUS les records ti=41 par le scanner dedie (matchWorldObjectRecord,
// robuste : il ne depend PAS d'une trame propre, contrairement a la sonde i10). Il ancre sur i0
// present (position, quasi tout record de projectile la porte) et tally les autres index de
// composant du masque. Rend (records ti=41 vus, histogramme index->compte). ROBUSTE et
// independant du rendement de DecodeFrameRecords : c'est le denominateur honnete de "i10 apparait-il".
func projOwnerMaskCensus(t *testing.T, dir string, n int) (int, map[int]int) {
	t.Helper()
	band := worldObjectSlotBand(dir, n, ProjectileTypeIndex)
	hist := map[int]int{}
	total := 0
	if len(band) == 0 {
		return 0, hist
	}
	posBits := projPosBits()
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits + posBits)
			for p := 0; p <= limit; p++ {
				rec, ok := matchWorldObjectRecord(pay, p, band)
				if !ok || rec.Idx[0] != 0 {
					continue
				}
				if _, ok := decodeWorldObjectPos(pay, rec.After, &projOwnerCensusRange); !ok {
					continue
				}
				total++
				for _, i := range rec.Idx {
					hist[i]++
				}
				p += posBits
			}
		}
	}
	return total, hist
}

// projOwnerCensusRange : bornes arbitraires pour decodeWorldObjectPos (on ne se sert que du
// booleen de validite du quantum, pas de la position en clair). Toute plage non degeneree suffit.
var projOwnerCensusRange = Vec3Range{{Min: 0, Max: 1}, {Min: 0, Max: 1}, {Min: 0, Max: 1}}

// projOwnerMaskLine formate l'histogramme des index de composant, tries.
func projOwnerMaskLine(hist map[int]int, total int) string {
	idxs := make([]int, 0, len(hist))
	for i := range hist {
		idxs = append(idxs, i)
	}
	sort.Ints(idxs)
	out := ""
	for _, i := range idxs {
		out += " i" + itoa(i) + "=" + itoa(hist[i])
	}
	return out
}

// utoa : uint64 -> decimal (evite fmt pour rester homogene avec itoa du paquet de tests).
func utoa(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
