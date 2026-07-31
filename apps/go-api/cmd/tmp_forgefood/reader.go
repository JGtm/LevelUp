package main

// Lecteur .module (mohd v53) en ReadAt : le module Forge ds fait 2 Go, on ne le
// charge JAMAIS en entier. Différences vs internal/himodule (calibrées sur les
// modules globals/forge) :
//   - la table des ressources commence 8 octets APRÈS la fin de la table des entrées ;
//   - dataBase = align(fin de la table des blocs, 0x1000) — le calcul
//     "tailleFichier - max(dataOffset+compSize)" de himodule est faux ici
//     (padding de fin de fichier).

import (
	"encoding/binary"
	"fmt"
	"os"

	"levelup/go-api/internal/ooz"
)

const (
	modHdr    = 0x48
	modStride = 0x58
)

type modFile struct {
	Index      int
	Group      string
	GlobalID   int32
	BlockCount int
	FirstBlock int
	DataOffset int
	CompSize   int
	UncompSize int
}

type modReader struct {
	f         *os.File
	ent       []byte
	blk       []byte
	fileCount int
	dataBase  int64
}

func openMod(path string) (*modReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	hdr := make([]byte, modHdr)
	if _, err := f.ReadAt(hdr, 0); err != nil {
		return nil, err
	}
	if string(hdr[:4]) != "mohd" {
		return nil, fmt.Errorf("magic mohd absent")
	}
	fileCount := int(binary.LittleEndian.Uint32(hdr[0x10:]))
	numRes := int(binary.LittleEndian.Uint32(hdr[0x28:]))
	numBlocks := int(binary.LittleEndian.Uint32(hdr[0x2C:]))

	ent := make([]byte, fileCount*modStride)
	if _, err := f.ReadAt(ent, modHdr); err != nil {
		return nil, err
	}
	blkTab := int64(modHdr + fileCount*modStride + 8 + numRes*4)
	blk := make([]byte, numBlocks*20)
	if _, err := f.ReadAt(blk, blkTab); err != nil {
		return nil, err
	}
	end := blkTab + int64(numBlocks*20)
	base := (end + 0xFFF) / 0x1000 * 0x1000
	return &modReader{f: f, ent: ent, blk: blk, fileCount: fileCount, dataBase: base}, nil
}

func (m *modReader) file(i int) modFile {
	e := m.ent[i*modStride:]
	g := binary.LittleEndian.Uint32(e[0x14:])
	name := ""
	b := []byte{byte(g >> 24), byte(g >> 16), byte(g >> 8), byte(g)}
	ok := true
	for _, c := range b {
		if c < 0x20 || c > 0x7e {
			ok = false
		}
	}
	if ok {
		name = string(b)
	}
	return modFile{
		Index:      i,
		Group:      name,
		GlobalID:   int32(binary.LittleEndian.Uint32(e[0x28:])),
		BlockCount: int(binary.LittleEndian.Uint16(e[0x0A:])),
		FirstBlock: int(binary.LittleEndian.Uint32(e[0x0C:])),
		DataOffset: int(binary.LittleEndian.Uint32(e[0x18:])),
		CompSize:   int(binary.LittleEndian.Uint32(e[0x20:])),
		UncompSize: int(binary.LittleEndian.Uint32(e[0x24:])),
	}
}

func (m *modReader) extract(fl modFile) ([]byte, error) {
	if fl.BlockCount == 0 {
		src := make([]byte, fl.CompSize)
		if _, err := m.f.ReadAt(src, m.dataBase+int64(fl.DataOffset)); err != nil {
			return nil, err
		}
		if fl.CompSize == fl.UncompSize {
			return src, nil
		}
		return ooz.Decompress(src, fl.UncompSize)
	}
	out := make([]byte, fl.UncompSize)
	for i := fl.FirstBlock; i < fl.FirstBlock+fl.BlockCount; i++ {
		b := m.blk[i*20:]
		co := int(binary.LittleEndian.Uint32(b[0:]))
		cs := int(binary.LittleEndian.Uint32(b[4:]))
		do := int(binary.LittleEndian.Uint32(b[8:]))
		ds := int(binary.LittleEndian.Uint32(b[12:]))
		comp := binary.LittleEndian.Uint32(b[16:])
		if do+ds > len(out) {
			return nil, fmt.Errorf("bloc %d hors borne (%d+%d > %d)", i, do, ds, len(out))
		}
		src := make([]byte, cs)
		if _, err := m.f.ReadAt(src, m.dataBase+int64(fl.DataOffset)+int64(co)); err != nil {
			return nil, err
		}
		if comp == 0 {
			copy(out[do:do+ds], src)
			continue
		}
		dec, err := ooz.Decompress(src, ds)
		if err != nil {
			return nil, fmt.Errorf("bloc %d: %w", i, err)
		}
		copy(out[do:do+ds], dec)
	}
	return out, nil
}
