// Package himap lit, dans les archives .module d'une carte Halo Infinite, les
// BORNES MONDE (`world bounds x/y/z`) portées par chaque tag scenario_structure_bsp.
//
// POURQUOI. Le moteur quantifie la position répliquée d'un objet dans la boîte du
// BSP courant : au chargement de carte (FUN_140be9a14 / FUN_140be9b88) il précalcule
// pour chaque région de compression r et chaque niveau de précision L
//
//	W[r][L][axe] = min(26, ceilLog2(ceil(extent_axe / (2*q(L)))))   q(L) = 2^(16-L)/120
//
// et la position d'objet utilise L = 16 câblé au site d'appel, d'où
// W = min(26, ceilLog2(ceil(60*extent))). Ni les bornes ni les largeurs ne transitent
// par le film : elles ne se lisent que dans le module de la carte. Sans elles, un
// quantum décodé n'est PAS une coordonnée monde.
//
// COMMENT (robustesse aux versions). Les offsets de champs bougent entre versions de
// Halo : rien n'est codé en dur. Le plugin sbsp.xml donne l'ORDRE des champs ; les
// tag-blocks `_40` du root sont appariés PAR RANG avec les TagBlocks de la
// struct-table du tag, et l'offset réel de `world bounds x` est déduit du tag-block
// qui le précède. La déduction n'est acceptée que si TOUS les écarts entre
// tag-blocks jusqu'à l'encadrement sont conservés entre le plugin et le tag.
package himap

import (
	_ "embed"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"math"
	"sort"
	"sync"

	"levelup/go-api/internal/himodule"
)

//go:embed sbsp.xml
var sbspPlugin []byte

// Bounds est l'AABB monde d'un BSP, en unités monde, axe par axe.
type Bounds struct {
	Min [3]float64 `json:"min"`
	Max [3]float64 `json:"max"`
}

// Extent est l'étendue de l'axe ax.
func (b Bounds) Extent(ax int) float64 { return b.Max[ax] - b.Min[ax] }

// Valid signale une AABB exploitable (non dégénérée, non aberrante).
func (b Bounds) Valid() bool {
	for ax := 0; ax < 3; ax++ {
		e := b.Extent(ax)
		if !(e > 0) || math.IsNaN(e) || math.IsInf(e, 0) || e > 1e6 {
			return false
		}
	}
	return true
}

// PositionLevel est le niveau de précision câblé au site d'appel de la position
// d'objet répliquée (MOV R9D,0x10 en 1406d008a).
const PositionLevel = 16

// maxAxisWidth est le plafond moteur de la largeur d'un axe quantifié.
const maxAxisWidth = 26

// quantDivisor : q(L) = 2^(16-L)/120, donc à L=16 le pas de grille vaut 2*q = 1/60.
const quantDivisor = 120

// AxisWidths applique la loi du moteur au niveau PositionLevel :
// W = min(26, ceilLog2(ceil(60*extent))).
func (b Bounds) AxisWidths() [3]int {
	var w [3]int
	inv := float64(quantDivisor) / 2 / math.Exp2(float64(16-PositionLevel))
	for ax := 0; ax < 3; ax++ {
		w[ax] = capWidth(ceilLog2(uint64(math.Ceil(inv * b.Extent(ax)))))
	}
	return w
}

func capWidth(w int) int {
	if w > maxAxisWidth {
		return maxAxisWidth
	}
	return w
}

func ceilLog2(v uint64) int {
	if v <= 1 {
		return 0
	}
	n := 0
	for x := v - 1; x > 0; x >>= 1 {
		n++
	}
	return n
}

// BSP est un tag scenario_structure_bsp d'un module, avec ses bornes monde.
type BSP struct {
	// FileIndex est l'index de l'entrée dans le .module.
	FileIndex int
	// UncompSize est la taille décompressée du tag (sert à désigner le BSP principal).
	UncompSize int
	// FieldOffset est l'offset RÉEL de `world bounds x` dans le root block (déduit).
	FieldOffset int
	Bounds      Bounds
}

// ReadModuleBSPBounds ouvre un .module et renvoie les bornes monde de tous ses tags
// sbsp, triés par taille décroissante (le BSP principal en premier).
func ReadModuleBSPBounds(modulePath string) ([]BSP, error) {
	m, err := himodule.Open(modulePath)
	if err != nil {
		return nil, err
	}
	files := m.Files("sbsp")
	sort.Slice(files, func(i, j int) bool { return files[i].UncompSize > files[j].UncompSize })
	out := make([]BSP, 0, len(files))
	var firstErr error
	for _, f := range files {
		tag, err := m.Extract(f)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("sbsp#%d: %w", f.Index, err)
			}
			continue
		}
		b, off, err := boundsFromTag(tag)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("sbsp#%d: %w", f.Index, err)
			}
			continue
		}
		out = append(out, BSP{FileIndex: f.Index, UncompSize: f.UncompSize, FieldOffset: off, Bounds: b})
	}
	if len(out) == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, fmt.Errorf("himap: aucun tag sbsp dans %s", modulePath)
	}
	return out, nil
}

// ---------------- plugin : ordre des champs ----------------

type xnode struct {
	XMLName xml.Name
	V       string  `xml:"v,attr"`
	Length  string  `xml:"length,attr"`
	Nodes   []xnode `xml:",any"`
}

// sizeTab = table des tailles de champs du format de plugin (group_lengths_dict).
// `_38` = struct inline (récursion), `_40` = tag-block (20 o), `_25` = real bounds (8 o).
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

type pluginField struct {
	name  string
	off   int
	block bool
}

var (
	pluginOnce   sync.Once
	pluginBlocks []pluginField
	pluginWBOff  = -1
	pluginErr    error
)

func loadPlugin() {
	var root xnode
	if err := xml.Unmarshal(sbspPlugin, &root); err != nil {
		pluginErr = fmt.Errorf("himap: plugin sbsp.xml illisible: %w", err)
		return
	}
	var all []pluginField
	walkPlugin(root, 0, &all)
	for _, f := range all {
		if f.block {
			pluginBlocks = append(pluginBlocks, f)
			continue
		}
		if f.name == "world bounds x" {
			pluginWBOff = f.off
		}
	}
	if pluginWBOff < 0 || len(pluginBlocks) == 0 {
		pluginErr = fmt.Errorf("himap: `world bounds x` absent du plugin sbsp.xml")
	}
}

func walkPlugin(n xnode, off int, out *[]pluginField) int {
	for _, c := range n.Nodes {
		name := c.XMLName.Local
		switch name {
		case "_40":
			*out = append(*out, pluginField{name: c.V, off: off, block: true})
			off += 20
		case "_25":
			*out = append(*out, pluginField{name: c.V, off: off})
			off += 8
		case "_38":
			off = walkPlugin(c, off, out)
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

// ---------------- tag : header + struct-table ----------------

func u16(b []byte, o int) uint16 { return binary.LittleEndian.Uint16(b[o:]) }
func u32(b []byte, o int) int    { return int(binary.LittleEndian.Uint32(b[o:])) }
func i32(b []byte, o int) int    { return int(int32(binary.LittleEndian.Uint32(b[o:]))) }
func u64(b []byte, o int) int    { return int(binary.LittleEndian.Uint64(b[o:])) }
func f32(b []byte, o int) float64 {
	return float64(math.Float32frombits(binary.LittleEndian.Uint32(b[o:])))
}

type tagInfo struct {
	tag        []byte
	headerSize int
	deps       int
	dataBlocks int
	structs    int
	blockTab   int
	structTab  int
}

// tagHeaderScan : bornes du balayage du couple (headerSize, dataSize) dans l'en-tête.
const (
	tagHeaderScanFrom = 0x28
	tagHeaderScanTo   = 0x80
	depEntrySize      = 0x18
	blockEntrySize    = 0x10
	structEntrySize   = 0x20
)

// tagCandidates énumère les positions plausibles du couple (headerSize, dataSize).
// La somme vaut la taille du tag, éventuellement augmentée d'une 3e section
// (certains sbsp, ex. ctf_bazaar, en portent une). L'ambiguïté est levée en aval par
// la validation structurelle.
func tagCandidates(tag []byte) []tagInfo {
	var out []tagInfo
	for _, withThird := range []bool{false, true} {
		for H := tagHeaderScanFrom; H <= tagHeaderScanTo && H+12 <= len(tag); H += 4 {
			hs, ds := u32(tag, H), u32(tag, H+4)
			if hs <= 0 || ds <= 0 || hs >= len(tag) {
				continue
			}
			sum := hs + ds
			if withThird {
				rs := u32(tag, H+8)
				if rs <= 0 {
					continue
				}
				sum += rs
			}
			if sum != len(tag) {
				continue
			}
			ti := tagInfo{
				tag:        tag,
				headerSize: hs,
				deps:       u32(tag, H-0x20),
				dataBlocks: u32(tag, H-0x1C),
				structs:    u32(tag, H-0x18),
			}
			ti.blockTab = H + 0x18 + ti.deps*depEntrySize
			ti.structTab = ti.blockTab + ti.dataBlocks*blockEntrySize
			if ti.structs <= 0 || ti.dataBlocks <= 0 || ti.deps < 0 ||
				ti.structTab+ti.structs*structEntrySize > len(tag) {
				continue
			}
			out = append(out, ti)
		}
	}
	return out
}

func (ti tagInfo) blockAbs(idx int) (int, int) {
	b := ti.blockTab + idx*blockEntrySize
	size := u32(ti.tag, b)
	abs := u64(ti.tag, b+8)
	if u16(ti.tag, b+6) != 0 {
		abs += ti.headerSize
	}
	return abs, size
}

func boundsFromTag(tag []byte) (Bounds, int, error) {
	pluginOnce.Do(loadPlugin)
	if pluginErr != nil {
		return Bounds{}, 0, pluginErr
	}
	cands := tagCandidates(tag)
	if len(cands) == 0 {
		return Bounds{}, 0, fmt.Errorf("en-tête de tag non reconnu")
	}
	var last error
	for _, ti := range cands {
		b, off, err := boundsFromTagInfo(ti)
		if err == nil {
			return b, off, nil
		}
		last = err
	}
	return Bounds{}, 0, last
}

func boundsFromTagInfo(ti tagInfo) (Bounds, int, error) {
	tag := ti.tag
	rootBlock := -1
	for i := 0; i < ti.structs; i++ {
		b := ti.structTab + i*structEntrySize
		if u16(tag, b+0x10) == 0 { // MainStruct
			rootBlock = i32(tag, b+0x14)
			break
		}
	}
	if rootBlock < 0 || ti.blockTab+(rootBlock+1)*blockEntrySize > len(tag) {
		return Bounds{}, 0, fmt.Errorf("root block introuvable")
	}
	// TagBlocks (type 1) référencés depuis le root block, triés par field_offset.
	var offs []int
	seen := map[int]bool{}
	for i := 0; i < ti.structs; i++ {
		b := ti.structTab + i*structEntrySize
		if u16(tag, b+0x10) != 1 || i32(tag, b+0x18) != rootBlock {
			continue
		}
		fo := u32(tag, b+0x1C)
		if fo < 0 || fo >= 0x10000 || seen[fo] {
			continue
		}
		seen[fo] = true
		offs = append(offs, fo)
	}
	sort.Ints(offs)
	if len(offs) > len(pluginBlocks) {
		return Bounds{}, 0, fmt.Errorf("rang incompatible: plugin %d blocs / tag %d refs",
			len(pluginBlocks), len(offs))
	}
	// dernier tag-block du plugin situé AVANT `world bounds x`
	k := -1
	for i, b := range pluginBlocks {
		if b.off < pluginWBOff {
			k = i
		}
	}
	if k < 0 || k+1 >= len(offs) {
		return Bounds{}, 0, fmt.Errorf("`world bounds x` hors encadrement")
	}
	// Validation : tous les écarts entre tag-blocks jusqu'à l'encadrement doivent être
	// conservés — sinon le préfixe du root block a changé de forme et l'interpolation
	// d'offset serait fausse.
	for i := 0; i <= k; i++ {
		if pluginBlocks[i+1].off-pluginBlocks[i].off != offs[i+1]-offs[i] {
			return Bounds{}, 0, fmt.Errorf("écart de tag-blocks non conservé au rang %d", i)
		}
	}
	fieldOff := offs[k] + (pluginWBOff - pluginBlocks[k].off)
	rootAbs, rootSize := ti.blockAbs(rootBlock)
	if rootAbs < 0 || fieldOff+24 > rootSize || rootAbs+fieldOff+24 > len(tag) {
		return Bounds{}, 0, fmt.Errorf("offset hors du root block")
	}
	base := rootAbs + fieldOff
	var b Bounds
	for ax := 0; ax < 3; ax++ {
		b.Min[ax] = f32(tag, base+ax*8)
		b.Max[ax] = f32(tag, base+ax*8+4)
	}
	return b, fieldOff, nil
}
