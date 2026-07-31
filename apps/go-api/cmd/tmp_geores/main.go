// tmp_geores — CARTE 2D : parse les resources runtime_geo (ucsh) du module, extrait
// les bornes monde par mesh (Per Mesh Data : Position@0x38, Bounds min@0x44, max@0x50)
// via le walker version-aware (rawg.xml + struct-table), et rend l'empreinte 2D.
//
// Usage : CGO_ENABLED=1 go run ./cmd/tmp_geores ["<module>"]
package main

import (
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/himodule"
)

const defMod = `D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/ds/levels/multi/catalyst/catalyst-rtx-new.module`

// Per Mesh Data (rawg.xml runtime_geo) : record 0x90 octets.
const pmdStride = 0x90
const offPos = 0x38
const offMin = 0x44
const offMax = 0x50

func u16(b []byte, o int) uint16  { return binary.LittleEndian.Uint16(b[o:]) }
func u32(b []byte, o int) int     { return int(binary.LittleEndian.Uint32(b[o:])) }
func i32(b []byte, o int) int     { return int(int32(binary.LittleEndian.Uint32(b[o:]))) }
func u64(b []byte, o int) int     { return int(binary.LittleEndian.Uint64(b[o:])) }
func f32(b []byte, o int) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(b[o:])) }

// ---- walker XML (rawg.xml) : offset du 1er `_40` = "Per Mesh Data" ----
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

type field struct {
	name string
	off  int
}

func walk(n xnode, off int, out *[]field) int {
	for _, c := range n.Nodes {
		switch nm := c.XMLName.Local; nm {
		case "_40":
			*out = append(*out, field{c.V, off})
			off += 20
		case "_38":
			off = walk(c, off, out)
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

func perMeshDataOffset(xmlPath string) (int, error) {
	data, err := os.ReadFile(xmlPath)
	if err != nil {
		return 0, err
	}
	var root xnode
	if err := xml.Unmarshal(data, &root); err != nil {
		return 0, err
	}
	var fields []field
	walk(root, 0, &fields)
	for _, f := range fields {
		if f.name == "Per Mesh Data" {
			return f.off, nil
		}
	}
	return -1, fmt.Errorf("Per Mesh Data introuvable dans le plugin")
}

// ---- parse tag (ucsh) : header + struct-table + data-blocks ----
type tagInfo struct {
	tag                                   []byte
	headerSize, deps, dataBlocks, structs int
	tablesStart, blockTab, structTab      int
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
			ti.tablesStart = H + 0x18
			ti.blockTab = ti.tablesStart + ti.deps*0x18
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

// perMeshBlock résout le data-block "Per Mesh Data" (le _40 d'offset pmdOff au root).
func (ti tagInfo) perMeshBlock(pmdOff int) (int, int, bool) {
	rootBlock := -1
	for i := 0; i < ti.structs; i++ {
		b := ti.structTab + i*0x20
		if int(u16(ti.tag, b+0x10)) == 0 {
			rootBlock = i32(ti.tag, b+0x14)
			break
		}
	}
	// TagBlocks depuis root, trié par field_offset ; match nearest à pmdOff.
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

type mesh struct {
	pos, min, max [3]float32
}

func main() {
	modPath := defMod
	if len(os.Args) > 1 {
		modPath = os.Args[1]
	}
	xmlPath := filepath.Join(filepath.Dir(os.Args[0]), "rawg.xml")
	if _, err := os.Stat(xmlPath); err != nil {
		xmlPath = "cmd/tmp_geores/rawg.xml"
	}
	pmdOff, err := perMeshDataOffset(xmlPath)
	if err != nil {
		fmt.Printf("plugin: %v\n", err)
		return
	}
	fmt.Printf("Per Mesh Data offset (rawg.xml) = %#x\n", pmdOff)

	m, err := himodule.Open(modPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	var meshes []mesh
	nRes := 0
	for _, f := range m.Files("") {
		data, err := m.Extract(f)
		if err != nil || len(data) < 8 || string(data[:4]) != "ucsh" {
			continue // pas un tag runtime_geo
		}
		ti, ok := parseTag(data)
		if !ok {
			continue
		}
		abs, size, ok := ti.perMeshBlock(pmdOff)
		if !ok || size < pmdStride {
			continue
		}
		n := size / pmdStride
		for i := 0; i < n; i++ {
			o := abs + i*pmdStride
			meshes = append(meshes, mesh{
				pos: [3]float32{f32(data, o+offPos), f32(data, o+offPos+4), f32(data, o+offPos+8)},
				min: [3]float32{f32(data, o+offMin), f32(data, o+offMin+4), f32(data, o+offMin+8)},
				max: [3]float32{f32(data, o+offMax), f32(data, o+offMax+4), f32(data, o+offMax+8)},
			})
		}
		nRes++
		fmt.Printf("  resource #%d : %d meshes (Per Mesh Data @%#x)\n", f.Index, n, abs)
	}
	fmt.Printf("\n%d resources runtime_geo, %d meshes au total\n", nRes, len(meshes))
	if len(meshes) == 0 {
		return
	}
	renderMap(meshes)
}

func renderMap(meshes []mesh) {
	// centre de chaque mesh = milieu de sa bbox (en coords monde).
	// bbox MONDE par mesh = Position + Bounds locaux ; filtre les meshes énormes (skybox).
	type box struct{ lo, hi [3]float32 }
	var boxes []box
	for _, m := range meshes {
		var lo, hi [3]float32
		ext, bad := float32(0), false
		for a := 0; a < 3; a++ {
			lo[a] = m.pos[a] + m.min[a]
			hi[a] = m.pos[a] + m.max[a]
			if math.IsNaN(float64(lo[a])) || math.IsInf(float64(lo[a]), 0) || hi[a] < lo[a] {
				bad = true
			}
			if hi[a]-lo[a] > ext {
				ext = hi[a] - lo[a]
			}
		}
		if !bad && ext < 40 { // <40m : exclut skybox + grandes dalles de sol
			boxes = append(boxes, box{lo, hi})
		}
	}
	// bornes robustes (2-98 pct des centres).
	pct := func(a, lo, hi int) (float32, float32) {
		v := make([]float32, len(boxes))
		for i, b := range boxes {
			v[i] = (b.lo[a] + b.hi[a]) / 2
		}
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
		hii := len(v) * hi / 100
		if hii >= len(v) {
			hii = len(v) - 1
		}
		return v[len(v)*lo/100], v[hii]
	}
	var mn, mx [3]float32
	for a := 0; a < 3; a++ {
		mn[a], mx[a] = pct(a, 1, 99)
	}
	axn := []string{"X", "Y", "Z"}
	sp := []struct {
		a int
		s float32
	}{{0, mx[0] - mn[0]}, {1, mx[1] - mn[1]}, {2, mx[2] - mn[2]}}
	sort.Slice(sp, func(i, j int) bool { return sp[i].s > sp[j].s })
	ax, ay := sp[0].a, sp[1].a
	fmt.Printf("%d meshes (skybox exclus) ; aire jouable X[%.1f,%.1f] Y[%.1f,%.1f] Z[%.1f,%.1f] ; plan %s/%s (up=%s)\n",
		len(boxes), mn[0], mx[0], mn[1], mx[1], mn[2], mx[2], axn[ax], axn[ay], axn[sp[2].a])

	const W, H = 100, 50
	grid := make([][]int, H)
	for i := range grid {
		grid[i] = make([]int, W)
	}
	spanX, spanY := mx[ax]-mn[ax], mx[ay]-mn[ay]
	if spanX == 0 || spanY == 0 {
		return
	}
	cell := func(v, lo, span float32, n int) int { return int(float64(v-lo) / float64(span) * float64(n-1)) }
	for _, b := range boxes {
		x0, x1 := cell(b.lo[ax], mn[ax], spanX, W), cell(b.hi[ax], mn[ax], spanX, W)
		y0, y1 := cell(b.lo[ay], mn[ay], spanY, H), cell(b.hi[ay], mn[ay], spanY, H)
		for gy := y0; gy <= y1; gy++ {
			for gx := x0; gx <= x1; gx++ {
				ry := H - 1 - gy
				if gx >= 0 && gx < W && ry >= 0 && ry < H {
					grid[ry][gx]++
				}
			}
		}
	}
	ramp := []byte(" .:-=+*#%@")
	maxc := 0
	for _, r := range grid {
		for _, c := range r {
			if c > maxc {
				maxc = c
			}
		}
	}
	bar := "+"
	for i := 0; i < W; i++ {
		bar += "-"
	}
	bar += "+"
	fmt.Println(bar)
	for _, r := range grid {
		s := make([]byte, W)
		for x, c := range r {
			if c == 0 {
				s[x] = ' '
				continue
			}
			idx := 1 + int(float64(c)/float64(maxc)*float64(len(ramp)-2))
			if idx >= len(ramp) {
				idx = len(ramp) - 1
			}
			s[x] = ramp[idx]
		}
		fmt.Println("|" + string(s) + "|")
	}
	fmt.Println(bar)
}
