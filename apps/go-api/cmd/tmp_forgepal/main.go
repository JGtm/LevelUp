// Commande jetable : exploration de la palette Forge (forge_objects-rtx-new.module).
// Lecture PAR ReadAt (le module fait 2 Go : on ne le charge JAMAIS en entier).
package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	hdrSize     = 0x48
	entryStride = 0x58
)

func u16(b []byte, o int) uint16 { return binary.LittleEndian.Uint16(b[o:]) }
func u32(b []byte, o int) uint32 { return binary.LittleEndian.Uint32(b[o:]) }
func u64(b []byte, o int) uint64 { return binary.LittleEndian.Uint64(b[o:]) }

func fourCC(v uint32) string {
	b := []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
	for _, c := range b {
		if c < 0x20 || c > 0x7e {
			return ""
		}
	}
	return string(b)
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: tmp_forgepal <cmd> <module> [args]")
		return
	}
	cmd, path := os.Args[1], os.Args[2]
	if cmd == "hunt" {
		hunt(path, os.Args[3])
		return
	}
	f, err := os.Open(path)
	must(err)
	defer f.Close()
	st, err := f.Stat()
	must(err)

	hdr := make([]byte, hdrSize)
	_, err = f.ReadAt(hdr, 0)
	must(err)
	if string(hdr[:4]) != "mohd" {
		fmt.Println("pas mohd")
		return
	}
	version := u32(hdr, 4)
	moduleID := u64(hdr, 8)
	fileCount := int(u32(hdr, 0x10))
	firstResIdx := int(u32(hdr, 0x20))
	fileNameSize := int(u32(hdr, 0x24))
	numRes := int(u32(hdr, 0x28))
	numBlocks := int(u32(hdr, 0x2C))
	build := u32(hdr, 0x30)

	fmt.Printf("file=%s size=%d\n", path, st.Size())
	fmt.Printf("version=%d moduleId=%#x fileCount=%d firstResourceIndex=%d fileNameSize=%d numResources=%d numBlocks=%d build=%d\n",
		version, moduleID, fileCount, firstResIdx, fileNameSize, numRes, numBlocks, build)

	// Table des entrées.
	entTblSize := fileCount * entryStride
	ent := make([]byte, entTblSize)
	_, err = f.ReadAt(ent, hdrSize)
	must(err)

	switch cmd {
	case "head":
		n := 8
		if len(os.Args) > 3 {
			n, _ = strconv.Atoi(os.Args[3])
		}
		for i := 0; i < n && i < fileCount; i++ {
			e := ent[i*entryStride : (i+1)*entryStride]
			fmt.Printf("#%d group=%q\n", i, fourCC(u32(e, 0x14)))
			for o := 0; o < entryStride; o += 8 {
				fmt.Printf("  +%02x: % 02x   u32=%d,%d  i32=%d,%d\n", o, e[o:o+8],
					u32(e, o), u32(e, o+4), int32(u32(e, o)), int32(u32(e, o+4)))
			}
		}
	case "groups":
		counts := map[string]int{}
		for i := 0; i < fileCount; i++ {
			counts[fourCC(u32(ent[i*entryStride:], 0x14))]++
		}
		for g, c := range counts {
			fmt.Printf("%-6q %d\n", g, c)
		}
	case "ids":
		// dump idx,group,globalId(candidat @0x28),compSize,uncompSize
		off := 0x28
		if len(os.Args) > 3 {
			v, _ := strconv.ParseInt(os.Args[3], 0, 32)
			off = int(v)
		}
		for i := 0; i < fileCount; i++ {
			e := ent[i*entryStride:]
			fmt.Printf("%d,%s,%d,%d,%d,%d\n", i, fourCC(u32(e, 0x14)), int32(u32(e, off)),
				u32(e, 0x18), u32(e, 0x20), u32(e, 0x24))
		}
	case "match":
		// Cherche les type_id cibles dans TOUS les offsets u32 de l'entrée.
		targets := map[int32]bool{}
		fh, err := os.Open(os.Args[3])
		must(err)
		defer fh.Close()
		var ids []int32
		buf := make([]byte, 1<<20)
		n, _ := fh.Read(buf)
		for _, line := range splitLines(string(buf[:n])) {
			if line == "" {
				continue
			}
			v, err := strconv.ParseInt(line, 10, 64)
			if err != nil {
				continue
			}
			if !targets[int32(v)] {
				targets[int32(v)] = true
				ids = append(ids, int32(v))
			}
		}
		fmt.Printf("targets=%d\n", len(ids))
		for off := 0; off+4 <= entryStride; off += 4 {
			hit := 0
			for i := 0; i < fileCount; i++ {
				if targets[int32(u32(ent[i*entryStride:], off))] {
					hit++
				}
			}
			if hit > 0 {
				fmt.Printf("offset +0x%02x : %d entrées matchent\n", off, hit)
			}
		}
	case "base":
		tableEnd := hdrSize + fileCount*entryStride + fileNameSize + numRes*4 + numBlocks*20
		maxEnd := 0
		for i := 0; i < fileCount; i++ {
			e := ent[i*entryStride:]
			if v := int(u32(e, 0x18)) + int(u32(e, 0x20)); v > maxEnd {
				maxEnd = v
			}
		}
		fmt.Printf("tableEnd=%d (0x%x)  maxEnd(dataOff+comp)=%d  size-maxEnd=%d\n", tableEnd, tableEnd, maxEnd, int(st.Size())-maxEnd)
		for _, al := range []int{0x1000, 0x10000, 1} {
			a := (tableEnd + al - 1) / al * al
			fmt.Printf("  align %#x -> %d\n", al, a)
		}
		// dump la table des blocs supposée
		bt := hdrSize + fileCount*entryStride + fileNameSize + numRes*4
		bb := make([]byte, 20*8)
		f.ReadAt(bb, int64(bt))
		fmt.Printf("blockTab@%d :\n", bt)
		for i := 0; i < 8; i++ {
			b := bb[i*20:]
			fmt.Printf("  blk%d compOff=%d compSize=%d decompOff=%d decompSize=%d comp=%d\n",
				i, u32(b, 0), u32(b, 4), u32(b, 8), u32(b, 12), u32(b, 16))
		}
	case "resolve":
		targets := readIDs(os.Args[3])
		for _, id := range targets {
			found := false
			for i := 0; i < fileCount; i++ {
				e := ent[i*entryStride:]
				if int32(u32(e, 0x28)) != id {
					continue
				}
				found = true
				fmt.Printf("%d\t#%d\t%s\tdataOff=%d comp=%d uncomp=%d blocks=%d firstBlock=%d res=%d\n",
					id, i, fourCC(u32(e, 0x14)), u32(e, 0x18), u32(e, 0x20), u32(e, 0x24),
					u16(e, 0x0A), u32(e, 0x0C), int32(u32(e, 0x10)))
			}
			if !found {
				fmt.Printf("%d\tNON TROUVE\n", id)
			}
		}
	}
}

func readIDs(path string) []int32 {
	b, err := os.ReadFile(path)
	must(err)
	seen := map[int32]bool{}
	var ids []int32
	for _, line := range splitLines(string(b)) {
		v, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			continue
		}
		if !seen[int32(v)] {
			seen[int32(v)] = true
			ids = append(ids, int32(v))
		}
	}
	return ids
}

func splitLines(s string) []string {
	var out []string
	cur := ""
	for _, c := range s {
		if c == '\n' || c == '\r' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(c)
	}
	out = append(out, cur)
	return out
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

// hunt : parcourt tous les .module d'une arborescence et signale, pour chaque
// global tag id cherché, le module + le groupe qui le contient. Ne lit que le
// header et la table des entrées (pas de décompression).
func hunt(root, idsFile string) {
	want := map[int32]bool{}
	for _, id := range readIDs(idsFile) {
		want[id] = true
	}
	found := map[int32]string{}
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".module") {
			return nil
		}
		f, err := os.Open(p)
		if err != nil {
			return nil
		}
		defer f.Close()
		hdr := make([]byte, hdrSize)
		if _, err := f.ReadAt(hdr, 0); err != nil || string(hdr[:4]) != "mohd" {
			return nil
		}
		fc := int(u32(hdr, 0x10))
		if fc <= 0 || fc > 5_000_000 {
			return nil
		}
		ent := make([]byte, fc*entryStride)
		if _, err := f.ReadAt(ent, hdrSize); err != nil {
			return nil
		}
		for i := 0; i < fc; i++ {
			e := ent[i*entryStride:]
			id := int32(u32(e, 0x28))
			if want[id] {
				found[id] = fmt.Sprintf("%s #%d %s", p, i, fourCC(u32(e, 0x14)))
			}
		}
		return nil
	})
	for id := range want {
		if v, ok := found[id]; ok {
			fmt.Printf("%d TROUVE %s\n", id, v)
		} else {
			fmt.Printf("%d ABSENT\n", id)
		}
	}
}
