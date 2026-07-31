// tmp_regschema — INVESTIGATION : les champs (kind u32, flags u32) IGNORÉS de chaque
// slot du registre (chunk_00) encodent-ils le TYPE + la PRÉCISION/largeur des composants ?
// Si oui = le schéma de quantification est dans le film (pas un mystère runtime/CE).
// On dumpe (kind, flags) par composant de l'archétype 35 (biped) et on croise avec les
// largeurs connues (i0 position = 6/6/6 IndexW=1 ; i1/i3 vélocité ; vec3...).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_regschema [filmDir] [archetype]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const slotSize = 260
const blockSlots = 64
const blockSize = slotSize * blockSlots

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

func slotName(d []byte, off int) string {
	s, e := off+8, off+slotSize
	if s >= len(d) {
		return ""
	}
	if e > len(d) {
		e = len(d)
	}
	raw := d[s:e]
	if z := bytes.IndexByte(raw, 0); z >= 0 {
		raw = raw[:z]
	}
	for _, c := range raw {
		if c < 0x20 || c > 0x7e {
			return ""
		}
	}
	return string(raw)
}

func main() {
	dir, arch := defFilm, 35
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if len(os.Args) > 2 {
		fmt.Sscanf(os.Args[2], "%d", &arch)
	}
	data := inflate(dir + "/chunk_00.bin")
	fmt.Printf("registre inflaté : %d octets, %d blocs (slotSize=%d, blockSize=%d)\n",
		len(data), len(data)/blockSize, slotSize, blockSize)

	base := arch * blockSize
	fmt.Printf("\n=== archétype %d : (kind, flags) + reste du slot non-nom ===\n", arch)
	fmt.Printf("%-4s %-10s %-10s %-44s %s\n", "i", "kind", "flags", "name", "octets 0..16 (hex)")
	for s := 0; s < blockSlots; s++ {
		off := base + s*slotSize
		if off+slotSize > len(data) {
			break
		}
		name := slotName(data, off)
		if name == "" {
			break
		}
		kind := binary.LittleEndian.Uint32(data[off:])
		flags := binary.LittleEndian.Uint32(data[off+4:])
		// dump aussi les octets du nom-end → slot-end (au cas où la précision est APRÈS le nom)
		tail := data[off : off+16]
		fmt.Printf("i%-3d 0x%08x 0x%08x %-44s % x\n", s, kind, flags, name, tail)
	}

	// Croisement direct : i0 position dynamic-precision (largeur connue 6/6/6, IndexW=1).
	// On regarde si kind/flags d'i0 contiennent 6 / 0x06 / 1 quelque part.
	fmt.Println("\n=== indices : valeurs distinctes de kind/flags sur tout l'archétype ===")
	kinds, flagsm := map[uint32]int{}, map[uint32]int{}
	for s := 0; s < blockSlots; s++ {
		off := base + s*slotSize
		if off+slotSize > len(data) || slotName(data, off) == "" {
			break
		}
		kinds[binary.LittleEndian.Uint32(data[off:])]++
		flagsm[binary.LittleEndian.Uint32(data[off+4:])]++
	}
	fmt.Printf("kinds distincts: %d ; flags distincts: %d\n", len(kinds), len(flagsm))
}
