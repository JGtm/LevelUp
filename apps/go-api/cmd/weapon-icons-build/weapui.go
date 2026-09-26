package main

// weapui.go — LE LIEN ARME -> ICONE, lu dans le jeu et auto-valide.
//
// Lit `UI display info` d un tag `weap` pour en sortir le couple
// (sprite = bitmap d atlas, sprite index = position de l icone dans cet atlas).
//
// Mecanique reprise de internal/himap (field-walker version-aware) : le plugin donne l ORDRE
// et les TAILLES des champs, jamais des offsets absolus codes en dur.
//
// L APPARIEMENT PAR RANG A ETE ESSAYE ET ECARTE. Aligner le k-ieme `_40` du plugin sur le
// k-ieme TagBlock du tag donne un bloc DECALE D UNE UNITE : celui trouve portait des
// references mode/jmad/aset, pas des sprites. Corriger par un +1 aurait ete un reglage
// qu aucune mesure ne garantit d une version a l autre. Le bloc est donc identifie PAR SON
// CONTENU : celui dont le champ `sprite` porte l un des deux atlas connus. C est
// auto-validant — si aucun bloc ne matche, la sonde le dit au lieu de rendre un chiffre faux.

import (
	_ "embed"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"sort"

	"levelup/go-api/internal/games/weapons"
)

// ---------------- plugin ----------------

type xnode struct {
	XMLName xml.Name
	V       string  `xml:"v,attr"`
	Length  string  `xml:"length,attr"`
	Nodes   []xnode `xml:",any"`
}

// sizeTab : tailles de champs du format de plugin (group_lengths_dict), copiee de
// internal/himap/sbsp.go — meme format de plugin, meme table.
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

type pfield struct {
	name  string
	off   int
	block bool
	node  xnode
}

// walkFields parcourt un niveau du plugin. Les `_38` sont des structs INLINE : on recurse
// dans le meme espace d offsets. Les `_40` sont des tableaux : on note leur position et on
// NE recurse PAS (leurs champs vivent dans un autre bloc de donnees).
func walkFields(n xnode, off int, out *[]pfield) int {
	for _, c := range n.Nodes {
		name := c.XMLName.Local
		switch name {
		case "_40":
			*out = append(*out, pfield{name: c.V, off: off, block: true, node: c})
			off += 20
		case "_38":
			off = walkFields(c, off, out)
		case "Flag", "":
			// valeur d enum / commentaire : pas un champ.
		default:
			l := sizeTab[name]
			if c.Length != "" {
				if _, err := fmt.Sscanf(c.Length, "%d", &l); err != nil {
					l = sizeTab[name]
				}
			}
			*out = append(*out, pfield{name: c.V, off: off, node: c})
			off += l
		}
	}
	return off
}

// weapPlugin : definition du tag `weap` (Gamergotten/Infinite-runtime-tagviewer,
// Plugins/weap.xml, 479 definitions dumpees par Lord Zedd et Exhibit). Embarquee comme
// internal/himap embarque sbsp.xml : le binaire doit pouvoir tourner sans depot tiers.
//
//go:embed weap.xml
var weapPlugin []byte

func loadWeapPlugin() ([]pfield, error) {
	var root xnode
	if err := xml.Unmarshal(weapPlugin, &root); err != nil {
		return nil, err
	}
	var all []pfield
	walkFields(root, 0, &all)
	return all, nil
}

// uiBlockOffsets rend les offsets internes des champs du bloc `UI display info`.
func uiBlockOffsets(fields []pfield) (offSprite, offIndex, offName int, ok bool) {
	offSprite, offIndex, offName = -1, -1, -1
	for _, f := range fields {
		if !f.block || f.name != "UI display info" {
			continue
		}
		var inner []pfield
		walkFields(f.node, 0, &inner)
		for _, g := range inner {
			switch g.name {
			case "sprite":
				offSprite = g.off
			case "sprite index":
				offIndex = g.off
			case "name":
				offName = g.off
			}
		}
	}
	return offSprite, offIndex, offName, offSprite >= 0 && offIndex >= 0
}

// ---------------- tag : tables ----------------

type weapTag struct {
	tag        []byte
	headerSize int
	blockTab   int
	structTab  int
	structs    int
	blocks     int
}

func openWeapTag(tag []byte) (weapTag, error) {
	h, ok := parseTagHeader(tag)
	if !ok {
		return weapTag{}, fmt.Errorf("en-tete non-ucsh")
	}
	wt := weapTag{
		tag: tag, headerSize: int(h.HeaderSize),
		structs: int(h.StructCount), blocks: int(h.BlockCount),
	}
	wt.blockTab = tagHdrFixed + int(h.DepCount)*depStride
	wt.structTab = wt.blockTab + wt.blocks*0x10
	if wt.structTab+wt.structs*0x20 > len(tag) {
		return weapTag{}, fmt.Errorf("tables hors bornes")
	}
	return wt, nil
}

func (w weapTag) blockAbs(idx int) (abs, size int) {
	b := w.blockTab + idx*0x10
	if idx < 0 || b+16 > len(w.tag) {
		return -1, 0
	}
	size = int(binary.LittleEndian.Uint32(w.tag[b:]))
	abs = int(binary.LittleEndian.Uint64(w.tag[b+8:]))
	if binary.LittleEndian.Uint16(w.tag[b+6:]) != 0 {
		abs += w.headerSize
	}
	return abs, size
}

// rootBlock : le bloc porte par la MainStruct (type 0).
func (w weapTag) rootBlock() (int, error) {
	for i := 0; i < w.structs; i++ {
		b := w.structTab + i*0x20
		if binary.LittleEndian.Uint16(w.tag[b+0x10:]) != 0 {
			continue
		}
		return int(int32(binary.LittleEndian.Uint32(w.tag[b+0x14:]))), nil
	}
	return 0, fmt.Errorf("MainStruct absente")
}

// tagBlocksOf rend, pour un bloc parent, les blocs cibles des TagBlocks (type 1) qui le
// referencent, tries par offset de champ croissant.
func (w weapTag) tagBlocksOf(parent int) []int {
	type ent struct{ fo, target int }
	var ents []ent
	seen := map[int]bool{}
	for i := 0; i < w.structs; i++ {
		b := w.structTab + i*0x20
		if binary.LittleEndian.Uint16(w.tag[b+0x10:]) != 1 {
			continue
		}
		if int(int32(binary.LittleEndian.Uint32(w.tag[b+0x18:]))) != parent {
			continue
		}
		fo := int(binary.LittleEndian.Uint32(w.tag[b+0x1C:]))
		if fo < 0 || fo >= 0x10000 || seen[fo] {
			continue
		}
		seen[fo] = true
		ents = append(ents, ent{fo, int(int32(binary.LittleEndian.Uint32(w.tag[b+0x14:])))})
	}
	sort.Slice(ents, func(i, j int) bool { return ents[i].fo < ents[j].fo })
	out := make([]int, 0, len(ents))
	for _, e := range ents {
		out = append(out, e.target)
	}
	return out
}

// refGID : identifiant de tag porte par un champ `_41` (28 octets). Position CALIBREE sur
// des references connues — le `mode` du BR75 (a2777b87) apparait a +8 du debut du champ.
func refGID(tag []byte, off int) uint32 {
	if off < 0 || off+12 > len(tag) {
		return 0
	}
	return binary.LittleEndian.Uint32(tag[off+8:])
}

// findUIBlock trouve le bloc `UI display info` par son CONTENU. Voir l en-tete du fichier.
func findUIBlock(wt weapTag, data []byte, offSprite int) (abs, size int, atlas uint32, ok bool) {
	root, err := wt.rootBlock()
	if err != nil {
		return 0, 0, 0, false
	}
	for _, blk := range wt.tagBlocksOf(root) {
		a, s := wt.blockAbs(blk)
		if a < 0 || s < offSprite+28 || a+s > len(data) {
			continue
		}
		g := refGID(data, a+offSprite)
		if g == 0xbc17adf1 || g == 0xe39747c8 {
			return a, s, g, true
		}
	}
	return 0, 0, 0, false
}

// ---------------- résolution ----------------

// weaponIcon : ce que le jeu déclare pour une arme du registre.
type weaponIcon struct {
	Key      string `json:"arme"`
	WeapTag  string `json:"weap_tag"`
	Atlas    string `json:"atlas_tag"`
	Index    int    `json:"sprite_index"`
	NameSID  string `json:"name_string_id"`
	AltIndex int    `json:"alt_sprite_index"`
}

// resolveWeaponIcons lit, pour chaque arme du registre, le bloc `UI display info` de son tag
// `weap` et en sort l'index de son icône dans l'atlas.
//
// Chaque ligne rendue est AUTO-VALIDÉE : elle n'existe que si le champ `sprite` du bloc porte
// l'un des deux atlas connus. Une arme dont le walk échoue est absente du résultat plutôt que
// présente avec un index arbitraire — l'inverse serait invisible et faux.
func resolveWeaponIcons(ix *tagIndex) ([]weaponIcon, error) {
	fields, err := loadWeapPlugin()
	if err != nil {
		return nil, fmt.Errorf("plugin weap.xml illisible: %w", err)
	}
	offSprite, offIndex, offName, ok := uiBlockOffsets(fields)
	if !ok {
		return nil, fmt.Errorf("bloc « UI display info » ou ses champs absents du plugin")
	}
	// `alt sprite index` suit `alt sprite` (28 o) qui suit `sprite index` (4 o).
	offAlt := offIndex + 4 + 28
	var out []weaponIcon
	for _, wf := range weaponFamilies() {
		refs := ix.byID[wf.ID]
		if len(refs) == 0 {
			continue
		}
		data, err := ix.extract(refs[0])
		if err != nil {
			continue
		}
		wt, err := openWeapTag(data)
		if err != nil {
			continue
		}
		abs, _, atlas, found := findUIBlock(wt, data, offSprite)
		if !found || abs+offAlt+4 > len(data) {
			continue
		}
		w := weaponIcon{
			Key:      wf.Key,
			WeapTag:  fmt.Sprintf("%08x", wf.ID),
			Atlas:    fmt.Sprintf("%08x", atlas),
			Index:    int(binary.LittleEndian.Uint32(data[abs+offIndex:])),
			AltIndex: int(binary.LittleEndian.Uint32(data[abs+offAlt:])),
		}
		if offName >= 0 && abs+offName+4 <= len(data) {
			w.NameSID = fmt.Sprintf("%08x", binary.LittleEndian.Uint32(data[abs+offName:]))
		}
		out = append(out, w)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out, nil
}

// weaponFamilies rend les armes du registre triées par weapon_key, avec le global tag id du
// `weap` (32 bits hauts de l'identifiant filmshell — cf. l'en-tête de main.go).
func weaponFamilies() []struct {
	Key string
	ID  uint32
} {
	byFam := weapons.FilmshellWeaponKeysByFamily()
	out := make([]struct {
		Key string
		ID  uint32
	}, 0, len(byFam))
	for id, key := range byFam {
		out = append(out, struct {
			Key string
			ID  uint32
		}{key, id})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}
