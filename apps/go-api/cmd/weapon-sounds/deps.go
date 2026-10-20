package main

// deps.go — les dependances SORTANTES d'un tag donne.
//
// Complement du mode `qui`, qui repond « qui pointe vers ce tag ? ». Ici on demande
// l'inverse : « vers quoi ce tag pointe-t-il ? ». Utile pour descendre d'un `snd!` vers sa
// bank quand l'arme n'a pas ete rattachee par la chaine habituelle.

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/himodule"
)

// dependancesEnOrdre affiche les references sortantes d'un tag DANS L'ORDRE DU FICHIER, sans
// regroupement ni tri.
//
// POURQUOI L'ORDRE EST UNE DONNEE, et pourquoi le trier le detruit. La vue groupee ci-dessous
// trie par identifiant : pratique pour repondre « vers quoi ce tag pointe-t-il ? », inutile
// pour repondre « QUELLE PLACE occupe cette reference ? ». Or dans ce format le RANG est
// parfois la seule semantique disponible — precedent etabli : la liste `gggl` des grenades,
// dont « l'ordre EST le rang de type » (`replay_labels.toml`, deux chaines independantes).
// Une table de groupes sonores se lit de la meme facon : le jeu y designe un son par sa
// place, pas par un nom, puisque les noms ne survivent pas a la cuisson.
func dependancesEnOrdre(cheminModule string, gid uint32, limite int) error {
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")
	for _, f := range m.Files("") {
		if f.GlobalID != gid {
			continue
		}
		data, err := m.Extract(f)
		if err != nil {
			return err
		}
		deps := dependances(data)
		fmt.Printf("tag %08x (groupe '%s') : %d dependance(s), DANS L'ORDRE DU FICHIER\n\n",
			gid, f.Group, len(deps))
		for i, d := range deps {
			if limite > 0 && i >= limite {
				fmt.Printf("  ... et %d autres\n", len(deps)-limite)
				break
			}
			fmt.Printf("  %4d  %-5s %08x\n", i, d.Groupe, d.IDGlobal)
		}
		return nil
	}
	return fmt.Errorf("tag %08x absent de ce module", gid)
}

// dependancesDe affiche les references sortantes d'un tag, groupees par classe.
func dependancesDe(cheminModule string, gid uint32) error {
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")
	for _, f := range m.Files("") {
		if f.GlobalID != gid {
			continue
		}
		data, err := m.Extract(f)
		if err != nil {
			return err
		}
		deps := dependances(data)
		fmt.Printf("tag %08x (groupe '%s') : %d dependance(s)\n\n", gid, f.Group, len(deps))
		parGroupe := map[string][]uint32{}
		for _, d := range deps {
			parGroupe[d.Groupe] = append(parGroupe[d.Groupe], d.IDGlobal)
		}
		classes := make([]string, 0, len(parGroupe))
		for g := range parGroupe {
			classes = append(classes, g)
		}
		sort.Strings(classes)
		for _, g := range classes {
			ids := parGroupe[g]
			sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
			fmt.Printf("  '%s' : %d\n", g, len(ids))
			for i, id := range ids {
				if i >= 8 {
					fmt.Printf("      ... et %d autres\n", len(ids)-8)
					break
				}
				fmt.Printf("      %08x  (decimal %d)\n", id, id)
			}
		}
		return nil
	}
	return fmt.Errorf("tag %08x absent de ce module", gid)
}
