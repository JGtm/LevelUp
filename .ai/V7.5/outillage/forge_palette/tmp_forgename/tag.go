package main

// Parsing CachedTag (magic ucsh, v27) — repris de cmd/tmp_forgedim : header
// auto-verrouille par l'invariant headerSize + dataSize == len(tag), tables
// dependencies / data-blocks / structs / data-refs / tag-refs / strings.

import (
	"encoding/binary"
	"math"
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
			return ""
		}
	}
	return string(b)
}

type tagInfo struct {
	tag                                          []byte
	headerSize, dataSize                         int
	deps, dataBlocks, structs, dataRefs, tagRefs int
	tablesStart, blockTab, structTab, dataRefTab int
	tagRefTab, strTab                            int
}

func parseTag(tag []byte) (tagInfo, bool) {
	for H := 0x28; H <= 0x4c; H += 4 {
		if H+8 > len(tag) || H < 0x20 {
			break
		}
		hs, ds := u32(tag, H), u32(tag, H+4)
		if hs > 0 && ds > 0 && hs+ds <= len(tag) && hs+ds >= len(tag)-16 {
			ti := tagInfo{tag: tag, headerSize: hs, dataSize: ds,
				deps: u32(tag, H-0x20), dataBlocks: u32(tag, H-0x1C), structs: u32(tag, H-0x18),
				dataRefs: u32(tag, H-0x14), tagRefs: u32(tag, H-0x10)}
			ti.tablesStart = H + 0x18
			ti.blockTab = ti.tablesStart + ti.deps*0x18
			ti.structTab = ti.blockTab + ti.dataBlocks*0x10
			ti.dataRefTab = ti.structTab + ti.structs*0x20
			ti.tagRefTab = ti.dataRefTab + ti.dataRefs*0x14
			ti.strTab = ti.tagRefTab + ti.tagRefs*0x10
			if ti.strTab > len(tag) {
				continue
			}
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

func (ti tagInfo) rootBlock() int {
	for i := 0; i < ti.structs; i++ {
		b := ti.structTab + i*0x20
		if u16(ti.tag, b+0x10) == 0 {
			return i32(ti.tag, b+0x14)
		}
	}
	return -1
}

type tagRef struct {
	Group    string
	GlobalID int32
	AssetID  uint64
	Block    int
	Offset   int
}

// refs renvoie les tag-refs NON VIDES. Layout du champ pointe (32o) :
// +0x00 pad(8, 0xbc) | +0x08 assetId u64 | +0x10 globalId i32 | +0x14 group fourCC LE.
func (ti tagInfo) refs() []tagRef {
	var out []tagRef
	for i := 0; i < ti.tagRefs; i++ {
		r := ti.tagRefTab + i*0x10
		fb, fo := i32(ti.tag, r), u32(ti.tag, r+4)
		abs, sz := ti.blockAbs(fb)
		if sz == 0 || abs+fo+0x18 > len(ti.tag) {
			continue
		}
		gid := int32(binary.LittleEndian.Uint32(ti.tag[abs+fo+0x08:]))
		grp := fourCC(binary.LittleEndian.Uint32(ti.tag[abs+fo+0x14:]))
		if gid == -1 || grp == "" {
			continue
		}
		out = append(out, tagRef{Group: grp, GlobalID: gid,
			AssetID: binary.LittleEndian.Uint64(ti.tag[abs+fo+0x08:]), Block: fb, Offset: fo})
	}
	return out
}
