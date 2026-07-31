// tmp_forgefood — jetable : extrait un tag `food` (forge object definition) du module
// any/globals/forge/forge_objects-rtx-new.module et dumpe sa structure CachedTag
// (header, dependencies, data-blocks, strings) pour identifier le modèle référencé.
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

func u16(b []byte, o int) uint16  { return binary.LittleEndian.Uint16(b[o:]) }
func u32(b []byte, o int) int     { return int(binary.LittleEndian.Uint32(b[o:])) }
func i32(b []byte, o int) int     { return int(int32(binary.LittleEndian.Uint32(b[o:]))) }
func u64(b []byte, o int) int     { return int(binary.LittleEndian.Uint64(b[o:])) }
func f32(b []byte, o int) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(b[o:])) }

func fourCC(v uint32) string {
	b := []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
	for _, c := range b {
		if c < 0x20 || c > 0x7e {
			return fmt.Sprintf("%08x", v)
		}
	}
	return string(b)
}

type tagInfo struct {
	tag                                          []byte
	headerSize, dataSize                         int
	deps, dataBlocks, structs, dataRefs, tagRefs int
	strTabSize                                   int
	tablesStart, blockTab, structTab, dataRefTab int
	tagRefTab, strTab                            int
	hOff                                         int
}

func parseTag(tag []byte) (tagInfo, bool) {
	for H := 0x28; H <= 0x4c; H += 4 {
		if H+8 > len(tag) {
			break
		}
		hs, ds := u32(tag, H), u32(tag, H+4)
		if hs > 0 && ds > 0 && hs+ds <= len(tag) && hs+ds >= len(tag)-16 {
			ti := tagInfo{tag: tag, headerSize: hs, dataSize: ds, hOff: H,
				deps: u32(tag, H-0x20), dataBlocks: u32(tag, H-0x1C), structs: u32(tag, H-0x18),
				dataRefs: u32(tag, H-0x14), tagRefs: u32(tag, H-0x10), strTabSize: u32(tag, H-0x0c)}
			ti.tablesStart = H + 0x18
			ti.blockTab = ti.tablesStart + ti.deps*0x18
			ti.structTab = ti.blockTab + ti.dataBlocks*0x10
			ti.dataRefTab = ti.structTab + ti.structs*0x20
			ti.tagRefTab = ti.dataRefTab + ti.dataRefs*0x14
			ti.strTab = ti.tagRefTab + ti.tagRefs*0x10
			return ti, true
		}
	}
	return tagInfo{}, false
}

func (ti tagInfo) blockAbs(idx int) (int, int) {
	if idx < 0 || idx >= ti.dataBlocks {
		return 0, 0
	}
	b := ti.blockTab + idx*0x10
	size := u32(ti.tag, b)
	section := int(u16(ti.tag, b+6))
	off := u64(ti.tag, b+8)
	if section != 0 {
		return ti.headerSize + off, size
	}
	return off, size
}

func main() {
	modPath := os.Args[1]
	m, err := openMod(modPath)
	must(err)
	idx, _ := strconv.Atoi(os.Args[2])
	target := m.file(idx)
	fmt.Printf("dataBase=%d entry: %+v\n", m.dataBase, target)
	data, err := m.extract(target)
	must(err)
	fmt.Printf("tag idx=%d group=%s len=%d magic=%q version=%d\n", idx, target.Group, len(data), data[:4], u32(data, 4))
	ti, ok := parseTag(data)
	if !ok {
		fmt.Println("parse header KO")
		hexdump(data, 0, 256)
		return
	}
	fmt.Printf("H=0x%x headerSize=%d dataSize=%d deps=%d blocks=%d structs=%d dataRefs=%d tagRefs=%d strTab=%d\n",
		ti.hOff, ti.headerSize, ti.dataSize, ti.deps, ti.dataBlocks, ti.structs, ti.dataRefs, ti.tagRefs, ti.strTabSize)

	fmt.Println("-- dependencies --")
	for i := 0; i < ti.deps; i++ {
		d := ti.tablesStart + i*0x18
		fmt.Printf("  dep%d group=%s nameOff=%d assetId=%#016x globalId=%d (%#x) pad=%d\n",
			i, fourCC(binary.LittleEndian.Uint32(ti.tag[d:])), u32(ti.tag, d+4),
			uint64(u64(ti.tag, d+8)), i32(ti.tag, d+0x10), uint32(i32(ti.tag, d+0x10)), i32(ti.tag, d+0x14))
	}
	fmt.Println("-- data blocks --")
	for i := 0; i < ti.dataBlocks && i < 40; i++ {
		abs, sz := ti.blockAbs(i)
		fmt.Printf("  blk%d abs=%d size=%d\n", i, abs, sz)
	}
	fmt.Println("-- structs --")
	for i := 0; i < ti.structs && i < 40; i++ {
		b := ti.structTab + i*0x20
		fmt.Printf("  st%d type=%d loc=%d target=%d fieldBlock=%d fieldOff=%d\n",
			i, u16(ti.tag, b+0x10), u16(ti.tag, b+0x12), i32(ti.tag, b+0x14), i32(ti.tag, b+0x18), u32(ti.tag, b+0x1c))
	}
	fmt.Println("-- tag refs --")
	for i := 0; i < ti.tagRefs; i++ {
		r := ti.tagRefTab + i*0x10
		fb, fo := i32(ti.tag, r), u32(ti.tag, r+4)
		abs, _ := ti.blockAbs(fb)
		fmt.Printf("  ref%d fieldBlock=%d fieldOff=%d nameOff=%d depIdx=%d | abs=%d data=% 02x\n",
			i, fb, fo, u32(ti.tag, r+8), i32(ti.tag, r+0x0c), abs+fo, safeSlice(data, abs+fo, 32))
	}
	fmt.Println("-- data refs --")
	for i := 0; i < ti.dataRefs; i++ {
		r := ti.dataRefTab + i*0x14
		fmt.Printf("  dref%d parent=%d unk=%d target=%d fieldBlock=%d fieldOff=%d\n",
			i, i32(ti.tag, r), i32(ti.tag, r+4), i32(ti.tag, r+8), i32(ti.tag, r+0x0c), u32(ti.tag, r+0x10))
	}
	fmt.Println("-- strings (printable runs >=4) --")
	for _, s := range strRuns(data, 4) {
		fmt.Printf("  %s\n", s)
	}
	fmt.Println("-- root block hexdump --")
	rootBlock := -1
	for i := 0; i < ti.structs; i++ {
		b := ti.structTab + i*0x20
		if u16(ti.tag, b+0x10) == 0 {
			rootBlock = i32(ti.tag, b+0x14)
			break
		}
	}
	abs, sz := ti.blockAbs(rootBlock)
	fmt.Printf("rootBlock=%d abs=%d size=%d\n", rootBlock, abs, sz)
	if sz > 0 {
		hexdump(data, abs, sz)
		fmt.Println("-- floats du root --")
		for o := 0; o+4 <= sz; o += 4 {
			v := f32(data, abs+o)
			if v != 0 && (abs32(v) > 1e-4 && abs32(v) < 1e5) {
				fmt.Printf("  +0x%03x %g\n", o, v)
			}
		}
	}
}

func abs32(f float32) float32 {
	if f < 0 {
		return -f
	}
	return f
}

func strRuns(b []byte, min int) []string {
	var out []string
	cur := []byte{}
	for _, c := range b {
		if c >= 0x20 && c < 0x7f {
			cur = append(cur, c)
			continue
		}
		if len(cur) >= min {
			out = append(out, string(cur))
		}
		cur = cur[:0]
	}
	if len(cur) >= min {
		out = append(out, string(cur))
	}
	return out
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

func safeSlice(b []byte, off, n int) []byte {
	if off < 0 || off >= len(b) {
		return nil
	}
	if off+n > len(b) {
		n = len(b) - off
	}
	return b[off : off+n]
}
