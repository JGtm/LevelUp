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
	bboxReport(meshes)
	histReport(meshes)
}

// ---- bboxReport : bornes monde brutes, sans percentile ni rendu ASCII ----

type box struct{ lo, hi [3]float32 }

// worldBox convertit un mesh en bbox monde (Position + Bounds locaux) et signale s'il est
// exploitable (pas de NaN/Inf, bornes ordonnées).
func worldBox(m mesh) (box, bool) {
	var b box
	for a := 0; a < 3; a++ {
		b.lo[a] = m.pos[a] + m.min[a]
		b.hi[a] = m.pos[a] + m.max[a]
		v := []float64{float64(b.lo[a]), float64(b.hi[a])}
		for _, x := range v {
			if math.IsNaN(x) || math.IsInf(x, 0) {
				return b, false
			}
		}
		if b.hi[a] < b.lo[a] {
			return b, false
		}
	}
	return b, true
}

func (b box) ext() float32 {
	e := float32(0)
	for a := 0; a < 3; a++ {
		if d := b.hi[a] - b.lo[a]; d > e {
			e = d
		}
	}
	return e
}

func union(boxes []box) (lo, hi [3]float32) {
	for a := 0; a < 3; a++ {
		lo[a], hi[a] = float32(math.Inf(1)), float32(math.Inf(-1))
	}
	for _, b := range boxes {
		for a := 0; a < 3; a++ {
			if b.lo[a] < lo[a] {
				lo[a] = b.lo[a]
			}
			if b.hi[a] > hi[a] {
				hi[a] = b.hi[a]
			}
		}
	}
	return
}

func show(label string, boxes []box) {
	if len(boxes) == 0 {
		fmt.Printf("%-34s : (vide)\n", label)
		return
	}
	lo, hi := union(boxes)
	fmt.Printf("%-34s : n=%3d  X[%9.2f,%9.2f] span %8.2f | Y[%9.2f,%9.2f] span %8.2f | Z[%9.2f,%9.2f] span %8.2f\n",
		label, len(boxes), lo[0], hi[0], hi[0]-lo[0], lo[1], hi[1], hi[1]-lo[1], lo[2], hi[2], hi[2]-lo[2])
}

func bboxReport(meshes []mesh) {
	var all, valid []box
	bad := 0
	for _, m := range meshes {
		b, ok := worldBox(m)
		if !ok {
			bad++
			continue
		}
		valid = append(valid, b)
		all = append(all, b)
	}
	fmt.Printf("\n=== BORNES MONDE (Position + Bounds, %d meshes valides, %d rejetés NaN/Inf) ===\n", len(valid), bad)
	show("BBOX GLOBALE (tous meshes)", all)

	for _, cut := range []float32{200, 100, 60, 40, 20} {
		var sub []box
		for _, b := range valid {
			if b.ext() < cut {
				sub = append(sub, b)
			}
		}
		show(fmt.Sprintf("meshes ext < %.0f wu", cut), sub)
	}

	// Distribution des extensions : montre où est la coupure naturelle skybox / props.
	ex := make([]float64, len(valid))
	for i, b := range valid {
		ex[i] = float64(b.ext())
	}
	sort.Float64s(ex)
	q := func(p int) float64 {
		i := len(ex) * p / 100
		if i >= len(ex) {
			i = len(ex) - 1
		}
		return ex[i]
	}
	fmt.Printf("\nextension max d'un mesh (wu) : p10 %.2f p50 %.2f p90 %.2f p99 %.2f max %.2f\n",
		q(10), q(50), q(90), q(99), ex[len(ex)-1])

	// Bornes robustes 1-99 pct des CENTRES, sur les meshes < 60 wu : proxy « aire jouable ».
	var sub []box
	for _, b := range valid {
		if b.ext() < 60 {
			sub = append(sub, b)
		}
	}
	if len(sub) == 0 {
		return
	}
	var rlo, rhi [3]float32
	for a := 0; a < 3; a++ {
		c := make([]float64, len(sub))
		for i, b := range sub {
			c[i] = float64(b.lo[a]+b.hi[a]) / 2
		}
		sort.Float64s(c)
		i99 := len(c) * 99 / 100
		if i99 >= len(c) {
			i99 = len(c) - 1
		}
		rlo[a], rhi[a] = float32(c[len(c)/100]), float32(c[i99])
	}
	fmt.Printf("centres 1-99 pct (< 60 wu)         : X[%9.2f,%9.2f] span %8.2f | Y[%9.2f,%9.2f] span %8.2f | Z[%9.2f,%9.2f] span %8.2f\n",
		rlo[0], rhi[0], rhi[0]-rlo[0], rlo[1], rhi[1], rhi[1]-rlo[1], rlo[2], rhi[2], rhi[2]-rlo[2])
}

// histReport : distribution des centres de mesh par axe. Sert à voir si la géométrie
// extraite forme UN volume compact (une map) ou plusieurs amas dans des repères distincts
// (positions locales de resource, non transformées en monde).
func histReport(meshes []mesh) {
	var valid []box
	for _, m := range meshes {
		if b, ok := worldBox(m); ok && b.ext() < 60 {
			valid = append(valid, b)
		}
	}
	axn := []string{"X", "Y", "Z"}
	for a := 0; a < 3; a++ {
		c := make([]float64, len(valid))
		for i, b := range valid {
			c[i] = float64(b.lo[a]+b.hi[a]) / 2
		}
		sort.Float64s(c)
		p := func(x int) float64 {
			i := len(c) * x / 100
			if i >= len(c) {
				i = len(c) - 1
			}
			return c[i]
		}
		fmt.Printf("centres %s (<60wu) : min %8.2f p05 %8.2f p25 %8.2f p50 %8.2f p75 %8.2f p95 %8.2f max %8.2f\n",
			axn[a], c[0], p(5), p(25), p(50), p(75), p(95), c[len(c)-1])
	}
	// Union des bbox du CŒUR : meshes dont le centre est dans les p2-p98 des 3 axes.
	var core []box
	for _, b := range valid {
		in := true
		for a := 0; a < 3; a++ {
			c := make([]float64, len(valid))
			for i, v := range valid {
				c[i] = float64(v.lo[a]+v.hi[a]) / 2
			}
			sort.Float64s(c)
			lo, hi := c[len(c)*2/100], c[len(c)*98/100]
			m := float64(b.lo[a]+b.hi[a]) / 2
			if m < lo || m > hi {
				in = false
			}
		}
		if in {
			core = append(core, b)
		}
	}
	show("CŒUR (centres p2-p98, <60 wu)", core)
}
