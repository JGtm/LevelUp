// tmp_kfdefstate — THROWAWAY. Teste l'hypothèse : le "380 bits" default-state biped du
// keyframe est un ARTEFACT de calibration ; le vrai default-state = le deser porté
// (consumeBipedDefaultState ~120 bits), et les ~260 bits "résidu" sont en fait la BOUCLE
// DE COMPOSANTS (position/vélocité au spawn, déjà portées via la recette registre).
//
// Méthode : sur le record Hydra du keyframe (start=194126), traverser avec :
//
//	(A) default-state = deser porté, résidu 0 (NewDefaultStateBits=0 => pas de skip)
//	(B) default-state = skip pur 380 (calibration actuelle)
//
// et comparer : atteint-on l'arme (weapon-state-type-info) ? capture-t-on une position i0
// SAINE au spawn ? où finit la traversée ? Le bon modèle = celui qui atteint l'arme au bon
// littéral ET capture une position plausible.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_kfdefstate
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

const hydraStart = 194126

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

func extractType2(d []byte) []byte {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == 2 {
			return d[off+16 : off+16+sz]
		}
		off += 16 + sz
	}
	return nil
}

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

func traverse(reg *filmdec.Registry, payload []byte, defBits int, label string) {
	var caps []filmdec.PositionSample
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { caps = append(caps, s) })
	filmdec.SetRecordStateParam(2)
	br := filmdec.NewBitReader(payload)
	br.Skip(hydraStart)
	t := filmdec.TraverseEntity(br, reg, defBits)
	filmdec.SetPositionCaptureHook(nil)

	fmt.Printf("--- %s (defaultStateBits=%d) ---\n", label, defBits)
	fmt.Printf("  typeIndex=%d desyncAt=i%d endBit=%d (nComps=%d)\n", t.TypeIndex, t.DesyncAt, t.EndBit, len(t.Comps))
	// arme atteinte ?
	wst := -1
	for _, c := range t.Comps {
		if c.Name == "weapon-state-type-info" {
			wst = c.StartBit
		}
	}
	if wst >= 0 {
		// le littéral d'arme suit le WST gate (h32<<32|low)
		h := uint32(bitsAt(payload, wst+1, 32))
		low := uint32(bitsAt(payload, wst+33, 32))
		id := (uint64(h) << 32) | uint64(low)
		name, known := analysis.WeaponIDToName[id]
		fmt.Printf("  weapon-state-type-info @bit%d ; littéral suivant = %s (connu=%v)\n", wst, name, known)
	} else {
		fmt.Printf("  weapon-state-type-info NON atteint\n")
	}
	// premiers composants + position capturée
	for i, c := range t.Comps {
		if i >= 6 {
			break
		}
		fmt.Printf("    i%-2d @%-7d %s (porté=%v)\n", c.Index, c.StartBit, c.Name, c.Ported)
	}
	for _, s := range caps {
		fmt.Printf("  POSITION i0 capturée : kind=%s vec=(%.2f, %.2f, %.2f) @bit%d\n", s.Kind, s.Vec[0], s.Vec[1], s.Vec[2], s.BitPos)
	}
	if len(caps) == 0 {
		fmt.Printf("  (aucune position i0 capturée)\n")
	}
	fmt.Println()
}

func main() {
	reg, _ := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	payload := extractType2(inflate(cache + "/chunk_02.bin"))
	fmt.Printf("keyframe type-2 = %d octets ; record Hydra @bit%d (R6 typeIndex)\n\n", len(payload), hydraStart)
	fmt.Printf("R(6) @%d = %d (attendu 35=biped)\n\n", hydraStart, bitsAt(payload, hydraStart, 6))

	// (B) calibration actuelle : skip pur 380 (le deser biped reste actif + skip résidu)
	traverse(reg, payload, 380, "(B) skip calibré 380")
	// (A) hypothèse : default-state = deser porté seul (résidu 0 => pas de skip)
	traverse(reg, payload, 0, "(A) deser porté, résidu 0 (default ~120)")
	// (C) hypothèse affinée : default-state = deser porté + défauts mouvement i0-i4 décodés.
	//     Gate calibré attendu = 194126+6+380 = 194512.
	filmdec.SetBipedDefaultStateDecodeMovement(true)
	traverse(reg, payload, 0, "(C) deser porté + défauts mouvement i0-i4 (gate attendu 194512)")
	filmdec.SetBipedDefaultStateDecodeMovement(false)
}
