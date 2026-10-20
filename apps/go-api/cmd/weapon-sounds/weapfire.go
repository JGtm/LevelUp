package main

// weapfire.go — ETAPE 3, DERNIER MAILLON : designer le son de TIR par le champ NOMME.
//
// C'est ici que la preuve se ferme. Jusqu'a present on savait qu'un `weap` mene a 8 `lsnd`,
// sans savoir lequel est le tir. `weap.xml` (definition du tag, champs nommes) donne le
// chemin exact :
//
//	_38 "sound"
//	  _40 "Weapon Fire Sounds"                  <- tableau
//	      _38 "Fire Sound"                      (struct inline)
//	          _40 "Weapon Fire Sound Variations"  <- tableau, offset de champ 12
//	              _41 "Weapon Fire Sound"       <- LA reference de tag cherchee
//
// Aucun nom Wwise, aucune heuristique acoustique : le champ s'appelle « Weapon Fire Sound »
// dans la definition du tag, point final.
//
// DUPLICATION ASSUMEE : la lecture du plugin et des tables de tag reprend la mecanique de
// `cmd/weapon-icons-build/weapui.go` (2e copie, la regle du depot en tolere 2). Voir la
// section Decouvertes du plan : une 3e imposerait de promouvoir tout ceci en package.

import (
	_ "embed"
	"encoding/binary"
	"encoding/xml"
	"fmt"
)

//go:embed weap.xml
var pluginWeap []byte

type noeudXML struct {
	XMLName xml.Name
	V       string     `xml:"v,attr"`
	Length  string     `xml:"length,attr"`
	Noeuds  []noeudXML `xml:",any"`
}

// taillesChamps : tailles du format de plugin (identique a weapon-icons-build/weapui.go).
var taillesChamps = map[string]int{
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

type champ struct {
	nom     string
	off     int
	tableau bool
	noeud   noeudXML
}

// parcourir calcule les offsets d'un niveau. `_38` = struct INLINE (meme espace d'offsets),
// `_40` = tableau (20 octets ici, ses champs vivent dans un autre bloc de donnees).
func parcourir(n noeudXML, off int, out *[]champ) int {
	for _, c := range n.Noeuds {
		nom := c.XMLName.Local
		switch nom {
		case "_40":
			*out = append(*out, champ{nom: c.V, off: off, tableau: true, noeud: c})
			off += 20
		case "_38":
			off = parcourir(c, off, out)
		case "Flag", "":
		default:
			l := taillesChamps[nom]
			if c.Length != "" {
				if _, err := fmt.Sscanf(c.Length, "%d", &l); err != nil {
					l = taillesChamps[nom]
				}
			}
			*out = append(*out, champ{nom: c.V, off: off, noeud: c})
			off += l
		}
	}
	return off
}

// champsPlugin rend les champs de premier niveau du tag `weap`.
func champsPlugin() ([]champ, error) {
	var racine noeudXML
	if err := xml.Unmarshal(pluginWeap, &racine); err != nil {
		return nil, err
	}
	var out []champ
	parcourir(racine, 0, &out)
	return out, nil
}

// trouverChamp rend le champ portant ce nom exact.
func trouverChamp(champs []champ, nom string) (champ, bool) {
	for _, c := range champs {
		if c.nom == nom {
			return c, true
		}
	}
	return champ{}, false
}

// ---------------- tables internes d'un tag ----------------

type tagWeap struct {
	octets    []byte
	tailleEnt int
	tabBlocs  int
	tabStruct int
	nBlocs    int
	nStructs  int
}

func ouvrirTagWeap(b []byte) (tagWeap, error) {
	if len(b) < tailleEnteteT || binary.LittleEndian.Uint32(b) != magicUCSH {
		return tagWeap{}, fmt.Errorf("en-tete non-ucsh")
	}
	nDeps := int(int32(binary.LittleEndian.Uint32(b[offNbDeps:])))
	t := tagWeap{
		octets:    b,
		nBlocs:    int(int32(binary.LittleEndian.Uint32(b[0x1c:]))),
		nStructs:  int(int32(binary.LittleEndian.Uint32(b[0x20:]))),
		tailleEnt: int(int32(binary.LittleEndian.Uint32(b[0x38:]))),
	}
	t.tabBlocs = tailleEnteteT + nDeps*tailleDep
	t.tabStruct = t.tabBlocs + t.nBlocs*0x10
	if t.tabStruct+t.nStructs*0x20 > len(b) {
		return tagWeap{}, fmt.Errorf("tables hors bornes")
	}
	return t, nil
}

// blocAbs rend l'offset absolu et la taille d'un bloc de donnees.
func (t tagWeap) blocAbs(idx int) (abs, taille int) {
	b := t.tabBlocs + idx*0x10
	if idx < 0 || b+16 > len(t.octets) {
		return -1, 0
	}
	taille = int(binary.LittleEndian.Uint32(t.octets[b:]))
	abs = int(binary.LittleEndian.Uint64(t.octets[b+8:]))
	if binary.LittleEndian.Uint16(t.octets[b+6:]) != 0 {
		abs += t.tailleEnt
	}
	return abs, taille
}

// blocRacine rend l'index du bloc porte par la MainStruct (type 0).
func (t tagWeap) blocRacine() (int, error) {
	for i := 0; i < t.nStructs; i++ {
		b := t.tabStruct + i*0x20
		if binary.LittleEndian.Uint16(t.octets[b+0x10:]) != 0 {
			continue
		}
		return int(int32(binary.LittleEndian.Uint32(t.octets[b+0x14:]))), nil
	}
	return 0, fmt.Errorf("MainStruct absente")
}

// enfantsDe rend, pour un bloc parent, les tableaux qu'il contient : offset de champ -> bloc.
// C'est cet offset qui permet d'apparier un `_40` du plugin a son bloc de donnees.
func (t tagWeap) enfantsDe(parent int) map[int]int {
	out := map[int]int{}
	for i := 0; i < t.nStructs; i++ {
		b := t.tabStruct + i*0x20
		if binary.LittleEndian.Uint16(t.octets[b+0x10:]) != 1 {
			continue
		}
		if int(int32(binary.LittleEndian.Uint32(t.octets[b+0x18:]))) != parent {
			continue
		}
		fo := int(binary.LittleEndian.Uint32(t.octets[b+0x1C:]))
		if fo < 0 || fo >= 0x10000 {
			continue
		}
		out[fo] = int(int32(binary.LittleEndian.Uint32(t.octets[b+0x14:])))
	}
	return out
}

// gidRef rend l'identifiant de tag porte par un champ `_41` (28 octets, gid a +8).
func gidRef(b []byte, off int) uint32 {
	if off < 0 || off+12 > len(b) {
		return 0
	}
	return binary.LittleEndian.Uint32(b[off+8:])
}
