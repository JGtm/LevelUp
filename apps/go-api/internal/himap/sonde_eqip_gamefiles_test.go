//go:build gamefiles

package himap

// SONDE (2026-08-18, plan PLAN_NOMMAGE_EQIP_TRANSLOCATEUR phase 0.2) — le NIVEAU `eqip`.
//
// CE QUE LA SONDE `sofd` A LAISSE OUVERT. La chaine `sofd -> sofa -> eqip` rattache 15 des
// 21 identifiants observes dans les poses. Les six autres n'ont AUCUN `sofa` qui les
// reference, et ils se separent en deux groupes que la mesure doit traiter differemment :
//
//	0x528fce46, 0x686b40c9   ont un POSEUR et une diagonale (« 2e identifiant du mur »).
//	                         Hypothese : ce sont des objets ENGENDRES par l'equipement pose,
//	                         donc references par l'`eqip` du parent, pas par le `sofa`.
//	0xbcabbe43, 0x0f5716ff,  n'ont PAS de poseur (les « plats »). Hypothese : objets poses par
//	0xcaaadcb0, 0xaada07f3   la carte, hors de toute palette de capacite.
//
// DEUX LECTURES, DEUX QUESTIONS SEPAREES : la chaine `eqip -> eqip` pour les premiers, et
// l'identifiant de chaine porte par l'`eqip` LUI-MEME pour les seconds.
//
// LECTURE SEULE, saute si le jeu n'est pas installe. UN SEUL module en memoire a la fois.

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/himodule"
)

// eqipInfo : ce qu'un tag `eqip` porte et qui sert au nommage.
type eqipInfo struct {
	// deps : toutes les dependances sous la forme `grp:id`, triees. La SIGNATURE d'objet.
	deps []string
	// eqip : les dependances du groupe `eqip` — les objets qu'il engendre.
	eqip []uint32
	// mots : les mots de 32 bits du bloc racine, avec leur offset. C'est la ou un
	// identifiant de chaine se trouverait ; on ne balaye PAS le tag entier (un tag de
	// 10 000 mots rendrait des collisions a la pelle contre un dictionnaire de 340 000
	// candidats — 0,8 par tag ; le bloc racine en fait quelques dizaines, soit 0,005).
	mots map[int]uint32
}

// balayeEqip lit tous les tags `eqip` des modules indexes, un module a la fois.
func balayeEqip(t *testing.T, root string) map[uint32]eqipInfo {
	t.Helper()
	out := map[uint32]eqipInfo{}
	for _, rel := range ti37Modules {
		m, err := himodule.Open(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		n := 0
		for _, f := range m.Files("eqip") {
			if _, deja := out[f.GlobalID]; deja {
				continue
			}
			raw, err := m.Extract(f)
			if err != nil {
				continue
			}
			if info, ok := lisEqip(raw); ok {
				n++
				out[f.GlobalID] = info
			}
		}
		t.Logf("  %-52s %4d `eqip` · cumul %d", rel, n, len(out))
		m = nil // un seul module en memoire a la fois
	}
	return out
}

// lisEqip rend les dependances d'un tag `eqip` et les mots de son bloc racine.
func lisEqip(raw []byte) (eqipInfo, bool) {
	ti, err := meilleurTagInfo(raw)
	if err != nil {
		return eqipInfo{}, false
	}
	rootIdx, err := ti.rootBlockIndex()
	if err != nil {
		return eqipInfo{}, false
	}
	var info eqipInfo
	depsTab := ti.blockTab - ti.deps*depEntrySize
	for i := 0; i < ti.deps; i++ {
		e := depsTab + i*depEntrySize
		if e+depEntrySize > len(raw) {
			continue
		}
		grp := fourCCBE(raw, e)
		id := binary.LittleEndian.Uint32(raw[e+0x10:])
		if grp == "eqip" {
			info.eqip = append(info.eqip, id)
		}
		if grp != "" && grp != "bitm" {
			info.deps = append(info.deps, fmt.Sprintf("%s:%08x", grp, id))
		}
	}
	sort.Strings(info.deps)
	abs, size := ti.blockAbs(rootIdx)
	info.mots = map[int]uint32{}
	for o := 0; o+4 <= size && abs+o+4 <= len(raw); o += 4 {
		info.mots[o] = binary.LittleEndian.Uint32(raw[abs+o:])
	}
	return info, true
}

// TestSondeEqipChaine repond aux deux questions laissees ouvertes : qui engendre les
// identifiants sans `sofa`, et l'`eqip` porte-t-il lui-meme un nom.
func TestSondeEqipChaine(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	tags := balayeEqip(t, root)
	obs := ti37Observes()
	t.Logf("== %d tags `eqip` lus · %d identifiants observes ==", len(tags), len(obs))

	// Qui reference qui, au niveau `eqip`.
	parents := map[uint32][]uint32{}
	for id, info := range tags {
		for _, e := range info.eqip {
			parents[e] = append(parents[e], id)
		}
	}
	keys := triIDs(obs)
	for _, k := range keys {
		ps := parents[k]
		sort.Slice(ps, func(i, j int) bool { return ps[i] < ps[j] })
		t.Logf("  0x%08x  engendre par [%s]  · engendre [%s]  · %s",
			k, hexIDs(ps), hexIDs(tags[k].eqip), obs[k])
	}

	// Le nom porte par l'`eqip` lui-meme, cherche dans son SEUL bloc racine.
	cand, n := tableCandidats()
	t.Logf("== table de %d candidats · esperance de collision par tag (%d mots) ==", n, 0)
	for _, k := range keys {
		info, ok := tags[k]
		if !ok {
			t.Logf("  0x%08x  TAG ABSENT des modules indexes", k)
			continue
		}
		var hits []string
		for _, o := range triOffsets(info.mots) {
			if nom, ok := cand[info.mots[o]]; ok {
				hits = append(hits, fmt.Sprintf("+%#x=%s", o, nom))
			}
		}
		t.Logf("  0x%08x  %3d mots racine · %s", k, len(info.mots), strings.Join(hits, " "))
	}
}

// TestSondeEqipSignature rapproche un `eqip` non nomme d'un `eqip` nomme par leurs
// dependances COMMUNES — le modele, l'animation, le son. Deux definitions qui partagent leur
// objet sont deux reglages de la meme chose ; c'est un argument de structure, pas de hasard.
func TestSondeEqipSignature(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	tags := balayeEqip(t, root)
	obs := ti37Observes()
	keys := triIDs(obs)
	for _, k := range keys {
		info, ok := tags[k]
		if !ok {
			continue
		}
		type score struct {
			id uint32
			n  int
		}
		var best []score
		for _, autre := range keys {
			if autre == k {
				continue
			}
			if n := communs(info.deps, tags[autre].deps); n > 0 {
				best = append(best, score{autre, n})
			}
		}
		sort.Slice(best, func(i, j int) bool { return best[i].n > best[j].n })
		var libs []string
		for i, b := range best {
			if i == 3 {
				break
			}
			libs = append(libs, fmt.Sprintf("%08x(%d)", b.id, b.n))
		}
		t.Logf("  0x%08x  %2d deps  proches: %s   [%s]",
			k, len(info.deps), strings.Join(libs, " "), strings.Join(info.deps, " "))
	}
}

// TestSondeGlpa cherche QUI reference les quatre identifiants « plats » (ceux qu'aucun `sofa`
// n'atteint et qui n'ont pas de poseur). Le candidat designe par la RECETTE §13 est le `glpa`,
// la palette globale qui porte deja les douze `sofd` ; s'il enumere aussi des `eqip`, l'ORDRE
// de cette enumeration est un rang, comme pour les capacites.
func TestSondeGlpa(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	m, err := himodule.Open(filepath.Join(root, filepath.FromSlash(
		"any/globals/globals-rtx-new.module")))
	if err != nil {
		t.Skipf("globals illisible : %v", err)
	}
	obs := ti37Observes()
	for _, grp := range []string{"glpa", "gggl"} {
		for _, f := range m.Files(grp) {
			raw, err := m.Extract(f)
			if err != nil {
				t.Logf("%s 0x%08x illisible : %v", grp, f.GlobalID, err)
				continue
			}
			ti, err := meilleurTagInfo(raw)
			if err != nil {
				t.Logf("%s 0x%08x : en-tete non reconnu : %v", grp, f.GlobalID, err)
				continue
			}
			t.Logf("== %s 0x%08x — %d o · deps=%d blocs=%d ==",
				grp, f.GlobalID, len(raw), ti.deps, ti.dataBlocks)
			depsTab := ti.blockTab - ti.deps*depEntrySize
			for i := 0; i < ti.deps; i++ {
				e := depsTab + i*depEntrySize
				id := binary.LittleEndian.Uint32(raw[e+0x10:])
				t.Logf("   dep %3d : %s 0x%08x%s", i, fourCCBE(raw, e), id, marqueObs(obs, id))
			}
			// L'ORDRE DE LA LISTE DE DEPENDANCES N'EST PAS LE RANG (mesure du chantier
			// kill feed, ETAT_DE_L_ART_KILLWEAPON 7ter.48) : le rang vient de l'ADRESSE de
			// la reference dans le bloc. On rend donc la position de chaque reference.
			for i := 0; i < ti.dataBlocks; i++ {
				abs, size := ti.blockAbs(i)
				for o := 0; o+28 <= size && abs+o+28 <= len(raw); o += 4 {
					if fourCCBE(raw, abs+o+sofdRefGroup) != "eqip" {
						continue
					}
					id := binary.LittleEndian.Uint32(raw[abs+o+sofdRefGlobalID:])
					t.Logf("   bloc %2d (%4d o) +%#04x : eqip 0x%08x%s",
						i, size, o, id, marqueObs(obs, id))
				}
			}
		}
	}
}

func marqueObs(obs map[uint32]string, id uint32) string {
	if c, ok := obs[id]; ok {
		return "  <== OBSERVE  " + c
	}
	return ""
}

func communs(a, b []string) int {
	set := make(map[string]bool, len(a))
	for _, v := range a {
		set[v] = true
	}
	n := 0
	for _, v := range b {
		if set[v] {
			n++
		}
	}
	return n
}

func triIDs(m map[uint32]string) []uint32 {
	out := make([]uint32, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func triOffsets(m map[int]uint32) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

// tableCandidats rend le dictionnaire COMPLET indexe par hachage, et sa taille. Il sert a
// chercher un identifiant de chaine a une position INCONNUE d'une structure — a n'employer
// que sur des fenetres etroites, sous peine de collisions (l'esperance vaut
// taille_table x mots_balayes / 2^32).
//
// Il vit ICI, et non avec le reste du dictionnaire (`sonde_stringid_dico_test.go`), parce que
// cette sonde est son seul appelant et qu'elle est derriere le tag `gamefiles` : le laisser
// dans un fichier non tague en ferait du code mort dans le build par defaut.
func tableCandidats() (map[uint32]string, int) {
	out := make(map[uint32]string, 400000)
	n := enumereCandidats(func(nom string, h uint32) {
		if _, deja := out[h]; !deja {
			out[h] = nom
		}
	})
	return out, n
}
