//go:build gamefiles

package himap

// SONDE (2026-08-18, plan PLAN_EQUIPEMENTS_MANQUANTS_SONS phase 1) — L'INVENTAIRE COMPLET
// des tags `eqip` du jeu installe, et pas seulement des 21 que le corpus montre.
//
// CE QUE LES SONDES PRECEDENTES NE FONT PAS. `sonde_eqip_gamefiles_test.go` et
// `sonde_sofd_gamefiles_test.go` partent des identifiants OBSERVES dans les films
// (`ti37Observes`) et remontent la chaine jusqu'a leur nom. Elles repondent donc « comment
// s'appelle ce que j'ai vu », jamais « qu'est-ce que le jeu contient que je n'ai pas vu ».
// La question de ce lot est la seconde : un equipement AJOUTE au jeu et jamais joue par le
// corpus n'a aucune raison d'apparaitre dans une sonde partant du corpus.
//
// LE DENOMINATEUR EST LE `sofa`, PAS LE `eqip`. Le groupe `eqip` compte des centaines de
// tags dont la plupart ne sont pas des equipements de joueur (objets de campagne, pieces de
// Forge, projectiles engendres). Ce qui fait un EQUIPEMENT DE JOUEUR, c'est d'etre reference
// par un `sofa` — le maillon que la palette `sofd` du match indexe par rang. L'inventaire
// publie donc les deux comptes, et le second est celui qui se croise avec le manifeste.
//
// LES MODULES D'ACCUEIL SONT RENDUS TOUS, pas seulement le premier. `ti37Modules` est un
// ordre de balayage, pas une chronologie : ne garder que le premier module ou un tag
// apparait ferait passer un ordre de commodite pour une date.
//
// LECTURE SEULE, saute si le jeu n'est pas installe. UN SEUL module en memoire a la fois.

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/himodule"
)

// eqipPresence : ou un tag `eqip` a ete trouve, et ce qu'il porte.
type eqipPresence struct {
	info    eqipInfo
	modules []string
}

// courtNomModule reduit `any/globals/multiplayer_r3-rtx-new.module` a `multiplayer_r3`.
func courtNomModule(rel string) string {
	base := filepath.Base(rel)
	return strings.TrimSuffix(strings.TrimSuffix(base, ".module"), "-rtx-new")
}

// balayeEqipTous lit TOUS les tags `eqip` des modules indexes, en notant chaque module ou le
// tag apparait. Un module a la fois.
func balayeEqipTous(t *testing.T, root string) map[uint32]*eqipPresence {
	t.Helper()
	out := map[uint32]*eqipPresence{}
	for _, rel := range ti37Modules {
		m, err := himodule.Open(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Logf("  module %s illisible : %v", rel, err)
			continue
		}
		court, vus, nouveaux := courtNomModule(rel), 0, 0
		for _, f := range m.Files("eqip") {
			vus++
			if p, deja := out[f.GlobalID]; deja {
				p.modules = append(p.modules, court)
				continue
			}
			raw, err := m.Extract(f)
			if err != nil {
				continue
			}
			info, ok := lisEqip(raw)
			if !ok {
				continue
			}
			nouveaux++
			out[f.GlobalID] = &eqipPresence{info: info, modules: []string{court}}
		}
		t.Logf("  %-52s %5d `eqip` (%d nouveaux) · cumul %d", rel, vus, nouveaux, len(out))
		m = nil // un seul module en memoire a la fois
	}
	return out
}

// depDuGroupe rend la premiere dependance du groupe demande, sous forme hexa, ou "-".
func depDuGroupe(deps []string, grp string) string {
	for _, d := range deps {
		if strings.HasPrefix(d, grp+":") {
			return d[len(grp)+1:]
		}
	}
	return "-"
}

// rangsParSofa inverse les palettes : `sofa` -> les couples palette/rang qui le servent.
func rangsParSofa(c corpusSofd) map[uint32][]string {
	out := map[uint32][]string{}
	ids := make([]uint32, 0, len(c.palettes))
	for k := range c.palettes {
		ids = append(ids, k)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, p := range ids {
		for rang, e := range c.palettes[p] {
			out[e.sofa] = append(out[e.sofa], fmt.Sprintf("%08x#%d", p, rang))
		}
	}
	return out
}

// TestSondeEqipInventaire est la mesure de la phase 1 : tout le groupe `eqip`, croise avec la
// chaine de nommage, avec le manifeste et avec le corpus.
func TestSondeEqipInventaire(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	t.Log("== PASSE 1 : le groupe `eqip` ==")
	tags := balayeEqipTous(t, root)
	t.Log("== PASSE 2 : `sofa` et `sofd` ==")
	c := balayeSofd(t, root)

	cibles := map[uint32]bool{}
	for _, s := range c.sofa {
		cibles[s.stringID] = true
	}
	noms, cand := casseIdentifiantsDeChaine(cibles)
	t.Logf("== %d `eqip` · %d `sofa` · %d `sofd` · %d identifiants de chaine, %d casses "+
		"(%d candidats, esperance de collision %.4f) ==",
		len(tags), len(c.sofa), len(c.palettes), len(cibles), len(noms), cand,
		float64(cand)*float64(len(cibles))/4294967296.0)

	parEqip := map[uint32][]uint32{} // eqip -> sofa qui le referencent
	for id, s := range c.sofa {
		for _, e := range s.eqip {
			parEqip[e] = append(parEqip[e], id)
		}
	}
	rangs := rangsParSofa(c)
	rapportEquipementsDuJeu(t, tags, c, parEqip, rangs, noms)
	rapportManquants(t, tags, parEqip, noms, c)
}

// nomDuSofa rend le nom casse d'un `sofa`, ou son identifiant de chaine a defaut.
func nomDuSofa(c corpusSofd, noms map[uint32]string, id uint32) string {
	s := c.sofa[id]
	if n := noms[s.stringID]; n != "" {
		return n
	}
	return fmt.Sprintf("str:%08x", s.stringID)
}

// rapportEquipementsDuJeu liste les `sofa` — les EQUIPEMENTS DE JOUEUR du jeu — avec leur
// nom, les palettes qui les servent et les `eqip` qu'ils posent dans le monde.
func rapportEquipementsDuJeu(
	t *testing.T, tags map[uint32]*eqipPresence, c corpusSofd,
	parEqip map[uint32][]uint32, rangs map[uint32][]string, noms map[uint32]string,
) {
	t.Helper()
	ids := make([]uint32, 0, len(c.sofa))
	for k := range c.sofa {
		ids = append(ids, k)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := nomDuSofa(c, noms, ids[i]), nomDuSofa(c, noms, ids[j])
		if a != b {
			return a < b
		}
		return ids[i] < ids[j]
	})
	obs := ti37Observes()
	t.Logf("== LES %d `sofa` DU JEU — un equipement de joueur par ligne ==", len(ids))
	for _, id := range ids {
		s := c.sofa[id]
		var objets []string
		for _, e := range s.eqip {
			marque := ""
			if _, ok := obs[e]; ok {
				marque = "*CORPUS*"
			}
			mods := "-"
			if p, ok := tags[e]; ok {
				mods = strings.Join(uniques(p.modules), "+")
			} else {
				marque += "[TAG ABSENT]"
			}
			objets = append(objets, fmt.Sprintf("%08x{%s}%s", e, mods, marque))
		}
		t.Logf("  sofa %08x  %-34s  palettes[%s]  eqip: %s",
			id, nomDuSofa(c, noms, id), strings.Join(rangs[id], " "),
			strings.Join(objets, " "))
	}
}

// rapportManquants croise l'inventaire avec le MANIFESTE (les 21 identifiants du corpus) et
// rend les trois populations : au manifeste ET au jeu, au jeu SEUL, au manifeste seul.
func rapportManquants(
	t *testing.T, tags map[uint32]*eqipPresence, parEqip map[uint32][]uint32,
	noms map[uint32]string, c corpusSofd,
) {
	t.Helper()
	obs := ti37Observes()
	var auJeuSeul []uint32
	for e := range parEqip {
		if _, ok := obs[e]; !ok {
			auJeuSeul = append(auJeuSeul, e)
		}
	}
	sort.Slice(auJeuSeul, func(i, j int) bool { return auJeuSeul[i] < auJeuSeul[j] })
	t.Logf("== AU JEU SEUL : %d `eqip` rattaches a un `sofa` et ABSENTS du corpus ==",
		len(auJeuSeul))
	for _, e := range auJeuSeul {
		var libs []string
		for _, s := range parEqip[e] {
			libs = append(libs, nomDuSofa(c, noms, s))
		}
		mods := "TAG ABSENT"
		if p, ok := tags[e]; ok {
			mods = strings.Join(uniques(p.modules), "+")
		}
		t.Logf("  %08x  {%s}  %s", e, mods, strings.Join(uniques(libs), " + "))
	}
	manquants := 0
	for id := range obs {
		if _, ok := tags[id]; !ok {
			manquants++
			t.Logf("  MANIFESTE SANS TAG : %08x (%s)", id, obs[id])
		}
	}
	t.Logf("== corpus : %d identifiants, %d sans tag `eqip` dans le jeu installe ==",
		len(obs), manquants)
}

// TestSondeEqipModulesParFamille rend, pour les `sofa` NOMMES, le module d'accueil de leurs
// `eqip` — la seule piece structurelle disponible pour ordonner les equipements dans le temps
// (hypothese H4 du plan, avec son controle : les equipements de lancement doivent tomber dans
// un module d'indice inferieur ou egal a ceux des saisons 4-5).
func TestSondeEqipModulesParFamille(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	tags := balayeEqipTous(t, root)
	c := balayeSofd(t, root)
	cibles := map[uint32]bool{}
	for _, s := range c.sofa {
		cibles[s.stringID] = true
	}
	noms, _ := casseIdentifiantsDeChaine(cibles)

	parModule := map[string][]string{}
	for id, s := range c.sofa {
		nom := nomDuSofa(c, noms, id)
		for _, e := range s.eqip {
			p, ok := tags[e]
			if !ok {
				continue
			}
			cle := strings.Join(uniques(p.modules), "+")
			parModule[cle] = append(parModule[cle], fmt.Sprintf("%s(%08x)", nom, e))
		}
	}
	cles := make([]string, 0, len(parModule))
	for k := range parModule {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	t.Logf("== EQUIPEMENTS PAR MODULE D'ACCUEIL (%d combinaisons) ==", len(cles))
	for _, k := range cles {
		v := uniques(parModule[k])
		sort.Strings(v)
		t.Logf("  %-40s %3d : %s", k, len(v), strings.Join(v, " "))
	}
}

func uniques(in []string) []string {
	vu := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		if vu[v] {
			continue
		}
		vu[v] = true
		out = append(out, v)
	}
	return out
}

// TestSondeEqipSignatureGlobale compare les `eqip` ANONYMES a TOUT le groupe, et non plus aux
// seuls identifiants du corpus.
//
// POURQUOI CE N'EST PAS UN DOUBLON DE `TestSondeEqipSignature`. Celle-la boucle sur
// `ti37Observes()` : elle ne pouvait rapprocher un anonyme que des 21 objets DEJA VUS dans un
// film. Le denominateur reel est de 116 tags, et un equipement jamais joue par le corpus en
// fait partie — c'est precisement la population que ce lot cherche.
//
// Elle rend aussi le RECENSEMENT DES GROUPES DE DEPENDANCES sur tout le groupe `eqip` : c'est
// la carte du format avant d'ecrire quoi que ce soit sur les sons (lecon n° 1 de
// RECETTE_SONS_ARMES — cartographier le format avant de coder).
func TestSondeEqipSignatureGlobale(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	tags := balayeEqipTous(t, root)

	groupes := map[string]int{}
	porteurs := map[string]int{}
	parModele := map[string][]uint32{}
	for id, p := range tags {
		vus := map[string]bool{}
		for _, d := range p.info.deps {
			g := d[:strings.IndexByte(d, ':')]
			groupes[g]++
			if !vus[g] {
				vus[g] = true
				porteurs[g]++
			}
		}
		if h := depDuGroupe(p.info.deps, "hlmt"); h != "-" {
			parModele[h] = append(parModele[h], id)
		}
	}
	cles := make([]string, 0, len(groupes))
	for g := range groupes {
		cles = append(cles, g)
	}
	sort.Slice(cles, func(i, j int) bool { return porteurs[cles[i]] > porteurs[cles[j]] })
	t.Logf("== GROUPES DE DEPENDANCES sur les %d `eqip` (porteurs / references) ==", len(tags))
	for _, g := range cles {
		t.Logf("  %-6s %3d tags sur %d  ·  %4d references",
			g, porteurs[g], len(tags), groupes[g])
	}

	// Les anonymes du rang 10 : les deux `eqip` du `sofa` 0xeb500815.
	for _, cible := range []uint32{0x4396db42, 0x4eebcb18} {
		p, ok := tags[cible]
		if !ok {
			t.Logf("  %08x ABSENT du groupe `eqip`", cible)
			continue
		}
		h := depDuGroupe(p.info.deps, "hlmt")
		var freres []string
		for _, autre := range parModele[h] {
			if autre != cible {
				freres = append(freres, fmt.Sprintf("%08x", autre))
			}
		}
		t.Logf("  %08x  hlmt=%s  freres de modele: [%s]  engendre [%s]",
			cible, h, strings.Join(freres, " "), hexIDs(p.info.eqip))
		t.Logf("      deps: %s", strings.Join(p.info.deps, " "))
	}

	// Les modeles PARTAGES : deux `eqip` qui portent le meme `hlmt` sont deux reglages du
	// meme objet. C'est la provenance `sofa_modele` du manifeste, ici sur tout le groupe.
	mods := make([]string, 0, len(parModele))
	for h, ids := range parModele {
		if len(ids) > 1 {
			mods = append(mods, h)
		}
	}
	sort.Strings(mods)
	t.Logf("== %d modeles `hlmt` partages par au moins deux `eqip` ==", len(mods))
	for _, h := range mods {
		ids := parModele[h]
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		t.Logf("  hlmt %s : %s", h, hexIDs(ids))
	}
}
