// tmp_traverse — THROWAWAY (P5) : valide la traversée d'entité (filmdec.TraverseEntity)
// sur le keyframe type-2. Étape 1 : la 1ère entité (typeIndex=40, default-state=76 bits
// mesuré par Front C) doit retrouver l'obje à i9 (variant 0x67abd42a) puis désync à i10.
// Étape 2 : balaie les default-state bits + cherche des entités biped (typeIndex=35)
// dont le held-weapon (i43) est atteint et matche WeaponIDToName.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

func inflate(path string) []byte {
	raw, _ := os.ReadFile(path)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

func extractType2(data []byte) []byte {
	off := 0
	for off+16 <= len(data) {
		typ := binary.LittleEndian.Uint16(data[off:])
		size := int(binary.LittleEndian.Uint32(data[off+4:]))
		if size <= 0 || off+16+size > len(data) {
			break
		}
		if typ == 2 {
			return data[off+16 : off+16+size]
		}
		off += 16 + size
	}
	return nil
}

func knownWeapon(v uint32) (string, bool) {
	if n, ok := analysis.WeaponIDToName[uint64(v)<<32|0x42c9679f]; ok {
		return n, true
	}
	for id, n := range analysis.WeaponIDToName {
		if uint32(id>>32) == v {
			return n, true
		}
	}
	return "", false
}

func main() {
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("registre : %d archétypes\n", len(reg.Archetypes))

	payload := extractType2(inflate(cache + "/chunk_02.bin"))
	fmt.Printf("keyframe type-2 : %d octets\n\n", len(payload))

	// --- Étape 1 : 1ère entité, default-state=76 (réf Front C : obje@i9 var=0x67abd42a) ---
	fmt.Println("=== entité @bit0, default-state=76 ===")
	br := filmdec.NewBitReader(payload)
	t := filmdec.TraverseEntity(br, reg, 76)
	printTrace(t)

	// --- Étape 1-bis : contenu COMPLET de l'obje i9 (bit 148) : owner/pi ? ---
	fmt.Println("\n=== contenu complet de l'obje i9 (multiplayer-properties) @bit148 ===")
	b9 := filmdec.NewBitReader(payload)
	b9.Skip(148)
	rec := filmdec.DecodeEntityRecordQ(b9, 13)
	fmt.Printf("  variant=0x%08x B1D=%d Field02=%d HeaderA=0x%02x Field0C=0x%x ID5=0x%x LocalID=0x%x\n",
		rec.VariantName, rec.B1D, rec.Field02, rec.HeaderA, rec.Field0C, rec.ID5, rec.LocalID)
	fmt.Printf("  statChans=%d bindings=%d posValid=%v pos=%v\n", len(rec.StatChans), len(rec.Bindings), rec.PosValid, rec.Position)
	for i, s := range rec.StatChans {
		fmt.Printf("    stat[%d] lo2=%d flag=%v raw=0x%08x (%d)\n", i, s.Lo2, s.Flag, s.Raw, s.Raw)
	}
	for i, bd := range rec.Bindings {
		fmt.Printf("    bind[%d] hdr4=0x%x present=%v sub=%d idx=%d word16=0x%04x(%d) vec=%v\n",
			i, bd.Hdr4, bd.Present, bd.SubVal, bd.Index, bd.Word16, bd.Word16, bd.Vec)
	}

	// --- Étape 2 : brute-force biped (typeIndex=35), ancré sur obje i9 plausible ---
	fmt.Println("\n=== scan biped (typeIndex=35) ancré sur obje i9 plausible ===")
	totalBits := len(payload) * 8
	scanMax := totalBits - 200
	if scanMax > 700000 {
		scanMax = 700000 // ~87KB, suffisant pour trouver des bipeds
	}
	type cand struct {
		off, d   int
		mask     uint64
		variant  uint32
		maxIdx   int
		desyncAt int
	}
	var cands []cand
	for off := 0; off < scanMax; off++ {
		// pré-filtre R6==35 (cheap)
		b0 := filmdec.NewBitReader(payload)
		b0.Skip(off)
		if uint32(b0.ReadBits(6)) != 35 {
			continue
		}
		for d := 60; d <= 140; d += 2 {
			b := filmdec.NewBitReader(payload)
			b.Skip(off)
			t := filmdec.TraverseEntity(b, reg, d)
			if t.TypeIndex != 35 {
				continue
			}
			// anchor : obje i9 présent + variant plausible (haute entropie)
			for _, c := range t.Comps {
				if c.Index == 9 && plausible(c.Variant) {
					maxIdx := 0
					for _, cc := range t.Comps {
						if cc.Index > maxIdx {
							maxIdx = cc.Index
						}
					}
					cands = append(cands, cand{off, d, t.Mask, c.Variant, maxIdx, t.DesyncAt})
				}
			}
		}
		if len(cands) >= 12 {
			break
		}
	}
	fmt.Printf("  %d candidat(s) biped (obje i9 plausible) :\n", len(cands))
	for i, c := range cands {
		if i >= 12 {
			break
		}
		fmt.Printf("    off=%d d=%d mask=0x%016x objeVar=0x%08x maxIdxAtteint=%d desyncAt=%d (popcount=%d)\n",
			c.off, c.d, c.mask, c.variant, c.maxIdx, c.desyncAt, popcount(c.mask))
	}
}

func plausible(v uint32) bool {
	if v == 0 || v == 0xFFFFFFFF || v&(v+1) == 0 {
		return false
	}
	if nv := ^v; nv&(nv+1) == 0 {
		return false
	}
	pc := 0
	for x := v; x != 0; x &= x - 1 {
		pc++
	}
	return pc >= 6 && pc <= 26
}

func popcount(v uint64) int {
	pc := 0
	for ; v != 0; v &= v - 1 {
		pc++
	}
	return pc
}

func printTrace(t filmdec.EntityTrace) {
	fmt.Printf("typeIndex=%d defaultBits=%d gate=%v mask=0x%016x desyncAt=%d endBit=%d\n",
		t.TypeIndex, t.DefaultBits, t.Gate, t.Mask, t.DesyncAt, t.EndBit)
	for _, c := range t.Comps {
		extra := ""
		if c.Variant != 0xFFFFFFFF {
			if n, ok := knownWeapon(c.Variant); ok {
				extra = "  <<< WEAPON " + n
			} else {
				extra = fmt.Sprintf("  var=0x%08x", c.Variant)
			}
		}
		fmt.Printf("   i%-2d %-46s ported=%v @bit%d%s\n", c.Index, c.Name, c.Ported, c.StartBit, extra)
	}
	if t.HeldWeapon != 0xFFFFFFFF {
		if n, ok := knownWeapon(t.HeldWeapon); ok {
			fmt.Printf("   HELD-WEAPON = %s (0x%08x)\n", n, t.HeldWeapon)
		} else {
			fmt.Printf("   HELD-WEAPON var=0x%08x (inconnu)\n", t.HeldWeapon)
		}
	}
}
