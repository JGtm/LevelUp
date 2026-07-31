// tmp_walker — localise `compression info` (bbox déquant par mesh) par MATCHING
// D'ORDRE entre le plugin sbsp.xml (liste ordonnée des champs tag-block `_40` du
// root) et la struct-table du tag (TagBlocks référencés depuis le root-block,
// triés par field_offset). Robuste aux décalages d'offset entre versions HI.
//
// Puis lit le bloc compression info → bbox par mesh, et déquantifie un buffer de
// positions int16 d'une resource pour rendre la carte 2D.
//
// Usage : CGO_ENABLED=1 go run ./cmd/tmp_walker ["<module>"] [resIndex]
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

func u16(b []byte, o int) uint16 { return binary.LittleEndian.Uint16(b[o:]) }
func u32(b []byte, o int) int    { return int(binary.LittleEndian.Uint32(b[o:])) }
func i32(b []byte, o int) int    { return int(int32(binary.LittleEndian.Uint32(b[o:]))) }
func u64(b []byte, o int) int    { return int(binary.LittleEndian.Uint64(b[o:])) }
func f32(b []byte, o int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b[o:]))
}

// ---- 1. plugin sbsp.xml : walk avec calcul d'OFFSET (table de tailles IRTV) ----
type xnode struct {
	XMLName xml.Name
	V       string  `xml:"v,attr"`
	Length  string  `xml:"length,attr"`
	Nodes   []xnode `xml:",any"`
}

// sizeTab = group_lengths_dict (IRTV TagLayouts.cs). _38 struct=0 (recurse inline),
// _40 block=20, _34/_35 = attr length.
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

// walk somme les tailles des champs ; recurse SEULEMENT dans `_38` (struct inline).
// Enregistre chaque `_40` (tag-block) avec son offset.
func walk(n xnode, off int, out *[]field) int {
	for _, c := range n.Nodes {
		name := c.XMLName.Local
		switch name {
		case "_40":
			*out = append(*out, field{c.V, off})
			off += 20
		case "_38":
			off = walk(c, off, out)
		case "_34", "_35":
			l := sizeTab[name]
			if c.Length != "" {
				fmt.Sscanf(c.Length, "%d", &l)
			}
			off += l
		case "Flag", "":
			// valeur d'enum/flag ou commentaire : pas un champ.
		default:
			off += sizeTab[name]
		}
	}
	return off
}

func loadXMLFields(path string) ([]field, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var root xnode
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	var fields []field
	walk(root, 0, &fields)
	return fields, nil
}

// ---- 2. tag : header + struct-table + data-blocks ----
type tagInfo struct {
	tag         []byte
	headerSize  int
	deps        int
	dataBlocks  int
	structs     int
	tablesStart int
	structTab   int
	blockTab    int
}

func parseTag(tag []byte) (tagInfo, bool) {
	for H := 0x28; H <= 0x44; H += 4 {
		if H+8 > len(tag) {
			break
		}
		hs, ds := u32(tag, H), u32(tag, H+4)
		if hs > 0 && ds > 0 && hs+ds == len(tag) {
			ti := tagInfo{
				tag:        tag,
				headerSize: hs,
				deps:       u32(tag, H-0x20),
				dataBlocks: u32(tag, H-0x1C),
				structs:    u32(tag, H-0x18),
			}
			ti.tablesStart = H + 0x18
			ti.blockTab = ti.tablesStart + ti.deps*0x18
			ti.structTab = ti.blockTab + ti.dataBlocks*0x10
			return ti, true
		}
	}
	return tagInfo{}, false
}

// blockAbs renvoie (abs, size) du data-block d'index idx.
func (ti tagInfo) blockAbs(idx int) (int, int) {
	b := ti.blockTab + idx*0x10
	size := u32(ti.tag, b)
	section := int(u16(ti.tag, b+6))
	off := u64(ti.tag, b+8)
	abs := off
	if section != 0 {
		abs = ti.headerSize + off
	}
	return abs, size
}

type tstruct struct {
	typ         int
	targetIndex int
	fieldBlock  int
	fieldOffset int
}

func (ti tagInfo) structAt(i int) tstruct {
	b := ti.structTab + i*0x20
	return tstruct{
		typ:         int(u16(ti.tag, b+0x10)),
		targetIndex: i32(ti.tag, b+0x14),
		fieldBlock:  i32(ti.tag, b+0x18),
		fieldOffset: u32(ti.tag, b+0x1C),
	}
}

func main() {
	modPath := defMod
	if len(os.Args) > 1 {
		modPath = os.Args[1]
	}
	xmlPath := filepath.Join(filepath.Dir(os.Args[0]), "sbsp.xml")
	if _, err := os.Stat(xmlPath); err != nil {
		xmlPath = "cmd/tmp_walker/sbsp.xml"
	}

	fields, err := loadXMLFields(xmlPath)
	if err != nil {
		fmt.Printf("xml: %v\n", err)
		return
	}
	ciOff := -1
	for _, f := range fields {
		if f.name == "compression info" {
			ciOff = f.off
		}
	}
	fmt.Printf("plugin sbsp.xml : %d champs tag-block au root ; 'compression info' offset calculé = %#x\n", len(fields), ciOff)

	m, err := himodule.Open(modPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	sbsps := m.Files("sbsp")
	sort.Slice(sbsps, func(i, j int) bool { return sbsps[i].UncompSize > sbsps[j].UncompSize })
	tag, err := m.Extract(sbsps[0])
	if err != nil {
		fmt.Println(err)
		return
	}
	ti, ok := parseTag(tag)
	if !ok {
		fmt.Println("header non parsé")
		return
	}
	fmt.Printf("tag : structs=%d dataBlocks=%d structTab=%#x\n", ti.structs, ti.dataBlocks, ti.structTab)

	// root block = target de la MainStruct (type 0).
	rootBlock := -1
	for i := 0; i < ti.structs; i++ {
		if ti.structAt(i).typ == 0 {
			rootBlock = ti.structAt(i).targetIndex
			break
		}
	}
	fmt.Printf("root block = %d\n", rootBlock)

	// TagBlocks (type 1) référencés depuis le root block, par field_offset.
	refByOff := map[int]int{} // field_offset -> target block
	for i := 0; i < ti.structs; i++ {
		s := ti.structAt(i)
		if s.typ == 1 && s.fieldBlock == rootBlock {
			refByOff[s.fieldOffset] = s.targetIndex
		}
	}
	fmt.Printf("struct-table : %d TagBlocks depuis root block\n", len(refByOff))

	// VALIDATION version : combien d'offsets calculés (XML) ont un ref struct-table ?
	matched := 0
	for _, f := range fields {
		if _, ok := refByOff[f.off]; ok {
			matched++
		}
	}
	fmt.Printf("validation : %d/%d offsets XML matchent un ref struct-table (≈100%% = version compatible)\n", matched, len(fields))

	_ = ciOff
	// MAPPING PAR RANG : refs triés par offset ; le k-ième ref = le k-ième champ XML
	// (l'ordre des champs est stable entre versions ; les tailles diffèrent mais pas l'ordre).
	var offs []int
	for o := range refByOff {
		offs = append(offs, o)
	}
	sort.Ints(offs)
	nameBlock := map[string]int{}
	for i, f := range fields {
		if i < len(offs) {
			nameBlock[f.name] = refByOff[offs[i]]
		}
	}
	_ = nameBlock
	fmt.Println("\n=== alignement complet par rang (champ XML ↔ block) ===")
	for i, f := range fields {
		if i >= len(offs) {
			fmt.Printf("  %2d  %-40s (pas de ref struct-table)\n", i, f.name)
			continue
		}
		t := refByOff[offs[i]]
		sz := -1
		if t >= 0 {
			_, sz = ti.blockAbs(t)
		}
		fmt.Printf("  %2d  off%#-6x  %-40s → block %d [%#x]\n", i, offs[i], f.name, t, sz)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
