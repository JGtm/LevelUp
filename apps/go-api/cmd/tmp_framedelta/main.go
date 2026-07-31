// tmp_framedelta — THROWAWAY (P1/P2/P3) : caractérise la boucle de records type-0
// (FRAME) du flux gameplay et tente l'OPTION B (NEW records auto-suffisants, World vide).
//
// Couvre les objectifs LAYERED du HANDOFF :
//
//	[A] structurel : combien de paquets type-0 dans un chunk gameplay, quelles tailles.
//	[B] boucle de records : sweep FrameConfig (HasExtraFields × IDLowBits 0..16) avec un
//	    World VIDE (OPTION B). Pour chaque combo, on mesure records décodés avant
//	    désync/fin, histogramme des types, bit final vs taille payload, terminaison
//	    propre (type==end, bit ~ fin).
//	[C] arme : si un record NEW décode un biped (typeIndex=35) avec weapon-state-type-info,
//	    on compare handle (StartBit+1,32) et variant (StartBit+33,32) au catalogue
//	    analysis.WeaponIDToName (high-32 = uint32(id>>32)).
//	[D] verdict : OPTION B viable (NEW présents/décodables) ou keyframe-init requis.
//
// IMPORTANT (confirmé Ghidra) : la boucle démarre au BIT 0 du payload (le marker
// a0 7b 42 éventuel est INCLUS, pas de skip 24 bits).
//
// Usage : tmp_framedelta [chunk_index]   (défaut 3 = chunk_03, 1er gameplay)
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

const bipedTypeIndex = 35 // registry block 35 = BIPED/player archetype (weapon-state-type-info @i43..46)

func inflate(p string) []byte {
	raw, _ := os.ReadFile(p)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

type packet struct {
	typ     uint16
	off     int
	size    int
	ts      uint64
	payload []byte
}

// listPackets parcourt le framing [u16 type][2o][u32 size][u64 ts][payload].
func listPackets(d []byte) []packet {
	var out []packet
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		out = append(out, packet{typ, off, sz, ts, d[off+16 : off+16+sz]})
		off += 16 + sz
	}
	return out
}

// bitsAt lit n bits MSB-first à la position bp (pour relire handle/variant hors BitReader).
func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

func knownHigh32(v uint32) (string, bool) {
	for id, n := range analysis.WeaponIDToName {
		if uint32(id>>32) == v {
			return n, true
		}
	}
	return "", false
}

// runResult agrège ce qu'a produit un decode complet d'un paquet pour un FrameConfig.
type runResult struct {
	records   int
	typeHist  map[int]int // new/del/delta/(end implicite)
	cleanEnd  bool        // boucle terminée proprement (err==nil)
	endBit    int
	totalBits int
	desyncMsg string
	bipeds    int // records (new/delta) ayant décodé typeIndex==35
}

// decodeOne lance DecodeFrameRecords sur un payload avec un World et un cfg donnés,
// en démarrant la boucle à startBit (0 = bit0 marker inclus ; 24 = après a0 7b 42).
func decodeOneAt(payload []byte, reg *filmdec.Registry, cfg filmdec.FrameConfig, startBit int) runResult {
	w := filmdec.NewWorld(reg)
	br := filmdec.NewBitReader(payload)
	br.Skip(startBit)
	recs, err := filmdec.DecodeFrameRecords(br, w, cfg)
	res := runResult{
		records:   len(recs),
		typeHist:  map[int]int{},
		cleanEnd:  err == nil,
		endBit:    br.BitPos(),
		totalBits: len(payload) * 8,
	}
	if err != nil {
		res.desyncMsg = err.Error()
	}
	for _, r := range recs {
		res.typeHist[r.Type]++
		if r.TypeIndex == bipedTypeIndex {
			res.bipeds++
		}
	}
	return res
}

// decodeOne lance DecodeFrameRecords sur un payload avec un World et un cfg donnés.
func decodeOne(payload []byte, reg *filmdec.Registry, cfg filmdec.FrameConfig) runResult {
	w := filmdec.NewWorld(reg)
	br := filmdec.NewBitReader(payload)
	recs, err := filmdec.DecodeFrameRecords(br, w, cfg)
	res := runResult{
		records:   len(recs),
		typeHist:  map[int]int{},
		cleanEnd:  err == nil,
		endBit:    br.BitPos(),
		totalBits: len(payload) * 8,
	}
	if err != nil {
		res.desyncMsg = err.Error()
	}
	for _, r := range recs {
		res.typeHist[r.Type]++
		if r.TypeIndex == bipedTypeIndex {
			res.bipeds++
		}
	}
	return res
}

func tname(t int) string {
	switch t {
	case 1:
		return "new"
	case 2:
		return "del"
	case 3:
		return "delta"
	default:
		return fmt.Sprintf("t%d", t)
	}
}

func histStr(h map[int]int) string {
	keys := make([]int, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	s := ""
	for _, k := range keys {
		s += fmt.Sprintf("%s=%d ", tname(k), h[k])
	}
	if s == "" {
		return "(0)"
	}
	return s
}

func main() {
	chunkIdx := 3
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &chunkIdx)
	}

	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	chunkPath := fmt.Sprintf("%s/chunk_%02d.bin", cache, chunkIdx)
	data := inflate(chunkPath)
	pkts := listPackets(data)

	// ----- [A] STRUCTUREL -----
	byType := map[uint16]int{}
	var t0 []packet
	for _, p := range pkts {
		byType[p.typ]++
		if p.typ == 0 {
			t0 = append(t0, p)
		}
	}
	fmt.Printf("=== [A] STRUCTUREL chunk_%02d (%d octets inflatés) ===\n", chunkIdx, len(data))
	fmt.Printf("registre %d archétypes ; %d paquets total ; par type=%v\n", len(reg.Archetypes), len(pkts), byType)
	fmt.Printf("%d paquets type-0 (FRAME)\n", len(t0))
	if len(t0) == 0 {
		fmt.Println("aucun type-0 — chunk non gameplay ?")
		return
	}
	// tailles
	minS, maxS, sumS := t0[0].size, t0[0].size, 0
	for _, p := range t0 {
		if p.size < minS {
			minS = p.size
		}
		if p.size > maxS {
			maxS = p.size
		}
		sumS += p.size
	}
	fmt.Printf("tailles type-0 : min=%d max=%d moy=%d total=%d octets\n", minS, maxS, sumS/len(t0), sumS)
	fmt.Println("préfixes (6 1ers octets) des 6 premiers type-0 :")
	for i := 0; i < len(t0) && i < 6; i++ {
		pre := t0[i].payload
		if len(pre) > 6 {
			pre = pre[:6]
		}
		fmt.Printf("  #%d size=%-5d ts=%d  %x\n", i, t0[i].size, t0[i].ts, pre)
	}

	// ----- [B] SWEEP FrameConfig sur le 1er paquet type-0 (World VIDE = OPTION B) -----
	target := t0[0]
	fmt.Printf("\n=== [B] SWEEP FrameConfig — type-0 #0 (size=%d, %d bits) — World VIDE ===\n",
		target.size, target.size*8)
	fmt.Printf("%-6s %-9s %-8s %-26s %-9s %-10s %s\n",
		"extra", "idLowBits", "records", "types", "cleanEnd", "endBit/tot", "bipeds")

	type combo struct {
		cfg filmdec.FrameConfig
		res runResult
	}
	var best []combo
	for _, extra := range []bool{false, true} {
		for idLow := 0; idLow <= 16; idLow++ {
			cfg := filmdec.FrameConfig{HasExtraFields: extra, IDLowBits: idLow, IDBase: 0, NewDefaultStateBits: 0}
			res := decodeOne(target.payload, reg, cfg)
			fmt.Printf("%-6v %-9d %-8d %-26s %-9v %d/%d  %d\n",
				extra, idLow, res.records, histStr(res.typeHist), res.cleanEnd,
				res.endBit, res.totalBits, res.bipeds)
			best = append(best, combo{cfg, res})
		}
	}
	// meilleur combo = plus de records, bonus si cleanEnd
	sort.SliceStable(best, func(i, j int) bool {
		bi, bj := best[i].res, best[j].res
		score := func(r runResult) int {
			s := r.records
			if r.cleanEnd {
				s += 1000
			}
			return s
		}
		return score(bi) > score(bj)
	})
	bc := best[0]
	fmt.Printf("\n>>> meilleur combo : extra=%v idLowBits=%d -> %d records, cleanEnd=%v, endBit=%d/%d\n",
		bc.cfg.HasExtraFields, bc.cfg.IDLowBits, bc.res.records, bc.res.cleanEnd, bc.res.endBit, bc.res.totalBits)
	if bc.res.desyncMsg != "" {
		fmt.Printf("    désync : %s\n", bc.res.desyncMsg)
	}

	// ----- [B'] même sweep sur les 5 premiers type-0 (le 1er combo est-il stable ?) -----
	fmt.Printf("\n=== [B'] meilleur combo appliqué aux 5 premiers type-0 ===\n")
	for i := 0; i < len(t0) && i < 5; i++ {
		res := decodeOne(t0[i].payload, reg, bc.cfg)
		fmt.Printf("  #%d size=%-5d : records=%-4d types=%-24s cleanEnd=%-5v endBit=%d/%d bipeds=%d\n",
			i, t0[i].size, res.records, histStr(res.typeHist), res.cleanEnd, res.endBit, res.totalBits, res.bipeds)
	}

	// ----- [B-agg] sur TOUS les type-0 : 1er bit (delta vs autre) + meilleur combo -----
	bit0set, sameMarker := 0, 0
	for _, p := range t0 {
		if len(p.payload) == 0 {
			continue
		}
		if p.payload[0]&0x80 != 0 {
			bit0set++ // bit0=1 -> readRecordType lit DELTA d'emblée
		}
		if len(p.payload) >= 3 && p.payload[0] == 0xa0 && p.payload[1] == 0x7b && p.payload[2] == 0x42 {
			sameMarker++
		}
	}
	fmt.Printf("\n=== [B-agg] sur %d type-0 : %d commencent par bit0=1 (delta d'emblée) ; %d ont le marker a0 7b 42 ===\n",
		len(t0), bit0set, sameMarker)

	// ----- [B''] le marker a0 7b 42 (bit0=1) force un delta : tester un départ post-marker (bit24) -----
	fmt.Printf("\n=== [B''] départ post-marker (bit 24) — type-0 #0 — World VIDE ===\n")
	fmt.Println("(le payload commence par a0 7b 42 = bit0=1 -> readRecordType lit delta d'emblée ;")
	fmt.Println(" on teste l'hypothèse alternative : le marker 24 bits est un en-tête de paquet, pas un record)")
	for _, extra := range []bool{false, true} {
		for _, idLow := range []int{0, 11, 12, 13, 14} {
			cfg := filmdec.FrameConfig{HasExtraFields: extra, IDLowBits: idLow}
			res := decodeOneAt(target.payload, reg, cfg, 24)
			fmt.Printf("  bit24 extra=%-5v idLow=%-2d : records=%-3d types=%-20s cleanEnd=%-5v endBit=%d/%d\n",
				extra, idLow, res.records, histStr(res.typeHist), res.cleanEnd, res.endBit, res.totalBits)
		}
	}

	// ----- [C] OPTION B brute-force : chercher un NEW biped auto-suffisant dans le type-0 -----
	// On scanne des offsets de départ et on tente readRecordType==new -> TraverseEntity.
	// Indépendant du World : un NEW est auto-suffisant. On sweepe aussi NewDefaultStateBits.
	fmt.Printf("\n=== [C] OPTION B brute-force NEW biped (typeIndex=35) dans type-0 #0 ===\n")
	scanNewBipeds(target.payload, reg, bc.cfg)

	// ----- [C-agg] scan NEW biped sur TOUS les type-0 (optionnel : arg2 == "all", LENT) -----
	if len(os.Args) >= 3 && os.Args[2] == "all" {
		fmt.Printf("\n=== [C-agg] scan NEW biped sur les %d type-0 du chunk (LENT, dump des hits) ===\n", len(t0))
		totHit := 0
		for i := range t0 {
			totHit += scanNewBipedsQuiet(t0[i].payload, reg, bc.cfg, i)
		}
		fmt.Printf(">>> total hits arme connue via NEW biped sur tout le chunk : %d\n", totHit)
		fmt.Println("(crédibilité : un vrai NEW biped donne un hit STABLE sur une plage de d/rsp avec desyncAt>=i43 ;")
		fmt.Println(" des hits isolés et éparpillés = collisions high-32 32-bit sur des milliards d'essais brute-force.)")
	}

	// ----- [D] tentative keyframe-init (dépendance notée, non bloquante) -----
	fmt.Printf("\n=== [D] note keyframe-init ===\n")
	kf := inflate(cache + "/chunk_02.bin")
	kpkts := listPackets(kf)
	nKey := 0
	for _, p := range kpkts {
		if p.typ == 2 {
			nKey++
		}
	}
	fmt.Printf("chunk_02 : %d paquets, dont %d type-2 (keyframe). Init World depuis keyframe = piste si OPTION B échoue.\n", len(kpkts), nKey)
}

// scanNewBipeds brute-force des offsets de départ dans un payload type-0 : à chaque
// offset où le header décode un record NEW (type==1), on lance TraverseEntity en
// sweepant NewDefaultStateBits ; si typeIndex==35 et un weapon-state-type-info matche
// le catalogue, on reporte. C'est l'OPTION B : un NEW est auto-suffisant (pas de World).
func scanNewBipeds(payload []byte, reg *filmdec.Registry, cfg filmdec.FrameConfig) {
	totalBits := len(payload) * 8
	nNewHdr, nBiped, nReachWeapon, nWeaponHit := 0, 0, 0, 0
	deepest := -1 // plus grand index de composant atteint sur une trace biped
	// On limite le scan pour rester rapide : 1 bit de pas sur tout le paquet.
	for start := 0; start+64 < totalBits; start++ {
		// décode le header de record à `start` (sans HasExtraFields prefix : on ancre direct)
		hb := filmdec.NewBitReader(payload)
		hb.Skip(start)
		typ := readType(hb)
		if typ != 1 { // recNew
			continue
		}
		_ = readID(hb, cfg.IDLowBits, cfg.IDBase)
		bodyStart := hb.BitPos()
		nNewHdr++
		// le corps NEW commence par R(6) typeIndex : filtre rapide avant TraverseEntity
		if uint32(bitsAt(payload, bodyStart, 6)) != bipedTypeIndex {
			continue
		}
		for d := 0; d <= 140; d++ {
			for r := uint32(0); r <= 3; r++ {
				filmdec.SetRecordStateParam(r)
				tb := filmdec.NewBitReader(payload)
				tb.Skip(bodyStart)
				t := filmdec.TraverseEntity(tb, reg, d)
				if t.TypeIndex != bipedTypeIndex {
					continue
				}
				nBiped++
				for _, c := range t.Comps {
					if c.Index > deepest {
						deepest = c.Index
					}
					if c.Name != "weapon-state-type-info" {
						continue
					}
					nReachWeapon++
					handle := uint32(bitsAt(payload, c.StartBit+1, 32))
					variant := uint32(bitsAt(payload, c.StartBit+filmdec.VariantBitOffsetInWST, 32))
					if hn, ok := knownHigh32(handle); ok {
						nWeaponHit++
						fmt.Printf("  HIT NEW@bit%d d=%d rsp=%d : i%d HANDLE=%s (variant=0x%08x) desyncAt=i%d\n",
							start, d, r, c.Index, hn, variant, t.DesyncAt)
					}
					if vn, ok := knownHigh32(variant); ok {
						nWeaponHit++
						fmt.Printf("  HIT NEW@bit%d d=%d rsp=%d : i%d VARIANT=%s (handle=0x%08x) desyncAt=i%d\n",
							start, d, r, c.Index, vn, handle, t.DesyncAt)
					}
				}
				if nWeaponHit >= 8 {
					fmt.Printf("  (%d headers NEW, %d traces biped, weapon-comp atteint %d fois, %d hits arme — assez)\n",
						nNewHdr, nBiped, nReachWeapon, nWeaponHit)
					filmdec.SetRecordStateParam(0)
					return
				}
			}
		}
	}
	filmdec.SetRecordStateParam(0)
	fmt.Printf("  %d headers NEW (type==1) ; %d traces biped (typeIdx=35) ; composant le + profond atteint=i%d ; weapon-comp atteint %d fois ; %d hits arme connue\n",
		nNewHdr, nBiped, deepest, nReachWeapon, nWeaponHit)
	if nNewHdr == 0 {
		fmt.Println("  => AUCUN header NEW décodable à un seul offset : OPTION B brute-force seule insuffisante (deltas dominent).")
	} else if nReachWeapon == 0 {
		fmt.Println("  => des 'bipeds' décodent (R6==35 par hasard) mais AUCUN n'atteint le weapon-comp i43 : faux positifs, pas de vrai NEW biped aligné.")
	}
}

// scanNewBipedsQuiet : version silencieuse de scanNewBipeds renvoyant juste le nb de
// hits arme (pour l'agrégat tout-le-chunk). Sweep d réduit pour rester tractable.
func scanNewBipedsQuiet(payload []byte, reg *filmdec.Registry, cfg filmdec.FrameConfig, pktIdx int) int {
	totalBits := len(payload) * 8
	hits := 0
	for start := 0; start+64 < totalBits; start++ {
		hb := filmdec.NewBitReader(payload)
		hb.Skip(start)
		if readType(hb) != 1 {
			continue
		}
		_ = readID(hb, cfg.IDLowBits, cfg.IDBase)
		bodyStart := hb.BitPos()
		if uint32(bitsAt(payload, bodyStart, 6)) != bipedTypeIndex {
			continue
		}
		for d := 0; d <= 140; d++ {
			for r := uint32(0); r <= 3; r++ {
				filmdec.SetRecordStateParam(r)
				tb := filmdec.NewBitReader(payload)
				tb.Skip(bodyStart)
				t := filmdec.TraverseEntity(tb, reg, d)
				if t.TypeIndex != bipedTypeIndex {
					continue
				}
				for _, c := range t.Comps {
					if c.Name != "weapon-state-type-info" {
						continue
					}
					h := uint32(bitsAt(payload, c.StartBit+1, 32))
					v := uint32(bitsAt(payload, c.StartBit+filmdec.VariantBitOffsetInWST, 32))
					if hn, ok := knownHigh32(h); ok {
						hits++
						fmt.Printf("  hit pkt#%d start=%d d=%d rsp=%d i%d HANDLE=%s desyncAt=i%d\n", pktIdx, start, d, r, c.Index, hn, t.DesyncAt)
					}
					if vn, ok := knownHigh32(v); ok {
						hits++
						fmt.Printf("  hit pkt#%d start=%d d=%d rsp=%d i%d VARIANT=%s desyncAt=i%d\n", pktIdx, start, d, r, c.Index, vn, t.DesyncAt)
					}
				}
			}
		}
	}
	filmdec.SetRecordStateParam(0)
	return hits
}

// readType : prefix-code R(1); 1->delta(3); sinon R(2). Copie locale de readRecordType.
func readType(br *filmdec.BitReader) int {
	if br.ReadBit() {
		return 3
	}
	return int(br.ReadBits(2))
}

// readID : low=R(idLowBits)+base ; tag=R(2)<<30. Copie locale de readRecordID.
func readID(br *filmdec.BitReader, idLowBits int, idBase uint32) uint32 {
	var low uint32
	if idLowBits > 0 {
		low = uint32(br.ReadBits(uint(idLowBits)))
	}
	low += idBase
	tag := uint32(br.ReadBits(2))
	return (tag << 30) | (low & 0x3fffffff)
}
