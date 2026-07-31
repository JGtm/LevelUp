// tmp_keyframe_traverse — THROWAWAY (Front C) : tente de TRAVERSER un entity du
// keyframe type-2 en chaînant ses composants, conformément à la structure RE :
//
//	new-entity FUN_1408f1aa4 :
//	  R(6) typeIndex -> archetype = *(*(world+0x18)+8+typeIndex*8)
//	  default-state read  (vtable+0x60)  [largeur inconnue -> balayée]
//	  binding-default     (vtable+0x88)  [idem]
//	  gate FUN_1406cf008  -> R(1)
//	  iterator FUN_14076cb60 :
//	     mask = FUN_1406d7610(...) : R(1) ; 1->R(64) ; 0->R(3)=N puis N*R(6)
//	     pour i<count si (mask>>((i-skip)&0x3f))&1 -> deser = descriptor[i].vtable+0x28
//
// On ne possède qu'UN deser (générique 'obje' = DecodeEntityRecordQ). Cet outil
// mesure : (a) où se trouve le mask qui précède l'ancre obje @bit148, (b) combien
// de composants présents on peut enchaîner avec le SEUL deser obje avant desync.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math/bits"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

func load(path string) []byte {
	raw, _ := os.ReadFile(path)
	if len(raw) >= 2 && raw[0] == 0x78 {
		zr, e := zlib.NewReader(bytes.NewReader(raw))
		if e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

func extractPacket(data []byte, target uint16) []byte {
	off := 0
	for off+16 <= len(data) {
		typ := binary.LittleEndian.Uint16(data[off:])
		size := int(binary.LittleEndian.Uint32(data[off+4:]))
		if size <= 0 || off+16+size > len(data) {
			break
		}
		if typ == target {
			return data[off+16 : off+16+size]
		}
		off += 16 + size
	}
	return nil
}

// readComponentMask mirrors FUN_1406d7610 exactly : R(1) ; 1->R(64) ;
// 0->R(3)=N puis N*R(6) index, mask |= 1<<index.
func readComponentMask(br *filmdec.BitReader) (mask uint64, indices []int, raw64 bool) {
	if br.ReadBit() {
		return br.ReadBits(64), nil, true
	}
	n := int(br.ReadBits(3))
	for i := 0; i < n; i++ {
		idx := int(br.ReadBits(6))
		mask |= 1 << uint(idx)
		indices = append(indices, idx)
	}
	return mask, indices, false
}

func dumpBits(payload []byte, start, n int) string {
	if start < 0 {
		start = 0
	}
	br := filmdec.NewBitReader(payload)
	br.Skip(start)
	out := ""
	for i := 0; i < n; i++ {
		if i > 0 && i%8 == 0 {
			out += " "
		}
		if br.ReadBit() {
			out += "1"
		} else {
			out += "0"
		}
	}
	return out
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tmp_keyframe_traverse <chunk.bin>")
		os.Exit(1)
	}
	data := load(os.Args[1])
	payload := extractPacket(data, 2)
	if payload == nil {
		fmt.Println("pas de keyframe type-2")
		return
	}
	fmt.Printf("keyframe type-2 : %d octets\n\n", len(payload))

	anchorBit := 148 // start du record obje validé (variant 0x67abd42a)
	fmt.Printf("=== A. Recherche du mask qui PRÉCÈDE l'ancre obje @bit %d ===\n", anchorBit)
	// Le mask est lu juste avant le 1er deser présent. On balaie un start de mask
	// dans [anchorBit-90, anchorBit-1] et on cherche celui dont le décodage
	// FUN_1406d7610 atterrit EXACTEMENT sur anchorBit (afterMaskBit == anchorBit),
	// avec un mask plausible (bit 0 ou un bit bas présent en premier).
	maskHits := 0
	for ms := anchorBit - 90; ms < anchorBit; ms++ {
		if ms < 0 {
			continue
		}
		func() {
			defer func() { _ = recover() }()
			br := filmdec.NewBitReader(payload)
			br.Skip(ms)
			mask, idxs, raw64 := readComponentMask(br)
			if br.BitPos() != anchorBit {
				return
			}
			fmt.Printf("  maskStart=%d -> afterMask=%d  mask=0x%016x raw64=%v indices=%v popcount=%d\n",
				ms, br.BitPos(), mask, raw64, idxs, bits.OnesCount64(mask))
			maskHits++
		}()
	}
	if maskHits == 0 {
		fmt.Println("  (aucun mask FUN_1406d7610 n'atterrit pile sur l'ancre)")
	}

	fmt.Printf("\n  bits [%d..%d] (avant l'ancre) :\n  %s\n",
		anchorBit-96, anchorBit, dumpBits(payload, anchorBit-96, 96))

	// === B. Chaînage avec le SEUL deser obje, depuis l'ancre ===
	fmt.Printf("\n=== B. Chaînage obje-only depuis l'ancre @bit %d ===\n", anchorBit)
	chainObjeOnly(payload, anchorBit, 13, 20)

	// === C. Tentative entity-header complète depuis bit 0 ===
	fmt.Printf("\n=== C. Décode entity-header (typeIndex R6 + sweep default-state) @bit 0 ===\n")
	sweepEntityHeader(payload, 0, anchorBit)

	// === D. Composant index-10 (le desync) : que lui faut-il ? ===
	fmt.Printf("\n=== D. Composant index-10 @bit 268 (le desync) : essais de deser ===\n")
	probeComponent10(payload, 268)

	// === E. Enchaînement des entities : trouver le 2e entity ===
	fmt.Printf("\n=== E. Recherche du 2e entity (autre typeIndex+mask atterrissant sur un obje) ===\n")
	findNextEntities(payload)
}

// probeComponent10 dumpe le voisinage du composant qui suit l'obje (index 10) et
// essaie plusieurs grammaires candidates (signed-varwidth, petit champ, transform).
func probeComponent10(payload []byte, start int) {
	fmt.Printf("  bits [%d..%d] :\n", start, start+160)
	for off := start; off < start+160; off += 32 {
		fmt.Printf("    [%3d] %s\n", off, dumpBits(payload, off, 32))
	}
	// candidat 1 : signed varwidth chain (statborg-like)
	func() {
		defer func() { _ = recover() }()
		br := filmdec.NewBitReader(payload)
		br.Skip(start)
		v1 := br.ReadSignedVarWidth()
		v2 := br.ReadSignedVarWidth()
		fmt.Printf("  signedVarWidth x2 -> %d, %d (ends bit %d)\n", v1, v2, br.BitPos())
	}()
	// candidat 2 : statborg record (2 slots interleaved)
	func() {
		defer func() { _ = recover() }()
		br := filmdec.NewBitReader(payload)
		br.Skip(start)
		var tgt filmdec.StatborgTarget
		filmdec.ParseStatborgRecord(br, filmdec.StatborgBinding{IdxA: 0, IdxB: 1}, &tgt)
		fmt.Printf("  statborg -> flags[%d,%d] vals[%d,%d] (ends bit %d)\n",
			tgt.Flags[0], tgt.Flags[1], tgt.Values[0], tgt.Values[1], br.BitPos())
	}()
	// candidat 3 : quantized vec3 (transform/position) à différentes largeurs
	for _, w := range []uint{4, 6, 12, 14, 16, 20} {
		func() {
			defer func() { _ = recover() }()
			br := filmdec.NewBitReader(payload)
			br.Skip(start)
			v := br.ReadQuantizedVec3(w, filmdec.QuantRangeWorld100)
			fmt.Printf("  vec3(w=%2d) -> [%.2f %.2f %.2f] (ends bit %d)\n",
				w, v[0], v[1], v[2], br.BitPos())
		}()
	}
}

// findNextEntities balaie tout le payload pour trouver des en-têtes new-entity
// (typeIndex R6 + default-state + gate + mask raw64) dont le mask atterrit sur un
// record obje plausible. Mesure combien d'entities on peut localiser.
func findNextEntities(payload []byte) {
	total := len(payload) * 8
	found := 0
	limit := total - 200
	if limit > 60000 {
		limit = 60000 // borne le scan (perf)
	}
	for s := 0; s < limit && found < 12; s++ {
		func() {
			defer func() { _ = recover() }()
			br := filmdec.NewBitReader(payload)
			br.Skip(s)
			// raw64 mask directement (R1=1 -> R64) : cherche le motif mask->obje.
			if !br.ReadBit() {
				return
			}
			mask := br.ReadBits(64)
			pc := bits.OnesCount64(mask)
			if pc < 3 || pc > 40 {
				return
			}
			objeStart := br.BitPos()
			rec := filmdec.DecodeEntityRecordQ(br, 13)
			if rec.Valid && plausibleVariant(rec.VariantName) && len(rec.Bindings) > 0 {
				fmt.Printf("  maskBit=%d objeStart=%d mask=0x%016x pc=%d var=0x%08x\n",
					s, objeStart, mask, pc, rec.VariantName)
				found++
			}
		}()
	}
	if found == 0 {
		fmt.Println("  (aucun autre motif mask-raw64 -> obje trouvé dans la fenêtre)")
	}
}

// chainObjeOnly applique DecodeEntityRecordQ en boucle (le seul deser dispo) et
// mesure combien de composants restent "plausibles" avant desync.
func chainObjeOnly(payload []byte, start int, w uint, maxRec int) {
	br := filmdec.NewBitReader(payload)
	br.Skip(start)
	good := 0
	for r := 0; r < maxRec; r++ {
		s := br.BitPos()
		if s+40 > len(payload)*8 {
			break
		}
		ok := func() (survived bool) {
			defer func() {
				if recover() != nil {
					survived = false
				}
			}()
			rec := filmdec.DecodeEntityRecordQ(br, w)
			e := br.BitPos()
			pl := plausibleVariant(rec.VariantName)
			fmt.Printf("  [%2d] %d..%d (len=%3d) var=0x%08x stats=%d binds=%d valid=%v plaus=%v\n",
				r, s, e, e-s, rec.VariantName, len(rec.StatChans), len(rec.Bindings), rec.Valid, pl)
			if pl {
				good++
			}
			return e > s && pl
		}()
		if !ok {
			fmt.Printf("  -> DESYNC au record %d (le composant suivant n'est PAS un 'obje')\n", r)
			break
		}
	}
	fmt.Printf("  => %d composant(s) obje plausible(s) enchaîné(s)\n", good)
}

// sweepEntityHeader essaie de décoder l'en-tête new-entity et de tomber pile sur
// l'ancre obje. typeIndex=R(6) ; puis on balaye une largeur de default-state d
// (0..120 bits) ; puis gate=R(1) ; puis mask FUN_1406d7610 doit atterrir == target.
func sweepEntityHeader(payload []byte, start, target int) {
	found := 0
	for d := 0; d <= 128 && found < 8; d++ {
		func() {
			defer func() { _ = recover() }()
			br := filmdec.NewBitReader(payload)
			br.Skip(start)
			ti := uint32(br.ReadBits(6)) // typeIndex
			br.Skip(d)                   // default-state (largeur inconnue, balayée)
			_ = br.ReadBit()             // gate FUN_1406cf008
			mask, idxs, raw64 := readComponentMask(br)
			if br.BitPos() == target {
				fmt.Printf("  HIT : typeIndex=%d defaultStateBits=%d gate+mask -> afterMask=%d == ancre\n",
					ti, d, br.BitPos())
				fmt.Printf("        mask=0x%016x raw64=%v indices=%v popcount=%d\n",
					mask, raw64, idxs, bits.OnesCount64(mask))
				found++
			}
		}()
	}
	if found == 0 {
		fmt.Println("  (aucune combinaison typeIndex+defaultState+mask n'atterrit sur l'ancre depuis bit 0)")
	}
}

func plausibleVariant(v uint32) bool {
	if v == 0 || v == 0xFFFFFFFF {
		return false
	}
	if v&(v+1) == 0 {
		return false
	}
	if nv := ^v; nv&(nv+1) == 0 {
		return false
	}
	pc := bits.OnesCount32(v)
	return pc >= 6 && pc <= 26
}
