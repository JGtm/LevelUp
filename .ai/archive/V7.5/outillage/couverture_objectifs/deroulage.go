package main

// deroulage.go — L'HYPOTHESE `assists` DE L'AUDIT §10, INSTRUITE PAR L'ORACLE.
//
// AJOUT DU LOT 6.7-B1, ITEM 6 (2026-09-11). L'audit du 2026-09-10 laissait la question
// ouverte faute de colonne : « les 40 806 `assists` des 3 films sont-elles fausses ? — l'export
// n'a pas de colonne `assists` » (§13). L'export du 2026-09-10 22:30
// (`oracle_vague6_participants.tsv`) porte `kills`, `assists` et `score` par joueur : la
// question se tranche.
//
// La sortie confronte, par film et par joueur, le nombre d'ACTIONS publiees sous chaque
// `stat` du calque `objectives` au compteur correspondant de la feuille de match. Ces deux
// grandeurs sont comparables par construction : `named.go` nomme `comp 2 A` `kills` et
// `comp 3 A` `assists`, et [incrementTimes] emet UN evenement par unite gagnee — le total
// publie doit donc valoir le total de la feuille.

import (
	"fmt"
	"strings"
)

// multipliciteMax rend, par (stat, xuid), le plus grand nombre d'actions publiees a un MEME
// instant — c'est-a-dire le plus gros DEROULAGE D'UN PAS que l'artefact laisse voir.
//
// POURQUOI CETTE GRANDEUR SE LIT DANS L'ARTEFACT. `incrementTimes` emet une action par unite
// gagnee et les DATE TOUTES a l'instant du point qui les porte : un pas qui deroule n unites
// publie donc n actions au meme `timeMs`. La borne `maxUnrollPerStep` se cale sur cette
// grandeur, et l'artefact suffit a la mesurer — aucun film a decoder.
func multipliciteMax(a *artefact) map[string]map[string]int {
	type cle struct {
		stat, xuid string
		t          int
	}
	compte := map[cle]int{}
	for _, o := range a.Objectives {
		compte[cle{o.Stat, o.XUID, o.TimeMS}]++
	}
	out := map[string]map[string]int{}
	for k, n := range compte {
		if out[k.stat] == nil {
			out[k.stat] = map[string]int{}
		}
		if n > out[k.stat][k.xuid] {
			out[k.stat][k.xuid] = n
		}
	}
	return out
}

// deroulageSuivi : la stat publiee et la colonne de la feuille de match qui la borne.
var deroulageSuivi = [][2]string{
	{"kills", "kills"},
	{"assists", "assists"},
}

// axe6Deroulage rend, par film et par joueur, les actions publiees des compteurs de combat
// contre la feuille de match — la preuve d'entree de la borne de deroulage.
func (c *ctxAudit) axe6Deroulage() []string {
	l := []string{strings.Join([]string{"film", "famille", "mode", "action", "xuid", "gamertag",
		"publie", "oracle_feuille", "ecart", "depasse_oracle", "max_pas"}, "\t")}
	for _, f := range c.films {
		fam := famille(f.mode)
		if fam == "SansObjectif" || fam == "INCONNU" {
			continue
		}
		pub := map[string]map[string]int{}
		for _, o := range f.art.Objectives {
			if pub[o.Stat] == nil {
				pub[o.Stat] = map[string]int{}
			}
			pub[o.Stat][o.XUID]++
		}
		mult := multipliciteMax(f.art)
		for _, a := range deroulageSuivi {
			xs := map[string]bool{}
			for x := range pub[a[0]] {
				xs[x] = true
			}
			for x := range c.feuille[f.match] {
				xs[x] = true
			}
			for _, x := range triees(xs) {
				o := 0
				if r, ok := c.feuille[f.match][x]; ok {
					o, _ = c.pTSV.i(r, a[1])
				}
				p := pub[a[0]][x]
				l = append(l, strings.Join([]string{f.court, fam, f.mode, a[0], x,
					c.gt[f.match][x], fmt.Sprint(p), fmt.Sprint(o), fmt.Sprint(p - o),
					fmt.Sprint(p > o), fmt.Sprint(mult[a[0]][x])}, "\t"))
			}
		}
	}
	return l
}

// axe6Bilan resume, par film, ce que le deroulage publie en trop sur les compteurs de combat.
func (c *ctxAudit) axe6Bilan() []string {
	l := []string{strings.Join([]string{"film", "famille", "action", "publie_total",
		"oracle_total", "ratio", "joueurs_au_dessus", "joueurs", "pire_xuid", "pire_ecart"}, "\t")}
	for _, f := range c.films {
		fam := famille(f.mode)
		if fam == "SansObjectif" || fam == "INCONNU" {
			continue
		}
		pub := map[string]map[string]int{}
		for _, o := range f.art.Objectives {
			if pub[o.Stat] == nil {
				pub[o.Stat] = map[string]int{}
			}
			pub[o.Stat][o.XUID]++
		}
		for _, a := range deroulageSuivi {
			var tp, to, au, n, pire int
			pireX := ""
			xs := map[string]bool{}
			for x := range pub[a[0]] {
				xs[x] = true
			}
			for x := range c.feuille[f.match] {
				xs[x] = true
			}
			for _, x := range triees(xs) {
				o := 0
				if r, ok := c.feuille[f.match][x]; ok {
					o, _ = c.pTSV.i(r, a[1])
				}
				p := pub[a[0]][x]
				tp, to, n = tp+p, to+o, n+1
				if p > o {
					au++
					if p-o > pire {
						pire, pireX = p-o, x
					}
				}
			}
			if n == 0 {
				continue
			}
			ratio := 0.0
			if to > 0 {
				ratio = float64(tp) / float64(to)
			}
			l = append(l, strings.Join([]string{f.court, fam, a[0], fmt.Sprint(tp),
				fmt.Sprint(to), fmt.Sprintf("%.3f", ratio), fmt.Sprint(au), fmt.Sprint(n),
				pireX, fmt.Sprint(pire)}, "\t"))
		}
	}
	return l
}
