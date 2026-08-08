package main

// extract.go — de l'archive au PNG : index des entrées, table de ressources, appariement
// image <-> ressource, décodage BC7 et rendu du glyphe.

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// moduleRoots : racines candidates, dans l'ordre. La première qui existe gagne.
// Surchargeable par HALO_DEPLOY (ou le drapeau -deploy).
var moduleRoots = []string{
	`C:/Program Files (x86)/Steam/steamapps/common/Halo Infinite/deploy`,
	`C:/XboxGames/Halo Infinite/Content/deploy`,
	`D:/SteamLibrary/steamapps/common/Halo Infinite/deploy`,
}

func moduleRoot() string {
	if v := os.Getenv("HALO_DEPLOY"); v != "" {
		return v
	}
	for _, r := range moduleRoots {
		if fi, err := os.Stat(r); err == nil && fi.IsDir() {
			return r
		}
	}
	return ""
}

// tagRef désigne une entrée de .module.
type tagRef struct {
	Module string
	Entry  int
	Group  string
	ID     uint32
}

type tagIndex struct {
	byID    map[uint32][]tagRef
	byGroup map[string][]tagRef
	nEntry  int
	nMods   int
	opened  map[string]*hmod
}

func listModules(only string) []string {
	root := moduleRoot()
	if root == "" {
		return nil
	}
	var mods []string
	_ = filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() || !strings.HasSuffix(p, ".module") {
			return nil
		}
		if only != "" && !strings.Contains(filepath.ToSlash(p), only) {
			return nil
		}
		mods = append(mods, p)
		return nil
	})
	sort.Strings(mods)
	return mods
}

func fourCCLE(v uint32) string {
	b := []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
	for i, c := range b {
		if c < 0x20 || c > 0x7e {
			b[i] = '.'
		}
	}
	return string(b)
}

// readEntryTable ne lit QUE l'en-tête + la table des entrées : les archives font plusieurs
// gigaoctets (globals-rtx-new = 7,8 Go), les charger pour indexer serait absurde.
func readEntryTable(path string) ([]byte, error) {
	f, err := os.Open(path) //nolint:gosec // chemin fourni par l'opérateur, lecture seule
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	head := make([]byte, 0x48)
	if _, err := f.ReadAt(head, 0); err != nil {
		return nil, err
	}
	if string(head[:4]) != "mohd" {
		return nil, fmt.Errorf("magic mohd absent")
	}
	n := int(binary.LittleEndian.Uint32(head[0x10:]))
	if n <= 0 || n > 4_000_000 {
		return nil, fmt.Errorf("fileCount aberrant: %d", n)
	}
	buf := make([]byte, 0x48+n*0x58)
	if _, err := f.ReadAt(buf, 0); err != nil {
		return nil, err
	}
	return buf, nil
}

func buildIndex(only string) *tagIndex {
	ix := &tagIndex{
		byID:    map[uint32][]tagRef{},
		byGroup: map[string][]tagRef{},
		opened:  map[string]*hmod{},
	}
	for _, mp := range listModules(only) {
		data, err := readEntryTable(mp)
		if err != nil {
			continue
		}
		n := int(binary.LittleEndian.Uint32(data[0x10:]))
		if n <= 0 || 0x48+n*0x58 > len(data) {
			continue
		}
		ix.nMods++
		ix.nEntry += n
		for i := 0; i < n; i++ {
			e := 0x48 + i*0x58
			id := binary.LittleEndian.Uint32(data[e+0x28:])
			grp := fourCCLE(binary.LittleEndian.Uint32(data[e+0x14:]))
			r := tagRef{Module: mp, Entry: i, Group: grp, ID: id}
			ix.byID[id] = append(ix.byID[id], r)
			ix.byGroup[grp] = append(ix.byGroup[grp], r)
		}
	}
	return ix
}

func (ix *tagIndex) open(path string) (*hmod, error) {
	if m, ok := ix.opened[path]; ok {
		return m, nil
	}
	m, err := hmOpen(path)
	if err != nil {
		return nil, err
	}
	ix.opened[path] = m
	return m, nil
}

func (ix *tagIndex) extract(r tagRef) ([]byte, error) {
	m, err := ix.open(r.Module)
	if err != nil {
		return nil, err
	}
	if r.Entry >= m.hdr.FileCount {
		return nil, fmt.Errorf("entrée %d hors borne (%d)", r.Entry, m.hdr.FileCount)
	}
	return m.extract(m.file(r.Entry))
}

// pickRef choisit la variante d'archive qui porte les pixels. La variante `ds/` (serveur
// dédié) déclare les mêmes tags mais son index de ressources vaut 0 : elle n'a pas les
// images. La variante `pc/` les a.
func pickRef(refs []tagRef) (tagRef, bool) {
	for _, r := range refs {
		if strings.Contains(filepath.ToSlash(r.Module), "/pc/") {
			return r, true
		}
	}
	if len(refs) > 0 {
		return refs[0], true
	}
	return tagRef{}, false
}

// resourceTable lit la table de ressources d'une archive : res[i] -> index d'entrée.
// Disposition : [entrées fichier][chaînes][2 u32 sentinelles][ressources u32][blocs].
func resourceTable(m *hmod) ([]uint32, error) {
	resOff := int64(0x48) + int64(m.hdr.FileCount)*0x58 + int64(m.hdr.StringsSize) + 8
	raw, err := m.readAt(resOff, m.hdr.ResourceCount*4)
	if err != nil {
		return nil, err
	}
	out := make([]uint32, m.hdr.ResourceCount)
	for i := range out {
		out[i] = binary.LittleEndian.Uint32(raw[i*4:])
	}
	return out, nil
}

// entryResourceIndex : champ +0x10 de l'entrée = index dans la table de ressources.
func entryResourceIndex(m *hmod, entry int) int {
	return int(binary.LittleEndian.Uint32(m.entries[entry*0x58+0x10:]))
}

// resBlob rend le contenu de la ressource appariée à l'image idx, et l'offset de ses pixels.
// La ressource est elle-même un blob à en-tête `ucsh` : les pixels commencent après
// headerSize + dataSize (vérifié à l'octet près, cf. l'en-tête du paquet).
func resBlob(ix *tagIndex, id uint32, idx int) (blob []byte, px int, im bitmImg, err error) {
	r, ok := pickRef(ix.byID[id])
	if !ok {
		return nil, 0, im, fmt.Errorf("tag %08x absent des archives", id)
	}
	m, err := ix.open(r.Module)
	if err != nil {
		return nil, 0, im, err
	}
	tab, err := resourceTable(m)
	if err != nil {
		return nil, 0, im, err
	}
	data, err := ix.extract(r)
	if err != nil {
		return nil, 0, im, err
	}
	h, ok := parseTagHeader(data)
	if !ok {
		return nil, 0, im, fmt.Errorf("tag %08x sans en-tête ucsh", id)
	}
	imgs := scanImgs(data[h.HeaderSize:])
	if idx < 0 || idx >= len(imgs) {
		return nil, 0, im, fmt.Errorf("index %d hors borne (%d images)", idx, len(imgs))
	}
	ri := entryResourceIndex(m, r.Entry) + idx
	if ri >= len(tab) {
		return nil, 0, im, fmt.Errorf("ressource %d hors table", ri)
	}
	blob, err = m.extract(m.file(int(tab[ri])))
	if err != nil {
		return nil, 0, im, err
	}
	rh, ok := parseTagHeader(blob)
	if !ok {
		return nil, 0, im, fmt.Errorf("ressource %d sans en-tête ucsh", ri)
	}
	return blob, int(rh.HeaderSize) + int(rh.DataSize), imgs[idx], nil
}

const imgStride = 0x2c

// bitmImg décrit une image déclarée par un tag `bitm`.
type bitmImg struct {
	Off    int
	W, H   int
	Depth  int
	Format int
	Mips   int
}

// scanImgs recense les images déclarées par le corps du tag. Ancrage INDÉPENDANT des offsets
// absolus (donc des versions du jeu) : chaque enregistrement de 44 octets répète ses
// dimensions à +0x00 et à +0x14 ; ce doublon sert de signature.
func scanImgs(data []byte) []bitmImg {
	var out []bitmImg
	u16 := func(o int) int { return int(binary.LittleEndian.Uint16(data[o:])) }
	for o := 0; o+imgStride <= len(data); o += 4 {
		w, h := u16(o), u16(o+2)
		if w < 4 || h < 4 || w > 8192 || h > 8192 {
			continue
		}
		if u16(o+0x14) != w || u16(o+0x16) != h {
			continue
		}
		out = append(out, bitmImg{Off: o, W: w, H: h, Depth: u16(o + 4), Format: u16(o + 8), Mips: u16(o + 0x0a)})
	}
	return out
}

// decodeAlphaGlyph rend l'image `idx` du tag `id` en glyphe BLANC sur fond transparent,
// recadré à sa boîte englobante. Retourne aussi le taux de blocs BC7 tombés en repli.
func decodeAlphaGlyph(ix *tagIndex, id uint32, idx int) (*image.NRGBA, float64, bitmImg, error) {
	blob, px, im, err := resBlob(ix, id, idx)
	if err != nil {
		return nil, 0, im, err
	}
	bw, bh := (im.W+3)/4, (im.H+3)/4
	need := bw * bh * 16
	if px+need > len(blob) {
		return nil, 0, im, fmt.Errorf("mip0 déborde la ressource (%d+%d > %d)", px, need, len(blob))
	}
	src, fb := decodeBC7(blob[px:px+need], im.W, im.H)
	glyph := image.NewNRGBA(image.Rect(0, 0, im.W, im.H))
	minX, minY, maxX, maxY := im.W, im.H, -1, -1
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			a := src.Pix[src.PixOffset(x, y)+3]
			o := glyph.PixOffset(x, y)
			glyph.Pix[o], glyph.Pix[o+1], glyph.Pix[o+2], glyph.Pix[o+3] = 255, 255, 255, a
			if a <= 8 {
				continue
			}
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	rate := 100 * float64(fb) / float64(bw*bh)
	if maxX < minX || maxY < minY {
		return glyph, rate, im, nil
	}
	crop := image.NewNRGBA(image.Rect(0, 0, maxX-minX+1, maxY-minY+1))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			so, do := glyph.PixOffset(x, y), crop.PixOffset(x-minX, y-minY)
			copy(crop.Pix[do:do+4], glyph.Pix[so:so+4])
		}
	}
	return crop, rate, im, nil
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path) //nolint:gosec // chemin de sortie fourni par l'opérateur
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	return enc.Encode(f, img)
}
