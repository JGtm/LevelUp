//go:build gamefiles

package himap

// SONDE — LE CANEVAS FORGE PORTE-T-IL DES « SCENARIO LOCATION NAME TRIGGERS » ?
//
// Question de l'utilisateur (2026-08-27) : « le jeu affiche des callouts sur TOUTES les
// cartes, Forge comprises ». Si l'asset UGC n'en porte pas, ils viendraient du CANEVAS —
// et toutes les cartes baties dessus les partageraient.
//
// callouts.go affirme « les canevas Forge en portent ZERO — c'est un fait de construction ».
// UN ZERO N'EST UNE MESURE QUE SI ON A VERIFIE QUE L'ON LIT AU BON ENDROIT : lire le
// compte de `named locations` a root+0x91C sur un tag dont la disposition serait autre
// rendrait zero aussi, et ce zero-la ne dirait rien. Cette sonde compare donc le tag levl
// de chaque canevas a celui d'une carte native de reference sur les temoins de disposition
// (taille du root block, comptes NOMS et VOLUMES, presence des blocs enfants), et affiche
// les volumes quand il y en a.
//
// Elle ne conclut pas : elle donne les nombres qui tranchent.

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/himodule"
)

// temoinLevl est ce qu'on mesure sur un tag levl sans rien supposer de son contenu.
type temoinLevl struct {
	tailleRoot  int
	nNoms       int
	nVolumes    int
	blocNoms    bool
	blocVolumes bool
	volumes     []Callout // rempli seulement quand la liaison nom<->volume tient
}

// mesureLevl lit les temoins de disposition du tag levl d'un module.
func mesureLevl(modulePath string) (temoinLevl, error) {
	var t temoinLevl
	m, err := himodule.Open(modulePath)
	if err != nil {
		return t, err
	}
	levls := m.Files("levl")
	if len(levls) != 1 {
		return t, errf("%d tags levl (attendu 1)", len(levls))
	}
	tag, err := m.Extract(levls[0])
	if err != nil {
		return t, err
	}
	ti, err := meilleurTagInfo(tag)
	if err != nil {
		return t, err
	}
	root, err := ti.rootBlockIndex()
	if err != nil {
		return t, err
	}
	rootAbs, rootSize := ti.blockAbs(root)
	t.tailleRoot = rootSize
	if levlChampNames+0x14 <= rootSize {
		t.nNoms = u32(ti.tag, rootAbs+levlChampNames+0x10)
	}
	if levlChampVolumes+0x14 <= rootSize {
		t.nVolumes = u32(ti.tag, rootAbs+levlChampVolumes+0x10)
	}
	for _, l := range liensBlocs(ti) {
		if l.owner != root {
			continue
		}
		switch l.fieldOff {
		case levlChampNames:
			t.blocNoms = true
		case levlChampVolumes:
			t.blocVolumes = true
		}
	}
	t.volumes, _ = calloutsFromTag(tag)
	return t, nil
}

func errf(format string, a ...any) error { return fmt.Errorf(format, a...) }

// TestSondeLevlCanevasForge — le tableau qui tranche : canevas vs carte native.
func TestSondeLevlCanevasForge(t *testing.T) {
	dir, err := LevelsDir("ds")
	if err != nil {
		t.Skipf("installation du jeu introuvable : %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%-24s %10s %6s %8s %6s %8s", "module", "rootBlock", "noms", "volumes", "blocN", "blocV")
	var canevasZero, nativesAvecNoms int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name(), e.Name()+"-rtx-new.module")
		if _, statErr := os.Stat(p); statErr != nil {
			continue
		}
		m, err := mesureLevl(p)
		if err != nil {
			t.Logf("%-24s LECTURE KO : %v", e.Name(), err)
			continue
		}
		t.Logf("%-24s %10d %6d %8d %6v %8v", e.Name(), m.tailleRoot, m.nNoms, m.nVolumes, m.blocNoms, m.blocVolumes)
		if m.nNoms == 0 {
			canevasZero++
		} else {
			nativesAvecNoms++
		}
		// Quand un module sans NOMS porte quand meme des VOLUMES, c'est la piste : des
		// zones geometriques sans libelle. On veut le savoir, pas le deviner.
		if m.nNoms == 0 && m.nVolumes > 0 {
			t.Logf("    ^^ %s : %d volumes SANS nom — a ouvrir", e.Name(), m.nVolumes)
		}
	}
	t.Logf("modules sans named location : %d ; avec : %d", canevasZero, nativesAvecNoms)
}

// TestSondeLevlWetlandDetaille — le canevas d'Isolation et de Vagabond, en detail, face a
// une carte native de reference. Si la disposition est la meme et le compte de noms nul,
// le zero est une mesure.
func TestSondeLevlWetlandDetaille(t *testing.T) {
	dir, err := LevelsDir("ds")
	if err != nil {
		t.Skipf("installation du jeu introuvable : %v", err)
	}
	for _, nom := range []string{CanevasWetland, CanevasBlank, "ridgeline"} {
		p := filepath.Join(dir, nom, nom+"-rtx-new.module")
		if _, err := os.Stat(p); err != nil {
			t.Logf("%s : module absent", nom)
			continue
		}
		m, err := mesureLevl(p)
		if err != nil {
			t.Errorf("%s : %v", nom, err)
			continue
		}
		t.Logf("%s : rootBlock=%d o, noms=%d, volumes=%d, blocNoms=%v, blocVolumes=%v, callouts rendus=%d",
			nom, m.tailleRoot, m.nNoms, m.nVolumes, m.blocNoms, m.blocVolumes, len(m.volumes))
		for i, c := range m.volumes {
			if i >= 5 {
				t.Logf("    ... (%d zones au total)", len(m.volumes))
				break
			}
			t.Logf("    vi=%3d %-34q sid=%08x pos=(%.1f, %.1f, %.1f) forme=%v poly=%d",
				c.VolumeIndex, c.Name, c.StringID, c.Pos[0], c.Pos[1], c.Pos[2], c.HasShape, len(c.Polygon))
		}
	}
}

// volumeBrut est un record du bloc `volumes` lu SANS passer par les named locations :
// c'est le seul moyen de voir les volumes qu'aucun nom ne designe.
type volumeBrut struct {
	Index    int
	StringID uint32
	HasShape bool
	Kind     int
	Pos      [3]float64
	Top, Bot float64
	NbPoly   int
}

// litVolumesBruts rend TOUS les volumes du tag levl d'un module, nommes ou non.
func litVolumesBruts(modulePath string) ([]volumeBrut, error) {
	m, err := himodule.Open(modulePath)
	if err != nil {
		return nil, err
	}
	levls := m.Files("levl")
	if len(levls) != 1 {
		return nil, fmt.Errorf("%d tags levl", len(levls))
	}
	tag, err := m.Extract(levls[0])
	if err != nil {
		return nil, err
	}
	ti, err := meilleurTagInfo(tag)
	if err != nil {
		return nil, err
	}
	root, err := ti.rootBlockIndex()
	if err != nil {
		return nil, err
	}
	parOwnerOff := map[[2]int]lienBloc{}
	for _, l := range liensBlocs(ti) {
		parOwnerOff[[2]int{l.owner, l.fieldOff}] = l
	}
	lv, ok := parOwnerOff[[2]int{root, levlChampVolumes}]
	if !ok {
		return nil, fmt.Errorf("bloc volumes absent")
	}
	vAbs, vSize, err := blocVerifie(ti, lv, levlStrideVolume)
	if err != nil {
		return nil, err
	}
	out := make([]volumeBrut, 0, vSize/levlStrideVolume)
	for i := 0; i < vSize/levlStrideVolume; i++ {
		r := vAbs + i*levlStrideVolume
		v := volumeBrut{
			Index:    i,
			StringID: uint32(u32(ti.tag, r+levlVolOffSID)),
			HasShape: ti.tag[r+levlVolOffHasShape] == 1,
			Kind:     int(ti.tag[r+levlVolOffKind]),
			Top:      f32(ti.tag, r+levlVolOffTop),
			Bot:      f32(ti.tag, r+levlVolOffBottom),
		}
		for a := 0; a < 3; a++ {
			v.Pos[a] = f32(ti.tag, r+levlVolOffPos+4*a)
		}
		if lp, okp := parOwnerOff[[2]int{lv.target, i*levlStrideVolume + levlVolOffPolygon}]; okp {
			_, pSize := ti.blockAbs(lp.target)
			v.NbPoly = pSize / 12
		}
		out = append(out, v)
	}
	return out, nil
}

// TestSondeVolumesSansNom — LES 12 VOLUMES DU CANEVAS SONT-ILS DES ZONES DE CALLOUT ?
//
// Chaque canevas porte 12 volumes et ZERO nom ; chaque carte native porte ~10 volumes de
// plus que de noms. Si ces volumes anonymes sont les MEMES des deux cotes (meme string_id,
// meme kind), ce sont des volumes de service du scenario, pas des zones nommees — et alors
// le canevas ne porte rien qui ressemble a un callout.
func TestSondeVolumesSansNom(t *testing.T) {
	dir, err := LevelsDir("ds")
	if err != nil {
		t.Skipf("installation du jeu introuvable : %v", err)
	}
	nommes := func(module string) map[uint32]bool {
		cs, e := ReadModuleCallouts(filepath.Join(dir, module, module+"-rtx-new.module"))
		if e != nil {
			return nil
		}
		s := map[uint32]bool{}
		for _, c := range cs {
			s[c.StringID] = true
		}
		return s
	}
	for _, module := range []string{CanevasWetland, CanevasBlank, "ridgeline", "catalyst"} {
		p := filepath.Join(dir, module, module+"-rtx-new.module")
		if _, e := os.Stat(p); e != nil {
			continue
		}
		vols, e := litVolumesBruts(p)
		if e != nil {
			t.Errorf("%s : %v", module, e)
			continue
		}
		avecNom := nommes(module)
		t.Logf("=== %s : %d volumes (%d designes par un nom)", module, len(vols), len(avecNom))
		for _, v := range vols {
			if avecNom[v.StringID] {
				continue
			}
			t.Logf("    ANONYME vi=%2d sid=%08x kind=%d forme=%v pos=(%9.1f,%9.1f,%9.1f) top=%.1f bot=%.1f poly=%d",
				v.Index, v.StringID, v.Kind, v.HasShape, v.Pos[0], v.Pos[1], v.Pos[2], v.Top, v.Bot, v.NbPoly)
		}
	}
}
