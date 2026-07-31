// tmp_ridgemesh — dump BRUT des bbox monde par mesh d'un .module (runtime_geo / rawg.xml),
// SANS filtre de taille ni percentile, vers un CSV. Sert à mesurer l'empreinte réelle
// avant tout overlay (cf .ai/HANDOFF_MAP_GEOMETRY_FROM_MODULES.md §9).
//
// Usage : CGO_ENABLED=1 go run ./cmd/tmp_ridgemesh <module> <out.csv>
package main

import (
	"bufio"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/himodule"
)

const pmdStride, offPos, offMin, offMax = 0x90, 0x38, 0x44, 0x50

func u16(b []byte, o int) uint16  { return binary.LittleEndian.Uint16(b[o:]) }
func u32(b []byte, o int) int     { return int(binary.LittleEndian.Uint32(b[o:])) }
func i32(b []byte, o int) int     { return int(int32(binary.LittleEndian.Uint32(b[o:]))) }
func u64(b []byte, o int) int     { return int(binary.LittleEndian.Uint64(b[o:])) }
func f32(b []byte, o int) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(b[o:])) }

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

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: tmp_ridgemesh <module> <out.csv>")
		return
	}
	modPath, outPath := os.Args[1], os.Args[2]
	xmlPath := filepath.Join(filepath.Dir(os.Args[0]), "rawg.xml")
	if _, err := os.Stat(xmlPath); err != nil {
		xmlPath = "cmd/tmp_geores/rawg.xml"
	}
	pmdOff := perMeshDataOffset(xmlPath)
	fmt.Printf("Per Mesh Data offset = %#x (plugin %s)\n", pmdOff, xmlPath)

	m, err := himodule.Open(modPath)
	if err != nil {
		fmt.Println("open:", err)
		return
	}
	all := m.Files("")
	fmt.Printf("module: %d entrées\n", len(all))

	f, err := os.Create(outPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()
	fmt.Fprintln(w, "res,mesh,px,py,pz,lox,loy,loz,hix,hiy,hiz")

	nRes, nMesh, nBad := 0, 0, 0
	var gLo, gHi [3]float64
	for a := 0; a < 3; a++ {
		gLo[a], gHi[a] = math.Inf(1), math.Inf(-1)
	}
	groups := map[string]int{}
	for _, fl := range all {
		data, err := m.Extract(fl)
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
		groups[fmt.Sprintf("%d", size/pmdStride)]++
		nRes++
		for i := 0; i < size/pmdStride; i++ {
			o := abs + i*pmdStride
			var p, lo, hi [3]float64
			bad := false
			for a := 0; a < 3; a++ {
				p[a] = float64(f32(data, o+offPos+a*4))
				lo[a] = p[a] + float64(f32(data, o+offMin+a*4))
				hi[a] = p[a] + float64(f32(data, o+offMax+a*4))
				if math.IsNaN(lo[a]) || math.IsInf(lo[a], 0) || math.IsNaN(hi[a]) || math.IsInf(hi[a], 0) || hi[a] < lo[a] {
					bad = true
				}
			}
			if bad {
				nBad++
				continue
			}
			nMesh++
			for a := 0; a < 3; a++ {
				gLo[a] = math.Min(gLo[a], lo[a])
				gHi[a] = math.Max(gHi[a], hi[a])
			}
			fmt.Fprintf(w, "%d,%d,%.4f,%.4f,%.4f,%.4f,%.4f,%.4f,%.4f,%.4f,%.4f\n",
				fl.Index, i, p[0], p[1], p[2], lo[0], lo[1], lo[2], hi[0], hi[1], hi[2])
		}
	}
	fmt.Printf("%d resources runtime_geo, %d meshes valides (%d rejetés NaN/inversés)\n", nRes, nMesh, nBad)
	fmt.Printf("bbox MONDE globale (aucun filtre) : X[%.2f,%.2f] Y[%.2f,%.2f] Z[%.2f,%.2f]\n",
		gLo[0], gHi[0], gLo[1], gHi[1], gLo[2], gHi[2])
	fmt.Printf("→ %s\n", outPath)
}
