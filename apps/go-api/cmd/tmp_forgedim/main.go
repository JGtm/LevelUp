// tmp_forgedim — jetable : résout les type_id Forge (map_objects.csv) vers les tags
// de la palette Forge et remonte la chaîne food -> bloc/scen -> hlmt -> mode/coll.
//
// Modules (Halo Infinite) :
//
//	any/globals/forge/forge_objects-rtx-new.module  (417 Mo) : food, bloc, scen, hlmt, coll, phmo
//	ds/globals/forge/forge_objects-rtx-new.module   (2,0 Go) : mode, rtgo (géométrie)
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type modIndex struct {
	name string
	m    *modReader
	byID map[int32][]modFile
}

func tryOpenIndex(path string) (*modIndex, error) {
	m, err := openMod(path)
	if err != nil {
		return nil, err
	}
	idx := &modIndex{name: shortName(path), m: m, byID: map[int32][]modFile{}}
	for i := 0; i < m.fileCount; i++ {
		f := m.file(i)
		idx.byID[f.GlobalID] = append(idx.byID[f.GlobalID], f)
	}
	return idx, nil
}

var indexes []*modIndex

// find renvoie la copie la PLUS COMPLETE du tag (plus gros uncompSize) parmi tous
// les modules indexes : les variantes any/ et ds/ d un meme tag n ont pas le meme
// contenu (geometrie strippee d un cote).
func find(gid int32) (*modIndex, modFile, bool) {
	var bi *modIndex
	var bf modFile
	found := false
	for _, ix := range indexes {
		for _, f := range ix.byID[gid] {
			if f.Group == "" {
				continue
			}
			if !found || f.UncompSize > bf.UncompSize {
				bi, bf, found = ix, f, true
			}
		}
	}
	return bi, bf, found
}

func shortName(p string) string {
	p = filepath.ToSlash(p)
	parts := strings.Split(p, "/")
	if n := len(parts); n >= 3 {
		return strings.Join(parts[n-3:], "/")
	}
	return p
}

func loadTag(gid int32) (modFile, tagInfo, []byte, bool) {
	ix, f, ok := find(gid)
	if !ok {
		return modFile{}, tagInfo{}, nil, false
	}
	data, err := ix.m.extract(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "extract %d (%s): %v\n", gid, f.Group, err)
		return f, tagInfo{}, nil, false
	}
	ti, ok := parseTag(data)
	return f, ti, data, ok
}

func main() {
	mode := os.Args[1]
	// os.Args[2] = racine deploy/ ; on indexe any/ et ds/ (pas pc/ : layout d'entrée
	// différent + offsets > 4 Go).
	root := os.Args[2]
	var paths []string
	for _, sub := range []string{"any", "ds"} {
		filepath.Walk(filepath.Join(root, sub), func(p string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && strings.HasSuffix(p, ".module") {
				paths = append(paths, p)
			}
			return nil
		})
	}
	sort.Strings(paths)
	for _, p := range paths {
		ix, err := tryOpenIndex(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", p, err)
			continue
		}
		indexes = append(indexes, ix)
	}
	fmt.Fprintf(os.Stderr, "modules indexés: %d\n", len(indexes))
	switch mode {
	case "chain":
		for _, a := range os.Args[3:] {
			v, _ := strconv.ParseInt(a, 10, 64)
			walk(int32(v), 0, map[int32]bool{})
		}
	case "chainfile":
		for _, id := range readIDs(os.Args[3]) {
			fmt.Printf("=== type_id %d\n", id)
			walk(id, 0, map[int32]bool{})
		}
	case "scan":
		// scan <u32|u64 hex> : cherche la valeur à TOUS les offsets de l'entrée.
		v, err := strconv.ParseUint(strings.TrimPrefix(os.Args[3], "0x"), 16, 64)
		must(err)
		for _, ix := range indexes {
			for off := 0; off+8 <= modStride; off += 4 {
				h4, h8 := 0, 0
				var ex4, ex8 int
				for i := 0; i < ix.m.fileCount; i++ {
					e := ix.m.ent[i*modStride:]
					if uint64(uint32(u32(e, off))) == v&0xffffffff {
						h4++
						ex4 = i
					}
					if uint64(u64(e, off)) == v {
						h8++
						ex8 = i
					}
				}
				if h4 > 0 {
					fmt.Printf("%s u32 +0x%02x : %d (ex #%d %s)\n", ix.name, off, h4, ex4, ix.m.file(ex4).Group)
				}
				if h8 > 0 {
					fmt.Printf("%s u64 +0x%02x : %d (ex #%d %s)\n", ix.name, off, h8, ex8, ix.m.file(ex8).Group)
				}
			}
		}
	case "table":
		runTable(os.Args[3], os.Args[4])
	case "cinfo":
		v, _ := strconv.ParseInt(os.Args[3], 10, 64)
		_, ti, data, ok := loadTag(int32(v))
		if !ok {
			return
		}
		for i := 0; i < ti.dataBlocks; i++ {
			abs, sz := ti.blockAbs(i)
			if sz == 0 || sz%84 != 0 || abs+sz > len(data) {
				continue
			}
			for r := 0; r+84 <= sz; r += 84 {
				o := abs + r
				fmt.Printf("blk%d rec%d flags=%d X(%.4f,%.4f) Y(%.4f,%.4f) Z(%.4f,%.4f)\n",
					i, r/84, u16(data, o), f32(data, o+4), f32(data, o+8), f32(data, o+12),
					f32(data, o+16), f32(data, o+20), f32(data, o+24))
			}
		}
	case "blk":
		v, _ := strconv.ParseInt(os.Args[3], 10, 64)
		bi, _ := strconv.Atoi(os.Args[4])
		_, ti, data, ok := loadTag(int32(v))
		if !ok {
			return
		}
		abs, sz := ti.blockAbs(bi)
		hexdump(data, abs, sz)
		for o := 0; o+4 <= sz; o += 4 {
			fmt.Printf("  +0x%03x f=%g i=%d\n", o, f32(data, abs+o), i32(data, abs+o))
		}
	case "dump":
		v, _ := strconv.ParseInt(os.Args[3], 10, 64)
		f, ti, data, ok := loadTag(int32(v))
		fmt.Printf("group=%s len=%d parsed=%v\n", f.Group, len(data), ok)
		if !ok {
			return
		}
		fmt.Printf("blocks=%d structs=%d refs=%d dataRefs=%d\n", ti.dataBlocks, ti.structs, ti.tagRefs, ti.dataRefs)
		for i := 0; i < ti.dataBlocks; i++ {
			abs, sz := ti.blockAbs(i)
			fmt.Printf("blk%d abs=%d size=%d\n", i, abs, sz)
		}
		for i := 0; i < ti.tagRefs; i++ {
			r := ti.tagRefTab + i*0x10
			fb, fo := i32(ti.tag, r), u32(ti.tag, r+4)
			a, sz := ti.blockAbs(fb)
			raw := []byte(nil)
			if sz > 0 && a+fo+28 <= len(data) {
				raw = data[a+fo : a+fo+28]
			}
			gid := int32(0)
			grp := ""
			if raw != nil {
				gid = int32(u32(raw, 0x08))
				grp = fourCC(uint32(u32(raw, 0x14)))
			}
			_, _, present := find(gid)
			fmt.Printf("ref%d blk=%d off=%d grp=%q gid=%d present=%v raw=% 02x\n", i, fb, fo, grp, gid, present, raw)
		}
		rb := ti.rootBlock()
		abs, sz := ti.blockAbs(rb)
		fmt.Printf("root=%d abs=%d size=%d\n", rb, abs, sz)
		hexdump(data, abs, sz)
		for o := 0; o+4 <= sz; o += 4 {
			v := f32(data, abs+o)
			if v != 0 && abs32(v) > 1e-4 && abs32(v) < 1e5 {
				fmt.Printf("  f+0x%03x %g\n", o, v)
			}
		}
	}
}

func walk(gid int32, depth int, seen map[int32]bool) {
	if depth > 4 || seen[gid] {
		return
	}
	seen[gid] = true
	pad := strings.Repeat("  ", depth)
	f, ti, data, ok := loadTag(gid)
	if data == nil {
		fmt.Printf("%s%d -> INTROUVABLE\n", pad, gid)
		return
	}
	if !ok {
		fmt.Printf("%s%d [%s] len=%d parse KO\n", pad, gid, f.Group, len(data))
		return
	}
	fmt.Printf("%s%d [%s] len=%d blocks=%d refs=%d\n", pad, gid, f.Group, len(data), ti.dataBlocks, ti.tagRefs)
	for _, r := range ti.refs() {
		if r.Group == "bloc" || r.Group == "scen" || r.Group == "hlmt" || r.Group == "mode" ||
			r.Group == "coll" || r.Group == "phmo" || r.Group == "mach" || r.Group == "food" {
			fmt.Printf("%s  ref %s %d\n", pad, r.Group, r.GlobalID)
			walk(r.GlobalID, depth+1, seen)
		}
	}
}

func readIDs(path string) []int32 {
	b, err := os.ReadFile(path)
	must(err)
	var ids []int32
	seen := map[int32]bool{}
	for _, l := range strings.Fields(string(b)) {
		v, err := strconv.ParseInt(l, 10, 64)
		if err != nil {
			continue
		}
		if !seen[int32(v)] {
			seen[int32(v)] = true
			ids = append(ids, int32(v))
		}
	}
	sort.SliceStable(ids, func(i, j int) bool { return false })
	return ids
}

func abs32(f float32) float32 {
	if f < 0 {
		return -f
	}
	return f
}

func hexdump(b []byte, off, n int) {
	if off+n > len(b) {
		n = len(b) - off
	}
	for i := 0; i < n; i += 16 {
		e := i + 16
		if e > n {
			e = n
		}
		line := b[off+i : off+e]
		var sb strings.Builder
		for _, c := range line {
			if c >= 0x20 && c < 0x7f {
				sb.WriteByte(c)
			} else {
				sb.WriteByte('.')
			}
		}
		fmt.Printf("  %04x: % 02x  %s\n", i, line, sb.String())
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
