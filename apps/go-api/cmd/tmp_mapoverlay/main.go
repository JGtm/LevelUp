// tmp_mapoverlay — IMAGE PNG : carte 2D reconstruite (.module catalyst) + positions
// joueurs décodées d'un film catalyst, alignées empiriquement (normalisation + recherche
// de la meilleure orientation), rendues en heatmap.
//
// Usage : CGO_ENABLED=1 go run ./cmd/tmp_mapoverlay [module] [filmDir] [out.png]
package main

import (
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"

	"bytes"
	"compress/zlib"
	"io"

	"levelup/go-api/internal/analysis/positions"
	"levelup/go-api/internal/himodule"
)

const defMod = `D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/ds/levels/multi/catalyst/catalyst-rtx-new.module`
const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/01e1f945`

const pmdStride, offPos, offMin, offMax = 0x90, 0x38, 0x44, 0x50

func u16(b []byte, o int) uint16  { return binary.LittleEndian.Uint16(b[o:]) }
func u32(b []byte, o int) int     { return int(binary.LittleEndian.Uint32(b[o:])) }
func i32(b []byte, o int) int     { return int(int32(binary.LittleEndian.Uint32(b[o:]))) }
func u64(b []byte, o int) int     { return int(binary.LittleEndian.Uint64(b[o:])) }
func f32(b []byte, o int) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(b[o:])) }

// ---------- carte : extraction des bbox monde par mesh (cf tmp_geores) ----------
type xnode struct {
	XMLName xml.Name
	V       string  `xml:"v,attr"`
	Length  string  `xml:"length,attr"`
	Nodes   []xnode `xml:",any"`
}

var sizeTab = map[string]int{
	"_0": 32, "_1": 256, "_2": 4, "_3": 4, "_4": 1, "_5": 2, "_6": 4, "_7": 8,
	"_8": 4, "_9": 4, "_A": 1, "_B": 2, "_C": 4, "_D": 4, "_E": 2, "_F": 1,
	"_10": 4, "_11": 4, "_12": 4, "_13": 4, "_14": 4, "_15": 4, "_16": 8, "_17": 12,
	"_18": 8, "_19": 12, "_1A": 16, "_1B": 8, "_1C": 12, "_1D": 12, "_1E": 16, "_1F": 12,
	"_20": 16, "_21": 4, "_22": 4, "_23": 4, "_24": 8, "_25": 8, "_26": 8, "_27": 4,
	"_28": 4, "_29": 4, "_2A": 4, "_2B": 4, "_2C": 1, "_2D": 1, "_2E": 2, "_2F": 2,
	"_30": 4, "_31": 4, "_32": 4, "_33": 4, "_36": 0, "_37": 0, "_38": 0, "_39": 32,
	"_3A": 4, "_3B": 0, "_3C": 1, "_3D": 2, "_3E": 4, "_3F": 8, "_40": 20, "_41": 28,
	"_42": 24, "_43": 16, "_44": 4, "_45": 4,
}

func walk(n xnode, off int, found *int, target string) int {
	for _, c := range n.Nodes {
		switch nm := c.XMLName.Local; nm {
		case "_40":
			if c.V == target && *found < 0 {
				*found = off
			}
			off += 20
		case "_38":
			off = walk(c, off, found, target)
		case "_34", "_35":
			l := sizeTab[nm]
			if c.Length != "" {
				fmt.Sscanf(c.Length, "%d", &l)
			}
			off += l
		case "Flag", "":
		default:
			off += sizeTab[nm]
		}
	}
	return off
}

func perMeshDataOffset(xmlPath string) int {
	data, _ := os.ReadFile(xmlPath)
	var root xnode
	if xml.Unmarshal(data, &root) != nil {
		return -1
	}
	found := -1
	walk(root, 0, &found, "Per Mesh Data")
	return found
}

type tagInfo struct {
	tag                                   []byte
	headerSize, deps, dataBlocks, structs int
	blockTab, structTab                   int
}

func parseTag(tag []byte) (tagInfo, bool) {
	for H := 0x28; H <= 0x44; H += 4 {
		if H+8 > len(tag) {
			break
		}
		hs, ds := u32(tag, H), u32(tag, H+4)
		if hs > 0 && ds > 0 && hs+ds == len(tag) {
			ti := tagInfo{tag: tag, headerSize: hs,
				deps: u32(tag, H-0x20), dataBlocks: u32(tag, H-0x1C), structs: u32(tag, H-0x18)}
			ti.blockTab = H + 0x18 + ti.deps*0x18
			ti.structTab = ti.blockTab + ti.dataBlocks*0x10
			return ti, true
		}
	}
	return tagInfo{}, false
}

func (ti tagInfo) blockAbs(idx int) (int, int) {
	b := ti.blockTab + idx*0x10
	size := u32(ti.tag, b)
	section := int(u16(ti.tag, b+6))
	off := u64(ti.tag, b+8)
	if section != 0 {
		return ti.headerSize + off, size
	}
	return off, size
}

func (ti tagInfo) perMeshBlock(pmdOff int) (int, int, bool) {
	rootBlock := -1
	for i := 0; i < ti.structs; i++ {
		b := ti.structTab + i*0x20
		if int(u16(ti.tag, b+0x10)) == 0 {
			rootBlock = i32(ti.tag, b+0x14)
			break
		}
	}
	type ref struct{ off, target int }
	var refs []ref
	for i := 0; i < ti.structs; i++ {
		b := ti.structTab + i*0x20
		if int(u16(ti.tag, b+0x10)) == 1 && i32(ti.tag, b+0x18) == rootBlock {
			refs = append(refs, ref{u32(ti.tag, b+0x1C), i32(ti.tag, b+0x14)})
		}
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].off < refs[j].off })
	best, bestD := -1, 1<<30
	for _, r := range refs {
		d := r.off - pmdOff
		if d < 0 {
			d = -d
		}
		if d < bestD {
			bestD, best = d, r.target
		}
	}
	if best < 0 {
		return 0, 0, false
	}
	abs, size := ti.blockAbs(best)
	return abs, size, size > 0 && abs+size <= len(ti.tag)
}

type box struct{ lo, hi [3]float32 }

func extractMap(modPath, xmlPath string) []box {
	pmdOff := perMeshDataOffset(xmlPath)
	if pmdOff < 0 {
		return nil
	}
	m, err := himodule.Open(modPath)
	if err != nil {
		return nil
	}
	var boxes []box
	for _, f := range m.Files("") {
		data, err := m.Extract(f)
		if err != nil || len(data) < 8 || string(data[:4]) != "ucsh" {
			continue
		}
		ti, ok := parseTag(data)
		if !ok {
			continue
		}
		abs, size, ok := ti.perMeshBlock(pmdOff)
		if !ok || size < pmdStride {
			continue
		}
		for i := 0; i < size/pmdStride; i++ {
			o := abs + i*pmdStride
			var lo, hi [3]float32
			ext, bad := float32(0), false
			for a := 0; a < 3; a++ {
				p := f32(data, o+offPos+a*4)
				lo[a] = p + f32(data, o+offMin+a*4)
				hi[a] = p + f32(data, o+offMax+a*4)
				if math.IsNaN(float64(lo[a])) || math.IsInf(float64(lo[a]), 0) || hi[a] < lo[a] {
					bad = true
				}
				if hi[a]-lo[a] > ext {
					ext = hi[a] - lo[a]
				}
			}
			if !bad && ext < 40 {
				boxes = append(boxes, box{lo, hi})
			}
		}
	}
	return boxes
}

// ---------- positions film (keyframe) ----------
func inflate(p string) []byte {
	raw, _ := os.ReadFile(p)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); (e2 == nil || e2 == io.ErrUnexpectedEOF) && len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

func filmPositions(dir string) []positions.PlayerPosition {
	files, _ := filepath.Glob(filepath.Join(dir, "chunk_*.bin"))
	sort.Strings(files)
	var chunks []positions.ChunkInput
	for _, f := range files {
		d := inflate(f)
		if d == nil {
			continue
		}
		chunks = append(chunks, positions.ChunkInput{Data: d, StartMS: 0, ChunkType: 2})
	}
	return positions.DecodeKeyframePositions(chunks)
}

// ---------- projection 2D robuste ----------
type pt2 struct{ u, v float64 } // normalisés [0,1]
type rect2 struct{ lo, hi pt2 } // bbox normalisée

// planeAndBounds choisit le plan (2 axes de plus grand span) et les bornes ROBUSTES
// (1-99 percentile par axe) à partir des CENTRES fournis.
func planeAndBounds(cx, cy, cz []float64) (ax, ay int, lo, hi [3]float64) {
	cols := [3][]float64{cx, cy, cz}
	for a := 0; a < 3; a++ {
		v := append([]float64(nil), cols[a]...)
		sort.Float64s(v)
		lo[a], hi[a] = v[len(v)/100], v[len(v)*99/100]
	}
	sp := []struct {
		a int
		s float64
	}{{0, hi[0] - lo[0]}, {1, hi[1] - lo[1]}, {2, hi[2] - lo[2]}}
	sort.Slice(sp, func(i, j int) bool { return sp[i].s > sp[j].s })
	return sp[0].a, sp[1].a, lo, hi
}

func clamp01(x float64) float64 { return math.Max(0, math.Min(1, x)) }

// orient applique l'une des 8 orientations (rotations+reflets) du carré unité.
func orient(p pt2, k int) pt2 {
	u, v := p.u, p.v
	switch k {
	case 0:
		return pt2{u, v}
	case 1:
		return pt2{1 - u, v}
	case 2:
		return pt2{u, 1 - v}
	case 3:
		return pt2{1 - u, 1 - v}
	case 4:
		return pt2{v, u}
	case 5:
		return pt2{1 - v, u}
	case 6:
		return pt2{v, 1 - u}
	default:
		return pt2{1 - v, 1 - u}
	}
}

func main() {
	modPath, filmDir, out := defMod, defFilm, "catalyst_overlay.png"
	if len(os.Args) > 1 {
		modPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		filmDir = os.Args[2]
	}
	if len(os.Args) > 3 {
		out = os.Args[3]
	}
	xmlPath := filepath.Join(filepath.Dir(os.Args[0]), "rawg.xml")
	if _, err := os.Stat(xmlPath); err != nil {
		xmlPath = "cmd/tmp_geores/rawg.xml"
	}

	boxes := extractMap(modPath, xmlPath)
	fmt.Printf("carte : %d meshes (play area)\n", len(boxes))
	pos := filmPositions(filmDir)
	fmt.Printf("film  : %d positions keyframe\n", len(pos))
	if len(boxes) == 0 || len(pos) == 0 {
		fmt.Println("données insuffisantes")
		return
	}

	// --- carte : plan + bornes robustes depuis les centres de bbox ---
	mcx, mcy, mcz := make([]float64, len(boxes)), make([]float64, len(boxes)), make([]float64, len(boxes))
	for i, b := range boxes {
		mcx[i] = float64((b.lo[0] + b.hi[0]) / 2)
		mcy[i] = float64((b.lo[1] + b.hi[1]) / 2)
		mcz[i] = float64((b.lo[2] + b.hi[2]) / 2)
	}
	max, may, mlo, mhi := planeAndBounds(mcx, mcy, mcz)
	nm := func(v float64, a int) float64 { return clamp01((v - mlo[a]) / (mhi[a] - mlo[a])) }
	var mapRects []rect2
	for _, b := range boxes {
		mapRects = append(mapRects, rect2{
			pt2{nm(float64(b.lo[max]), max), nm(float64(b.lo[may]), may)},
			pt2{nm(float64(b.hi[max]), max), nm(float64(b.hi[may]), may)},
		})
	}

	// --- film : plan + bornes robustes ---
	fcx, fcy, fcz := make([]float64, len(pos)), make([]float64, len(pos)), make([]float64, len(pos))
	for i, p := range pos {
		fcx[i], fcy[i], fcz[i] = float64(p.X), float64(p.Y), float64(p.Z)
	}
	fax, fay, flo, fhi := planeAndBounds(fcx, fcy, fcz)
	nf := func(v float64, a int) float64 { return clamp01((v - flo[a]) / (fhi[a] - flo[a])) }
	var filmPts []pt2
	for _, p := range pos {
		c := [3]float64{float64(p.X), float64(p.Y), float64(p.Z)}
		filmPts = append(filmPts, pt2{nf(c[fax], fax), nf(c[fay], fay)})
	}

	// raster footprint carte (densité de remplissage) pour scorer l'orientation film.
	const G = 80
	foot := make([]int, G*G)
	for _, r := range mapRects {
		x0, x1 := int(r.lo.u*(G-1)), int(r.hi.u*(G-1))
		y0, y1 := int(r.lo.v*(G-1)), int(r.hi.v*(G-1))
		for gy := y0; gy <= y1; gy++ {
			for gx := x0; gx <= x1; gx++ {
				if gx >= 0 && gx < G && gy >= 0 && gy < G {
					foot[gy*G+gx]++
				}
			}
		}
	}
	bestK, bestScore := 0, -1
	for k := 0; k < 8; k++ {
		s := 0
		for _, p := range filmPts {
			q := orient(p, k)
			gx, gy := int(q.u*(G-1)), int(q.v*(G-1))
			if foot[gy*G+gx] > 0 {
				s++
			}
		}
		if s > bestScore {
			bestScore, bestK = s, k
		}
	}
	fmt.Printf("alignement : orientation #%d (%d/%d positions sur le footprint)\n", bestK, bestScore, len(filmPts))

	var mapCenters []pt2
	for _, r := range mapRects {
		mapCenters = append(mapCenters, pt2{(r.lo.u + r.hi.u) / 2, (r.lo.v + r.hi.v) / 2})
	}
	renderPNG(out, mapRects, mapCenters, filmPts, bestK)
	fmt.Printf("→ %s\n", out)
	_ = fax
}

func renderPNG(path string, mapRects []rect2, mapCenters, filmPts []pt2, k int) {
	const S = 760
	const pad = 36
	span := float64(S - 2*pad)
	img := image.NewRGBA(image.Rect(0, 0, S, S))
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			img.Set(x, y, color.RGBA{16, 18, 24, 255})
		}
	}
	px := func(u, v float64) (int, int) { return pad + int(u*span), S - pad - int(v*span) }

	// carte : footprint en gris (rectangles semi-transparents accumulés via raster).
	const G = 200
	foot := make([]int, G*G)
	maxc := 1
	for _, r := range mapRects {
		x0, x1 := int(r.lo.u*(G-1)), int(r.hi.u*(G-1))
		y0, y1 := int(r.lo.v*(G-1)), int(r.hi.v*(G-1))
		for gy := y0; gy <= y1; gy++ {
			for gx := x0; gx <= x1; gx++ {
				if gx >= 0 && gx < G && gy >= 0 && gy < G {
					foot[gy*G+gx]++
					if foot[gy*G+gx] > maxc {
						maxc = foot[gy*G+gx]
					}
				}
			}
		}
	}
	for gy := 0; gy < G; gy++ {
		for gx := 0; gx < G; gx++ {
			c := foot[gy*G+gx]
			if c == 0 {
				continue
			}
			lum := uint8(45 + 120*math.Min(1, float64(c)/float64(maxc)*3))
			x0, y0 := px(float64(gx)/G, float64(gy+1)/G)
			x1, y1 := px(float64(gx+1)/G, float64(gy)/G)
			for py := y0; py <= y1; py++ {
				for pxx := x0; pxx <= x1; pxx++ {
					if pxx >= 0 && pxx < S && py >= 0 && py < S {
						img.Set(pxx, py, color.RGBA{lum, lum, uint8(float64(lum) * 1.15), 255})
					}
				}
			}
		}
	}
	// structure : splat lumineux additif par centre de mesh (densité géométrie).
	acc := make([]float64, S*S)
	amax := 0.0
	for _, c := range mapCenters {
		cx, cy := px(c.u, c.v)
		for dy := -7; dy <= 7; dy++ {
			for dx := -7; dx <= 7; dx++ {
				d2 := float64(dx*dx + dy*dy)
				if d2 > 49 {
					continue
				}
				x, y := cx+dx, cy+dy
				if x >= 0 && x < S && y >= 0 && y < S {
					acc[y*S+x] += math.Exp(-d2 / 14)
					if acc[y*S+x] > amax {
						amax = acc[y*S+x]
					}
				}
			}
		}
	}
	if amax > 0 {
		for y := 0; y < S; y++ {
			for x := 0; x < S; x++ {
				v := acc[y*S+x] / amax
				if v <= 0.03 {
					continue
				}
				r0, g0, b0, _ := img.At(x, y).RGBA()
				add := math.Min(1, v*2.2)
				img.Set(x, y, color.RGBA{
					uint8(math.Min(255, float64(r0>>8)+90*add)),
					uint8(math.Min(255, float64(g0>>8)+120*add)),
					uint8(math.Min(255, float64(b0>>8)+160*add)), 255})
			}
		}
	}
	// positions joueurs : disques orange (halo + cœur).
	disc := func(cx, cy, r int, col color.RGBA) {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if dx*dx+dy*dy <= r*r {
					x, y := cx+dx, cy+dy
					if x >= 0 && x < S && y >= 0 && y < S {
						img.Set(x, y, col)
					}
				}
			}
		}
	}
	for _, p := range filmPts {
		q := orient(p, k)
		cx, cy := px(q.u, q.v)
		disc(cx, cy, 6, color.RGBA{255, 100, 40, 230})
		disc(cx, cy, 3, color.RGBA{255, 225, 130, 255})
	}
	f, err := os.Create(path)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	png.Encode(f, img)
}
