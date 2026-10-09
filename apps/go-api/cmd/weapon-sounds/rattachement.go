package main

// rattachement.go — relier une arme a son tag `weap` SANS passer par le son de tir.
//
// POURQUOI. Le nommage passait par la chaine du tir : une arme sans champ « Weapon Fire
// Sound » n'avait ni nom ni icone. C'etait une contrainte que je m'etais imposee sans
// raison — l'epee a energie, le marteau antigravite, les tourelles et les variantes PNJ
// sont de vraies armes, elles ONT une icone dans le jeu. Le rattachement doit donc reposer
// sur le seul lien `weap -> ... -> sbnk`, dont le tir n'est qu'un cas particulier.
//
// Methode : un tag de son declare la bank dont il depend. On construit donc
// `tag de son -> bank`, puis `weap -> tags de son (relais compris) -> banks`, et on inverse.

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/himodule"
)

// banksDesSons rend, pour chaque tag de son, la bank dont il depend.
func banksDesSons(m *himodule.Module) map[uint32]uint32 {
	out := map[uint32]uint32{}
	for _, g := range []string{"snd!", "lsnd"} {
		for _, f := range m.Files(g) {
			data, err := m.Extract(f)
			if err != nil {
				continue
			}
			for _, d := range dependances(data) {
				if d.Groupe == "sbnk" {
					out[f.GlobalID] = d.IDGlobal
					break
				}
			}
		}
	}
	return out
}

// weapsParBank rend, pour chaque bank, les `weap` qui l'atteignent par leurs sons.
//
// La profondeur est bornee : au-dela, on traverse des points de passage communs et tout se
// rattache a tout. Deux niveaux suffisent au cas `stai` observe (weap -> snd! -> stai -> snd!).
func weapsParBank(m *himodule.Module, profondeur int) map[uint32][]uint32 {
	bankDeSon := banksDesSons(m)
	parGid := make(map[uint32]himodule.File, 1<<15)
	for _, g := range []string{"snd!", "lsnd", "stai"} {
		for _, f := range m.Files(g) {
			parGid[f.GlobalID] = f
		}
	}

	out := map[uint32]map[uint32]bool{}
	for _, f := range m.Files("weap") {
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		atteints := map[uint32]bool{}
		frontiere := []uint32{}
		for _, d := range dependances(data) {
			if groupesSonores[d.Groupe] {
				atteints[d.IDGlobal] = true
				frontiere = append(frontiere, d.IDGlobal)
			}
		}
		for n := 0; n < profondeur && len(frontiere) > 0; n++ {
			var suivante []uint32
			for _, gid := range frontiere {
				fs, ok := parGid[gid]
				if !ok {
					continue
				}
				ds, err := m.Extract(fs)
				if err != nil {
					continue
				}
				for _, d := range dependances(ds) {
					if groupesSonores[d.Groupe] && !atteints[d.IDGlobal] {
						atteints[d.IDGlobal] = true
						suivante = append(suivante, d.IDGlobal)
					}
				}
			}
			frontiere = suivante
		}
		for son := range atteints {
			if bank, ok := bankDeSon[son]; ok {
				if out[bank] == nil {
					out[bank] = map[uint32]bool{}
				}
				out[bank][f.GlobalID] = true
			}
		}
	}

	// PLAFOND LARGE, ET C'EST VOULU. Un premier essai a 6 candidats ecartait l'epee a
	// energie, dont la bank est atteinte par 8 `weap` — trois qui sont bien l'epee, deux
	// qui sont un « skull » de Forge, trois inconnus. Trancher ICI reviendrait a deviner :
	// on rend donc TOUS les candidats, et c'est la couche de nommage (qui, elle, dispose de
	// l'index d'icones) qui retient celui qui resout vers une vraie entree produit.
	// Le plafond ne sert plus qu'a ecarter les banks vraiment universelles.
	const maxWeapParBank = 24
	final := map[uint32][]uint32{}
	for bank, s := range out {
		if len(s) > maxWeapParBank {
			continue
		}
		l := make([]uint32, 0, len(s))
		for w := range s {
			l = append(l, w)
		}
		sort.Slice(l, func(i, j int) bool { return l[i] < l[j] })
		final[bank] = l
	}
	return final
}

// completerRattachements ajoute, pour les armes sans son de tir, leurs `weap` candidats.
func completerRattachements(m *himodule.Module, p1 rapportLot, deja []tirArme) []tirArme {
	parBank := weapsParBank(m, 4)
	fmt.Printf("banks rattachees a au moins un weap : %d\n", len(parBank))

	couvertes := map[string]bool{}
	for _, t := range deja {
		couvertes[t.Arme] = true
	}
	ajouts := 0
	for _, a := range p1.Armes {
		if couvertes[a.Arme] {
			continue
		}
		weaps, ok := parBank[a.SbnkGlobal]
		if !ok || len(weaps) == 0 {
			continue
		}
		t := tirArme{
			Arme: a.Arme, Pck: a.Pck, WeapRetenu: fmt.Sprintf("%08x", weaps[0]),
			WemsDuPck: a.WemsDuPck,
		}
		for _, w := range weaps {
			t.WeapTags = append(t.WeapTags, fmt.Sprintf("%08x", w))
		}
		deja = append(deja, t)
		ajouts++
	}
	fmt.Printf("armes rattachees sans son de tir : %d\n", ajouts)
	sort.Slice(deja, func(i, j int) bool { return deja[i].Arme < deja[j].Arme })
	return deja
}
