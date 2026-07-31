// tmp_offlinereplay — PROUVE l'équivalence entre un World décodé OFFLINE (mapping
// slot->typeIndex reconstruit depuis le KEYFRAME type-2 du film, ZÉRO Cheat Engine, via
// filmdec.WorldFromKeyframe) et le World-CE (oracle world_dump capturé au debugger), en
// rejouant les MÊMES deltas type-0 avec chacun des deux Worlds via filmdec.DecodeFrameRecords.
//
// Le World-offline == World-CE à 249/250 slots (le 250e n'est pas retrouvé par le walker
// keyframe) : l'équivalence des traces est donc attendue PAR CONSTRUCTION. Ce tool ne la
// "réussit" pas, il la MESURE :
//   - compare slot par slot le typeIndex offline vs CE (match / mismatch + liste) ;
//   - décode chaque paquet type-0 avec les deux Worlds et compare la SÉQUENCE de records
//     (signature slot|type|typeIndex|mask|comps|desync|endBit) + les positions i0 bipeds ;
//   - écrit trace_offline.txt, trace_ce.txt, diff.txt dans le scratchpad.
//
// AUCUN input Cheat Engine n'est requis pour le chemin OFFLINE : il ne lit que chunk_00
// (registre) + chunk_02 (keyframe) du film. Le world_dump CE n'est lu que pour la RÉFÉRENCE.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_offlinereplay [chunkIdx] [npkts]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	cache     = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
	ceDump    = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/crack/world_dump_000d5950.txt`
	scratch   = `c:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration/d6f109c3-2822-4e19-8fd4-cf65c50fec3d/scratchpad`
	bipedTI   = 35
	idLowBits = 11 // combo calibré (cf. tmp_worldreplay : extra=false idLowBits=11)
)

var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}

func inflate(p string) []byte {
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); (e2 == nil || e2 == io.ErrUnexpectedEOF) && len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

// framePayload : extrait le payload de la 1re frame de type `want` (framing chunk_02).
func framePayload(d []byte, want uint16) []byte {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == want {
			return d[off+16 : off+16+sz]
		}
		off += 16 + sz
	}
	return nil
}

type packet struct {
	typ     uint16
	size    int
	payload []byte
}

func listType0(d []byte) []packet {
	var out []packet
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, packet{typ, sz, d[off+16 : off+16+sz]})
		}
		off += 16 + sz
	}
	return out
}

type binding struct {
	full uint32 // id complet (gen<<30 | slot)
	ti   uint32
}

// parseCEBindings lit le world_dump CE (paires slot:typeIndex ; slots bruts). Le gen n'y
// est pas capturé -> on suppose gen=1 (namespace par défaut) comme parseWorld de tmp_worldreplay.
func parseCEBindings(path string) ([]binding, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []binding
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for _, tok := range strings.Fields(line) {
			parts := strings.SplitN(tok, ":", 2)
			if len(parts) != 2 {
				continue
			}
			slot, e1 := strconv.ParseUint(parts[0], 10, 32)
			ti, e2 := strconv.ParseUint(parts[1], 10, 32)
			if e1 != nil || e2 != nil {
				continue
			}
			out = append(out, binding{full: uint32(slot), ti: uint32(ti)})
		}
	}
	return out, nil
}

// offlineBindings walke le keyframe UNE fois et retourne les bindings (id complet, ti).
func offlineBindings(pay []byte) []binding {
	recs := filmdec.WalkKeyframeWorld(pay)
	out := make([]binding, 0, len(recs))
	for _, r := range recs {
		out = append(out, binding{full: uint32((r.Gen << 30) | r.Slot), ti: uint32(r.TI)})
	}
	return out
}

func buildWorld(reg *filmdec.Registry, bs []binding) *filmdec.World {
	w := filmdec.NewWorld(reg)
	for _, b := range bs {
		w.BindFull(b.full, b.ti)
	}
	return w
}

// tiBySlot réduit une liste de bindings en map slot->ti (dernier binding gagnant, comme BindFull).
func tiBySlot(bs []binding) map[uint32]uint32 {
	m := map[uint32]uint32{}
	for _, b := range bs {
		m[b.full&0x3fffffff] = b.ti
	}
	return m
}

// decodeRes capture le résultat de décodage d'un paquet.
type decodeRes struct {
	recs    []filmdec.FrameRecord
	endBit  int
	total   int
	cleanEr bool
	samples map[int][3]float32 // bitpos i0 -> position monde décodée
}

func decodePacket(payload []byte, reg *filmdec.Registry, bs []binding) decodeRes {
	w := buildWorld(reg, bs)
	samples := map[int][3]float32{}
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) {
		if _, dup := samples[s.BitPos]; !dup {
			samples[s.BitPos] = s.Vec
		}
	})
	br := filmdec.NewBitReader(payload)
	recs, err := filmdec.DecodeFrameRecords(br, w, filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idLowBits})
	filmdec.SetPositionCaptureHook(nil)
	return decodeRes{recs: recs, endBit: br.BitPos(), total: len(payload) * 8, cleanEr: err == nil, samples: samples}
}

// recSig : signature bit-exacte d'un record décodé (ce qui doit être identique offline vs CE
// pour un slot bindé identiquement). endBit = position bit après le record -> capte toute
// dérive de largeur.
func recSig(r filmdec.FrameRecord) string {
	return fmt.Sprintf("t%d slot=%d ti=%d dz=%d mask=0x%016x nc=%d end=%d",
		r.Type, r.Slot, r.TypeIndex, r.DesyncAt, r.Trace.Mask, len(r.Trace.Comps), r.Trace.EndBit)
}

// bipedPos : position i0 d'un record biped (via le StartBit de son 1er composant présent).
func bipedPos(r filmdec.FrameRecord, samples map[int][3]float32) ([3]float32, bool) {
	for _, c := range r.Trace.Comps {
		if v, ok := samples[c.StartBit]; ok {
			return v, true
		}
	}
	return [3]float32{}, false
}

func typeName(t int) string {
	switch t {
	case 1:
		return "new"
	case 2:
		return "del"
	case 3:
		return "delta"
	default:
		return "t" + strconv.Itoa(t)
	}
}

func writeTrace(path string, label string, pkts []packet, reses []decodeRes) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# trace %s — %d paquets type-0 — decode via filmdec.DecodeFrameRecords\n", label, len(pkts))
	for i, res := range reses {
		fmt.Fprintf(&b, "\n=== pkt#%d size=%d records=%d endBit=%d/%d cleanEnd=%v ===\n",
			i, pkts[i].size, len(res.recs), res.endBit, res.total, res.cleanEr)
		for j, r := range res.recs {
			line := fmt.Sprintf("  #%-4d %-5s slot=%-5d ti=%-3d dz=%-3d mask=0x%016x nc=%-2d end=%d",
				j, typeName(r.Type), r.Slot, r.TypeIndex, r.DesyncAt, r.Trace.Mask, len(r.Trace.Comps), r.Trace.EndBit)
			if bipedSlots[r.Slot] {
				if v, ok := bipedPos(r, res.samples); ok {
					line += fmt.Sprintf("  i0=(%.2f, %.2f, %.2f)", v[0], v[1], v[2])
				} else {
					line += "  i0=(none)"
				}
			}
			b.WriteString(line + "\n")
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func main() {
	chunkIdx := 3
	npkts := -1 // -1 = tous
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &chunkIdx)
	}
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[2], "%d", &npkts)
	}
	filmdec.SetRecordStateParam(2) // bipeds (comme au keyframe / tmp_worldreplay)

	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	pay02 := framePayload(inflate(cache+"/chunk_02.bin"), 2)
	if pay02 == nil {
		fmt.Println("chunk_02 : frame type-2 introuvable — abandon")
		return
	}

	// ---- Worlds ----
	offBs := offlineBindings(pay02)
	ceBs, err := parseCEBindings(ceDump)
	if err != nil {
		panic(err)
	}
	wOff := buildWorld(reg, offBs)
	wCE := buildWorld(reg, ceBs)
	fmt.Printf("=== WORLDS ===\n")
	fmt.Printf("OFFLINE (keyframe chunk_02, zéro CE) : records walkés=%d, slots bindés=%d\n", len(offBs), wOff.Bound())
	fmt.Printf("CE (world_dump crack)                : paires=%d, slots bindés=%d\n", len(ceBs), wCE.Bound())

	// ---- comparaison slot par slot ----
	offMap := tiBySlot(offBs)
	ceMap := tiBySlot(ceBs)
	var match, mismatch, ceOnly, offOnly int
	var mismatchList, ceOnlyList []string
	ceSlots := make([]int, 0, len(ceMap))
	for s := range ceMap {
		ceSlots = append(ceSlots, int(s))
	}
	sort.Ints(ceSlots)
	for _, s := range ceSlots {
		ce := ceMap[uint32(s)]
		off, ok := offMap[uint32(s)]
		switch {
		case !ok:
			ceOnly++
			ceOnlyList = append(ceOnlyList, fmt.Sprintf("slot=%d(ti=%d absent offline)", s, ce))
		case off == ce:
			match++
		default:
			mismatch++
			mismatchList = append(mismatchList, fmt.Sprintf("slot=%d off=%d ce=%d", s, off, ce))
		}
	}
	for s := range offMap {
		if _, ok := ceMap[s]; !ok {
			offOnly++
		}
	}
	fmt.Printf("\n=== COMPARAISON slot->typeIndex (référence = %d slots CE) ===\n", len(ceMap))
	fmt.Printf("MATCH (off==ce)=%d  MISMATCH(ti diffère)=%d  CE-only(absent offline)=%d  OFFLINE-only(extra)=%d\n",
		match, mismatch, ceOnly, offOnly)
	if len(mismatchList) > 0 {
		fmt.Printf("  slots MISMATCH: %s\n", strings.Join(mismatchList, " "))
	}
	if len(ceOnlyList) > 0 {
		fmt.Printf("  slots CE-only : %s\n", strings.Join(ceOnlyList, " "))
	}
	// bipeds
	nbOff, nbCE := 0, 0
	for s := uint32(512); s <= 519; s++ {
		if offMap[s] == bipedTI {
			nbOff++
		}
		if ceMap[s] == bipedTI {
			nbCE++
		}
	}
	fmt.Printf("  bipeds 512-519 ti=35 : offline=%d/8  CE=%d/8\n", nbOff, nbCE)

	// ---- décodage des deltas ----
	data := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, chunkIdx))
	t0 := listType0(data)
	if npkts < 0 || npkts > len(t0) {
		npkts = len(t0)
	}
	t0 = t0[:npkts]
	fmt.Printf("\n=== DECODE DELTAS chunk_%02d : %d paquets type-0 (idLowBits=%d) ===\n", chunkIdx, len(t0), idLowBits)

	offReses := make([]decodeRes, len(t0))
	ceReses := make([]decodeRes, len(t0))
	var totOff, totCE int
	for i := range t0 {
		offReses[i] = decodePacket(t0[i].payload, reg, offBs)
		ceReses[i] = decodePacket(t0[i].payload, reg, ceBs)
		totOff += len(offReses[i].recs)
		totCE += len(ceReses[i].recs)
	}
	fmt.Printf("records décodés OFFLINE=%d  CE=%d\n", totOff, totCE)

	// ---- DIFF ----
	var diff strings.Builder
	fmt.Fprintf(&diff, "# diff offline vs CE — %d paquets type-0 chunk_%02d\n", len(t0), chunkIdx)
	pktIdentical, pktDiffer := 0, 0
	identRecs, firstDivergences := 0, 0
	var bipedDiffLines []string
	for i := range t0 {
		of, ce := offReses[i], ceReses[i]
		n := len(of.recs)
		if len(ce.recs) < n {
			n = len(ce.recs)
		}
		div := -1
		for j := 0; j < n; j++ {
			if recSig(of.recs[j]) != recSig(ce.recs[j]) {
				div = j
				break
			}
			identRecs++
		}
		sameLen := len(of.recs) == len(ce.recs)
		if div == -1 && sameLen {
			pktIdentical++
			continue
		}
		pktDiffer++
		firstDivergences++
		fmt.Fprintf(&diff, "\npkt#%d DIFFÈRE : offRecords=%d ceRecords=%d endBit off=%d ce=%d\n",
			i, len(of.recs), len(ce.recs), of.endBit, ce.endBit)
		if div >= 0 {
			fmt.Fprintf(&diff, "  1re divergence @record#%d :\n    OFF: %s\n    CE : %s\n",
				div, recSig(of.recs[div]), recSig(ce.recs[div]))
		} else {
			fmt.Fprintf(&diff, "  records communs identiques (%d) mais longueurs différentes (queue divergente)\n", n)
		}
	}
	// comparaison ciblée bipeds : pour chaque paquet, séquence de positions i0 par slot biped
	for i := range t0 {
		of, ce := offReses[i], ceReses[i]
		obp := map[uint32][3]float32{}
		cbp := map[uint32][3]float32{}
		for _, r := range of.recs {
			if bipedSlots[r.Slot] && r.DesyncAt == -1 {
				if v, ok := bipedPos(r, of.samples); ok {
					obp[r.Slot] = v
				}
			}
		}
		for _, r := range ce.recs {
			if bipedSlots[r.Slot] && r.DesyncAt == -1 {
				if v, ok := bipedPos(r, ce.samples); ok {
					cbp[r.Slot] = v
				}
			}
		}
		for s := uint32(512); s <= 519; s++ {
			ov, ohas := obp[s]
			cv, chas := cbp[s]
			if ohas != chas || (ohas && ov != cv) {
				bipedDiffLines = append(bipedDiffLines,
					fmt.Sprintf("pkt#%d slot=%d off(%v %v) != ce(%v %v)", i, s, ohas, ov, chas, cv))
			}
		}
	}
	fmt.Fprintf(&diff, "\n=== RÉSUMÉ ===\n")
	fmt.Fprintf(&diff, "paquets identiques (record-stream complet)=%d/%d ; paquets divergents=%d\n", pktIdentical, len(t0), pktDiffer)
	fmt.Fprintf(&diff, "records identiques avant 1re divergence (cumul)=%d ; total off=%d ce=%d\n", identRecs, totOff, totCE)
	if len(bipedDiffLines) == 0 {
		fmt.Fprintf(&diff, "positions i0 bipeds (records clean) : IDENTIQUES offline==CE sur tous les paquets\n")
	} else {
		fmt.Fprintf(&diff, "positions i0 bipeds DIVERGENTES (%d) :\n  %s\n", len(bipedDiffLines), strings.Join(bipedDiffLines, "\n  "))
	}

	// ---- écriture des fichiers ----
	_ = os.MkdirAll(scratch, 0o755)
	fOff := scratch + "/trace_offline.txt"
	fCE := scratch + "/trace_ce.txt"
	fDiff := scratch + "/diff.txt"
	if err := writeTrace(fOff, "OFFLINE (World keyframe)", t0, offReses); err != nil {
		fmt.Printf("write offline: %v\n", err)
	}
	if err := writeTrace(fCE, "CE (World dump)", t0, ceReses); err != nil {
		fmt.Printf("write ce: %v\n", err)
	}
	if err := os.WriteFile(fDiff, []byte(diff.String()), 0o644); err != nil {
		fmt.Printf("write diff: %v\n", err)
	}

	fmt.Printf("\n=== ÉQUIVALENCE ===\n")
	fmt.Printf("paquets record-stream identiques = %d/%d ; divergents = %d\n", pktIdentical, len(t0), pktDiffer)
	fmt.Printf("positions i0 bipeds : %s\n", func() string {
		if len(bipedDiffLines) == 0 {
			return "IDENTIQUES offline==CE"
		}
		return fmt.Sprintf("%d divergences (voir diff.txt)", len(bipedDiffLines))
	}())
	fmt.Printf("fichiers : %s | %s | %s\n", fOff, fCE, fDiff)
}
