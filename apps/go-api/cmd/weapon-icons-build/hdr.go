package main

// hdr.go — HEADER `ucsh` d un fichier de tag Halo Infinite + table de dependances.
// Repris de cmd/tmp_tagname (supprime par d53c133a9), reduit a ce dont la sonde a besoin.

import (
	"encoding/binary"
	"fmt"
)

type tagHeader struct {
	Magic       uint32
	Version     int32
	DepCount    int32
	BlockCount  int32
	StructCount int32
	DataRefCnt  int32
	TagRefCnt   int32
	StrTabSize  int32
	ZoneSetSize int32
	HeaderSize  int32
	DataSize    int32
	ResSize     int32
}

const (
	tagMagicUCSH = 0x68736375 // 'ucsh' en little-endian
	tagHdrFixed  = 0x50
	depStride    = 0x18
)

func parseTagHeader(b []byte) (tagHeader, bool) {
	if len(b) < tagHdrFixed {
		return tagHeader{}, false
	}
	g := func(o int) int32 { return int32(binary.LittleEndian.Uint32(b[o:])) }
	h := tagHeader{
		Magic:       binary.LittleEndian.Uint32(b[0:]),
		Version:     g(0x04),
		DepCount:    g(0x18),
		BlockCount:  g(0x1c),
		StructCount: g(0x20),
		DataRefCnt:  g(0x24),
		TagRefCnt:   g(0x28),
		StrTabSize:  g(0x2c),
		ZoneSetSize: g(0x30),
		HeaderSize:  g(0x38),
		DataSize:    g(0x3c),
		ResSize:     g(0x40),
	}
	return h, h.Magic == tagMagicUCSH
}

type tagDep struct {
	Group   string
	AssetID uint64
	GlobalD uint32
	Parent  int32
}

func parseDeps(b []byte, h tagHeader) []tagDep {
	out := make([]tagDep, 0, h.DepCount)
	for i := 0; i < int(h.DepCount); i++ {
		o := tagHdrFixed + i*depStride
		if o+depStride > len(b) {
			break
		}
		out = append(out, tagDep{
			Group:   fourCCLE(binary.LittleEndian.Uint32(b[o:])),
			AssetID: binary.LittleEndian.Uint64(b[o+8:]),
			GlobalD: binary.LittleEndian.Uint32(b[o+0x10:]),
			Parent:  int32(binary.LittleEndian.Uint32(b[o+0x14:])),
		})
	}
	return out
}

//nolint:unused // garde-fou de diagnostic
func hexdump(b []byte, n int) {
	if n > len(b) {
		n = len(b)
	}
	for i := 0; i < n; i += 16 {
		end := i + 16
		if end > n {
			end = n
		}
		fmt.Printf("  %04x  ", i)
		for j := i; j < end; j++ {
			fmt.Printf("%02x ", b[j])
		}
		fmt.Println()
	}
}
