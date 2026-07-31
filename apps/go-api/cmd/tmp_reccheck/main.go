// tmp_reccheck — THROWAWAY. Valide le décodage d'un record NEW biped COMPLET
// (R6 + representation + boucle composants) contre la GROUND TRUTH live (CE) : un record
// biped de ~1847-1935 bits. Reconstruit le flux depuis l'état BitReader du jeu et lance
// TraverseEntity, en dumpant les composants + positions.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

// knownHigh32 : le variant d'un weapon-state-type-info est le high-32 d'un id d'arme connu.
func knownHigh32(v uint32) bool {
	if v == 0 || v == 0xffffffff {
		return false
	}
	for id := range analysis.WeaponIDToName {
		if uint32(id>>32) == v {
			return true
		}
	}
	return false
}

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

// Capture live (CE) : record NEW biped, arch=35, ~1847-1935 bits.
const (
	acc      = uint64(0x8E1B180000000000)
	used     = 43
	bytesHex = "0ED7B4028033A0240844D660256000411241CAFEB7AE3982197E0361F3712381BA43DF0E43A97381BAEB6000000000000000000001240B02C160280E0001704C0B070320000903C000000D0500A0805500006040000001280B4A0BD00802CA0D0100C20F01002B4A0E0B4A0710D68498EB0400C20D0100CB184A042A010000001CD01638046FFFFFF6060EC8CAC57AD200004B124A4BBE00200000100000C00009983D0B40C80CC32B1824D542C9679F12403008A7430ECE429D4FF6C3F920B3DC3B619D84A42C9679F12C030089A85F15DCE51039C43F920B3DC29EE14F7088D8BDE6C8EB4631E6B68003F0BF03001A12800434FC0F200A425000823B80C407"
	target   = 1924 // longueur EXACTE capturee live (CE) de ce record NEW biped
)

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

func main() {
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	bs, _ := hex.DecodeString(bytesHex)
	var bits []byte
	for pos := 63; pos >= used; pos-- { // bits non-lus de l'accumulateur (MSB d'abord)
		bits = append(bits, byte((acc>>uint(pos))&1))
	}
	for _, b := range bs {
		for j := 7; j >= 0; j-- {
			bits = append(bits, (b>>uint(j))&1)
		}
	}
	buf := make([]byte, (len(bits)+7)/8)
	for i, bit := range bits {
		if bit == 1 {
			buf[i/8] |= 1 << uint(7-(i%8))
		}
	}
	fmt.Printf("flux reconstruit : %d bits (cible record ~1847-1935)\n", len(bits))

	filmdec.SetRecordStateParam(2)
	filmdec.SetSimStateComplete(true) // oracle: décode i60 en entier + continue vers i61-63

	fmt.Printf("--- SWEEP simStateExtra (tail i60) -> endBit vs 1924 + tags i63 (valides=0..5) ---\n")
	for k := 0; k <= 130; k++ {
		filmdec.SetBipedActionDebug(true) // reset + enable
		filmdec.SetSimStateExtra(k)
		br := filmdec.NewBitReader(buf)
		tt := filmdec.TraverseEntity(br, reg, 0)
		// lire les tags AVANT de désactiver (SetBipedActionDebug(false) efface BiOkSeqs)
		var tags []uint64
		if len(filmdec.BiOkSeqs) > 0 {
			tags = filmdec.BiOkSeqs[0]
		}
		clean := true
		for _, tg := range tags {
			if tg > 5 {
				clean = false
			}
		}
		if k >= 15 && k <= 40 {
			fmt.Printf("  simStateExtra=%-3d endBit=%-5d desync=i%-3d count1=%d tagsI63=%v cleanTags=%v\n", k, tt.EndBit, tt.DesyncAt, len(tags), tags, clean)
		}
	}
	filmdec.SetBipedActionDebug(false)
	filmdec.SetSimStateExtra(0) // défaut honnête : pas de tail (le tail runtime reste non résolu offline)

	fmt.Printf("--- TRACE du rep (R6@0..6 ; rep du jeu = 166 bits -> fin attendue @bit172) ---\n")
	filmdec.SetRepTraceHook(func(label string, pos int) { fmt.Printf("  REP %-12s @bit%d\n", label, pos) })

	var caps []filmdec.PositionSample
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { caps = append(caps, s) })
	filmdec.SetBipedActionDebug(true)
	br := filmdec.NewBitReader(buf)
	t := filmdec.TraverseEntity(br, reg, 0)
	filmdec.SetPositionCaptureHook(nil)
	filmdec.SetRepTraceHook(nil)
	fmt.Printf("  (version rep = %d)\n", filmdec.LastRepVersion())
	fmt.Printf("  i63 biped-action tags (count1=len) : OK=%v BAD=%v\n", filmdec.BiOkSeqs, filmdec.BiBadSeqs)
	filmdec.SetBipedActionDebug(false)

	fmt.Printf("\ntypeIndex=%d (attendu 35) desyncAt=i%d endBit=%d nComps=%d\n", t.TypeIndex, t.DesyncAt, t.EndBit, len(t.Comps))
	fmt.Printf("mask=%016X gate=%v  (bits 59-63 = %d%d%d%d%d)\n", t.Mask, t.Gate,
		(t.Mask>>59)&1, (t.Mask>>60)&1, (t.Mask>>61)&1, (t.Mask>>62)&1, (t.Mask>>63)&1)
	fmt.Printf("=== BIT-EXACT ? endBit=%d vs longueur reelle=%d => %s ===\n", t.EndBit, target,
		map[bool]string{true: "OUI, BIT-EXACT", false: fmt.Sprintf("NON, ecart=%d", t.EndBit-target)}[t.EndBit == target])
	wstReached := false
	for _, c := range t.Comps {
		if c.Name == "weapon-state-type-info" {
			wstReached = true
		}
	}
	fmt.Printf("weapon-state-type-info atteint = %v\n", wstReached)
	fmt.Printf("TOUS les composants (largeur = gap au suivant) :\n")
	for i, c := range t.Comps {
		w := "?"
		if i+1 < len(t.Comps) {
			w = fmt.Sprintf("%d", t.Comps[i+1].StartBit-c.StartBit)
		} else {
			w = fmt.Sprintf("%d", t.EndBit-c.StartBit)
		}
		extra := ""
		if c.Name == "weapon-state-type-info" {
			extra = fmt.Sprintf("  variant=0x%08X known=%v", c.Variant, knownHigh32(c.Variant))
		}
		fmt.Printf("  i%-2d @%-4d w=%-4s %s (porté=%v)%s\n", c.Index, c.StartBit, w, c.Name, c.Ported, extra)
	}
	for _, s := range caps {
		fmt.Printf("POSITION i0 : kind=%s vec=(%.2f, %.2f, %.2f) @bit%d\n", s.Kind, s.Vec[0], s.Vec[1], s.Vec[2], s.BitPos)
	}
}
