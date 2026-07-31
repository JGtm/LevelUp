// tmp_kf_bootstrap — THROWAWAY : bootstrappe le décodeur filmdec sur un film SANS
// world_dump CE (auto-calibration keyframe). Vérifie sur un nouveau match (jgtm) :
//  1. le registre d'archétypes parse depuis le header (filmChunk0) — archétype #35
//     = BIPED avec weapon-state-type-info à i43..46.
//  2. le keyframe type-2 (dans filmChunk1) porte les littéraux d'armes complets
//     (id64 catalogué) = les ancres de loadout pour la calibration des 8 records biped.
//
// Usage : tmp_kf_bootstrap <dir_chunks>
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

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

// extractType2 renvoie le payload du 1er paquet type-2 (keyframe) d'un chunk inflaté.
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

func knownHigh32(v uint32) bool {
	for id := range analysis.WeaponIDToName {
		if uint32(id>>32) == v {
			return true
		}
	}
	return false
}

type weaponLit struct {
	bit  int
	id64 uint64
	name string
}

func findWeaponLits(payload []byte) []weaponLit {
	var lits []weaponLit
	total := len(payload) * 8
	for bp := 0; bp+64 <= total; bp++ {
		v := uint32(bitsAt(payload, bp, 32))
		if !knownHigh32(v) {
			continue
		}
		low := uint32(bitsAt(payload, bp+32, 32))
		id64 := (uint64(v) << 32) | uint64(low)
		if nm, ok := analysis.WeaponIDToName[id64]; ok {
			lits = append(lits, weaponLit{bp, id64, nm})
		}
	}
	return lits
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: tmp_kf_bootstrap <dir_chunks>")
		return
	}
	dir := os.Args[1]
	files, _ := filepath.Glob(filepath.Join(dir, "filmChunk*"))
	sort.Slice(files, func(i, j int) bool { return chunkNum(files[i]) < chunkNum(files[j]) })
	if len(files) == 0 {
		fmt.Println("aucun filmChunk dans", dir)
		return
	}

	// 1. registre depuis le header (filmChunk0)
	header := inflate(files[0])
	fmt.Printf("header %s : inflaté %d octets\n", filepath.Base(files[0]), len(header))
	reg, err := filmdec.ParseRegistryChunk(header)
	if err != nil {
		fmt.Println("ParseRegistryChunk:", err)
		return
	}
	fmt.Printf("registre : %d archétypes\n", len(reg.Archetypes))
	if arch, ok := reg.Archetype(35); ok {
		fmt.Printf("archétype #35 (%d composants) :\n", len(arch.Components))
		for i, c := range arch.Components {
			mark := ""
			if c == "weapon-state-type-info" {
				mark = "  <-- HELD WEAPON"
			}
			if i <= 1 || c == "weapon-state-type-info" || c == "object-dead-state-component" ||
				c == "object-multiplayer-properties-component" {
				fmt.Printf("    i%-2d %s%s\n", i, c, mark)
			}
		}
	} else {
		fmt.Println("PAS d'archétype #35 — layout registre différent !")
	}

	// 2. keyframe type-2 : cherché dans les premiers chunks réplication
	var kf []byte
	var kfFrom string
	for _, f := range files[1:] {
		if p := extractType2(inflate(f)); p != nil {
			kf = p
			kfFrom = filepath.Base(f)
			break
		}
	}
	if kf == nil {
		fmt.Println("aucun paquet type-2 (keyframe) trouvé")
		return
	}
	fmt.Printf("\nkeyframe type-2 trouvé dans %s : %d octets\n", kfFrom, len(kf))

	lits := findWeaponLits(kf)
	fmt.Printf("littéraux d'armes complets (id64 catalogué) dans le keyframe : %d\n", len(lits))
	// regroupe en records (gap > 1000 bits) = ~les 8 joueurs
	var groups [][]weaponLit
	for _, l := range lits {
		if len(groups) == 0 || l.bit-groups[len(groups)-1][len(groups[len(groups)-1])-1].bit > 1000 {
			groups = append(groups, []weaponLit{l})
		} else {
			groups[len(groups)-1] = append(groups[len(groups)-1], l)
		}
	}
	fmt.Printf("regroupés en %d records (gap>1000) :\n", len(groups))
	for i, g := range groups {
		s := ""
		for _, w := range g {
			s += fmt.Sprintf(" %s@%d", w.name, w.bit)
		}
		fmt.Printf("  record %d (%d armes):%s\n", i, len(g), s)
	}
}

func chunkNum(p string) int {
	base := filepath.Base(p)
	n, started := 0, false
	for _, c := range base {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
			started = true
		} else if started {
			break
		}
	}
	return n
}
