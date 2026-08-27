package himap

import (
	"encoding/xml"
	"fmt"
	"sort"
	"testing"
)

// OU SONT LES DRAPEAUX DE FILTRE, ET COMMENT ON LE SAIT.
//
// Ce fichier ne compte rien : il ETABLIT les offsets et les numeros de bit sur lesquels
// repose filtres_reclaimer.go, par deux voies independantes.
//
//	1. le PLUGIN embarque (sbsp.xml) — l'ordre et la taille des champs declares par le jeu ;
//	2. les OCTETS du jeu installe — la distribution des valeurs lues.
//
// Aucune des deux ne suffit seule. Le plugin peut derailler d'un build a l'autre ; une
// lecture qui « donne des chiffres » ne prouve rien tant qu'un offset voisin en donne
// d'aussi plausibles. C'est leur ACCORD qui fait la preuve.

// ---------------------------------------------------------------------------
// 1. GARDE-RAIL DE PLUGIN : les bits sont-ils bien ceux qu'on croit ?
// ---------------------------------------------------------------------------

// champPlugin decrit un champ du plugin : son offset dans l'enregistrement et, si c'est un
// champ de bits, le nom de chacun de ses drapeaux DANS L'ORDRE.
type champPlugin struct {
	nom  string
	off  int
	bits []string
}

// champsDuBloc enumere les champs d'un bloc nomme du plugin, avec leurs offsets.
//
// POURQUOI CE PARCOURS ET PAS `walkPlugin`. Celui de la production ne retient que les
// tag-blocks (c'est tout ce dont il a besoin) et tabule `_39` — un tableau de taille fixe —
// a 32 octets forfaitaires. Ici on SOMME les enfants du `_39`, ce qui est la seule facon
// d'obtenir l'offset des champs qui le SUIVENT : `vertex buffer indices` declare 19 entrees
// de 2 octets, pas 16.
func champsDuBloc(t *testing.T, bloc string) ([]champPlugin, int) {
	t.Helper()
	var root xnode
	if err := xml.Unmarshal(sbspPlugin, &root); err != nil {
		t.Fatal(err)
	}
	n := findPluginNode(root, bloc)
	if n == nil {
		t.Fatalf("bloc %q absent du plugin sbsp.xml", bloc)
	}
	var out []champPlugin
	return out, marcheChamps(*n, 0, &out)
}

func marcheChamps(n xnode, off int, out *[]champPlugin) int {
	for _, c := range n.Nodes {
		nom := c.XMLName.Local
		switch nom {
		case "Flag", "":
			continue
		case "_38": // struct inline : les champs continuent au meme niveau
			off = marcheChamps(c, off, out)
			continue
		}
		champ := champPlugin{nom: c.V, off: off}
		for _, f := range c.Nodes {
			if f.XMLName.Local == "Flag" {
				champ.bits = append(champ.bits, f.V)
			}
		}
		*out = append(*out, champ)
		off += tailleChamp(c)
	}
	return off
}

// tailleChamp rend la taille d'un champ du plugin, `_39` compris (somme des enfants).
func tailleChamp(c xnode) int {
	nom := c.XMLName.Local
	switch nom {
	case "_39":
		n := 0
		for _, f := range c.Nodes {
			n += tailleChamp(f)
		}
		return n
	case "_34", "_35":
		l := sizeTab[nom]
		if c.Length != "" {
			if _, err := fmt.Sscanf(c.Length, "%d", &l); err != nil {
				l = sizeTab[nom]
			}
		}
		return l
	default:
		return sizeTab[nom]
	}
}

func champParNom(t *testing.T, champs []champPlugin, nom string) champPlugin {
	t.Helper()
	for _, c := range champs {
		if c.nom == nom {
			return c
		}
	}
	t.Fatalf("champ %q absent du bloc", nom)
	return champPlugin{}
}

func exigeBit(t *testing.T, c champPlugin, bit int, attendu string) {
	t.Helper()
	if bit >= len(c.bits) {
		t.Fatalf("champ %q : %d drapeaux declares, bit %d demande", c.nom, len(c.bits), bit)
	}
	if c.bits[bit] != attendu {
		t.Fatalf("champ %q bit %d = %q, attendu %q", c.nom, bit, c.bits[bit], attendu)
	}
}

// TestPluginDrapeauxDeFiltre verrouille l'accord entre les constantes de filtres_reclaimer.go
// et le plugin embarque. Une renumerotation de drapeau entre builds du jeu doit CASSER ICI,
// pas produire une carte silencieusement amputee.
func TestPluginDrapeauxDeFiltre(t *testing.T) {
	ins, strideIns := champsDuBloc(t, instancesBlockName)
	if strideIns != instanceStride {
		t.Fatalf("pas du bloc instances = %d, attendu %d", strideIns, instanceStride)
	}
	flags := champParNom(t, ins, "flags")
	if flags.off != insOffFlags {
		t.Fatalf("`flags` @%d, attendu %#x", flags.off, insOffFlags)
	}
	exigeBit(t, flags, 12, "exclude from intel map")
	if FlagExcludeFromIntelMap != 1<<12 {
		t.Fatalf("FlagExcludeFromIntelMap = %#x, attendu le bit 12", FlagExcludeFromIntelMap)
	}

	sect, strideSect := champsDuBloc(t, "meshes")
	if strideSect != sectionStride {
		t.Fatalf("pas du bloc meshes = %d, attendu %d", strideSect, sectionStride)
	}
	mf := champParNom(t, sect, "mesh flags")
	if mf.off != sectionOffMeshFlags {
		t.Fatalf("`mesh flags` @%d, attendu %d", mf.off, sectionOffMeshFlags)
	}
	exigeBit(t, mf, 3, "mesh is custom shadow caster")

	lod, strideLod := champsDuBloc(t, "LOD render data")
	ib := champParNom(t, lod, "index buffer index")
	if ib.off != lodOffIndexBuffer {
		t.Fatalf("`index buffer index` @%d, attendu %#x — le report du `_39` a bouge", ib.off, lodOffIndexBuffer)
	}
	lf := champParNom(t, lod, "lod flags")
	if lf.off != lodOffLodFlags {
		t.Fatalf("`lod flags` @%d, attendu %#x", lf.off, lodOffLodFlags)
	}
	lrf := champParNom(t, lod, "lod render flags")
	if lrf.off != lodOffLodRenderFlags {
		t.Fatalf("`lod render flags` @%d, attendu %#x", lrf.off, lodOffLodRenderFlags)
	}
	exigeBit(t, lrf, 0, "LOD has shadow proxies")
	// Le plugin s'arrete 4 octets avant le pas REEL. On le CONSTATE au lieu de le taire :
	// c'est la trace du champ non declare qui suit `lod render flags`.
	t.Logf("`LOD render data` : pas plugin %d, pas reel %d (%d octets non declares en queue)",
		strideLod, lodStride, lodStride-strideLod)
}

// ---------------------------------------------------------------------------
// 2. TEMOIN D'OFFSETS : ce que disent les octets du jeu
// ---------------------------------------------------------------------------

// TestTemoinOffsetsLOD departage l'offset de `lod render flags` sur les octets du jeu.
//
// LE TEMOIN QUI SEPARE : un champ a UN SEUL drapeau declare ne peut valoir que 0 ou 1. On lit
// donc le meme enregistrement a plusieurs offsets concurrents et on regarde lequel respecte
// cette contrainte. Une lecture qui « donne des chiffres » ne prouve rien ; une lecture qui
// donne exactement la distribution attendue la ou les voisines l'etalent, si.
func TestTemoinOffsetsLOD(t *testing.T) {
	assets := assetsNatifs(t, "ctf_forbidden", 80)
	if len(assets) == 0 {
		t.Skip("aucun tag rtgo lisible")
	}
	offsets := []int{0x84, 0x86, 0x88, 0x8A, 0x8C, 0x8E, 0x90, 0x92}
	hist := map[int]map[uint16]int{}
	for _, o := range offsets {
		hist[o] = map[uint16]int{}
	}
	nLods := 0
	for _, a := range assets {
		for mi := 0; mi < a.MeshCount(); mi++ {
			for k := 0; k < a.NbLODsDuMaillage(mi); k++ {
				nLods++
				for _, o := range offsets {
					if v, ok := a.U16DuLOD(mi, k, o); ok {
						hist[o][v]++
					}
				}
			}
		}
	}
	if nLods == 0 {
		t.Skip("aucun enregistrement de LOD")
	}
	t.Logf("%d enregistrements de LOD sur %d tags rtgo", nLods, len(assets))
	for _, o := range offsets {
		t.Logf("  u16 @%#04x : %2d valeurs distinctes | %s", o, len(hist[o]), sommetsHist(hist[o], 6))
	}
	binaire := func(o int) bool {
		for v := range hist[o] {
			if v > 1 {
				return false
			}
		}
		return true
	}
	if !binaire(lodOffLodRenderFlags) {
		t.Fatalf("`lod render flags` @%#x prend des valeurs > 1 : l'offset est faux", lodOffLodRenderFlags)
	}
	for _, o := range []int{0x8A, 0x8C, 0x90} {
		if binaire(o) {
			t.Fatalf("@%#x est binaire aussi : le temoin ne separe plus, l'offset n'est pas etabli", o)
		}
	}
}

func sommetsHist(h map[uint16]int, n int) string {
	type vc struct {
		v uint16
		n int
	}
	vs := make([]vc, 0, len(h))
	for v, c := range h {
		vs = append(vs, vc{v, c})
	}
	sort.Slice(vs, func(i, j int) bool { return vs[i].n > vs[j].n })
	s := ""
	for i, e := range vs {
		if i >= n {
			break
		}
		s += fmt.Sprintf("%#x:%d  ", e.v, e.n)
	}
	return s
}
