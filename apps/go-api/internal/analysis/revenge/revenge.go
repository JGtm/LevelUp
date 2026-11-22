// Package revenge compte les vengeances d'un match (médaille Vengeur).
//
// Règle : un frag de K sur V est une vengeance quand V est le tueur de la dernière mort de K. Une
// vengeance par mort : la mort est consommée par le premier frag qui la venge, un second frag de K
// sur V ne compte pas. Seule la dernière mort de K compte : une mort plus récente remplace la
// précédente, vengée ou non.
//
// Sont exclus : les frags sans tueur ou sans victime nommée (bot, joueur non résolu), les suicides et
// les trahisons (tueur et victime dans la même équipe connue). Une mort exclue n'est pas vengeable :
// elle efface la dernière mort de sa victime.
//
// Le paquet est pur : ni base, ni horloge, ni libellé.
package revenge

import "sort"

// Frag est un frag du fil des morts. TimeMS ordonne les frags ; à égalité, l'ordre de l'entrée.
type Frag struct {
	Killer string
	Victim string
	TimeMS int
}

// Count rend, par xuid de tueur, son nombre de vengeances. Seuls les joueurs qui ont au moins une
// vengeance figurent dans le résultat. teamOf donne l'équipe de chaque xuid ; un xuid absent est
// d'équipe inconnue, et son frag n'est pas écarté comme trahison. L'entrée n'est pas modifiée.
func Count(frags []Frag, teamOf map[string]int) map[string]int {
	ordered := append([]Frag(nil), frags...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].TimeMS < ordered[j].TimeMS })

	counts := map[string]int{}
	lastKiller := map[string]string{} // victime -> tueur de sa dernière mort vengeable
	for _, f := range ordered {
		valid := vengeable(f, teamOf)
		if valid && lastKiller[f.Killer] == f.Victim {
			counts[f.Killer]++
			delete(lastKiller, f.Killer) // la mort est consommée
		}
		if f.Victim == "" {
			continue
		}
		if valid {
			lastKiller[f.Victim] = f.Killer
		} else {
			delete(lastKiller, f.Victim)
		}
	}
	return counts
}

// vengeable dit si le frag est un frag entre deux joueurs nommés, adverses.
func vengeable(f Frag, teamOf map[string]int) bool {
	if f.Killer == "" || f.Victim == "" || f.Killer == f.Victim {
		return false
	}
	tk, okK := teamOf[f.Killer]
	tv, okV := teamOf[f.Victim]
	return !okK || !okV || tk != tv || tk < 0
}
