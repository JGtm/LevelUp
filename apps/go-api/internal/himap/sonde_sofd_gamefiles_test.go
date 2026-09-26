//go:build gamefiles

package himap

// SONDE (2026-08-18, plan .ai/V7.5/replay2d/PLAN_NOMMAGE_EQIP_TRANSLOCATEUR.md phase 0) —
// NOMMER les objets d'equipement `ti=37` PAR LA STRUCTURE DU JEU, sans statistique.
//
// LA CHAINE, telle que la mesure l'a rendue — et elle CORRIGE le plan. Le plan annoncait
// `eqip -> entree sofd -> string_id`. Le bloc `sofd` ne reference PAS d'`eqip` : il reference
// un `sofa` par rang, et c'est le `sofa` qui porte a la fois l'identifiant de chaine ET les
// references `eqip`. La chaine reelle est donc :
//
//	sofd (palette du match)  --rang-->  sofa  --+--> string_id (murmur3 du nom)
//	                                            +--> eqip (les objets poses dans le monde)
//
// C'est la MEME structure que celle du chantier voisin pour l'armement de vehicule
// (`vcdd -> sofd -> sofa -> uwfa -> weap`) : le `sofa` est le maillon commun.
//
// LES PIECES AMONT. RECETTE_LOADOUT_2026-07-27 §9 : `FUN_1407E7648` parcourt le bloc du
// groupe `sofd` — donnees a `tag+0x28`, nombre a `tag+0x38`, pas `0x20` en MEMOIRE — et
// compare `entree+0x18` au handle de definition. §13 : les noms sont des murmur3
// d'identifiants de chaine. La forme SERIALISEE (module) range le bloc dans un data-block
// distinct ; c'est ce que la sonde MESURE au lieu de le postuler.
//
// LECTURE SEULE, saute si le jeu n'est pas installe. UN SEUL module en memoire a la fois :
// `himodule.Open` lit le fichier ENTIER.

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/himodule"
)

// Reference de tag SERIALISEE, 28 octets — meme convention que `physique_sonde` :
//
//	+0x00 pointeur d'execution (rempli de 0xbc dans le fichier)
//	+0x08 GlobalID de la cible
//	+0x0c asset id (u64)
//	+0x14 fourCC du groupe, GROS-BOUTIEN
//	+0x18 offset de nom (0xbcbcbcbc : les noms sont depouilles)
const (
	sofdRefGlobalID = 8
	sofdRefGroup    = 20
)

// sofdEntrySize : pas d'une entree de palette, MESURE (608/19, 544/17, 672/21, 704/22,
// 864/27 sur les palettes de globals) — et il coincide avec le pas `0x20` que le decompile
// lit en memoire.
const sofdEntrySize = 32

// sofdOffCategorie : l'octet qui suit la reference de tag dans une entree de palette. Il vaut
// 1 pour les rangs qui sont des capacites et 0 pour ceux que la RECETTE §13 appelle
// « categorie nulle » (`mobility_sprint` rang 0, `melee_default` rang 7, et le rang 3). C'est
// un CONTROLE de la lecture, pas une hypothese : trois rangs designes d'avance, trois zeros.
const sofdOffCategorie = 29

// sofaOffStringID : position de l'identifiant de chaine dans le bloc racine d'un `sofa`.
//
// L'OFFSET N'EST PAS DEVINE, il est ETABLI par cinq egalites exactes sur la palette de la
// famille A (`sofd` 0xd91958af), murmur3 x86_32 seed 0 des noms que la RECETTE §13 avait
// obtenus par un autre chemin :
//
//	rang 0  0xf08fa6e6 = mobility_sprint          rang 4  0x87b1d7a4 = ability_grapple_hook
//	rang 1  0x566bb170 = ability_location_sensor  rang 5  0xed76a664 = ability_evade
//	rang 2  0xedebd7b7 = ability_deployable_wall
//
// Cinq collisions fortuites a la meme position sont exclues.
const sofaOffStringID = 0x10

// sofaInfo : ce qu'un `sofa` porte et qui sert au nommage.
type sofaInfo struct {
	stringID uint32
	eqip     []uint32
	// modeles : les dependances qui ne sont ni `eqip` ni `bitm`, sous la forme `grp:id`,
	// triees. C'est la SIGNATURE D'OBJET d'une capacite — deux `sofa` qui partagent leur
	// modele, leur animation et leur son sont deux reglages du meme objet. Elle sert a
	// rapprocher une variante d'un nom connu SANS passer par le dictionnaire.
	modeles []string
}

// sofdEntree : une entree de palette d'un `sofd`, a son RANG — celui que le film transmet
// dans `i48`.
type sofdEntree struct {
	sofa      uint32
	categorie byte
}

// corpusSofd : ce qu'une passe sur les modules ramasse. Une seule passe, un module a la fois.
type corpusSofd struct {
	sofa     map[uint32]sofaInfo
	palettes map[uint32][]sofdEntree
}

// balayeSofd lit `sofa` et `sofd` de tous les modules indexes, un module a la fois.
func balayeSofd(t *testing.T, root string) corpusSofd {
	t.Helper()
	c := corpusSofd{sofa: map[uint32]sofaInfo{}, palettes: map[uint32][]sofdEntree{}}
	for _, rel := range ti37Modules {
		m, err := himodule.Open(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Logf("module %s illisible : %v", rel, err)
			continue
		}
		nA, nD := 0, 0
		for _, f := range m.Files("sofa") {
			raw, err := m.Extract(f)
			if err != nil {
				continue
			}
			if info, ok := lisSofa(raw); ok {
				nA++
				c.sofa[f.GlobalID] = info
			}
		}
		for _, f := range m.Files("sofd") {
			raw, err := m.Extract(f)
			if err != nil {
				continue
			}
			if ent := sofdPalette(raw); len(ent) > 0 {
				nD++
				c.palettes[f.GlobalID] = ent
			}
		}
		t.Logf("  %-52s %4d `sofa` · %3d `sofd` · cumul %d/%d",
			rel, nA, nD, len(c.sofa), len(c.palettes))
		m = nil // un seul module en memoire a la fois
	}
	return c
}

// sofdPalette rend les entrees de la palette d'un tag `sofd`, dans l'ordre.
//
// La selection du bloc n'est pas devinee : on prend le data-block dont la taille est un
// multiple du pas ET dont TOUTES les entrees portent le fourCC `sofa` a `+0x14`. Un bloc qui
// n'est pas la palette echoue ce test en masse.
func sofdPalette(raw []byte) []sofdEntree {
	ti, err := meilleurTagInfo(raw)
	if err != nil {
		return nil
	}
	var best []sofdEntree
	for i := 0; i < ti.dataBlocks; i++ {
		abs, size := ti.blockAbs(i)
		if size < sofdEntrySize || size%sofdEntrySize != 0 || abs+size > len(raw) {
			continue
		}
		ent, ok := sofdEntrees(raw, abs, size/sofdEntrySize)
		if ok && len(ent) > len(best) {
			best = ent
		}
	}
	return best
}

func sofdEntrees(raw []byte, abs, n int) ([]sofdEntree, bool) {
	ent := make([]sofdEntree, 0, n)
	for k := 0; k < n; k++ {
		o := abs + k*sofdEntrySize
		if fourCCBE(raw, o+sofdRefGroup) != "sofa" {
			return nil, false
		}
		ent = append(ent, sofdEntree{
			sofa:      binary.LittleEndian.Uint32(raw[o+sofdRefGlobalID:]),
			categorie: raw[o+sofdOffCategorie],
		})
	}
	return ent, true
}

// lisSofa rend l'identifiant de chaine d'un `sofa`, les `eqip` qu'il reference et sa
// signature d'objet.
//
// Les dependances portent le fourCC du groupe pour chaque cible : aucune appartenance a
// deviner, aucun index a croiser.
func lisSofa(raw []byte) (sofaInfo, bool) {
	ti, err := meilleurTagInfo(raw)
	if err != nil {
		return sofaInfo{}, false
	}
	rootIdx, err := ti.rootBlockIndex()
	if err != nil {
		return sofaInfo{}, false
	}
	var info sofaInfo
	abs, size := ti.blockAbs(rootIdx)
	if size < sofaOffStringID+4 || abs+sofaOffStringID+4 > len(raw) {
		return sofaInfo{}, false
	}
	info.stringID = binary.LittleEndian.Uint32(raw[abs+sofaOffStringID:])
	depsTab := ti.blockTab - ti.deps*depEntrySize
	for i := 0; i < ti.deps; i++ {
		e := depsTab + i*depEntrySize
		if e+depEntrySize > len(raw) {
			continue
		}
		grp := fourCCBE(raw, e)
		id := binary.LittleEndian.Uint32(raw[e+0x10:])
		switch grp {
		case "eqip":
			info.eqip = append(info.eqip, id)
		case "bitm", "":
			// Les images d'interface ne signent pas un objet : deux capacites sans rapport
			// partagent le meme atlas (`0xc6b7ad41` revient partout).
		default:
			info.modeles = append(info.modeles, fmt.Sprintf("%s:%08x", grp, id))
		}
	}
	sort.Strings(info.modeles)
	return info, true
}

// fourCCBE lit un fourCC GROS-BOUTIEN (l'ordre des references de tag), ou "" s'il n'est pas
// imprimable.
func fourCCBE(b []byte, o int) string {
	if o+4 > len(b) {
		return ""
	}
	v := binary.LittleEndian.Uint32(b[o:])
	out := []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
	for _, c := range out {
		if c < 0x20 || c > 0x7e {
			return ""
		}
	}
	return string(out)
}

// TestSondeSofdNommage est la mesure du gate 0 : les palettes lues rang par rang, puis
// l'index INVERSE `eqip -> sofa -> nom` croise avec les identifiants observes dans les films.
func TestSondeSofdNommage(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	c := balayeSofd(t, root)
	cibles := map[uint32]bool{}
	for _, s := range c.sofa {
		cibles[s.stringID] = true
	}
	noms, cand := casseIdentifiantsDeChaine(cibles)
	t.Logf("== %d `sofa` · %d `sofd` · %d identifiants de chaine, %d casses "+
		"(%d candidats, esperance de collision %.4f) ==",
		len(c.sofa), len(c.palettes), len(cibles), len(noms), cand,
		float64(cand)*float64(len(cibles))/4294967296.0)

	rapportPalettes(t, c, noms)
	rapportInverse(t, c, noms)
}

// rapportPalettes liste les palettes qui portent un jeu de capacites de joueur (10 rangs ou
// plus), rang par rang.
func rapportPalettes(t *testing.T, c corpusSofd, noms map[uint32]string) {
	t.Helper()
	ids := make([]uint32, 0, len(c.palettes))
	for k, ent := range c.palettes {
		if len(ent) >= 10 {
			ids = append(ids, k)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	t.Logf("== %d palettes de 10 rangs ou plus ==", len(ids))
	for _, id := range ids {
		t.Logf("  sofd 0x%08x — %d rangs", id, len(c.palettes[id]))
		for rang, e := range c.palettes[id] {
			s := c.sofa[e.sofa]
			t.Logf("      rang %2d  sofa 0x%08x  str 0x%08x  cat=%d  eqip=%s  %s",
				rang, e.sofa, s.stringID, e.categorie, hexIDs(s.eqip), noms[s.stringID])
		}
	}
}

// rapportInverse croise les identifiants observes dans les poses des films avec les `sofa`
// qui les referencent.
func rapportInverse(t *testing.T, c corpusSofd, noms map[uint32]string) {
	t.Helper()
	par := map[uint32][]uint32{} // eqip -> sofa
	for id, s := range c.sofa {
		for _, e := range s.eqip {
			par[e] = append(par[e], id)
		}
	}
	obs := ti37Observes()
	keys := make([]uint32, 0, len(obs))
	for k := range obs {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	nommes, rattaches := 0, 0
	t.Logf("== index inverse : %d `eqip` references par un `sofa` ==", len(par))
	for _, k := range keys {
		ss := par[k]
		if len(ss) == 0 {
			t.Logf("  0x%08x  AUCUN `sofa`                                    %s", k, obs[k])
			continue
		}
		rattaches++
		sort.Slice(ss, func(i, j int) bool { return ss[i] < ss[j] })
		var libs []string
		nomme := false
		for _, id := range ss {
			s := c.sofa[id]
			n := noms[s.stringID]
			if n == "" {
				n = fmt.Sprintf("str:0x%08x", s.stringID)
			} else {
				nomme = true
			}
			libs = append(libs, fmt.Sprintf("sofa:%08x=%s", id, n))
		}
		if nomme {
			nommes++
		}
		t.Logf("  0x%08x  %-56s %s", k, strings.Join(libs, " + "), obs[k])
	}
	t.Logf("== %d des %d identifiants observes rattaches a un `sofa`, %d NOMMES ==",
		rattaches, len(obs), nommes)
}

func hexIDs(ids []uint32) string {
	if len(ids) == 0 {
		return "-"
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, fmt.Sprintf("%08x", id))
	}
	return strings.Join(out, ",")
}
