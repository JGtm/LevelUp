//go:build research

// m4b_barillets_research_test.go — SONDE M4b (campagne « retours rejeu », 2026-09-24) : LA CADENCE
// ET LE TYPE DE PREDICTION DES BARILLETS, LUS DANS LE TAG `weap`. Lecture SEULE des modules
// installes, aucun film ouvert, rien d ecrit.
//
// CE QUE LE JEU LIT (sonde P1-S3, Ghidra lecture seule) : `FUN_140de87fc` decide l emission d un
// record de tir par le TYPE DE PREDICTION du barillet (`barillet + 0x70`, court) ; types 1 et 3 :
// seau a jetons dont le debit est `barillet + 0x74` (`+ 0x78` sur le serveur), seuil
// `barillet + 0x7c`. Le barillet tire a sa cadence (`FUN_1407fa928`) : le champ « rounds per
// second » (min, max) en tete de l element (`cmd/weapon-sounds/cadence.go`, +4).
//
// CE QUE L INSTRUMENT FAIT, pour chaque arme demandee (M4B_WEAP, GlobalID en hexadecimal) : il
// ouvre le tag, trouve le bloc `barrels` (tableau enfant de la racine dont les elements portent une
// cadence plausible a +4 — le plugin derive, cf. cadence.go), et publie par barillet : la cadence,
// le type de prediction, le debit d evenements et les deux champs suivants. Il remonte aussi, pour
// chaque chassis `vehi` demande (M4B_VEHI), les armes que ses configurations MULTIJOUEUR declarent
// (`vcdd` -> `sofd` -> `sofa` -> `uwfa` -> `weap`, meme chaine que la sonde M6.2).
//
//	M4B_DEPLOY=<bibliotheque>/<jeux>/Halo Infinite/deploy M4B_WEAP=00015435,a0955e9e \
//	M4B_VEHI=5b80c406 go test -tags=research -count=1 -v -run '^TestM4bBarillets$' ./internal/himodule/

package himodule_test

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/himodule"
)

// Disposition d un tag `ucsh` (cmd/weapon-sounds/tags.go, weapfire.go).
const (
	m4bEntete     = 0x50
	m4bDep        = 0x18
	m4bBloc       = 0x10
	m4bStruct     = 0x20
	m4bRPS        = 4 // « rounds per second » dans l element de `barrels`
	m4bPrediction = 0x70
)

// m4bTag est un tag `ucsh` et ses tables.
type m4bTag struct {
	b                             []byte
	tailleEnt, tabBlocs, tabStruc int
	nBlocs, nStructs              int
}

func m4bOuvrir(b []byte) (m4bTag, bool) {
	if len(b) < m4bEntete || binary.LittleEndian.Uint32(b) != 0x68736375 {
		return m4bTag{}, false
	}
	nDeps := int(int32(binary.LittleEndian.Uint32(b[0x18:])))
	t := m4bTag{b: b, nBlocs: int(int32(binary.LittleEndian.Uint32(b[0x1c:]))),
		nStructs: int(int32(binary.LittleEndian.Uint32(b[0x20:]))), tailleEnt: int(int32(binary.LittleEndian.Uint32(b[0x38:])))}
	t.tabBlocs = m4bEntete + nDeps*m4bDep
	t.tabStruc = t.tabBlocs + t.nBlocs*m4bBloc
	return t, t.tabStruc+t.nStructs*m4bStruct <= len(b)
}

func (t m4bTag) bloc(i int) (abs, taille int) {
	o := t.tabBlocs + i*m4bBloc
	if i < 0 || o+16 > len(t.b) {
		return -1, 0
	}
	taille = int(binary.LittleEndian.Uint32(t.b[o:]))
	abs = int(binary.LittleEndian.Uint64(t.b[o+8:]))
	if binary.LittleEndian.Uint16(t.b[o+6:]) != 0 {
		abs += t.tailleEnt
	}
	return abs, taille
}

// racine rend le bloc de la MainStruct ; enfants, les tableaux d un bloc : offset de champ -> bloc.
func (t m4bTag) racine() int {
	for i := 0; i < t.nStructs; i++ {
		o := t.tabStruc + i*m4bStruct
		if binary.LittleEndian.Uint16(t.b[o+0x10:]) == 0 {
			return int(int32(binary.LittleEndian.Uint32(t.b[o+0x14:])))
		}
	}
	return -1
}

func (t m4bTag) enfants(parent int) map[int]int {
	out := map[int]int{}
	for i := 0; i < t.nStructs; i++ {
		o := t.tabStruc + i*m4bStruct
		if binary.LittleEndian.Uint16(t.b[o+0x10:]) != 1 ||
			int(int32(binary.LittleEndian.Uint32(t.b[o+0x18:]))) != parent {
			continue
		}
		out[int(binary.LittleEndian.Uint32(t.b[o+0x1C:]))] = int(int32(binary.LittleEndian.Uint32(t.b[o+0x14:])))
	}
	return out
}

func (t m4bTag) f32(o int) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(t.b[o:])) }

// m4bBarillets trouve le bloc `barrels` et rend ses elements : l offset du champ dans la racine,
// le nombre d elements (lu dans le champ du tableau, +0x10 du champ de 20 octets), et leur pas.
func m4bBarillets(t m4bTag) (champ, abs, n, pas int, ok bool) {
	rac := t.racine()
	racAbs, _ := t.bloc(rac)
	enf := t.enfants(rac)
	champs := make([]int, 0, len(enf))
	for fo := range enf {
		champs = append(champs, fo)
	}
	sort.Ints(champs) // ordre DETERMINISTE : la table des enfants est une map
	for _, fo := range champs {
		blk := enf[fo]
		a, taille := t.bloc(blk)
		if a < 0 || taille < 12 || a+12 > len(t.b) {
			continue
		}
		bas, haut := t.f32(a+m4bRPS), t.f32(a+m4bRPS+4)
		if bas < 0.3 || haut > 60 || bas > haut || math.IsNaN(float64(bas)) {
			continue
		}
		cnt := int(binary.LittleEndian.Uint32(t.b[racAbs+fo+0x10:]))
		if cnt <= 0 || taille%cnt != 0 || taille/cnt < m4bPrediction+16 {
			continue // un element de barillet porte au moins le type de prediction et ses trois flottants
		}
		return fo, a, cnt, taille / cnt, true
	}
	return 0, 0, 0, 0, false
}

// TestM4bBarillets publie, par arme, les barillets lus dans le tag (en-tete du fichier).
func TestM4bBarillets(t *testing.T) {
	racine := strings.TrimSpace(os.Getenv("M4B_DEPLOY"))
	if racine == "" {
		t.Skip("M4B_DEPLOY non pose (racine deploy de l installation)")
	}
	modules := strings.Split(os.Getenv("M4B_MODULES"), ",")
	if os.Getenv("M4B_MODULES") == "" {
		modules = []string{"common-rtx-new.module", "globals-rtx-new.module"}
	}
	weaps := m4bGids(os.Getenv("M4B_WEAP"))
	restants := map[uint32]bool{}
	for _, g := range weaps {
		restants[g] = true
	}
	for _, nom := range modules {
		mc, ic := ca9Index(t, filepath.Join(racine, "any", "globals", strings.TrimSpace(nom)))
		var ix *m6Index
		for _, v := range m4bGids(os.Getenv("M4B_VEHI")) {
			if _, ok := ic[v]; !ok {
				continue
			}
			if ix == nil {
				x := m6Indexer(t, mc.Files(""), mc.Extract)
				ix = &x
			}
			for _, g := range m4bArmesDuVehicule(t, *ix, ic, v) {
				restants[g] = true
			}
		}
		for _, g := range m4bGids(os.Getenv("M4B_DEPS")) {
			if f, ok := ic[g]; ok {
				if data, err := mc.Extract(f); err == nil {
					var parts []string
					for _, d := range ca9Deps(data) {
						parts = append(parts, fmt.Sprintf("%s %08x", d.groupe, d.gid))
					}
					t.Logf("%s %s %08x : dependances %s", nom, f.Group, g, strings.Join(parts, ", "))
				}
			}
		}
		if os.Getenv("M4B_TOUTES") != "" {
			m4bToutesLesArmes(t, mc, nom)
		}
		for g := range restants {
			f, ok := ic[g]
			if !ok || f.Group != "weap" {
				continue
			}
			data, err := mc.Extract(f)
			if err != nil {
				t.Logf("weap %08x : extraction %v", g, err)
				continue
			}
			t.Logf("module %s :", nom)
			m4bPublier(t, g, data)
			delete(restants, g)
		}
	}
	for g := range restants {
		t.Logf("weap %08x : absent des modules %v", g, modules)
	}
}

// m4bPublier publie les barillets d un tag `weap`.
func m4bPublier(t *testing.T, g uint32, data []byte) {
	t.Helper()
	tag, ok := m4bOuvrir(data)
	if !ok {
		t.Logf("weap %08x : en-tete non ucsh", g)
		return
	}
	fo, abs, n, pas, ok := m4bBarillets(tag)
	if !ok {
		t.Logf("weap %08x : aucun tableau de barillets a cadence plausible", g)
		return
	}
	t.Logf("weap %08x : barrels au champ +%d, %d element(s) de %d octets", g, fo, n, pas)
	for i := 0; i < n; i++ {
		e := abs + i*pas
		if e+m4bPrediction+16 > len(data) {
			break
		}
		t.Logf("   barillet %d : cadence %.3f..%.3f coups/s (acceleration %.3f s, deceleration %.3f s) · "+
			"prediction (+0x70) %d · +0x74 %.3f · +0x78 %.3f · +0x7c %.3f", i, tag.f32(e+m4bRPS),
			tag.f32(e+m4bRPS+4), tag.f32(e+0x10), tag.f32(e+0x18),
			binary.LittleEndian.Uint16(data[e+m4bPrediction:]), tag.f32(e+0x74), tag.f32(e+0x78), tag.f32(e+0x7c))
	}
}

// m4bGids lit une liste de GlobalID hexadecimaux.
func m4bGids(s string) []uint32 {
	var out []uint32
	for _, x := range strings.Split(s, ",") {
		var v uint32
		if _, err := fmt.Sscanf(strings.TrimSpace(x), "%x", &v); err == nil {
			out = append(out, v)
		}
	}
	return out
}

// m4bArmesDuVehicule rend les `weap` que les configurations MULTIJOUEUR d un chassis declarent :
// chaque `vcdd` qui declare le `vehi`, puis la chaine descendante sofd -> sofa -> uwfa -> weap.
func m4bArmesDuVehicule(t *testing.T, ix m6Index, ic map[uint32]himodule.File, vehi uint32) []uint32 {
	t.Helper()
	var out []uint32
	for _, vcdd := range ix.referents[vehi] {
		if ic[vcdd].Group != "vcdd" {
			continue
		}
		front := []uint32{vcdd}
		for prof := 0; prof < 5 && len(front) > 0; prof++ {
			var suivant []uint32
			for _, id := range front {
				for gid, groupe := range ix.deps[id] {
					switch groupe {
					case "weap":
						out = append(out, gid)
					case "sofd", "sofa", "uwfa":
						suivant = append(suivant, gid)
					}
				}
			}
			front = suivant
		}
		groupes := map[string]int{}
		for _, g := range ix.deps[vcdd] {
			groupes[g]++
		}
		t.Logf("vehi %08x : vcdd %08x (dependances %v)", vehi, vcdd, groupes)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	t.Logf("vehi %08x : armes des configurations multijoueur %x", vehi, out)
	return out
}

// m4bToutesLesArmes publie, pour CHAQUE tag `weap` du module, les barillets dont le type de
// prediction n est pas « evenement a chaque coup » (1 continu, 3 rafale continue) : les armes dont le
// tir ne vit que dans la vue de controle quand leur debit d evenements est nul.
func m4bToutesLesArmes(t *testing.T, mc *himodule.Module, nom string) {
	t.Helper()
	n, continues := 0, 0
	for _, f := range mc.Files("weap") {
		data, err := mc.Extract(f)
		if err != nil {
			continue
		}
		tag, ok := m4bOuvrir(data)
		if !ok {
			continue
		}
		_, abs, nb, pas, ok := m4bBarillets(tag)
		if !ok {
			continue
		}
		n++
		var parts []string
		for i := 0; i < nb; i++ {
			e := abs + i*pas
			if e+m4bPrediction+16 > len(data) {
				break
			}
			p := binary.LittleEndian.Uint16(data[e+m4bPrediction:])
			if p == 1 || p == 3 {
				parts = append(parts, fmt.Sprintf("b%d p%d %.2f..%.2f/s debit %.2f", i, p,
					tag.f32(e+m4bRPS), tag.f32(e+m4bRPS+4), tag.f32(e+0x74)))
			}
		}
		if len(parts) > 0 {
			continues++
			t.Logf("%s weap %08x : %s", nom, f.GlobalID, strings.Join(parts, " | "))
		}
	}
	t.Logf("%s : %d tags weap lus, %d a barillet de prediction 1 ou 3", nom, n, continues)
}
