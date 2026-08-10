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
	atlases map[uint32]*atlas
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
		atlases: map[uint32]*atlas{},
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

// imgStride : pas du tableau de descripteurs d'images, MESURÉ (0xe8 → 0x110 → 0x138 sur les
// trois atlas examinés). imgProbe est la fenêtre nécessaire pour vérifier la signature.
const (
	imgStride = 0x28
	imgProbe  = 0x2c
)

// bitmImg décrit une image déclarée par un tag `bitm`.
type bitmImg struct {
	Off    int
	W, H   int
	Depth  int
	Format int
	Mips   int
}

// scanImgs recense les images déclarées par le corps du tag.
//
// DEUX TEMPS, ET LE SECOND EST CE QUI A SUPPRIMÉ LA DÉRIVE. Le balayage seul (chercher
// partout une signature « dimensions répétées à +0x14 ») trouvait 91 enregistrements là où le
// tag en déclare 88, et 24 là où il en déclare 22 : les faux positifs consommaient des
// ressources et décalaient tout ce qui suivait, d'où les images rayées en queue d'atlas.
//
// Le balayage ne sert donc plus qu'à TROUVER LE PREMIER enregistrement — son offset n'est pas
// codé en dur, il est retrouvé, ce qui garde la lecture robuste aux versions. Ensuite le
// tableau est lu tel qu'il est : un compte (u32 juste avant le premier enregistrement) et un
// pas régulier de 0x28 octets, mesurés tous deux sur les trois atlas examinés.
func scanImgs(data []byte) []bitmImg {
	u16 := func(o int) int { return int(binary.LittleEndian.Uint16(data[o:])) }
	first := -1
	for o := 0; o+imgProbe <= len(data); o += 4 {
		w, h := u16(o), u16(o+2)
		if w < 4 || h < 4 || w > 8192 || h > 8192 {
			continue
		}
		if u16(o+0x14) != w || u16(o+0x16) != h {
			continue
		}
		first = o
		break
	}
	if first < 0 {
		return nil
	}
	count := 0
	if first >= 4 {
		count = int(binary.LittleEndian.Uint32(data[first-4:]))
	}
	// Le compte doit rester plausible ET tenir dans le corps du tag ; sinon on retombe sur
	// ce que le corps peut porter, plutôt que de faire confiance à un champ mal lu.
	if max := (len(data) - first) / imgStride; count <= 0 || count > max {
		count = max
	}
	out := make([]bitmImg, 0, count)
	for i := 0; i < count; i++ {
		o := first + i*imgStride
		out = append(out, bitmImg{
			Off: o, W: u16(o), H: u16(o + 2),
			Depth: u16(o + 4), Format: u16(o + 8), Mips: u16(o + 0x0a),
		})
	}
	return out
}

// glyphStats rapporte ce que le décodage a dû approximer sur une image.
type glyphStats struct {
	Blocks   int // blocs 4x4 du mip0
	Rebuilt  int // blocs de mode 7 reconstruits par ajustement sur le mip inférieur
	Opaque   int // blocs modes 0-3 : pas de canal alpha, alpha 255 EXACT, RGB approché (jeté)
	Degraded int // blocs mode 7 sans témoin : alpha en aplat — la seule vraie dégradation
}

// mipGuide décode le niveau de mip SUIVANT (deux fois plus petit) et l'agrandit, pour servir
// de témoin à la reconstruction des blocs de mode 7. Retourne nil si la ressource ne porte
// pas ce niveau — le décodeur retombe alors sur le repli en aplat, comme avant.
func mipGuide(blob []byte, px, w, h int) *image.NRGBA {
	off := px + bcSize(w, h)
	w1, h1 := max1(w/2), max1(h/2)
	need := bcSize(w1, h1)
	if off+need > len(blob) {
		return nil
	}
	small, _, _, _ := decodeBC7(blob[off:off+need], w1, h1, nil)
	return upscale2(small, w, h)
}

// decodeAlphaGlyph rend l'image `idx` du tag `id` en glyphe BLANC sur fond transparent,
// recadré à sa boîte englobante. Retourne aussi ce que le décodage a dû approximer.
func decodeAlphaGlyph(ix *tagIndex, id uint32, idx int) (*image.NRGBA, glyphStats, bitmImg, error) {
	var st glyphStats
	blob, px, im, err := resBlob(ix, id, idx)
	if err != nil {
		return nil, st, im, err
	}
	bw, bh := (im.W+3)/4, (im.H+3)/4
	need := bw * bh * 16
	if px+need > len(blob) {
		return nil, st, im, fmt.Errorf("mip0 déborde la ressource (%d+%d > %d)", px, need, len(blob))
	}
	src, rebuilt, opaque, degraded := decodeBC7(blob[px:px+need], im.W, im.H, mipGuide(blob, px, im.W, im.H))
	st = glyphStats{Blocks: bw * bh, Rebuilt: rebuilt, Opaque: opaque, Degraded: degraded}
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
	if maxX < minX || maxY < minY {
		return glyph, st, im, nil
	}
	crop := image.NewNRGBA(image.Rect(0, 0, maxX-minX+1, maxY-minY+1))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			so, do := glyph.PixOffset(x, y), crop.PixOffset(x-minX, y-minY)
			copy(crop.Pix[do:do+4], glyph.Pix[so:so+4])
		}
	}
	return crop, st, im, nil
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

// alphaNoise mesure la densité de TRANSITIONS d'opacité par pixel, ligne par ligne.
//
// À quoi ça sert : le contrôle arithmétique valide le POIDS d'une ressource, pas son
// contenu. En queue d'atlas, des descripteurs faux positifs tombent parfois sur une
// ressource du bon poids : le décodage rend alors du bruit rayé, plausible pour la machine
// et évident à l'oeil. Un dessin d'interface a des contours nets et peu nombreux (mesuré
// < 0,06 transition par pixel) ; un décodage désaligné en a un ordre de grandeur de plus.
func alphaNoise(img *image.NRGBA) float64 {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 2 || h < 1 {
		return 0
	}
	trans := 0
	for y := 0; y < h; y++ {
		prev := img.Pix[img.PixOffset(0, y)+3] > 127
		for x := 1; x < w; x++ {
			cur := img.Pix[img.PixOffset(x, y)+3] > 127
			if cur != prev {
				trans++
				prev = cur
			}
		}
	}
	return float64(trans) / float64(w*h)
}
