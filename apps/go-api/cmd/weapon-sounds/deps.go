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
